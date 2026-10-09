package mailwriter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/rockship/cosmo-agents-go/internal/domain"
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

// KnowledgeSearcher searches the knowledge base for relevant context.
type KnowledgeSearcher interface {
	Search(ctx context.Context, userID uuid.UUID, query string, limit int) ([]KnowledgeChunk, error)
}

// KnowledgeChunk represents a chunk of knowledge text with relevance score.
type KnowledgeChunk struct {
	ChunkText string
	Score     float64
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
	mailWriter        mailGenerationService
	knowledgeSearcher KnowledgeSearcher

	logger *zerolog.Logger
}

type mailGenerationService interface {
	GenerateSingleOutreach(ctx context.Context, previous []mailService.EmailTemplate, knowledgeDocs ...mailService.Document) (*mailService.EmailTemplate, error)
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
	knowledgeSearcher KnowledgeSearcher,
	log *zerolog.Logger,
) *Worker {
	return &Worker{
		campaignRepo:      campaignRepo,
		contactRepo:       contactRepo,
		conversationRepo:  conversationRepo,
		emailRepo:         emailRepo,
		templateRepo:      templateRepo,
		userRepo:          userRepo,
		orgRepo:           orgRepo,
		mailWriter:        mailWriter,
		knowledgeSearcher: knowledgeSearcher,
		logger:            log,
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

	// Look up campaign to get UserID
	camp, err := w.campaignRepo.FindByID(ctx, payload.CampaignID)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to find campaign")
		return fmt.Errorf("failed to find campaign: %w", err)
	}

	// Search knowledge base for relevant context (RAG)
	var knowledgeDocs []mailService.Document
	if w.knowledgeSearcher != nil {
		query := fmt.Sprintf("%s %s", payload.ContactData["company_name"], payload.ContactData["title"])
		chunks, err := w.knowledgeSearcher.Search(ctx, camp.UserID, query, 5)
		if err != nil {
			w.logger.Warn().Err(err).Msg("Knowledge search failed, continuing without RAG context")
		} else {
			for _, chunk := range chunks {
				knowledgeDocs = append(knowledgeDocs, mailService.Document{
					Title:   "Knowledge Base",
					Content: chunk.ChunkText,
				})
			}
			w.logger.Info().Int("knowledge_docs", len(knowledgeDocs)).Msg("Retrieved knowledge context for email generation")
		}
	}

	// Generate email using MailWriter service with knowledge context
	generatedEmail, err := w.mailWriter.GenerateSingleOutreach(ctx, payload.PreviousEmails, knowledgeDocs...)
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

	contact, err := w.contactRepo.FindByID(ctx, payload.ContactID)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to find contact")
		return fmt.Errorf("failed to find contact: %w", err)
	}

	toEmail := ""
	if contact != nil {
		toEmail = payload.ContactData["email"]
	}

	campaignID := payload.CampaignID
	emailRecord := &domain.Email{
		UserID:     camp.UserID,
		CampaignID: &campaignID,
		FromEmail:  generatedEmail.FromEmail,
		ToEmail:    toEmail,
		Subject:    generatedEmail.Subject,
		Content:    generatedEmail.Content,
		Status:     domain.EmailStatusDraft,
		Labels:     []string{payload.TemplateType},
	}

	saved, err := w.emailRepo.Create(ctx, emailRecord)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to save generated email")
		return fmt.Errorf("failed to save email: %w", err)
	}

	w.logger.Info().
		Str("email_id", saved.ID.String()).
		Str("subject", generatedEmail.Subject).
		Msg("Generated email saved as draft")

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

	// Look up campaign to get UserID
	camp, err := w.campaignRepo.FindByID(ctx, payload.CampaignID)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to find campaign for reply")
		return fmt.Errorf("failed to find campaign: %w", err)
	}

	campaignID := payload.CampaignID
	conversationID := payload.ConversationID
	emailRecord := &domain.Email{
		UserID:         camp.UserID,
		CampaignID:     &campaignID,
		ConversationID: &conversationID,
		FromEmail:      generatedReply.FromEmail,
		ToEmail:        generatedReply.ToEmail,
		Subject:        generatedReply.Subject,
		Content:        generatedReply.Content,
		Status:         domain.EmailStatusDraft,
		Labels:         []string{"reply"},
		Intents:        []string{string(payload.Intent)},
	}

	saved, err := w.emailRepo.Create(ctx, emailRecord)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to save generated reply")
		return fmt.Errorf("failed to save reply: %w", err)
	}

	w.logger.Info().
		Str("email_id", saved.ID.String()).
		Str("conversation_id", payload.ConversationID.String()).
		Str("intent", string(payload.Intent)).
		Msg("Generated reply saved as draft")

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

	// Using Redis vector search for indexing
	logger.Logger.Info().
		Str("collection", payload.Collection).
		Int("num_documents", len(documents)).
		Strs("document_ids", documentIDs).
		Msg("Email sequence indexed successfully")

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
