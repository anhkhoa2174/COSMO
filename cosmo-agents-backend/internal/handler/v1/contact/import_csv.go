package contact

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	contactService "github.com/rockship/cosmo-agents-go/internal/service/contact"
)

// ImportCSV handles POST /v1/contacts/import-csv
// @Summary Import contacts from CSV
// @Description Imports contacts from a CSV file with optional field mapping for custom fields
// @Tags Contacts
// @Accept multipart/form-data
// @Produce json
// @Param csv_file formData file true "CSV file to import"
// @Param field_mapping formData string false "JSON object mapping CSV column names to contact fields. Non-standard fields are stored in profile. Example: {\"Company Name\": \"company\", \"Custom Field\": \"profile.custom_field\"}"
// @Success 200 {object} map[string]interface{} "Import result with total, imported, and skipped counts"
// @Failure 400 {object} any "Invalid request"
// @Failure 401 {object} any "Unauthorized"
// @Failure 500 {object} any "Internal server error"
// @Security BearerAuth
// @Router /v1/contacts/import-csv [post]
func (h *Handler) ImportCSV(c fiber.Ctx) error {
	// Use auth helper
	user, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	// Get uploaded file
	fileHeader, err := c.FormFile("csv_file")
	if err != nil {
		return h.responseHelper.BadRequest(c, "csv_file is required", err)
	}

	// Validate content type
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType != "text/csv" && !strings.HasPrefix(contentType, "text/csv") && contentType != "application/vnd.ms-excel" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(
			schema.ErrorResponse(fiber.StatusUnprocessableEntity, "Only CSVs are allowed.", nil),
		)
	}

	// Parse optional field mapping
	var fieldMapping map[string]string
	if mappingStr := c.FormValue("field_mapping"); mappingStr != "" {
		if err := json.Unmarshal([]byte(mappingStr), &fieldMapping); err != nil {
			return h.responseHelper.BadRequest(c, "Invalid field_mapping JSON", err)
		}
	}

	// Save to temporary file
	tempDir := "temp"
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		if err := os.Mkdir(tempDir, 0755); err != nil {
			return h.responseHelper.InternalServerError(c, "Failed to create temp directory", err)
		}
	}

	tempFile := fmt.Sprintf("%s/%s.csv", tempDir, uuid.New().String())
	defer os.Remove(tempFile)
	if err := c.SaveFile(fileHeader, tempFile); err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to save uploaded file", err)
	}

	// Build import config
	config := contactService.ImportConfig{
		FilePath:       tempFile,
		UserID:         user.ID,
		RequiredFields: []string{"email"}, // Only email is required now
		FieldMapping:   fieldMapping,
	}

	// Set organization ID if available
	if orgID != uuid.Nil {
		config.OrganizationID = &orgID
	}

	result, err := h.csvImporter.Import(c.Context(), config)
	if err != nil {
		return h.responseHelper.InternalServerError(c, fmt.Sprintf("Failed to import CSV: %v", err), nil)
	}

	// Return detailed response
	return h.responseHelper.Success(c, fiber.Map{
		"message":       "CSV import completed",
		"total_rows":    result.TotalRows,
		"imported_rows": result.ImportedRows,
		"skipped_rows":  result.SkippedRows,
	})
}
