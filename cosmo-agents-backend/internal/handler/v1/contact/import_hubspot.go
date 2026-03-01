package contact

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// ImportHubspot handles POST /v1/contacts/import-hubspot
// @Summary Import contacts from HubSpot
// @Description Imports contacts from HubSpot via async operation
// @Tags Contacts
// @Accept json
// @Produce json
// @Param body body v1schema.ContactImportHubspotRequest true "HubSpot import configuration"
// @Success 200 {object} v1schema.ContactImportHubspotResponse "Successfully started import operation"
// @Failure 400 {object} any "Invalid request"
// @Failure 401 {object} any "Unauthorized"
// @Failure 500 {object} any "Internal server error"
// @Security BearerAuth
// @Router /v1/contacts/import-hubspot [post]
func (h *Handler) ImportHubspot(c fiber.Ctx) error {
	// Use auth helper
	user, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	// Parse request
	var req v1schema.ContactImportHubspotRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	// Use field mapper service to flatten mapping
	mappingFields := h.fieldMapper.FlattenMapping(req.MappingFields)
	if len(mappingFields) == 0 {
		return h.responseHelper.BadRequest(c, "Mapping fields cannot be empty", nil)
	}

	// Get custom fields and build allowed fields
	customFields, err := h.customFieldRepo.FindByOrganizationID(c.Context(), organizationID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to fetch custom fields", err)
	}

	allowedFields := h.fieldMapper.BuildAllowedFields(customFields)
	for key := range mappingFields {
		if _, ok := allowedFields[key]; !ok {
			return h.responseHelper.BadRequest(c, fmt.Sprintf("Field %s is not allowed", key), nil)
		}
	}

	// Create operation
	input := map[string]interface{}{
		"user_id":         user.ID.String(),
		"organization_id": organizationID.String(),
		"mapping_fields":  mappingFields,
		"list_ids":        req.ListIDs,
	}

	operation := &domain.Operation{
		Name:   "machine.controllers.v1.contact.import_hubspot",
		Status: domain.OperationStatusInProgress,
	}

	if err := operation.Input.Marshal(input); err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to initialise operation input", err)
	}

	createdOperation, err := h.operationRepo.Create(c.Context(), operation)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to create operation", err)
	}

	// Build response
	responseInput := make(map[string]interface{})
	if err := createdOperation.Input.Unmarshal(&responseInput); err != nil {
		responseInput = input
	}

	var responseOutput interface{}
	if len(createdOperation.Output) > 0 {
		var output map[string]interface{}
		if err := createdOperation.Output.Unmarshal(&output); err == nil {
			responseOutput = output
		}
	}

	response := v1schema.ContactImportHubspotResponse{
		ID:        createdOperation.ID,
		Name:      createdOperation.Name,
		Status:    string(createdOperation.Status),
		Input:     responseInput,
		Output:    responseOutput,
		CreatedAt: createdOperation.CreatedAt,
		UpdatedAt: createdOperation.UpdatedAt,
	}

	return h.responseHelper.Success(c, response)
}
