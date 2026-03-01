package campaign

import (
	"github.com/openai/openai-go"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	knowledgeRepo "github.com/rockship/cosmo-agents-go/internal/repository/knowledge"
	"github.com/rockship/cosmo-agents-go/internal/repository/organization"
	template "github.com/rockship/cosmo-agents-go/internal/repository/template"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
)

// Handler handles V3 campaign API requests
type Handler struct {
	campaignRepo  *campaignRepo.CampaignRepository
	templateRepo  *template.TemplateRepository
	userRepo      *user.UserRepository
	orgRepo       *organization.OrganizationRepository
	knowledgeRepo *knowledgeRepo.KnowledgeRepository
	openAIClient  *openai.Client
}

// NewHandler creates a new V3 Campaign Handler
func NewHandler(
	campaignRepo *campaignRepo.CampaignRepository,
	templateRepo *template.TemplateRepository,
	userRepo *user.UserRepository,
	orgRepo *organization.OrganizationRepository,
	knowledgeRepo *knowledgeRepo.KnowledgeRepository,
	openAIClient *openai.Client,
) *Handler {
	return &Handler{
		campaignRepo:  campaignRepo,
		templateRepo:  templateRepo,
		userRepo:      userRepo,
		orgRepo:       orgRepo,
		knowledgeRepo: knowledgeRepo,
		openAIClient:  openAIClient,
	}
}
