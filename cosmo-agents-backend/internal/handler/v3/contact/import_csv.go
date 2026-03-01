package contact

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// ImportCSV handles POST /v3/contacts/import (enhanced with field mapping)
func (h *Handler) ImportCSV(c fiber.Ctx) error {
	// Use common auth helper
	userID, organizationID, err := h.commonAuthHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.Unauthorized(c, "Unauthorized", nil)
	}

	// Validate and get uploaded file using common file helper
	file, err := h.fileHelper.GetUploadedFile(c, "file")
	if err != nil {
		return h.responseHelper.BadRequest(c, "Missing or invalid file", err.Error())
	}

	// Validate file type using common file helper
	if err := h.fileHelper.ValidateFileType(file, []string{".csv"}); err != nil {
		return h.responseHelper.BadRequest(c, err.Error(), nil)
	}

	// Validate file size (10MB max)
	if err := h.fileHelper.ValidateFileSize(file, 10*1024*1024); err != nil {
		return h.responseHelper.BadRequest(c, err.Error(), nil)
	}

	// Parse field mappings from form data
	metadata := h.parseFieldMappings(c, file.Filename)

	// Validate field mappings
	var orgID uuid.UUID
	if organizationID != nil {
		orgID = *organizationID
	}
	customFields, err := h.customFieldRepo.FindByOrganizationID(c.Context(), orgID)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to get custom fields")
		return h.responseHelper.InternalServerError(c, "Failed to validate fields", err.Error())
	}

	// Build allowed fields using field mapper service
	allowedFields := h.fieldMapper.BuildAllowedFields(customFields)

	// Validate metadata keys
	for key := range metadata {
		if key != "name_import" {
			if _, ok := allowedFields[key]; !ok {
				return h.responseHelper.BadRequest(c, fmt.Sprintf("Field '%s' is not allowed", key), nil)
			}
		}
	}

	// Save uploaded file to temp directory using common file helper
	tempPath, tempFileName, err := h.fileHelper.SaveToTempFile(file, "cosmo-contact-imports")
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to persist uploaded file")
		return h.responseHelper.InternalServerError(c, "Failed to persist uploaded file", err.Error())
	}

	// Build mapping fields (exclude name_import)
	mappingFields := make(map[string]string)
	for key, value := range metadata {
		if key != "name_import" {
			mappingFields[key] = value
		}
	}

	// Create operation for async processing with timeout
	operationTimeout := 30 * time.Minute // 30 minutes timeout for CSV import
	input := map[string]interface{}{
		"user_id":           userID.String(),
		"organization_id":   organizationID.String(),
		"metadata":          metadata,
		"mapping_fields":    mappingFields,
		"file_path":         tempPath,
		"original_filename": file.Filename,
		"temp_filename":     tempFileName,
		"name_import":       metadata["name_import"],
		"uploaded_at":       time.Now().UTC().Format(time.RFC3339Nano),
		"timeout_seconds":   int64(operationTimeout.Seconds()),
		"deadline":          time.Now().Add(operationTimeout).UTC().Format(time.RFC3339Nano),
	}

	// Create context with timeout for operation creation
	ctx, cancel := context.WithTimeout(c.Context(), operationTimeout)
	defer cancel()

	operation := &domain.Operation{
		Name:   "machine.controllers.v3.contact.import_csv",
		Status: domain.OperationStatusInProgress,
	}

	if err := operation.Input.Marshal(input); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to marshal operation input")
		return h.responseHelper.InternalServerError(c, "Failed to initialise import operation", err.Error())
	}

	// Create operation with timeout context
	createdOperation, err := h.operationRepo.Create(ctx, operation)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logger.Logger.Error().Err(err).Msg("Operation creation timed out")
			return h.responseHelper.InternalServerError(c, "Import operation timed out", err.Error())
		}
		logger.Logger.Error().Err(err).Msg("Failed to create operation")
		return h.responseHelper.InternalServerError(c, "Failed to initialise import operation", err.Error())
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

	responseData := fiber.Map{
		"id":         createdOperation.ID,
		"name":       createdOperation.Name,
		"status":     string(createdOperation.Status),
		"input":      responseInput,
		"output":     responseOutput,
		"created_at": createdOperation.CreatedAt.Format(time.RFC3339Nano),
		"updated_at": createdOperation.UpdatedAt.Format(time.RFC3339Nano),
	}

	return h.responseHelper.Success(c, responseData)
}

// parseFieldMappings extracts field mappings from form data
func (h *Handler) parseFieldMappings(c fiber.Ctx, filename string) map[string]string {
	metadata := make(map[string]string)

	// Parse name_import
	nameImport := c.FormValue("name_import")
	if nameImport != "" {
		metadata["name_import"] = nameImport
	} else {
		metadata["name_import"] = filename
	}

	// Parse field mappings
	fieldMappings := []string{
		"first_name", "last_name", "email", "phone", "company",
		"job_title", "address", "city", "country", "state", "zip",
	}

	for _, field := range fieldMappings {
		value := c.FormValue(field)
		if value != "" {
			metadata[field] = value
		}
	}

	return metadata
}
