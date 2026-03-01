package contact

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	contactService "github.com/rockship/cosmo-agents-go/internal/service/contact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateContactRequest_UnmarshalJSON tests that extra fields are captured
func TestCreateContactRequest_UnmarshalJSON(t *testing.T) {
	t.Run("captures extra fields for custom field support", func(t *testing.T) {
		jsonData := `{
			"name": "John Doe",
			"email": "john@example.com",
			"custom_field_1": "value1",
			"custom_field_2": 123
		}`

		var req v1schema.CreateContactRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "John Doe", req.Name)
		assert.Equal(t, "john@example.com", req.Email)
		assert.Len(t, req.ExtraFields, 2)
		assert.Equal(t, "value1", req.ExtraFields["custom_field_1"])
		assert.Equal(t, float64(123), req.ExtraFields["custom_field_2"])
	})

	t.Run("handles standard fields only", func(t *testing.T) {
		jsonData := `{
			"name": "Jane Smith",
			"email": "jane@example.com"
		}`

		var req v1schema.CreateContactRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "Jane Smith", req.Name)
		assert.Len(t, req.ExtraFields, 0)
	})
}

// TestCreateContactRequest_Validation tests required field validation
func TestCreateContactRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		request     v1schema.CreateContactRequest
		shouldError bool
		errorField  string
	}{
		{
			name: "valid request with all required fields",
			request: v1schema.CreateContactRequest{
				Name:  "John Doe",
				Email: "john@example.com",
			},
			shouldError: false,
		},
		{
			name: "missing name",
			request: v1schema.CreateContactRequest{
				Email: "john@example.com",
			},
			shouldError: true,
			errorField:  "Name",
		},
		{
			name: "missing email",
			request: v1schema.CreateContactRequest{
				Name: "John Doe",
			},
			shouldError: false,
		},
		{
			name: "invalid email format",
			request: v1schema.CreateContactRequest{
				Name:  "John Doe",
				Email: "not-an-email",
			},
			shouldError: true,
			errorField:  "Email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v1validation.ValidateStruct(tt.request)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestContactResponse_Structure tests the response matches spec
func TestContactResponse_Structure(t *testing.T) {
	t.Run("response has correct JSON structure", func(t *testing.T) {
		app := fiber.New()

		app.Post("/test", func(c fiber.Ctx) error {
			orgID := uuid.New()
			response := v1schema.ContactResponse{
				ID:             uuid.New(),
				UserID:         uuid.New(),
				OrganizationID: &orgID,
				SourceID:       stringPtr(uuid.New().String()),
				Source:         stringPtr("cosmo-agents"),
				Name:           stringPtr("John Doe"),
				Email:          stringPtr("john@example.com"),
				CreatedAt:      "2025-01-15T10:30:00Z",
				UpdatedAt:      "2025-01-15T10:30:00Z",
			}

			return c.JSON(fiber.Map{
				"status": "success",
				"data":   response,
			})
		})

		req := httptest.NewRequest("POST", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, "success", result["status"])
		assert.NotNil(t, result["data"])

		data := result["data"].(map[string]interface{})
		assert.NotEmpty(t, data["id"])
		assert.NotEmpty(t, data["user_id"])
		assert.NotEmpty(t, data["organization_id"])
		assert.Equal(t, "cosmo-agents", data["source"])
		assert.Equal(t, "John Doe", data["name"])
		assert.Equal(t, "john@example.com", data["email"])
	})
}

// TestErrorResponse_Structure tests error response matches spec
func TestErrorResponse_Structure(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
		detail     interface{}
	}{
		{
			name:       "400 Bad Request",
			statusCode: 400,
			message:    "Invalid request body",
			detail:     nil,
		},
		{
			name:       "401 Unauthorized",
			statusCode: 401,
			message:    "You are not authorized to access this resource",
			detail:     nil,
		},
		{
			name:       "422 Validation Error",
			statusCode: 422,
			message:    "Field name not found",
			detail:     nil,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: 500,
			message:    "Internal server error",
			detail:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			app.Get("/test", func(c fiber.Ctx) error {
				return c.Status(tt.statusCode).JSON(fiber.Map{
					"status": "error",
					"error": fiber.Map{
						"error_code": tt.statusCode,
						"message":    tt.message,
						"detail":     tt.detail,
					},
				})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.statusCode, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.Equal(t, "error", result["status"])
			assert.NotNil(t, result["error"])

			errorObj := result["error"].(map[string]interface{})
			assert.Equal(t, float64(tt.statusCode), errorObj["error_code"])
			assert.Equal(t, tt.message, errorObj["message"])
		})
	}
}

// TestStandardContactFields verifies the field map is complete
func TestStandardContactFields(t *testing.T) {
	// Use the field validator service instead of the removed function
	validator := contactService.NewContactFieldValidator(nil)
	fields := validator.GetStandardFields()

	// Convert to map for easier lookup
	fieldMap := make(map[string]bool)
	for _, field := range fields {
		fieldMap[field] = true
	}

	// Note: "id" is not included because it has json:"-" tag in domain.Contact
	// The field validator uses reflection on JSON tags
	requiredFields := []string{
		"user_id", "organization_id", "source_id", "source",
		"name", "email", "phone", "company",
		"job_title", "address", "city", "country", "state", "zip",
		"created_at", "updated_at", "profile", "tags", "attributes",
	}

	for _, field := range requiredFields {
		assert.True(t, fieldMap[field], "Field %s should be in standard fields", field)
	}

	// Verify we have a reasonable number of fields
	assert.GreaterOrEqual(t, len(fields), 20, "Should have at least 20 standard fields")
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
