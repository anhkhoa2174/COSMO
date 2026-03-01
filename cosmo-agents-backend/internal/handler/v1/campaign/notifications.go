package campaign

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// DeleteNotifications handles DELETE /v1/campaigns/{id}/notifications.
// @Summary Delete Campaign Notifications
// @Description Delete notifications for a campaign.
// @Tags Campaigns
// @Accept json
// @Produce json
// @Param id path string true "Campaign ID"
// @Param body body v1schema.DeleteNotificationRequest true "Delete Notification Request"
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[string] "Successful Response"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 404 {object} schema.APIResponse[any] "Not Found"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v1/campaigns/{id}/notifications [delete]
func (h *Handler) DeleteNotifications(c fiber.Ctx) error {
	campaignID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	var req v1schema.DeleteNotificationRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return badRequest(c, "Validation failed", err)
	}

	campaign, err := h.campaignRepo.FindByID(c.Context(), campaignID)
	if err != nil {
		return internalError(c, "Failed to fetch campaign", err)
	}
	if campaign == nil || campaign.UserID != userID {
		return notFound(c, "Campaign not found")
	}

	if err := h.notificationRepo.DeleteByIDs(c.Context(), campaignID, req.IDs); err != nil {
		return internalError(c, "Failed to delete notifications", err)
	}

	return c.JSON(schema.SuccessResponse("Notification deleted successfully"))
}

// syncNotifications synchronizes campaign notifications
func (h *Handler) syncNotifications(ctx context.Context, campaignID uuid.UUID, organizationID *uuid.UUID, userID uuid.UUID, provided []uuid.UUID) ([]domain.Notification, error) {
	if organizationID == nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Campaign must belong to an organization")
	}

	// Ensure owner is subscribed.
	desired := make(map[uuid.UUID]struct{})
	desired[userID] = struct{}{}

	for _, id := range provided {
		desired[id] = struct{}{}
	}

	userIDs := make([]uuid.UUID, 0, len(desired))
	for id := range desired {
		ok, err := h.roleRepo.ExistsByOrgAndUser(ctx, *organizationID, id, domain.RoleNameAdmin)
		if err != nil {
			return nil, internalError(nil, "Failed to verify organization membership", err)
		}
		if !ok {
			role, err := h.roleRepo.FindByUserAndOrganization(ctx, id, *organizationID)
			if err != nil {
				return nil, internalError(nil, "Failed to verify organization membership", err)
			}
			if role == nil {
				return nil, fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("User %s not found in organization", id))
			}
		}
		userIDs = append(userIDs, id)
	}

	notifications := make([]domain.Notification, len(userIDs))
	for i, id := range userIDs {
		notifications[i] = domain.Notification{UserID: id, CampaignID: campaignID}
	}

	created, err := h.notificationRepo.UpsertMany(ctx, notifications)
	if err != nil {
		return nil, internalError(nil, "Failed to update notifications", err)
	}
	return created, nil
}
