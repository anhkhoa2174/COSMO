package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	taskRepo "github.com/rockship/cosmo-agents-go/internal/repository/task"
	workerpayloads "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	queueworker "github.com/rockship/cosmo-agents-go/pkg/worker"
)

// extractEmailFromProfile extracts email from contact's profile JSONB
func extractEmailFromProfile(profile domain.JSONB) string {
	if len(profile) == 0 {
		return ""
	}
	var profileData map[string]interface{}
	if err := json.Unmarshal(profile, &profileData); err != nil {
		return ""
	}
	if email, ok := profileData["email"].(string); ok {
		return email
	}
	return ""
}

type campaignFinder interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error)
}

type contactFinder interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Contact, error)
}

type templateFinder interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Template, error)
}

type knowledgeFinder interface {
	FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.Knowledge], error)
}

// Worker handles AI-related background tasks.
type Worker struct {
	db            *gorm.DB
	workerClient  queueClient
	openaiClient  openaiEmailClient
	campaignRepo  campaignFinder
	contactRepo   contactFinder
	templateRepo  templateFinder
	taskRepo      *taskRepo.TaskRepository
	knowledgeRepo knowledgeFinder
}

type openaiEmailClient interface {
	GenerateEmailContent(ctx context.Context, params ai.EmailGenerationParams) (*ai.EmailContent, error)
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

type queueClient interface {
	EnqueueTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error)
	EnqueueTaskAt(ctx context.Context, taskType string, payload interface{}, processAt time.Time, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// New creates a new AI worker.
func New(
	db *gorm.DB,
	workerClient queueClient,
	openaiClient openaiEmailClient,
	campaignRepo campaignFinder,
	contactRepo contactFinder,
	templateRepo templateFinder,
	taskRepo *taskRepo.TaskRepository,
	knowledgeRepo knowledgeFinder,
) *Worker {
	return &Worker{
		db:            db,
		workerClient:  workerClient,
		openaiClient:  openaiClient,
		campaignRepo:  campaignRepo,
		contactRepo:   contactRepo,
		templateRepo:  templateRepo,
		taskRepo:      taskRepo,
		knowledgeRepo: knowledgeRepo,
	}
}

// HandleGenerateEmail generates personalized email content using AI.
func (w *Worker) HandleGenerateEmail(ctx context.Context, task *asynq.Task) error {
	var payload workerpayloads.GenerateEmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	logger.Logger.Info().
		Str("campaign_id", payload.CampaignID.String()).
		Str("contact_id", payload.ContactID.String()).
		Str("template_id", payload.TemplateID.String()).
		Msg("Generating AI email content")

	// Get campaign
	campaign, err := w.campaignRepo.FindByID(ctx, payload.CampaignID)
	if err != nil {
		return fmt.Errorf("campaign not found: %w", err)
	}

	// Get contact
	contact, err := w.contactRepo.FindByID(ctx, payload.ContactID)
	if err != nil {
		return fmt.Errorf("contact not found: %w", err)
	}

	// Get template
	template, err := w.templateRepo.FindByID(ctx, payload.TemplateID)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Get knowledge base context (RAG)
	knowledgeContext := ""
	if campaign.UserID != uuid.Nil {
		knowledgeContext, err = w.getKnowledgeContext(ctx, campaign.UserID, template.Content)
		if err != nil {
			logger.Logger.Warn().Err(err).Msg("Failed to get knowledge context, continuing without it")
		}
	}

	// Extract email from profile
	contactEmail := extractEmailFromProfile(contact.Profile)

	// Build generation parameters
	params := ai.EmailGenerationParams{
		ContactName:       contact.Name,
		ContactEmail:      contactEmail,
		ContactCompany:    contact.Company,
		ContactTitle:      contact.JobTitle,
		TemplateSubject:   template.Subject,
		TemplateContent:   template.Content,
		CampaignGoal:      string(campaign.Status), // Could be improved with campaign.description
		AdditionalContext: knowledgeContext,
		SystemPrompt: `You are an expert email writer for B2B sales outreach.
Your goal is to write personalized, engaging emails that get responses.
Keep the tone professional yet friendly, and always include a clear call-to-action.`,
	}

	// Generate email content
	emailContent, err := w.openaiClient.GenerateEmailContent(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to generate email: %w", err)
	}
	if emailContent == nil {
		return fmt.Errorf("generated email content is nil")
	}

	logger.Logger.Info().
		Str("contact_id", contact.ID.String()).
		Int("body_length", len(emailContent.Body)).
		Msg("Email content generated successfully")

	// Get campaign agent
	agentID := campaign.AgentID
	if agentID == nil {
		return fmt.Errorf("campaign has no agent assigned")
	}

	// Enqueue send email task with generated content
	sendPayload := workerpayloads.SendEmailPayload{
		AgentID:    *agentID,
		ContactID:  contact.ID,
		CampaignID: &campaign.ID,
		TemplateID: &template.ID,
		TaskID:     payload.TaskID,
		To:         contactEmail,
		Subject:    emailContent.Subject,
		Body:       emailContent.Body,
		IsHTML:     true,
	}

	if _, err := w.workerClient.EnqueueTask(ctx, queueworker.TypeSendEmail, sendPayload); err != nil {
		return fmt.Errorf("failed to enqueue send email: %w", err)
	}

	logger.Logger.Info().
		Str("contact_email", contactEmail).
		Msg("Send email task enqueued with AI-generated content")

	return nil
}

// HandleGenerateEmbedding generates embeddings for knowledge base entries (simplified).
func (w *Worker) HandleGenerateEmbedding(ctx context.Context, task *asynq.Task) error {
	var payload workerpayloads.GenerateEmbeddingPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	logger.Logger.Info().
		Str("knowledge_id", payload.KnowledgeID.String()).
		Msg("Generating embedding")

	// Generate embedding
	embedding, err := w.openaiClient.GenerateEmbedding(ctx, payload.Text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// TODO: Store embedding in vector database
	// For now just log success
	logger.Logger.Info().
		Str("knowledge_id", payload.KnowledgeID.String()).
		Int("embedding_dimensions", len(embedding)).
		Msg("Embedding generated successfully (storage not implemented)")

	return nil
}

// getKnowledgeContext retrieves relevant knowledge base context (simplified).
func (w *Worker) getKnowledgeContext(ctx context.Context, userID uuid.UUID, query string) (string, error) {
	// For now, just get recent knowledge entries without semantic search
	filter := baseRepo.Filter{
		"user_id": userID,
	}

	result, err := w.knowledgeRepo.FindAll(ctx, filter, &baseRepo.PaginationParams{Offset: 0, Limit: 3})
	if err != nil {
		return "", err
	}

	// Combine results into context
	context := ""
	for i, k := range result.List {
		// Use summary_pair if available
		summary := "No summary available"
		if len(k.SummaryPair) > 0 {
			summary = k.SummaryPair[0]
		}
		context += fmt.Sprintf("\n\n--- Knowledge %d ---\n%s", i+1, summary)
	}

	return context, nil
}
