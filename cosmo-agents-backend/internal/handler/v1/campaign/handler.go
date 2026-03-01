package campaign

import (
	"context"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	draftTemplateRepo "github.com/rockship/cosmo-agents-go/internal/repository/draft_template"
	notificationRepo "github.com/rockship/cosmo-agents-go/internal/repository/notification"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	template "github.com/rockship/cosmo-agents-go/internal/repository/template"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	aiService "github.com/rockship/cosmo-agents-go/internal/service/ai"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// Handler handles campaign HTTP requests
type Handler struct {
	campaignRepo      *campaignRepo.CampaignRepository
	roleRepo          *roleRepo.RoleRepository
	notificationRepo  *notificationRepo.NotificationRepository
	listContactRepo   *contactRepo.ListContactRepository
	templateRepo      *template.TemplateRepository
	draftTemplateRepo *draftTemplateRepo.DraftTemplateRepository
	agentRepo         *agentRepo.AgentRepository
	conversationRepo  *conversationRepo.ConversationRepository
	aiEmailService    *aiService.AIEmailService
	workerClient      *worker.Client
}

// NewHandler creates a new campaign handler with injected dependencies
func NewHandler(
	campaignRepoParam *campaignRepo.CampaignRepository,
	roleRepoParam *roleRepo.RoleRepository,
	notificationRepoParam *notificationRepo.NotificationRepository,
	listContactRepoParam *contactRepo.ListContactRepository,
	templateRepoParam *template.TemplateRepository,
	draftTemplateRepoParam *draftTemplateRepo.DraftTemplateRepository,
	agentRepoParam *agentRepo.AgentRepository,
	conversationRepoParam *conversationRepo.ConversationRepository,
	aiEmailService *aiService.AIEmailService,
	workerClient *worker.Client,
) *Handler {
	return &Handler{
		campaignRepo:      campaignRepoParam,
		roleRepo:          roleRepoParam,
		notificationRepo:  notificationRepoParam,
		listContactRepo:   listContactRepoParam,
		templateRepo:      templateRepoParam,
		draftTemplateRepo: draftTemplateRepoParam,
		agentRepo:         agentRepoParam,
		conversationRepo:  conversationRepoParam,
		aiEmailService:    aiEmailService,
		workerClient:      workerClient,
	}
}

// loadCampaignNotifications loads notifications for a campaign and converts to response format
func (h *Handler) loadCampaignNotifications(ctx context.Context, campaignID uuid.UUID) ([]v1schema.NotificationRelationshipListItem, error) {
	// Load domain models
	domainNotifications, err := h.notificationRepo.FindByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	// Convert to response entities
	notifications := make([]v1schema.NotificationRelationshipListItem, len(domainNotifications))
	for i, notification := range domainNotifications {
		notifications[i] = v1schema.NotificationRelationshipListItem{
			UserID: notification.UserID,
		}
	}

	return notifications, nil
}

// loadCampaignTemplates loads templates for a campaign and converts to response format
func (h *Handler) loadCampaignTemplates(ctx context.Context, campaignID uuid.UUID) ([]v1schema.TemplateRelationshipListItem, error) {
	// Load domain models
	domainTemplates, err := h.templateRepo.FindByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	// Convert to response entities
	templates := make([]v1schema.TemplateRelationshipListItem, len(domainTemplates))
	for i, template := range domainTemplates {
		templates[i] = v1schema.TemplateRelationshipListItem{
			ID:        template.ID,
			Category:  string(template.Category),
			Type:      template.Type,
			SendAfter: template.SendAfter,
			Content:   template.Content,
		}
	}

	return templates, nil
}

// loadCampaignDraftTemplates loads draft templates for a campaign and converts to response format
func (h *Handler) loadCampaignDraftTemplates(ctx context.Context, campaignID uuid.UUID) ([]v1schema.DraftTemplateRelationshipListItem, error) {
	// Load domain models
	domainDraftTemplates, err := h.draftTemplateRepo.FindByCampaignID(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	// Convert to response entities
	draftTemplates := make([]v1schema.DraftTemplateRelationshipListItem, len(domainDraftTemplates))
	for i, draftTemplate := range domainDraftTemplates {
		draftTemplates[i] = v1schema.DraftTemplateRelationshipListItem{
			ID:     draftTemplate.ID,
			Intent: draftTemplate.Intent,
		}
	}

	return draftTemplates, nil
}

// createCampaignDetailResponse creates a campaign detail response with loaded relationships
func (h *Handler) createCampaignDetailResponse(
	campaign *domain.Campaign,
	notifications []v1schema.NotificationRelationshipListItem,
	templates []v1schema.TemplateRelationshipListItem,
	draftTemplates []v1schema.DraftTemplateRelationshipListItem,
) v1schema.CampaignDetailGetResponse {
	if campaign == nil {
		return v1schema.CampaignDetailGetResponse{}
	}

	status := string(campaign.Status)

	response := v1schema.CampaignDetailGetResponse{
		ID:             campaign.ID,
		UserID:         campaign.UserID,
		Playbook:       stringPtrOrNil(campaign.Playbook),
		Name:           stringPtrOrNil(campaign.Name),
		ListContactID:  campaign.ListContactID,
		OrganizationID: campaign.OrganizationID,
		Schedule:       campaign.Schedule,
		CreatedAt:      campaign.CreatedAt,
		UpdatedAt:      campaign.UpdatedAt,
		Status:         stringPtrOrNil(status),
		AgentID:        campaign.AgentID,
		CMetadata:      ensureEmptyMap(campaignMetadataToMap(campaign.CMetadata)),
		Notifications:  notifications,
		Templates:      templates,
		DraftTemplates: draftTemplates,
	}

	return response
}

// Helper functions for response creation
func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ensureEmptyMap(metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		return make(map[string]interface{})
	}
	return metadata
}

func campaignMetadataToMap(metadata interface{}) map[string]interface{} {
	if metadata == nil {
		return map[string]interface{}{
			"config": make([]interface{}, 0),
		}
	}

	// Just return the raw CMetadata from database
	if campaignMeta, ok := metadata.(domain.CampaignMetadata); ok {
		// Convert struct to map for JSON serialization
		return map[string]interface{}{
			"config":   campaignMeta.Config,
			"sequence": campaignMeta.Sequence,
			"client":   campaignMeta.Client,
		}
	}

	// Handle map[string]interface{} directly
	if metaMap, ok := metadata.(map[string]interface{}); ok {
		return metaMap
	}

	// Fallback
	return map[string]interface{}{
		"config": make([]interface{}, 0),
	}
}
