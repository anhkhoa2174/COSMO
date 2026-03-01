package contact

import (
	"github.com/gofiber/fiber/v3"
)

// RecalculateStatus recalculates status for all contacts in the organization
// POST /v1/contacts/recalculate-status
// @Summary Recalculate contact statuses
// @Description Recalculates status (ready/pending) for all contacts based on current required fields
// @Tags Contacts
// @Produce json
// @Success 200 {object} schema.APIResponse[map[string]int] "Successfully recalculated statuses"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/contacts/recalculate-status [post]
func (h *Handler) RecalculateStatus(c fiber.Ctx) error {
	// Use auth helper to get user and organization
	_, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	// Recalculate statuses for all contacts in the organization
	updated, err := h.repo.RecalculateAllStatuses(c.Context(), &organizationID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to recalculate statuses", err)
	}

	return h.responseHelper.Success(c, fiber.Map{
		"message":       "Statuses recalculated successfully",
		"updated_count": updated,
	})
}
