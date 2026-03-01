package contact

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	"gorm.io/gorm"
)

var _ = schema.APIResponse[any]{}

// Get retrieves a contact by ID
// @Summary Get contact by ID
// @Description Retrieves contact details by ID. User must be the owner or have access through organization membership.
// @Tags V2 Contacts
// @Accept json
// @Produce json
// @Param id path string true "Contact ID (UUID)"
// @Success 200 {object} schema.APIResponse[ContactEntity] "Successfully retrieved contact"
// @Failure 400 {object} schema.APIResponse[any] "Invalid contact ID"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 403 {object} schema.APIResponse[any] "Forbidden - User doesn't have access to this contact"
// @Failure 404 {object} schema.APIResponse[any] "Contact not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v2/contacts/{id} [get]
func (h *Handler) Get(c fiber.Ctx) error {
	// Use auth helper
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	// Parse ID
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid contact ID",
		})
	}

	// Find contact with access control
	contact, err := h.contactRepo.FindByIDAndUserID(c.Context(), userID, id)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": "Contact not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to get contact",
		})
	}

	entity, err := h.toContactEntitySingle(contact)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to convert contact to entity",
		})
	}
	return c.JSON(fiber.Map{
		"status": "success",
		"data":   entity,
	})
}
