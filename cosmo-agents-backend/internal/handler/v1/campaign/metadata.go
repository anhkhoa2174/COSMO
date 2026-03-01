package campaign

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// UpdateClientMetadata handles PATCH /v1/campaigns/{id}/client-metadata.
// @Summary Update Campaign Client Metadata
// @Description Update the client-specific metadata for a campaign.
// @Tags Campaigns
// @Accept json
// @Produce json
// @Param id path string true "Campaign ID"
// @Param body body v1schema.UpdateCampaignMetadataRequest true "Client Metadata Request"
// @Success 200 {object} schema.APIResponse[v1schema.CampaignResponse] "Successful Response"
// @Security BearerAuth
// @Failure 400 {object} schema.APIResponse[any] "Bad Request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 403 {object} schema.APIResponse[any] "Forbidden"
// @Failure 404 {object} schema.APIResponse[any] "Not Found"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v1/campaigns/{id}/client-metadata [patch]
func (h *Handler) UpdateClientMetadata(c fiber.Ctx) error {
	campaignID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	var req v1schema.UpdateCampaignMetadataRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	campaign, err := h.campaignRepo.FindByID(c.Context(), campaignID)
	if err != nil {
		return internalError(c, "Failed to fetch campaign", err)
	}
	if campaign == nil {
		return notFound(c, "Campaign not found")
	}

	// Authorization: user must be campaign owner OR member of campaign's organization
	if campaign.UserID != userID {
		if campaign.OrganizationID == nil {
			return forbidden(c, "Access denied")
		}

		// Check if user is active member of campaign's organization
		userRoles, err := h.roleRepo.FindByUserID(c.Context(), userID)
		if err != nil {
			return internalError(c, "Failed to verify user organizations", err)
		}

		if !userInOrganization(*campaign.OrganizationID, userRoles) {
			return forbidden(c, "You must be a member of the campaign's organization")
		}
	}

	if campaign.CMetadata.Client == nil {
		campaign.CMetadata.Client = map[string]interface{}{}
	}

	for k, v := range req.Client {
		campaign.CMetadata.Client[k] = v
	}

	if err := h.campaignRepo.UpdateAttributes(c.Context(), campaignID, map[string]interface{}{"cmetadata": campaign.CMetadata}); err != nil {
		return internalError(c, "Failed to update metadata", err)
	}

	return c.JSON(schema.SuccessResponse(v1schema.ToCampaignResponse(campaign)))
}
