package reply

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	intentService "github.com/rockship/cosmo-agents-go/internal/service/intent"
)

const (
	TypeHandleReply = "email:handle_reply"
)

// HandleReplyPayload is the payload for handle reply task
type HandleReplyPayload struct {
	EmailID uuid.UUID `json:"email_id"`
}

// Worker processes email replies with intent classification
type Worker struct {
	emailRepo        emailRepository
	conversationRepo conversationRepository
	campaignRepo     campaignRepository
	intentClassifier intentClassifier
	handlerFactory   intentHandlerFactory
	logger           *zerolog.Logger
}

type emailRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Email, error)
	Update(ctx context.Context, id uuid.UUID, email *domain.Email) error
}

type conversationRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error)
	Update(ctx context.Context, id uuid.UUID, conversation *domain.Conversation) error
}

type campaignRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error)
}

type intentClassifier interface {
	Classify(ctx context.Context, content string) (domain.IntentType, error)
}

type intentHandlerFactory interface {
	Build(campaign *domain.Campaign, intent domain.IntentType) (intentService.IntentHandler, error)
}

// New creates a new handle reply worker
func New(
	emailRepo emailRepository,
	conversationRepo conversationRepository,
	campaignRepo campaignRepository,
	intentClassifier intentClassifier,
	handlerFactory intentHandlerFactory,
	log *zerolog.Logger,
) *Worker {
	return &Worker{
		emailRepo:        emailRepo,
		conversationRepo: conversationRepo,
		campaignRepo:     campaignRepo,
		intentClassifier: intentClassifier,
		handlerFactory:   handlerFactory,
		logger:           log,
	}
}

// ProcessTask processes a handle reply task
func (w *Worker) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload HandleReplyPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		w.logger.Error().Err(err).Msg("Failed to unmarshal handle reply payload")
		return fmt.Errorf("json.Unmarshal failed: %w", err)
	}

	w.logger.Info().
		Str("email_id", payload.EmailID.String()).
		Msg("Processing handle reply task")

	// Fetch the email
	email, err := w.emailRepo.FindByID(ctx, payload.EmailID)
	if err != nil {
		w.logger.Warn().
			Str("email_id", payload.EmailID.String()).
			Msg("Reply email not found")
		return fmt.Errorf("email not found: %w", err)
	}

	// Fetch the conversation
	conversation, err := w.conversationRepo.FindByID(ctx, *email.ConversationID)
	if err != nil {
		w.logger.Error().Err(err).Msg("Conversation not found")
		return fmt.Errorf("conversation not found: %w", err)
	}

	// Check if conversation was already handled
	if conversation.Replied {
		w.logger.Info().
			Str("email_id", payload.EmailID.String()).
			Str("conversation_id", conversation.ID.String()).
			Msg("Conversation was already handled by the campaign. Skipped.")
		return nil
	}

	// Fetch the campaign
	campaign, err := w.campaignRepo.FindByID(ctx, *email.CampaignID)
	if err != nil {
		w.logger.Error().Err(err).Msg("Campaign not found")
		return fmt.Errorf("campaign not found: %w", err)
	}

	// Check if campaign is active
	if campaign.Status != domain.CampaignStatusActive {
		w.logger.Info().
			Str("campaign_id", campaign.ID.String()).
			Str("email_id", email.ID.String()).
			Msg("Campaign is not active. Skipped")
		return nil
	}

	// Classify intent using AI
	intent, err := w.intentClassifier.Classify(ctx, email.Content)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to classify intent")
		return fmt.Errorf("intent classification failed: %w", err)
	}

	w.logger.Info().
		Str("email_id", email.ID.String()).
		Str("intent", string(intent)).
		Msg("Classified intent")

	// Update email with intent (convert to StringArray)
	email.Intents = []string{string(intent)}
	if err := w.emailRepo.Update(ctx, email.ID, email); err != nil {
		w.logger.Error().Err(err).Msg("Failed to update email intent")
		return fmt.Errorf("failed to update email: %w", err)
	}

	// Update conversation with intent (convert to StringArray)
	conversation.Intents = []string{string(intent)}
	if err := w.conversationRepo.Update(ctx, conversation.ID, conversation); err != nil {
		w.logger.Error().Err(err).Msg("Failed to update conversation intent")
		return fmt.Errorf("failed to update conversation: %w", err)
	}

	// Build the appropriate handler
	handler, err := w.handlerFactory.Build(campaign, intent)
	if err != nil {
		w.logger.Error().Err(err).Msg("Failed to build intent handler")
		return fmt.Errorf("failed to build handler: %w", err)
	}

	if handler == nil {
		w.logger.Error().
			Str("campaign_id", campaign.ID.String()).
			Str("intent", string(intent)).
			Msg("Failed to resolve handler")
		return fmt.Errorf("no handler available for intent: %s", intent)
	}

	// Execute the handler
	success, err := handler.Execute(ctx, campaign, intent, email)
	if err != nil {
		w.logger.Error().
			Err(err).
			Str("intent", string(intent)).
			Str("email_id", email.ID.String()).
			Msg("Failed to execute intent handler")
		return fmt.Errorf("handler execution failed: %w", err)
	}

	if !success {
		w.logger.Warn().
			Str("intent", string(intent)).
			Str("email_id", email.ID.String()).
			Msg("Intent handler execution was not successful")
		return fmt.Errorf("handler execution unsuccessful")
	}

	w.logger.Info().
		Str("intent", string(intent)).
		Str("email_id", email.ID.String()).
		Msg("Contact email handled successfully with intent handler")

	return nil
}

// NewHandleReplyTask creates a new handle reply task
func NewHandleReplyTask(emailID uuid.UUID) (*asynq.Task, error) {
	payload, err := json.Marshal(HandleReplyPayload{EmailID: emailID})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeHandleReply, payload), nil
}
