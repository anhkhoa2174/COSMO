package campaign

import (
	"github.com/openai/openai-go"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	draftTemplateRepo "github.com/rockship/cosmo-agents-go/internal/repository/draft_template"
	knowledgeRepo "github.com/rockship/cosmo-agents-go/internal/repository/knowledge"
	"github.com/rockship/cosmo-agents-go/internal/repository/organization"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	template "github.com/rockship/cosmo-agents-go/internal/repository/template"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	intentService "github.com/rockship/cosmo-agents-go/internal/service/intent"
)

// Handler handles V2 campaign HTTP requests
type Handler struct {
	campaignRepo      *campaignRepo.CampaignRepository
	templateRepo      *template.TemplateRepository
	draftTemplateRepo *draftTemplateRepo.DraftTemplateRepository
	userRepo          *user.UserRepository
	roleRepo          *roleRepo.RoleRepository
	orgRepo           *organization.OrganizationRepository
	knowledgeRepo     *knowledgeRepo.KnowledgeRepository
	conversationRepo  *conversationRepo.ConversationRepository
	openAIClient      *openai.Client
	intentClassifier  *intentService.IntentClassifier
}

// NewHandler creates a new V2 campaign handler with injected dependencies
func NewHandler(
	campaignRepo *campaignRepo.CampaignRepository,
	templateRepo *template.TemplateRepository,
	draftTemplateRepo *draftTemplateRepo.DraftTemplateRepository,
	userRepo *user.UserRepository,
	roleRepo *roleRepo.RoleRepository,
	orgRepo *organization.OrganizationRepository,
	knowledgeRepo *knowledgeRepo.KnowledgeRepository,
	conversationRepo *conversationRepo.ConversationRepository,
	openAIClient *openai.Client,
	intentClassifier *intentService.IntentClassifier,
) *Handler {
	return &Handler{
		campaignRepo:      campaignRepo,
		templateRepo:      templateRepo,
		draftTemplateRepo: draftTemplateRepo,
		userRepo:          userRepo,
		roleRepo:          roleRepo,
		orgRepo:           orgRepo,
		knowledgeRepo:     knowledgeRepo,
		conversationRepo:  conversationRepo,
		openAIClient:      openAIClient,
		intentClassifier:  intentClassifier,
	}
}
