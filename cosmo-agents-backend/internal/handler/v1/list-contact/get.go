package listcontact

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// GetByID handles GET /v1/list-contacts/:id
// @Summary Get a list contact by ID
// @Description Retrieves a single contact list by its ID. User must own the list.
// @Tags List Contacts
// @Accept json
// @Produce json
// @Param id path string true "List contact ID (UUID)"
// @Success 200 {object} schema.APIResponse[v1schema.ListContactResponse] "List contact retrieved successfully"
// @Failure 400 {object} schema.APIResponse[any] "Invalid list contact ID"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Access denied - user doesn't own this list"
// @Failure 404 {object} schema.APIResponse[any] "List contact not found"
// @Router /v1/list-contacts/{id} [get]
// @Security BearerAuth
func (h *Handler) GetByID(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return unauthorized(c, "User not authenticated")
	}

	id := c.Params("id")
	listContactID, err := uuid.Parse(id)
	if err != nil {
		return badRequest(c, "Invalid list contact ID", err)
	}

	listContact, err := h.listContactRepo.FindByID(c.Context(), listContactID)
	if err != nil {
		return internalError(c, "Failed to retrieve list contact", err)
	}

	if listContact == nil {
		return notFound(c, "List contact not found")
	}

	// Check ownership
	if listContact.UserID != userID {
		return forbidden(c, "Access denied")
	}

	// Load InboundLeadForms associated with this ListContact
	inboundForms, err := h.loadInboundLeadForms(c.Context(), listContactID)
	if err != nil {
		// Log the error but don't fail the request
		// InboundLeadForms is supplementary data, so we can continue with empty array
		inboundForms = []v1schema.InboundLeadFormEntity{}
	}

	// Convert Contacts to response entities
	contacts := make([]v1schema.ContactResponse, len(listContact.Contacts))
	for i, contact := range listContact.Contacts {
		contactResp := v1schema.ToContactResponse(&contact)
		if contactResp != nil {
			contacts[i] = *contactResp
		}
	}

	response := v1schema.ListContactResponse{
		ID:               listContact.ID,
		Name:             listContact.Name,
		Source:           &listContact.Source,
		SourceID:         &listContact.SourceID,
		HubspotID:        listContact.HubspotID,
		InboundLeadForms: inboundForms,
		Contacts:         contacts,
		CreatedAt:        listContact.CreatedAt,
		UpdatedAt:        listContact.UpdatedAt,
	}

	return c.JSON(schema.SuccessResponse(response))
}
