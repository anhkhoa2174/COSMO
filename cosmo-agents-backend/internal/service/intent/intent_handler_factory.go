package intent

import (
	"github.com/rockship/cosmo-agents-go/internal/domain"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	knowledgeRepo "github.com/rockship/cosmo-agents-go/internal/repository/knowledge"
	saleRepRepo "github.com/rockship/cosmo-agents-go/internal/repository/sale_rep"
	taskRepo "github.com/rockship/cosmo-agents-go/internal/repository/task"
	template "github.com/rockship/cosmo-agents-go/internal/repository/template"
	"github.com/rs/zerolog"

	"github.com/google/uuid"
)

// IntentHandlerFactory creates appropriate intent handlers based on campaign and intent
type IntentHandlerFactory struct {
	// Repositories
	emailRepo        *emailRepo.Repository
	contactRepo      *contactRepo.ContactRepository
	conversationRepo *conversationRepo.ConversationRepository
	agentRepo        *agentRepo.AgentRepository
	saleRepRepo      *saleRepRepo.SaleRepRepository
	knowledgeRepo    *knowledgeRepo.KnowledgeRepository
	taskRepo         *taskRepo.TaskRepository
	templateRepo     *template.TemplateRepository

	logger *zerolog.Logger
}

// NewIntentHandlerFactory creates a new intent handler factory
func NewIntentHandlerFactory(
	emailRepo *emailRepo.Repository,
	contactRepo *contactRepo.ContactRepository,
	conversationRepo *conversationRepo.ConversationRepository,
	agentRepo *agentRepo.AgentRepository,
	saleRepRepo *saleRepRepo.SaleRepRepository,
	knowledgeRepo *knowledgeRepo.KnowledgeRepository,
	taskRepo *taskRepo.TaskRepository,
	templateRepo *template.TemplateRepository,
	log *zerolog.Logger,
) *IntentHandlerFactory {
	return &IntentHandlerFactory{
		emailRepo:        emailRepo,
		contactRepo:      contactRepo,
		conversationRepo: conversationRepo,
		agentRepo:        agentRepo,
		saleRepRepo:      saleRepRepo,
		knowledgeRepo:    knowledgeRepo,
		taskRepo:         taskRepo,
		templateRepo:     templateRepo,
		logger:           log,
	}
}

// Build creates the appropriate intent handler based on campaign configuration and intent
func (f *IntentHandlerFactory) Build(campaign *domain.Campaign, intent domain.IntentType) (IntentHandler, error) {
	// Priority handlers that don't depend on campaign config
	switch intent {
	case domain.IntentDoNotContact:
		return NewDoNotContactHandler(
			f.emailRepo,
			f.contactRepo,
			f.conversationRepo,
			f.logger,
		), nil

	case domain.IntentOutOfOffice:
		return NewOutOfOfficeHandler(
			f.emailRepo,
			f.contactRepo,
			f.conversationRepo,
			f.logger,
		), nil

	case domain.IntentUnknown:
		// For unknown intent, assign to campaign owner
		return NewAssignToPersonHandler(
			f.conversationRepo,
			&campaign.UserID,
			f.logger,
		), nil
	}

	// For other intents, check campaign configuration
	cfg := campaign.GetIntentAssignee(intent)
	if cfg == nil {
		f.logger.Info().
			Str("intent", string(intent)).
			Str("campaign_id", campaign.ID.String()).
			Msg("Campaign doesn't have handler configured for intent")
		return nil, nil
	}

	// Route based on handler type
	switch cfg.Who {
	case domain.HandlerAI:
		return NewAIReplyHandler(
			f.emailRepo,
			f.contactRepo,
			f.conversationRepo,
			f.agentRepo,
			f.knowledgeRepo,
			f.saleRepRepo,
			f.logger,
		), nil

	case domain.HandlerDraft:
		return NewDraftReplyHandler(
			f.emailRepo,
			f.contactRepo,
			f.conversationRepo,
			f.logger,
		), nil

	case domain.HandlerHuman:
		// Extract user ID from payload
		payload, ok := cfg.Payload.(map[string]interface{})
		if !ok {
			f.logger.Warn().Msg("Invalid payload in handler config")
			return nil, nil
		}

		userIDStr, ok := payload["user_id"].(string)
		if !ok {
			f.logger.Warn().Msg("Missing user_id in payload")
			return nil, nil
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			f.logger.Error().Err(err).Msg("Invalid user_id in payload")
			return nil, err
		}

		return NewAssignToPersonHandler(
			f.conversationRepo,
			&userID,
			f.logger,
		), nil

	default:
		f.logger.Warn().
			Str("handler_type", string(cfg.Who)).
			Str("intent", string(intent)).
			Msg("Unknown handler type")
		return nil, nil
	}
}
