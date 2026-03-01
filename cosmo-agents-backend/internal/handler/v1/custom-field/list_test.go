package customfield

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

func TestListCustomFields(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	field1ID := uuid.New()
	field2ID := uuid.New()

	t.Run("Success - List all custom fields", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		mockCustomFieldRepo.On("FindAll", mock.Anything, mock.MatchedBy(func(f baseRepo.Filter) bool {
			return f["user_id"] == userID.String()
		}), mock.MatchedBy(func(p *baseRepo.PaginationParams) bool {
			return p.Offset == 0 && p.Limit == 25
		})).Return(&baseRepo.PaginatedResult[domain.CustomField]{
			List: []domain.CustomField{
				{
					Base:           domain.Base{ID: field1ID},
					UserID:         userID,
					OrganizationID: &orgID,
					Name:           "Company Size",
					DataType:       domain.CustomFieldDataTypeSelect,
					EntityType:     domain.CustomFieldEntityContact,
				},
				{
					Base:           domain.Base{ID: field2ID},
					UserID:         userID,
					OrganizationID: &orgID,
					Name:           "Department",
					DataType:       domain.CustomFieldDataTypeText,
					EntityType:     domain.CustomFieldEntityContact,
				},
			},
			Total:  2,
			Offset: 0,
			Limit:  25,
		}, nil).Once()

		app.Get("/custom-fields", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.List(c)
		})

		req := httptest.NewRequest("GET", "/custom-fields", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		mockCustomFieldRepo.AssertExpectations(t)
	})

	t.Run("Success - Filter by entity_type", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		mockCustomFieldRepo.On("FindAll", mock.Anything, mock.MatchedBy(func(f baseRepo.Filter) bool {
			return f["user_id"] == userID.String() && f["entity_type"] == "company"
		}), mock.Anything).Return(&baseRepo.PaginatedResult[domain.CustomField]{
			List: []domain.CustomField{
				{
					Base:           domain.Base{ID: field1ID},
					UserID:         userID,
					OrganizationID: &orgID,
					Name:           "Industry",
					DataType:       domain.CustomFieldDataTypeText,
					EntityType:     domain.CustomFieldEntityCompany,
				},
			},
			Total:  1,
			Offset: 0,
			Limit:  25,
		}, nil).Once()

		app.Get("/custom-fields", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.List(c)
		})

		req := httptest.NewRequest("GET", "/custom-fields?entity_type=company", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		mockCustomFieldRepo.AssertExpectations(t)
	})

	t.Run("Success - Pagination", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		mockCustomFieldRepo.On("FindAll", mock.Anything, mock.Anything, mock.MatchedBy(func(p *baseRepo.PaginationParams) bool {
			return p.Offset == 10 && p.Limit == 10
		})).Return(&baseRepo.PaginatedResult[domain.CustomField]{
			List:   []domain.CustomField{},
			Total:  15,
			Offset: 10,
			Limit:  10,
		}, nil).Once()

		app.Get("/custom-fields", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.List(c)
		})

		req := httptest.NewRequest("GET", "/custom-fields?offset=10&limit=10", nil)
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

		app.Get("/custom-fields", handler.List)

		req := httptest.NewRequest("GET", "/custom-fields", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("Fail - Repository error", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		mockCustomFieldRepo.On("FindAll", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, assert.AnError).Once()

		app.Get("/custom-fields", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.List(c)
		})

		req := httptest.NewRequest("GET", "/custom-fields", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		assert.Equal(t, 500, resp.StatusCode)

		mockCustomFieldRepo.AssertExpectations(t)
	})
}
