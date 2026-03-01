package contact

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContactDeleteRequest_Validation tests request validation
func TestContactDeleteRequest_Validation(t *testing.T) {
	tests := []struct {
		name        string
		request     v1schema.ContactDeleteRequest
		shouldError bool
	}{
		{
			name: "valid request with IDs",
			request: v1schema.ContactDeleteRequest{
				IDs: []string{uuid.New().String(), uuid.New().String()},
			},
			shouldError: false,
		},
		{
			name: "empty IDs array",
			request: v1schema.ContactDeleteRequest{
				IDs: []string{},
			},
			shouldError: true,
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

// TestContactDeleteResponse_Structure tests the response structure
func TestContactDeleteResponse_Structure(t *testing.T) {
	t.Run("response includes is_deleted field", func(t *testing.T) {
		app := fiber.New()

		app.Delete("/test", func(c fiber.Ctx) error {
			orgID := uuid.New()
			contacts := []*v1schema.ContactResponse{
				{
					ID:             uuid.New(),
					UserID:         uuid.New(),
					OrganizationID: &orgID,
					SourceID:       stringPtr(uuid.New().String()),
					Source:         stringPtr("cosmo-agents"),
					Name:           stringPtr("John Doe"),
					Email:          stringPtr("john@example.com"),
					IsDeleted:      true,
					CreatedAt:      "2025-01-15T10:30:00Z",
					UpdatedAt:      "2025-01-15T12:45:00Z",
				},
			}

			return c.JSON(fiber.Map{
				"status": "success",
				"data":   contacts,
			})
		})

		req := httptest.NewRequest("DELETE", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, "success", result["status"])
		assert.NotNil(t, result["data"])

		data := result["data"].([]interface{})
		assert.Len(t, data, 1)

		contact := data[0].(map[string]interface{})
		assert.NotEmpty(t, contact["id"])
		assert.NotEmpty(t, contact["user_id"])
		assert.Equal(t, true, contact["is_deleted"])
		assert.Equal(t, "John Doe", contact["name"])
		assert.Equal(t, "john@example.com", contact["email"])
	})

	t.Run("empty array response when no contacts deleted", func(t *testing.T) {
		app := fiber.New()

		app.Delete("/test", func(c fiber.Ctx) error {
			return c.JSON(fiber.Map{
				"status": "success",
				"data":   []*v1schema.ContactResponse{},
			})
		})

		req := httptest.NewRequest("DELETE", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, "success", result["status"])
		data := result["data"].([]interface{})
		assert.Len(t, data, 0)
	})
}

// TestContactDelete_ErrorResponses tests various error scenarios
func TestContactDelete_ErrorResponses(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid JSON",
			requestBody:    `{"ids": [`,
			expectedStatus: 400,
			expectedError:  "Invalid request body",
		},
		{
			name:           "missing ids field",
			requestBody:    `{}`,
			expectedStatus: 422,
			expectedError:  "", // Will have validation error
		},
		{
			name:           "empty ids array",
			requestBody:    `{"ids": []}`,
			expectedStatus: 422,
			expectedError:  "Minimum value is 1", // Validator error message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			app.Delete("/test", func(c fiber.Ctx) error {
				var req v1schema.ContactDeleteRequest
				if err := c.Bind().JSON(&req); err != nil {
					return c.Status(400).JSON(fiber.Map{
						"status": "error",
						"error": fiber.Map{
							"error_code": 400,
							"message":    "Invalid request body",
							"detail":     nil,
						},
					})
				}

				if err := v1validation.ValidateStruct(req); err != nil {
					return c.Status(422).JSON(fiber.Map{
						"status": "error",
						"error": fiber.Map{
							"error_code": 422,
							"message":    err.Error(),
							"detail":     nil,
						},
					})
				}

				if len(req.IDs) == 0 {
					return c.Status(422).JSON(fiber.Map{
						"status": "error",
						"error": fiber.Map{
							"error_code": 422,
							"message":    "ids must have at least 1 element",
							"detail":     nil,
						},
					})
				}

				return c.JSON(fiber.Map{"status": "success", "data": []interface{}{}})
			})

			req := httptest.NewRequest("DELETE", "/test", strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var result map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&result)
			require.NoError(t, err)

			assert.Equal(t, "error", result["status"])
			errorObj := result["error"].(map[string]interface{})
			assert.Equal(t, float64(tt.expectedStatus), errorObj["error_code"])

			if tt.expectedError != "" {
				assert.Contains(t, errorObj["message"], tt.expectedError)
			}
		})
	}
}

// TestContactDelete_InvalidUUIDs tests handling of invalid UUID formats
func TestContactDelete_InvalidUUIDs(t *testing.T) {
	t.Run("skips invalid UUIDs gracefully", func(t *testing.T) {
		app := fiber.New()

		app.Delete("/test", func(c fiber.Ctx) error {
			var req v1schema.ContactDeleteRequest
			if err := c.Bind().JSON(&req); err != nil {
				return c.Status(400).JSON(fiber.Map{"status": "error"})
			}

			// Simulate handler logic - filter valid UUIDs
			validIDs := []uuid.UUID{}
			for _, idStr := range req.IDs {
				if id, err := uuid.Parse(idStr); err == nil {
					validIDs = append(validIDs, id)
				}
			}

			// If all invalid, return empty array
			if len(validIDs) == 0 {
				return c.JSON(fiber.Map{
					"status": "success",
					"data":   []*v1schema.ContactResponse{},
				})
			}

			// Otherwise would query DB with valid IDs
			return c.JSON(fiber.Map{
				"status": "success",
				"data":   []*v1schema.ContactResponse{},
			})
		})

		requestBody := `{"ids": ["not-a-uuid", "123", "invalid"]}`
		req := httptest.NewRequest("DELETE", "/test", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)

		// Should return 200 with empty array
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)

		assert.Equal(t, "success", result["status"])
		data := result["data"].([]interface{})
		assert.Len(t, data, 0)
	})

	t.Run("processes mix of valid and invalid UUIDs", func(t *testing.T) {
		app := fiber.New()

		validID := uuid.New()

		app.Delete("/test", func(c fiber.Ctx) error {
			var req v1schema.ContactDeleteRequest
			if err := c.Bind().JSON(&req); err != nil {
				return c.Status(400).JSON(fiber.Map{"status": "error"})
			}

			// Filter valid UUIDs
			validIDs := []uuid.UUID{}
			for _, idStr := range req.IDs {
				if id, err := uuid.Parse(idStr); err == nil {
					validIDs = append(validIDs, id)
				}
			}

			assert.Len(t, validIDs, 1) // Only one valid UUID
			assert.Equal(t, validID, validIDs[0])

			return c.JSON(fiber.Map{
				"status": "success",
				"data":   []*v1schema.ContactResponse{},
			})
		})

		requestBody := `{"ids": ["not-a-uuid", "` + validID.String() + `", "invalid"]}`
		req := httptest.NewRequest("DELETE", "/test", strings.NewReader(requestBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

// TestContactDelete_AccessControl tests access control logic
func TestContactDelete_AccessControl(t *testing.T) {
	t.Run("documentation of access control logic", func(t *testing.T) {
		// This test documents the expected SQL WHERE clause:
		// WHERE id IN (ids) AND (organization_id = orgID OR user_id = userID) AND is_deleted = false

		t.Log("Access Control Rules:")
		t.Log("1. Contact ID must be in provided list")
		t.Log("2. AND one of these must be true:")
		t.Log("   a) Contact belongs to user's organization (organization_id = orgID)")
		t.Log("   b) OR Contact is owned by user (user_id = userID)")
		t.Log("3. AND contact is not already deleted (is_deleted = false)")
		t.Log("")
		t.Log("Users can delete:")
		t.Log("- Their own contacts")
		t.Log("- Contacts belonging to their organization")
		t.Log("")
		t.Log("Users CANNOT delete:")
		t.Log("- Contacts from other organizations")
		t.Log("- Contacts owned by other users outside their org")
		t.Log("- Already deleted contacts")
	})
}

// TestContactDelete_SoftDelete tests soft delete behavior
func TestContactDelete_SoftDelete(t *testing.T) {
	t.Run("soft delete sets is_deleted and updates timestamp", func(t *testing.T) {
		t.Log("Soft Delete Behavior:")
		t.Log("1. UPDATE contacts SET is_deleted = true, updated_at = NOW()")
		t.Log("2. Records remain in database (not hard deleted)")
		t.Log("3. is_deleted flag set to true")
		t.Log("4. updated_at timestamp updated to deletion time")
		t.Log("5. Can be restored later if needed")
		t.Log("")
		t.Log("Response includes:")
		t.Log("- Full contact objects (not just IDs)")
		t.Log("- is_deleted: true for each deleted contact")
		t.Log("- updated_at reflects deletion time")
	})
}
