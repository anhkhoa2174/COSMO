package listcontact

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// Delete handles DELETE /v1/list-contacts
// @Summary Delete list contacts (batch)
// @Description Deletes multiple contact lists by their IDs. Only lists owned by the user will be deleted. Non-existent or unauthorized IDs are skipped.
// @Tags List Contacts
// @Accept json
// @Produce json
// @Param request body v1schema.DeleteListContactRequest true "List of list contact IDs to delete"
// @Success 200 {object} schema.APIResponse[string] "List contacts deleted successfully"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request body or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Router /v1/list-contacts [delete]
// @Security BearerAuth
func (h *Handler) Delete(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return unauthorized(c, "User not authenticated")
	}

	var req v1schema.DeleteListContactRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return badRequest(c, "Validation failed", err)
	}

	// Delete all specified IDs (with ownership check in query)
	for _, id := range req.IDs {
		listContact, err := h.listContactRepo.FindByID(c.Context(), id)
		if err != nil {
			continue // Skip if error occurred
		}

		if listContact == nil {
			continue // Skip if not found or already deleted
		}

		if listContact.UserID != userID {
			continue // Skip if no access
		}

		_ = h.listContactRepo.Delete(c.Context(), id)
	}

	return c.JSON(schema.SuccessResponse("List contacts deleted successfully"))
}
