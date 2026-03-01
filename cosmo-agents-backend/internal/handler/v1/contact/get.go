package contact

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// Get godoc
// @Summary Get contact by ID
// @Tags Contacts
// @Accept json
// @Produce json
// @Param id path string true "Contact ID (UUID)"
// @Success 200 {object} v1schema.ContactResponse
// @Failure 400 {object} any
// @Failure 404 {object} any
// @Failure 500 {object} any
// @Security BearerAuth
// @Router /v1/contact/{id} [get]
func (h *Handler) Get(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

	user, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	contact, err := h.repo.FindByIDAndUserID(c.Context(), user.ID, id)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to fetch contact", err)
	}

	if contact == nil {
		return h.responseHelper.NotFound(c, "Contact not found", nil)
	}

	if contact.OrganizationID == nil || *contact.OrganizationID != organizationID {
		return h.responseHelper.NotFound(c, "Contact not found", nil)
	}

	response := v1schema.ToContactResponse(contact)
	return h.responseHelper.Success(c, response)
}
