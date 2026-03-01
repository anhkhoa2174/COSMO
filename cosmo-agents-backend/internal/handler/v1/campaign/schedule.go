package campaign

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// SetFollowUpSchedule handles POST /v1/campaigns/{id}/follow-up-schedule.
// @Summary Set Follow-Up Schedule
// @Description Set the follow-up email schedule for a campaign.
// @Tags Campaigns
// @Accept json
// @Produce json
// @Param id path string true "Campaign ID"
// @Param body body v1schema.FollowUpScheduleRequest true "Follow-Up Schedule Request"
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[v1schema.CampaignResponse] "Successful Response"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 404 {object} schema.APIResponse[any] "Not Found"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v1/campaigns/{id}/follow-up-schedule [post]
func (h *Handler) SetFollowUpSchedule(c fiber.Ctx) error {
	campaignID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	var req v1schema.FollowUpScheduleRequest
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
	if campaign.UserID != userID {
		return notFound(c, "Campaign not found")
	}

	if campaign.CMetadata.Client == nil {
		campaign.CMetadata.Client = map[string]interface{}{}
	}

	if req.FollowUp1Schedule != nil {
		campaign.CMetadata.Client["follow_up_1_schedule"] = *req.FollowUp1Schedule
	}
	if req.FollowUp2Schedule != nil {
		campaign.CMetadata.Client["follow_up_2_schedule"] = *req.FollowUp2Schedule
	}

	if err := h.campaignRepo.UpdateAttributes(c.Context(), campaignID, map[string]interface{}{"cmetadata": campaign.CMetadata}); err != nil {
		return internalError(c, "Failed to save schedule", err)
	}

	return c.JSON(schema.SuccessResponse(v1schema.ToCampaignResponse(campaign)))
}
