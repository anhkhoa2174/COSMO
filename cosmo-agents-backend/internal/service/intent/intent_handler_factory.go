package intent

import (
	"context"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	knowledgeRepo "github.com/rockship/cosmo-agents-go/internal/repository/knowledge"
	saleRepRepo "github.com/rockship/cosmo-agents-go/internal/repository/sale_rep"
	taskRepo "github.com/rockship/cosmo-agents-go/internal/repository/task"
	template "github.com/rockship/cosmo-agents-go/internal/repository/template"
	cozeService "github.com/rockship/cosmo-agents-go/internal/service/coze"
	"github.com/rockship/cosmo-agents-go/internal/skills"
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
	knowledgeSearch  *skills.KnowledgeSearchSkill
	taskRepo         *taskRepo.TaskRepository
	templateRepo     *template.TemplateRepository
	cozeSvc          *cozeService.CozeService

	// orgSettings is handed to AI reply handlers so drafts follow the
	// organisation's per-intent guidance. Optional.
	orgSettings func(ctx context.Context, userID uuid.UUID) ([]byte, error)

	logger *zerolog.Logger
}

// WithOrgSettings makes AI-written replies follow the organisation's guidance.
func (f *IntentHandlerFactory) WithOrgSettings(load func(ctx context.Context, userID uuid.UUID) ([]byte, error)) *IntentHandlerFactory {
	f.orgSettings = load
	return f
}

// NewIntentHandlerFactory creates a new intent handler factory
func NewIntentHandlerFactory(
	emailRepo *emailRepo.Repository,
	contactRepo *contactRepo.ContactRepository,
	conversationRepo *conversationRepo.ConversationRepository,
	agentRepo *agentRepo.AgentRepository,
	saleRepRepo *saleRepRepo.SaleRepRepository,
	knowledgeRepo *knowledgeRepo.KnowledgeRepository,
	knowledgeSearch *skills.KnowledgeSearchSkill,
	taskRepo *taskRepo.TaskRepository,
	templateRepo *template.TemplateRepository,
	cozeSvc *cozeService.CozeService,
	log *zerolog.Logger,
) *IntentHandlerFactory {
	return &IntentHandlerFactory{
		emailRepo:        emailRepo,
		contactRepo:      contactRepo,
		conversationRepo: conversationRepo,
		agentRepo:        agentRepo,
		saleRepRepo:      saleRepRepo,
		knowledgeRepo:    knowledgeRepo,
		knowledgeSearch:  knowledgeSearch,
		taskRepo:         taskRepo,
		templateRepo:     templateRepo,
		cozeSvc:          cozeSvc,
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
		// Không có cấu hình riêng thì rơi về route mặc định theo taxonomy,
		// thay vì bỏ rơi reply (nil handler làm asynq retry vô ích 25 lần).
		f.logger.Info().
			Str("intent", string(intent)).
			Str("campaign_id", campaign.ID.String()).
			Msg("No handler configured for intent; using default route")
		return f.defaultHandler(campaign, intent), nil
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
			f.knowledgeSearch,
			f.saleRepRepo,
			f.cozeSvc,
			f.logger,
		).WithOrgSettings(f.orgSettings), nil

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
			f.logger.Warn().Msg("Invalid payload in handler config; assigning campaign owner")
			return NewAssignToPersonHandler(f.conversationRepo, &campaign.UserID, f.logger), nil
		}

		userIDStr, ok := payload["user_id"].(string)
		if !ok {
			f.logger.Warn().Msg("Missing user_id in payload; assigning campaign owner")
			return NewAssignToPersonHandler(f.conversationRepo, &campaign.UserID, f.logger), nil
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
			Msg("Unknown handler type; using default route")
		return f.defaultHandler(campaign, intent), nil
	}
}

// defaultHandler routes an intent that has no (valid) campaign-specific
// configuration, following the intent taxonomy's routed actions:
// actionable asks get a grounded AI reply, referrals create a lead, and
// everything else goes to the campaign owner for human follow-up.
func (f *IntentHandlerFactory) defaultHandler(campaign *domain.Campaign, intent domain.IntentType) IntentHandler {
	switch intent {
	case domain.IntentInterested, domain.IntentRequestForPricing, domain.IntentRequestForInfo:
		return NewAIReplyHandler(
			f.emailRepo,
			f.contactRepo,
			f.conversationRepo,
			f.agentRepo,
			f.knowledgeRepo,
			f.knowledgeSearch,
			f.saleRepRepo,
			f.cozeSvc,
			f.logger,
		).WithOrgSettings(f.orgSettings)
	case domain.IntentReferral:
		return NewReferralHandler(
			f.emailRepo,
			f.contactRepo,
			f.conversationRepo,
			f.logger,
		)
	default: // NURTURE, NOT_INTERESTED, ... : human follow-up by campaign owner
		return NewAssignToPersonHandler(f.conversationRepo, &campaign.UserID, f.logger)
	}
}
