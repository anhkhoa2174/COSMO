package campaign

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// Delete handles DELETE /v1/campaigns/{id}.
// @Summary Delete Campaign
// @Description Delete a specific campaign by its ID.
// @Tags Campaigns
// @Accept json
// @Produce json
// @Param id path string true "Campaign ID"
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[string] "Successful Response"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 403 {object} schema.APIResponse[any] "Forbidden"
// @Failure 404 {object} schema.APIResponse[any] "Not Found"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v1/campaigns/{id} [delete]
func (h *Handler) Delete(c fiber.Ctx) error {
	campaignID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	campaign, err := h.campaignRepo.FindByID(c.Context(), campaignID)
	if err != nil {
		return internalError(c, "Failed to fetch campaign", err)
	}
	if campaign == nil {
		return notFound(c, "Campaign not found")
	}

	if campaign.UserID != userID {
		return forbidden(c, "You don't have permission to delete this campaign")
	}

	if err := h.campaignRepo.Delete(c.Context(), campaignID); err != nil {
		return internalError(c, "Failed to delete campaign", err)
	}

	return c.JSON(schema.SuccessResponse("Deleted successfully"))
}
