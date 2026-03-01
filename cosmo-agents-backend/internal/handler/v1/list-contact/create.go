package listcontact

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// Create handles POST /v1/list-contacts
// @Summary Create a new list contact
// @Description Creates a new contact list for the authenticated user
// @Tags List Contacts
// @Accept json
// @Produce json
// @Param request body v1schema.CreateListContactRequest true "List contact creation request"
// @Success 201 {object} schema.APIResponse[v1schema.ListContactResponse] "List contact created successfully"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request body or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 500 {object} schema.APIResponse[any] "Failed to create list contact"
// @Router /v1/list-contacts [post]
// @Security BearerAuth
func (h *Handler) Create(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return unauthorized(c, "User not authenticated")
	}

	var req v1schema.CreateListContactRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return badRequest(c, "Validation failed", err)
	}

	listContact := &domain.ListContact{
		UserID:         userID,
		OrganizationID: nil,
		Name:           req.Name,
		Source:         "",
		SourceID:       "",
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

	if _, err := h.listContactRepo.Create(c.Context(), listContact); err != nil {
		return internalError(c, "Failed to create list contact", err)
	}

	// If inbound lead form IDs were provided, link them to the new list with ownership validation
	if len(req.InboundLeadFormIDs) > 0 {
		if err := h.listContactRepo.AddInboundFormsWithOwnership(c.Context(), userID, listContact.ID, req.InboundLeadFormIDs); err != nil {
			if errors.Is(err, contactRepo.ErrListContactUnauthorized) {
				return forbidden(c, err.Error())
			}
			return internalError(c, "Failed to associate inbound lead forms", err)
		}
	}

	if len(req.ContactIDs) > 0 {
		if err := h.listContactRepo.AddContactsWithOwnership(c.Context(), userID, listContact.ID, req.ContactIDs); err != nil {
			if errors.Is(err, contactRepo.ErrListContactUnauthorized) {
				return forbidden(c, err.Error())
			}
			return internalError(c, "Failed to associate contacts", err)
		}
	}

	// Reload to get relationships populated
	createdListContact, _ := h.listContactRepo.FindByID(c.Context(), listContact.ID)
	if createdListContact != nil {
		listContact = createdListContact
	}

	// Load InboundLeadForms associated with this ListContact
	inboundForms, err := h.loadInboundLeadForms(c.Context(), listContact.ID)
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

	return c.Status(fiber.StatusCreated).JSON(schema.SuccessResponse(response))
}
