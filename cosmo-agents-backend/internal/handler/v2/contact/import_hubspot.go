package contact

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	schemaV2 "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	contactworker "github.com/rockship/cosmo-agents-go/internal/worker/contact"
)

var ContactColumns = []string{"id", "user_id", "source_id", "hubspot_id", "source",
	"first_name", "last_name", "email", "phone", "company", "job_title", "address",
	"city", "country", "state", "zip", "created_at", "updated_at",
	"profile", "do_not_contact", "organization_id", "tags"}

// ImportHubspot handles POST /v1/contacts/import-hubspot
// @Summary Import contacts from HubSpot
// @Description Imports contacts from HubSpot via async operation
// @Tags V2 Contacts
// @Accept json
// @Produce json
// @Param body body schemaV2.ContactImportHubspotRequest true "HubSpot import configuration"
// @Success 200 {object} schema.APIResponse[string] "Successfully started import operation"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v2/contacts/import-hubspot [post]
func (h *Handler) ImportHubspot(c fiber.Ctx) error {
	// Use auth helper
	user, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"",
		))
	}

	// Parse request
	var req schemaV2.ContactImportHubspotRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid request body",
			err.Error(),
		))
	}

	if len(req.ListIDs) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Maximum 100 lists allowed", ""))
	}

	for _, id := range req.ListIDs {
		if id == "" {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest, "list_id cannot be empty", ""))
		}
	}

	if h.workerClient == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Worker client not configured", "",
		))
	}

	// Get allowed fields
	var customFields []domain.CustomField
	if customFields, err = h.customFieldRepo.FindByOrganizationID(c.Context(), organizationID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to get custom fields",
			err.Error(),
		))
	}
	normalizedNames := make([]string, 0, len(customFields))
	for _, cf := range customFields {
		normalizedNames = append(normalizedNames, cf.NormalizedName)
	}
	allowedFields := append(ContactColumns, normalizedNames...)

	// Get hubspot integration
	var hubspotIntegration *domain.Integration
	if hubspotIntegration, err = h.integrationRepo.FindByUserIDAndSource(c.Context(), user.ID, domain.SourceIntegrationHubspot); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to get hubspot integration",
			err.Error(),
		))
	} else if hubspotIntegration == nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusNotFound,
			"Hubspot account haven't authorized yet. Please connect to your Hubspot account",
			"",
		))
	}

	// Start: Check if key of field_mapping is allowed
	var configMap map[string]any
	if err := json.Unmarshal(hubspotIntegration.Config, &configMap); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to parse config",
			err.Error(),
		))
	}

	// Get field_mapping
	fieldMappingAny, ok := configMap["field_mapping"]
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"field_mapping not found",
			"",
		))
	}

	// Convert to map[string]string (or map[string]interface{})
	fieldMapping, ok := fieldMappingAny.(map[string]interface{})
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"field_mapping invalid format",
			"",
		))
	}

	// Make set
	allowedSet := make(map[string]struct{})
	for _, f := range allowedFields {
		allowedSet[f] = struct{}{}
	}

	// Check key of field_mapping
	for key := range fieldMapping {
		if _, ok := allowedSet[key]; !ok {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest,
				fmt.Sprintf("Field %s is not allowed", key),
				"",
			))
		}
	}
	// End: Check if key of field_mapping is allowed

	// Create operation
	input := map[string]interface{}{
		"user_id":         user.ID.String(),
		"organization_id": organizationID.String(),
		"mapping_fields":  fieldMappingAny,
		"list_ids":        req.ListIDs,
	}

	operation := &domain.Operation{
		Name:   "contact.ImportHubspot",
		Status: domain.OperationStatusInProgress,
	}

	if err := operation.Input.Marshal(input); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to initialize operation",
			err.Error(),
		))
	}

	createdOperation, err := h.operationRepo.Create(c.Context(), operation)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to create operation",
			err.Error(),
		))
	}

	payload := schemaV2.ContactImportHubspotTask{
		OperationID: createdOperation.ID,
	}
	if _, err := h.workerClient.EnqueueLowPriorityTask(c.Context(), contactworker.TypeContactImportHubspot, payload); err != nil {
		errOutput, _ := json.Marshal(map[string]interface{}{
			"error": err.Error(),
		})
		if err2 := h.operationRepo.UpdateFailedStatusWithOutput(c.Context(), createdOperation.ID, domain.JSONB(errOutput)); err2 != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
				fiber.StatusInternalServerError,
				"Failed to update operation status to failed",
				err2.Error(),
			))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to import hubspot contacts",
			err.Error(),
		))
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"id":     createdOperation.ID,
			"status": createdOperation.Status,
		},
	})
}
