package contact

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	contactService "github.com/rockship/cosmo-agents-go/internal/service/contact"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// importTimeout bounds one import; a file too large to finish in it fails
// instead of leaving the operation in progress forever.
const importTimeout = 30 * time.Minute

// importRequiredFields must be mapped to a column; email is the duplicate key
// and the outreach address.
var importRequiredFields = []string{"email"}

// ImportCSV handles POST /v3/contacts/import (enhanced with field mapping).
//
// The form carries the file, an optional name_import, and one value per
// contact field naming the CSV column it comes from. The import runs in the
// background; the response is an operation the page polls at
// /v3/operations/:id until it succeeds or fails.
func (h *Handler) ImportCSV(c fiber.Ctx) error {
	userID, err := h.commonAuthHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.Unauthorized(c, "Unauthorized", nil)
	}

	// The auth middleware does not put an organisation in the context, so it
	// is looked up here, the same way the v1 handlers resolve it.
	orgID, err := h.roleRepo.FindPrimaryOrganization(c.Context(), userID)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to resolve organization")
		return h.responseHelper.InternalServerError(c, "Failed to resolve organization", nil)
	}

	file, err := h.fileHelper.GetUploadedFile(c, "file")
	if err != nil {
		return h.responseHelper.BadRequest(c, "Missing or invalid file", err.Error())
	}
	if err := h.fileHelper.ValidateFileType(file, []string{".csv"}); err != nil {
		return h.responseHelper.BadRequest(c, err.Error(), nil)
	}
	if err := h.fileHelper.ValidateFileSize(file, 10*1024*1024); err != nil {
		return h.responseHelper.BadRequest(c, err.Error(), nil)
	}

	nameImport, mapping, err := h.parseFieldMappings(c, file.Filename)
	if err != nil {
		return h.responseHelper.BadRequest(c, err.Error(), nil)
	}

	// The import page lists the user's own custom fields, which may be stored
	// without an organisation or under another one, so both sets are allowed.
	var customFields []domain.CustomField
	if orgID != nil {
		customFields, err = h.customFieldRepo.FindByOrganizationID(c.Context(), *orgID)
		if err != nil {
			logger.Logger.Error().Err(err).Msg("Failed to get custom fields")
			return h.responseHelper.InternalServerError(c, "Failed to validate fields", nil)
		}
	}
	own, err := h.customFieldRepo.FindAll(c.Context(), baseRepo.Filter{"user_id": userID}, &baseRepo.PaginationParams{Offset: 0, Limit: 1000})
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to get the user's custom fields")
		return h.responseHelper.InternalServerError(c, "Failed to validate fields", nil)
	}
	customFields = append(customFields, own.List...)
	allowedFields := h.fieldMapper.BuildAllowedFields(customFields)
	for _, field := range importOnlyFields {
		allowedFields[field] = struct{}{}
	}
	for field := range mapping {
		if _, ok := allowedFields[field]; !ok {
			return h.responseHelper.BadRequest(c, fmt.Sprintf("Field '%s' is not allowed", field), nil)
		}
	}
	// The importer maps each column to one field, so a column picked for two
	// fields would keep one of them at random and drop the other.
	fieldForColumn := make(map[string]string, len(mapping))
	for field, column := range mapping {
		key := strings.ToLower(strings.TrimSpace(column))
		if prev, dup := fieldForColumn[key]; dup {
			return h.responseHelper.BadRequest(c, fmt.Sprintf("Column '%s' is mapped to both '%s' and '%s'; pick one", column, prev, field), nil)
		}
		fieldForColumn[key] = field
	}
	for _, field := range importRequiredFields {
		if mapping[field] == "" {
			return h.responseHelper.BadRequest(c, fmt.Sprintf("Select a column for '%s'", field), nil)
		}
	}

	tempPath, tempFileName, err := h.fileHelper.SaveToTempFile(file, "cosmo-contact-imports")
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to persist uploaded file")
		return h.responseHelper.InternalServerError(c, "Failed to persist uploaded file", nil)
	}

	now := time.Now().UTC()
	input := map[string]interface{}{
		"user_id":           userID.String(),
		"mapping_fields":    mapping,
		"original_filename": file.Filename,
		"temp_filename":     tempFileName,
		"name_import":       nameImport,
		"uploaded_at":       now.Format(time.RFC3339Nano),
		"timeout_seconds":   int64(importTimeout.Seconds()),
		"deadline":          now.Add(importTimeout).Format(time.RFC3339Nano),
	}
	if orgID != nil {
		input["organization_id"] = orgID.String()
	}

	operation := &domain.Operation{
		Name:   "machine.controllers.v3.contact.import_csv",
		Status: domain.OperationStatusInProgress,
	}
	if err := operation.Input.Marshal(input); err != nil {
		_ = os.Remove(tempPath)
		return h.responseHelper.InternalServerError(c, "Failed to initialise import operation", nil)
	}
	createdOperation, err := h.operationRepo.Create(c.Context(), operation)
	if err != nil {
		_ = os.Remove(tempPath)
		logger.Logger.Error().Err(err).Msg("Failed to create operation")
		return h.responseHelper.InternalServerError(c, "Failed to initialise import operation", nil)
	}

	// The file maps column -> field, the reverse of the form.
	columnMapping := make(map[string]string, len(mapping))
	for field, column := range mapping {
		columnMapping[column] = field
	}
	config := contactService.ImportConfig{
		FilePath:        tempPath,
		UserID:          userID,
		OrganizationID:  orgID,
		RequiredFields:  importRequiredFields,
		FieldMapping:    columnMapping,
		ExplicitMapping: true,
	}
	h.runAsync(func() { h.runImport(createdOperation.ID, config) })

	return h.responseHelper.Success(c, fiber.Map{
		"id":         createdOperation.ID,
		"name":       createdOperation.Name,
		"status":     string(createdOperation.Status),
		"input":      input,
		"output":     nil,
		"created_at": createdOperation.CreatedAt.Format(time.RFC3339Nano),
		"updated_at": createdOperation.UpdatedAt.Format(time.RFC3339Nano),
	})
}

// runImport imports the saved file and records the outcome on the operation.
// It runs detached from the request, so it has its own deadline, and it always
// settles the operation: the page polls until it leaves in_progress.
func (h *Handler) runImport(operationID uuid.UUID, config contactService.ImportConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), importTimeout)
	defer cancel()
	defer os.Remove(config.FilePath)

	log := logger.Logger.With().Str("operation_id", operationID.String()).Logger()
	// Status writes get a context of their own: when the import ran out of
	// time, ctx is already done, and writing "failed" with it failed too and
	// left the page polling an operation stuck in progress.
	statusCtx := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.Background(), 30*time.Second)
	}
	fail := func(msg string) {
		out, _ := json.Marshal(map[string]string{"error": msg})
		sctx, scancel := statusCtx()
		defer scancel()
		if err := h.operationRepo.UpdateFailedStatusWithOutput(sctx, operationID, out); err != nil {
			log.Error().Err(err).Msg("Failed to mark CSV import as failed")
		}
	}
	defer func() {
		if r := recover(); r != nil {
			log.Error().Interface("panic", r).Msg("CSV import panicked")
			fail("Import failed unexpectedly")
		}
	}()

	result, err := h.csvImporter.Import(ctx, config)
	if err != nil {
		log.Warn().Err(err).Msg("CSV import failed")
		fail(err.Error())
		return
	}

	out, _ := json.Marshal(map[string]interface{}{
		"total_rows":       result.TotalRows,
		"imported_rows":    result.ImportedRows,
		"skipped_rows":     result.SkippedRows,
		"rejected_rows":    result.RejectedRows,
		"rejected_reasons": result.RejectedReasons,
	})
	sctx, scancel := statusCtx()
	defer scancel()
	if err := h.operationRepo.UpdateOutput(sctx, operationID, out); err != nil {
		log.Error().Err(err).Msg("Failed to save CSV import result")
	}
	if err := h.operationRepo.UpdateStatus(sctx, operationID, domain.OperationStatusSuccess); err != nil {
		log.Error().Err(err).Msg("Failed to mark CSV import as done")
		// Never leave the page polling forever: the rows are in, say so.
		fail("Import finished but its status could not be saved; refresh the contact list")
	}
	log.Info().Int("imported", result.ImportedRows).Int("rejected", result.RejectedRows).Msg("CSV import finished")
}

// importOnlyFields are contact fields the importer fills that the shared field
// mapper does not list.
var importOnlyFields = []string{"name", "industry", "contact_channel"}

// parseFieldMappings reads name_import and the field -> column pairs from the
// form. Every form value other than the file and name_import is a mapping,
// so custom fields come through as well as the standard ones; before, only a
// fixed list of standard fields was read and custom-field columns were dropped.
func (h *Handler) parseFieldMappings(c fiber.Ctx, filename string) (string, map[string]string, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return "", nil, fmt.Errorf("invalid form data")
	}
	nameImport := strings.TrimSpace(c.FormValue("name_import"))
	if nameImport == "" {
		nameImport = filename
	}
	mapping := make(map[string]string)
	for key, values := range form.Value {
		field := strings.ToLower(strings.TrimSpace(key))
		if field == "name_import" || field == "file" || len(values) == 0 {
			continue
		}
		if column := strings.TrimSpace(values[0]); column != "" {
			mapping[field] = column
		}
	}
	return nameImport, mapping, nil
}
