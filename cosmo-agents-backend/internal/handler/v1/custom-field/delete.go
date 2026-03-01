package customfield

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// Delete handles DELETE /v1/custom-fields/:id
// @Summary Delete a custom field
// @Description Deletes a custom field by ID. User must own the custom field.
// @Tags Custom Fields
// @Accept json
// @Produce json
// @Param id path string true "Custom field ID (UUID)"
// @Success 200 {object} schema.APIResponse[string] "Custom field deleted successfully"
// @Failure 400 {object} schema.APIResponse[any] "Invalid custom field ID"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Access denied - user doesn't own this custom field"
// @Failure 404 {object} schema.APIResponse[any] "Custom field not found"
// @Failure 500 {object} schema.APIResponse[any] "Failed to delete custom field"
// @Router /v1/custom-fields/{id} [delete]
// @Security BearerAuth
func (h *Handler) Delete(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return unauthorized(c, "User not authenticated")
	}

	customFieldID, err := parseUUIDParam(c, "id")
	if err != nil {
		return badRequest(c, "Invalid custom field ID", err)
	}

	// Get existing custom field
	customField, err := h.customFieldRepo.FindByID(c.Context(), customFieldID)
	if err != nil {
		return internalError(c, "Failed to fetch custom field", err)
	}

	if customField == nil {
		return notFound(c, "Custom field not found")
	}

	// Check ownership
	if customField.UserID != userID {
		return forbidden(c, "Access denied")
	}

	if err := h.customFieldRepo.Delete(c.Context(), customFieldID); err != nil {
		return internalError(c, "Failed to delete custom field", err)
	}

	return c.JSON(schema.SuccessResponse("Custom field deleted successfully"))
}
