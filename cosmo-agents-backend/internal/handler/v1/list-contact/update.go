package listcontact

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// Update handles PATCH /v1/list-contacts/:id
// @Summary Update a list contact
// @Description Updates an existing contact list. Only provided fields will be updated. User must own the list.
// @Tags List Contacts
// @Accept json
// @Produce json
// @Param id path string true "List contact ID (UUID)"
// @Param request body v1schema.UpdateListContactRequest true "Fields to update (all optional)"
// @Success 200 {object} schema.APIResponse[v1schema.ListContactResponse] "List contact updated successfully"
// @Failure 400 {object} schema.APIResponse[any] "Invalid list contact ID or request body"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Access denied - user doesn't own this list"
// @Failure 404 {object} schema.APIResponse[any] "List contact not found"
// @Failure 500 {object} schema.APIResponse[any] "Failed to update list contact"
// @Router /v1/list-contacts/{id} [patch]
// @Security BearerAuth
func (h *Handler) Update(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return unauthorized(c, "User not authenticated")
	}

	id := c.Params("id")
	listContactID, err := uuid.Parse(id)
	if err != nil {
		return badRequest(c, "Invalid list contact ID", err)
	}

	var req v1schema.UpdateListContactRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	listContact, err := h.listContactRepo.FindByID(c.Context(), listContactID)
	if err != nil {
		return internalError(c, "Failed to fetch list contact", err)
	}

	if listContact == nil {
		return notFound(c, "List contact not found")
	}

	if listContact.UserID != userID {
		return forbidden(c, "Access denied")
	}

	if req.Name != nil {
		listContact.Name = *req.Name
	}
	if req.Source != nil {
		listContact.Source = *req.Source
	}
	if req.SourceID != nil {
		listContact.SourceID = *req.SourceID
	}
	if req.HubspotID != nil {
		listContact.HubspotID = req.HubspotID
	}

	// If inbound lead form IDs provided, attach them to the list with ownership validation
	if len(req.InboundLeadFormIDs) > 0 {
		if err := h.listContactRepo.AddInboundFormsWithOwnership(c.Context(), userID, listContactID, req.InboundLeadFormIDs); err != nil {
			if errors.Is(err, contactRepo.ErrListContactUnauthorized) {
				return forbidden(c, err.Error())
			}
			return internalError(c, "Failed to associate inbound lead forms", err)
		}
	}

	if len(req.ContactIDs) > 0 {
		if err := h.listContactRepo.AddContactsWithOwnership(c.Context(), userID, listContactID, req.ContactIDs); err != nil {
			if errors.Is(err, contactRepo.ErrListContactUnauthorized) {
				return forbidden(c, err.Error())
			}
			return internalError(c, "Failed to associate contacts", err)
		}
	}

	if err := h.listContactRepo.Update(c.Context(), listContactID, listContact); err != nil {
		return internalError(c, "Failed to update list contact", err)
	}

	// Reload to get relationships populated
	updatedListContact, _ := h.listContactRepo.FindByID(c.Context(), listContactID)
	if updatedListContact != nil {
		listContact = updatedListContact
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
