package contact

import (
	"bytes"
	"encoding/csv"
	"io"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// FieldMapping represents a single field mapping
type FieldMapping struct {
	Name    string      `json:"name"`
	Mapping interface{} `json:"mapping"` // string for system fields, nil for custom fields
}

// ExtractHeadersResponse represents the response structure for the ExtractCSVHeaders endpoint
type ExtractHeadersResponse struct {
	System []FieldMapping `json:"system"`
	Custom []FieldMapping `json:"custom"`
}

var CONTACT_REVERSE_MAPPING = map[string][]string{
	"first_name": {"first_name", "First Name", "fname", "Firstname", "first_name", "firstname"},
	"last_name":  {"last_name", "Last Name", "LName", "surname", "lastname"},
	"email":      {"email", "Email"},
	"phone":      {"phone", "Phone", "telephone", "telephone_number", "phone_number"},
	"company":    {"company", "Company", "organization", "organization_name", "company_name", "corporation_name", "corporation"},
	"job_title":  {"job_title", "Job Title"},
	"address":    {"address", "Address"},
	"city":       {"city", "City"},
	"country":    {"country", "Country"},
	"state":      {"state", "State"},
	"zip":        {"zip", "Zip"},
	"source_id":  {"source_id", "id"},
}

// ExtractCSVHeaders handles POST /v2/contacts/import/extract-csv-headers
// @Summary Extract CSV Headers
// @Description Extracts headers from a CSV file and maps them to system fields
// @Tags V2 Contacts
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "CSV file to process"
// @Success 200 {object} schema.APIResponse[ExtractHeadersResponse] "Successfully extracted headers"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request - Invalid or missing file"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Security BearerAuth
// @Router /v2/contacts/import/extract-csv-headers [post]
func (h *Handler) ExtractCSVHeaders(c fiber.Ctx) error {
	// Use auth helper
	_, err := h.authHelper.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			nil,
		))
	}

	// Get uploaded file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"file is required",
			nil,
		))
	}

	// Check if the file is a CSV
	filename := fileHeader.Filename
	if len(filename) < 4 || filename[len(filename)-4:] != ".csv" {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Only CSV files are allowed",
			nil,
		))
	}

	// Open file
	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to open file",
			err.Error(),
		))
	}
	defer file.Close()

	// Read the entire file content
	content, err := io.ReadAll(file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to read file content",
			err.Error(),
		))
	}
	if len(content) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"CSV file is empty",
			nil,
		))
	}

	// Read CSV headers
	reader := csv.NewReader(bytes.NewReader(content))
	headers, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest,
				"CSV file is empty",
				nil,
			))
		}
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Error parsing CSV file. Please check the file format",
			err.Error(),
		))
	}

	// Initialize response
	response := ExtractHeadersResponse{
		System: make([]FieldMapping, 0),
		Custom: make([]FieldMapping, 0),
	}

	// Process each header
	for _, header := range headers {
		// Remove BOM if exists
		header = strings.TrimLeft(header, "\uFEFF")
		// Normalize header for comparison
		normalizedHeader := strings.ToLower(strings.TrimSpace(header))
		normalizedHeader = strings.ReplaceAll(normalizedHeader, " ", "_")

		mapped := false

		for stdField, aliases := range CONTACT_REVERSE_MAPPING {
			for _, alias := range aliases {
				normalizedAlias := strings.ToLower(strings.TrimSpace(alias))
				normalizedAlias = strings.ReplaceAll(normalizedAlias, " ", "_")

				if normalizedHeader == normalizedAlias {
					response.System = append(response.System, FieldMapping{
						Name:    strings.TrimSpace(header),
						Mapping: stdField,
					})
					mapped = true
					break
				}
			}
			if mapped {
				break
			}
		}

		if !mapped {
			response.Custom = append(response.Custom, FieldMapping{
				Name:    strings.TrimSpace(header),
				Mapping: nil,
			})
		}
	}

	return c.JSON(schema.SuccessResponse(response))
}
