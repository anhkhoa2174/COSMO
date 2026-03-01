package campaign

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// AssignMember handles POST /v1/campaigns/{id}/assign.
// @Summary Assign Members to Campaign
// @Description Assign members to be responsible for a campaign.
// @Tags Campaigns
// @Accept json
// @Produce json
// @Param id path string true "Campaign ID"
// @Param body body v1schema.CampaignAssignRequest true "Campaign Assignment Request"
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[v1schema.CampaignResponse] "Successful Response"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 403 {object} schema.APIResponse[any] "Forbidden"
// @Failure 404 {object} schema.APIResponse[any] "Not Found"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v1/campaigns/{id}/assign [post]
func (h *Handler) AssignMember(c fiber.Ctx) error {
	campaignID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	primaryOrgID, err := h.roleRepo.FindPrimaryOrganization(c.Context(), userID)
	if err != nil {
		return internalError(c, "Failed to determine primary organization", err)
	}
	if primaryOrgID == nil {
		return forbidden(c, "You need to be a member of an organization to assign responsibility")
	}

	var req v1schema.CampaignAssignRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return badRequest(c, "Validation failed", err)
	}

	members, err := convertAssignConfig(req.Config)
	if err != nil {
		return badRequest(c, "Invalid assignment configuration", err)
	}

	campaign, err := h.campaignRepo.FindByID(c.Context(), campaignID)
	if err != nil {
		return internalError(c, "Failed to fetch campaign", err)
	}
	if campaign == nil {
		return notFound(c, "Campaign not found")
	}
	if campaign.OrganizationID == nil || *campaign.OrganizationID != *primaryOrgID {
		return forbidden(c, "You need to be a member of an organization to assign responsibility")
	}

	campaign.CMetadata.Config = members

	if err := h.campaignRepo.UpdateAttributes(c.Context(), campaignID, map[string]interface{}{"cmetadata": campaign.CMetadata}); err != nil {
		return internalError(c, "Failed to save assignment", err)
	}

	return c.JSON(schema.SuccessResponse(v1schema.ToCampaignResponse(campaign)))
}
