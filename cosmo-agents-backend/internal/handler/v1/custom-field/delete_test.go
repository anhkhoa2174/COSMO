package customfield

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

func TestDeleteCustomField(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	fieldID := uuid.New()
	otherUserID := uuid.New()

	t.Run("Success - Delete custom field", func(t *testing.T) {
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
			Name:           "To Delete",
			DataType:       domain.CustomFieldDataTypeText,
			EntityType:     domain.CustomFieldEntityContact,
		}, nil).Once()

		// Mock Delete
		mockCustomFieldRepo.On("Delete", mock.Anything, fieldID).Return(nil).Once()

		app.Delete("/custom-fields/:id", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.Delete(c)
		})

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/custom-fields/%s", fieldID.String()), nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		mockCustomFieldRepo.AssertExpectations(t)
	})

	t.Run("Fail - Unauthorized", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		app.Delete("/custom-fields/:id", handler.Delete)

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/custom-fields/%s", fieldID.String()), nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("Fail - Invalid ID", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		app.Delete("/custom-fields/:id", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.Delete(c)
		})

		req := httptest.NewRequest("DELETE", "/custom-fields/invalid-uuid", nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("Fail - Custom field not found", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		// With original code, FindByID returns nil when not found (not gorm.ErrRecordNotFound)
		mockCustomFieldRepo.On("FindByID", mock.Anything, fieldID).Return(nil, nil).Once()

		app.Delete("/custom-fields/:id", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.Delete(c)
		})

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/custom-fields/%s", fieldID.String()), nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 404, resp.StatusCode)

		mockCustomFieldRepo.AssertExpectations(t)
	})

	t.Run("Fail - Forbidden - not owner", func(t *testing.T) {
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
			Name:           "To Delete",
			DataType:       domain.CustomFieldDataTypeText,
			EntityType:     domain.CustomFieldEntityContact,
		}, nil).Once()

		app.Delete("/custom-fields/:id", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.Delete(c)
		})

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/custom-fields/%s", fieldID.String()), nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)

		mockCustomFieldRepo.AssertExpectations(t)
	})

	t.Run("Fail - Delete error", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		mockCustomFieldRepo.On("FindByID", mock.Anything, fieldID).Return(&domain.CustomField{
			Base:           domain.Base{ID: fieldID},
			UserID:         userID,
			OrganizationID: &orgID,
			Name:           "To Delete",
			DataType:       domain.CustomFieldDataTypeText,
			EntityType:     domain.CustomFieldEntityContact,
		}, nil).Once()

		mockCustomFieldRepo.On("Delete", mock.Anything, fieldID).Return(assert.AnError).Once()

		app.Delete("/custom-fields/:id", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.Delete(c)
		})

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/custom-fields/%s", fieldID.String()), nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 500, resp.StatusCode)

		mockCustomFieldRepo.AssertExpectations(t)
	})
}
