package contact

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"gorm.io/gorm"
)

// Delete handles DELETE /v1/contacts
// @Summary Delete contacts
// @Description Soft deletes contacts by IDs
// @Tags Contacts
// @Accept json
// @Produce json
// @Param body body v1schema.ContactDeleteRequest true "Contact IDs to delete"
// @Success 200 {object} schema.APIResponse[[]v1schema.ContactResponse] "Successfully deleted contacts"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/contacts [delete]
func (h *Handler) Delete(c fiber.Ctx) error {
	// Use auth helper
	user, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return h.responseHelper.HandleAuthError(c, errors.New("You are not authorized to access this resource"))
		}
		return h.responseHelper.HandleAuthError(c, err)
	}

	// Parse request
	var req v1schema.ContactDeleteRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	// Validate request
	if err := v1validation.ValidateStruct(req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(
			schema.ErrorResponse(fiber.StatusUnprocessableEntity, err.Error(), ""),
		)
	}

	if len(req.IDs) == 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(
			schema.ErrorResponse(fiber.StatusUnprocessableEntity, "ids must have at least 1 element", ""),
		)
	}

	// Convert string IDs to UUIDs
	ids := make([]uuid.UUID, 0, len(req.IDs))
	for _, idStr := range req.IDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue // Skip invalid UUIDs
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return h.responseHelper.Success(c, []*v1schema.ContactResponse{})
	}

	// Delete contacts
	orgIDPtr := &organizationID
	deletedContacts, err := h.repo.DeleteByIDs(c.Context(), ids, user.ID, orgIDPtr)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to delete contacts", err)
	}

	// Clean up vectors from Redis for deleted contacts
	if h.intelSvc != nil && len(deletedContacts) > 0 {
		deletedIDs := make([]uuid.UUID, len(deletedContacts))
		for i, contact := range deletedContacts {
			deletedIDs[i] = contact.ID
		}
		go h.intelSvc.DeleteContactVectors(context.Background(), deletedIDs)
	}

	// Convert to responses
	responses := make([]*v1schema.ContactResponse, len(deletedContacts))
	for i, contact := range deletedContacts {
		responses[i] = v1schema.ToContactResponse(contact)
	}

	return h.responseHelper.Success(c, responses)
}
