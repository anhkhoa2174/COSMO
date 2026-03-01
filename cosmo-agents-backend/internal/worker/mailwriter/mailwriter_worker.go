package mailwriter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	"github.com/rockship/cosmo-agents-go/internal/repository/organization"
	templateRepo "github.com/rockship/cosmo-agents-go/internal/repository/template"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	mailService "github.com/rockship/cosmo-agents-go/internal/service/mail"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

const (
	// Campaign email generation tasks
	TypeGenerateCampaignEmail = "campaign:generate_campaign_email"
	TypeGenerateReplyEmail    = "campaign:generate_reply"

	// Email indexing tasks (RAG)
	TypeEmailIndexing = "email:indexing"
)

// GenerateCampaignEmailPayload is the payload for campaign email generation
type GenerateCampaignEmailPayload struct {
	CampaignID     uuid.UUID                   `json:"campaign_id"`
	ContactID      uuid.UUID                   `json:"contact_id"`
	TemplateType   string                      `json:"template_type"` // "first", "followup1", "followup2"
	ContactData    map[string]string           `json:"contact_data"`
	PreviousEmails []mailService.EmailTemplate `json:"previous_emails,omitempty"`
}

// GenerateReplyEmailPayload is the payload for reply email generation
type GenerateReplyEmailPayload struct {
	CampaignID     uuid.UUID                   `json:"campaign_id"`
	ConversationID uuid.UUID                   `json:"conversation_id"`
	Intent         mailService.IntentType      `json:"intent"`
	Conversation   []mailService.EmailTemplate `json:"conversation"`
}

// EmailIndexingPayload represents the payload for email indexing task
type EmailIndexingPayload struct {
	Collection        string                 `json:"collection"`
	SequenceEmailsStr string                 `json:"sequence_emails_str"`
	EmbeddingGID      string                 `json:"embedding_gid"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// Worker processes email generation and indexing tasks
type Worker struct {
	// Repositories
	campaignRepo     *campaignRepo.CampaignRepository
	contactRepo      *contactRepo.ContactRepository
	conversationRepo *conversationRepo.ConversationRepository
	emailRepo        *emailRepo.Repository
	templateRepo     *templateRepo.TemplateRepository
	userRepo         *userRepo.UserRepository
	orgRepo          *organization.OrganizationRepository

	// Services
	mailWriter mailGenerationService

	logger *zerolog.Logger
}

type mailGenerationService interface {
	GenerateSingleOutreach(ctx context.Context, previous []mailService.EmailTemplate) (*mailService.EmailTemplate, error)
	GenerateReply(ctx context.Context, conversation []mailService.EmailTemplate, intent mailService.IntentType) (*mailService.EmailTemplate, error)
}

// New creates a new mail writer worker
func New(
	campaignRepo *campaignRepo.CampaignRepository,
	contactRepo *contactRepo.ContactRepository,
	conversationRepo *conversationRepo.ConversationRepository,
	emailRepo *emailRepo.Repository,
	templateRepo *templateRepo.TemplateRepository,
	userRepo *userRepo.UserRepository,
	orgRepo *organization.OrganizationRepository,
	mailWriter mailGenerationService,
	log *zerolog.Logger,
) *Worker {
	return &Worker{
		campaignRepo:     campaignRepo,
		contactRepo:      contactRepo,
		conversationRepo: conversationRepo,
		emailRepo:        emailRepo,
		templateRepo:     templateRepo,
		userRepo:         userRepo,
		orgRepo:          orgRepo,
		mailWriter:       mailWriter,
		logger:           log,
	}
}

// ProcessTask processes an email generation or indexing task
func (w *Worker) ProcessTask(ctx context.Context, task *asynq.Task) error {
	switch task.Type() {
	case TypeGenerateCampaignEmail:
		return w.processGenerateCampaignEmail(ctx, task)
	case TypeGenerateReplyEmail:
		return w.processGenerateReplyEmail(ctx, task)
	case TypeEmailIndexing:
		return w.handleEmailIndexing(ctx, task)
	default:
		return fmt.Errorf("unknown task type: %s", task.Type())
	}
}

// processGenerateCampaignEmail generates a campaign outreach email
func (w *Worker) processGenerateCampaignEmail(ctx context.Context, task *asynq.Task) error {
	var payload GenerateCampaignEmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.logger.Error().Err(err).Msg("Failed to unmarshal generate campaign email payload")
		return fmt.Errorf("json.Unmarshal failed: %w", err)
	}

	w.logger.Info().
		Str("campaign_id", payload.CampaignID.String()).
		Str("contact_id", payload.ContactID.String()).
		Str("template_type", string(payload.TemplateType)).
		Msg("Processing campaign email generation task")

	// Generate email using MailWriter service
	// previousEmails will be from payload (can be nil for first email)
	generatedEmail, err := w.mailWriter.GenerateSingleOutreach(ctx, payload.PreviousEmails)
	if err != nil {
		w.logger.Error().
			Err(err).
			Str("campaign_id", payload.CampaignID.String()).
			Msg("Failed to generate email")
		return fmt.Errorf("email generation failed: %w", err)
	}

	w.logger.Info().
		Str("campaign_id", payload.CampaignID.String()).
		Str("contact_id", payload.ContactID.String()).
		Str("subject", generatedEmail.Subject).
		Int("content_length", len(generatedEmail.Content)).
		Msg("Campaign email generated successfully")

	// TODO: Save generated email to database
	// For now, we'll just log the result
	w.logger.Debug().
		Str("subject", generatedEmail.Subject).
		Str("content", generatedEmail.Content).
		Msg("Generated email content")

	return nil
}

// processGenerateReplyEmail generates a reply based on intent
func (w *Worker) processGenerateReplyEmail(ctx context.Context, task *asynq.Task) error {
	var payload GenerateReplyEmailPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.logger.Error().Err(err).Msg("Failed to unmarshal generate reply email payload")
		return fmt.Errorf("json.Unmarshal failed: %w", err)
	}

	w.logger.Info().
		Str("campaign_id", payload.CampaignID.String()).
		Str("conversation_id", payload.ConversationID.String()).
		Str("intent", string(payload.Intent)).
		Msg("Processing reply email generation task")

	// Generate reply using MailWriter service
	// Conversation emails should be provided in payload
	generatedReply, err := w.mailWriter.GenerateReply(ctx, payload.Conversation, payload.Intent)
	if err != nil {
		w.logger.Error().
			Err(err).
			Str("intent", string(payload.Intent)).
			Msg("Failed to generate reply")
		return fmt.Errorf("reply generation failed: %w", err)
	}

	w.logger.Info().
		Str("conversation_id", payload.ConversationID.String()).
		Str("intent", string(payload.Intent)).
		Str("subject", generatedReply.Subject).
		Int("content_length", len(generatedReply.Content)).
		Msg("Reply email generated successfully")

	// TODO: Save generated reply or create draft
	// For now, we'll just log it
	w.logger.Debug().
		Str("subject", generatedReply.Subject).
		Str("content", generatedReply.Content).
		Msg("Generated reply content")

	return nil
}

// handleEmailIndexing indexes email sequences in vector store for RAG
// This stores email templates for later retrieval and similarity search
func (w *Worker) handleEmailIndexing(ctx context.Context, task *asynq.Task) error {
	var payload EmailIndexingPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	logger.Logger.Info().
		Str("collection", payload.Collection).
		Str("embedding_gid", payload.EmbeddingGID).
		Msg("Indexing email sequence")

	// Validate input
	if payload.SequenceEmailsStr == "" {
		return fmt.Errorf("sequence of emails is empty")
	}

	// Prepare documents and metadata
	documents := []string{payload.SequenceEmailsStr}

	// Create metadata for each document
	metadatas := make([]map[string]interface{}, len(documents))
	documentIDs := make([]string, len(documents))

	for idx := range documents {
		// Combine embedding_gid with index
		docID := fmt.Sprintf("%s_%d", payload.EmbeddingGID, idx)
		documentIDs[idx] = docID

		// Merge provided metadata with ID
		metadatas[idx] = make(map[string]interface{})
		metadatas[idx]["id"] = docID
		for k, v := range payload.Metadata {
			metadatas[idx][k] = v
		}
	}

	// TODO: Integrate with actual vector store (Qdrant, Weaviate, or Pinecone)
	// For now, just log the operation
	logger.Logger.Info().
		Str("collection", payload.Collection).
		Int("num_documents", len(documents)).
		Strs("document_ids", documentIDs).
		Msg("Email sequence indexed successfully")

	// Production implementation would be:
	// err := w.vectorStore.AddDocuments(ctx, payload.Collection, documents, metadatas, documentIDs)
	// if err != nil {
	//     return fmt.Errorf("failed to index documents: %w", err)
	// }

	return nil
}

// NewGenerateCampaignEmailTask creates a new campaign email generation task
func NewGenerateCampaignEmailTask(
	campaignID, contactID uuid.UUID,
	templateType string,
	contactData map[string]string,
	previousEmails []mailService.EmailTemplate,
) (*asynq.Task, error) {
	payload, err := json.Marshal(GenerateCampaignEmailPayload{
		CampaignID:     campaignID,
		ContactID:      contactID,
		TemplateType:   templateType,
		ContactData:    contactData,
		PreviousEmails: previousEmails,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeGenerateCampaignEmail, payload), nil
}

// NewGenerateReplyEmailTask creates a new reply email generation task
func NewGenerateReplyEmailTask(
	campaignID, conversationID uuid.UUID,
	intent mailService.IntentType,
	conversation []mailService.EmailTemplate,
) (*asynq.Task, error) {
	payload, err := json.Marshal(GenerateReplyEmailPayload{
		CampaignID:     campaignID,
		ConversationID: conversationID,
		Intent:         intent,
		Conversation:   conversation,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeGenerateReplyEmail, payload), nil
}

// EnqueueEmailIndexing enqueues an email indexing task
func EnqueueEmailIndexing(client *asynq.Client, payload EmailIndexingPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TypeEmailIndexing, data)

	info, err := client.Enqueue(task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	logger.Logger.Info().
		Str("task_id", info.ID).
		Str("queue", info.Queue).
		Msg("Email indexing task enqueued")

	return nil
}
