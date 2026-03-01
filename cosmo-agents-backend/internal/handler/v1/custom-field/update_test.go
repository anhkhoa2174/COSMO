package customfield

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

func TestUpdateCustomField(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	fieldID := uuid.New()
	otherUserID := uuid.New()

	t.Run("Success - Update name only", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		// Mock FindByID
		mockCustomFieldRepo.On("FindByID", mock.Anything, fieldID).Return(&domain.CustomField{
			Base:           domain.Base{ID: fieldID},
			UserID:         userID,
			OrganizationID: &orgID,
			Name:           "Old Name",
			DataType:       domain.CustomFieldDataTypeText,
			EntityType:     domain.CustomFieldEntityContact,
		}, nil).Once()

		// Mock UpdateFields
		newName := "New Name"
		mockCustomFieldRepo.On("UpdateFields", mock.Anything, fieldID, mock.MatchedBy(func(updates map[string]interface{}) bool {
			return updates["name"] == newName
		})).Return(nil).Once()

		// Mock FindByID after update
		mockCustomFieldRepo.On("FindByID", mock.Anything, fieldID).Return(&domain.CustomField{
			Base:           domain.Base{ID: fieldID},
			UserID:         userID,
			OrganizationID: &orgID,
			Name:           newName,
			DataType:       domain.CustomFieldDataTypeText,
			EntityType:     domain.CustomFieldEntityContact,
		}, nil).Once()

		app.Patch("/custom-fields/:id", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.Update(c)
		})

		reqBody := v1schema.UpdateCustomFieldRequest{
			Name: &newName,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", fmt.Sprintf("/custom-fields/%s", fieldID.String()), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		mockCustomFieldRepo.AssertExpectations(t)
	})

	t.Run("Fail - Not found", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		// With original code, FindByID returns nil when not found
		mockCustomFieldRepo.On("FindByID", mock.Anything, fieldID).Return(nil, nil).Once()

		app.Patch("/custom-fields/:id", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.Update(c)
		})

		newName := "New Name"
		reqBody := v1schema.UpdateCustomFieldRequest{
			Name: &newName,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", fmt.Sprintf("/custom-fields/%s", fieldID.String()), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode)

		mockCustomFieldRepo.AssertExpectations(t)
	})

	t.Run("Fail - Forbidden", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		// Mock FindByID - custom field belongs to different user
		mockCustomFieldRepo.On("FindByID", mock.Anything, fieldID).Return(&domain.CustomField{
			Base:           domain.Base{ID: fieldID},
			UserID:         otherUserID, // Different user
			OrganizationID: &orgID,
			Name:           "Old Name",
			DataType:       domain.CustomFieldDataTypeText,
			EntityType:     domain.CustomFieldEntityContact,
		}, nil).Once()

		app.Patch("/custom-fields/:id", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.Update(c)
		})

		newName := "New Name"
		reqBody := v1schema.UpdateCustomFieldRequest{
			Name: &newName,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("PATCH", fmt.Sprintf("/custom-fields/%s", fieldID.String()), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)

		mockCustomFieldRepo.AssertExpectations(t)
	})
}
