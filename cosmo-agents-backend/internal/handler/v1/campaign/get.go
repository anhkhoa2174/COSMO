package campaign

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// GetByID handles GET /v1/campaigns/{id}.
// @Summary Get Campaign by ID
// @Description Retrieve detailed information about a specific campaign by its ID.
// @Tags Campaigns
// @Accept json
// @Produce json
// @Param id path string true "Campaign ID"
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[v1schema.CampaignDetailGetResponse] "Successful Response"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 404 {object} schema.APIResponse[any] "Not Found"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v1/campaigns/{id} [get]
func (h *Handler) GetByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	userRoles, err := h.roleRepo.FindByUserID(c.Context(), userID)
	if err != nil {
		return internalError(c, "Failed to verify user organizations", err)
	}
	if len(userRoles) == 0 {
		return unauthorizedAccessResponse(c)
	}

	campaign, err := h.campaignRepo.FindWithRelations(c.Context(), id)
	if err != nil {
		return internalError(c, "Failed to fetch campaign", err)
	}
	if campaign == nil {
		return notFound(c, "Campaign not found")
	}

	if !campaignAccessible(campaign, userID, userRoles) {
		return notFound(c, "Campaign not found")
	}

	// Load campaign relationships manually
	notifications, err := h.loadCampaignNotifications(c.Context(), campaign.ID)
	if err != nil {
		// Log error but don't fail the request
		notifications = []v1schema.NotificationRelationshipListItem{}
	}

	templates, err := h.loadCampaignTemplates(c.Context(), campaign.ID)
	if err != nil {
		// Log error but don't fail the request
		templates = []v1schema.TemplateRelationshipListItem{}
	}

	draftTemplates, err := h.loadCampaignDraftTemplates(c.Context(), campaign.ID)
	if err != nil {
		// Log error but don't fail the request
		draftTemplates = []v1schema.DraftTemplateRelationshipListItem{}
	}

	// Create custom response with loaded relationships
	response := h.createCampaignDetailResponse(campaign, notifications, templates, draftTemplates)

	return c.JSON(schema.SuccessResponse(response))
}
