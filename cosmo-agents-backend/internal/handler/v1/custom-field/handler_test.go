package customfield

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// MockCustomFieldRepository implements the CustomFieldRepository interface with mock functionality
type MockCustomFieldRepository struct {
	mock.Mock
}

func NewMockCustomFieldRepository() *MockCustomFieldRepository {
	return &MockCustomFieldRepository{}
}

func (m *MockCustomFieldRepository) Create(ctx context.Context, field *domain.CustomField) (*domain.CustomField, error) {
	args := m.Called(ctx, field)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CustomField), args.Error(1)
}

func (m *MockCustomFieldRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.CustomField, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CustomField), args.Error(1)
}

func (m *MockCustomFieldRepository) FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.CustomField], error) {
	args := m.Called(ctx, filter, pagination)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*baseRepo.PaginatedResult[domain.CustomField]), args.Error(1)
}

func (m *MockCustomFieldRepository) UpdateFields(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockCustomFieldRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockUserRepository embeds the real repository and adds mock methods
type MockUserRepository struct {
	mock.Mock
	*user.UserRepository
}

func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{}
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

// MockRoleRepository embeds the real repository and adds mock methods
type MockRoleRepository struct {
	mock.Mock
	*roleRepo.RoleRepository
}

func NewMockRoleRepository() *MockRoleRepository {
	return &MockRoleRepository{}
}

func (m *MockRoleRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Role), args.Error(1)
}

func setupTestApp() *fiber.App {
	app := fiber.New()
	return app
}

func TestCreateCustomField(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	fieldID := uuid.New()

	t.Run("Success - Create with organization", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		// Mock role to return organization_id
		mockRoleRepo.On("FindByUserID", mock.Anything, userID).Return([]domain.Role{
			{
				Base:           domain.Base{ID: uuid.New()},
				UserID:         userID,
				OrganizationID: orgID,
				Status:         "active",
			},
		}, nil).Once()

		// Mock create
		mockCustomFieldRepo.On("Create", mock.Anything, mock.MatchedBy(func(cf *domain.CustomField) bool {
			return cf.Name == "Company Size" && cf.OrganizationID != nil && *cf.OrganizationID == orgID
		})).Return(&domain.CustomField{
			Base:           domain.Base{ID: fieldID},
			UserID:         userID,
			OrganizationID: &orgID,
			Name:           "Company Size",
			NormalizedName: "company_size",
			DataType:       domain.CustomFieldDataTypeSelect,
			EntityType:     domain.CustomFieldEntityContact,
			IsRequired:     false,
			Options:        pq.StringArray{"1-10", "11-50", "51-200"},
		}, nil).Once()

		app.Post("/custom-fields", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.Create(c)
		})

		reqBody := v1schema.CreateCustomFieldRequest{
			Name:       "Company Size",
			DataType:   domain.CustomFieldDataTypeSelect,
			EntityType: domain.CustomFieldEntityContact,
			IsRequired: false,
			Options:    []string{"1-10", "11-50", "51-200"},
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/custom-fields", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)

		mockRoleRepo.AssertExpectations(t)
		mockCustomFieldRepo.AssertExpectations(t)
	})

	t.Run("Success - Create without organization", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		// Mock role returns no roles
		mockRoleRepo.On("FindByUserID", mock.Anything, userID).Return([]domain.Role{}, nil).Once()

		// Mock create
		mockCustomFieldRepo.On("Create", mock.Anything, mock.MatchedBy(func(cf *domain.CustomField) bool {
			return cf.Name == "Department" && cf.OrganizationID == nil
		})).Return(&domain.CustomField{
			Base:           domain.Base{ID: fieldID},
			UserID:         userID,
			OrganizationID: nil,
			Name:           "Department",
			NormalizedName: "department",
			DataType:       domain.CustomFieldDataTypeText,
			EntityType:     domain.CustomFieldEntityContact,
		}, nil).Once()

		app.Post("/custom-fields", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.Create(c)
		})

		reqBody := v1schema.CreateCustomFieldRequest{
			Name:       "Department",
			DataType:   domain.CustomFieldDataTypeText,
			EntityType: domain.CustomFieldEntityContact,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/custom-fields", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)

		mockRoleRepo.AssertExpectations(t)
		mockCustomFieldRepo.AssertExpectations(t)
	})

	t.Run("Fail - Unauthorized", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		app.Post("/custom-fields", handler.Create)

		reqBody := v1schema.CreateCustomFieldRequest{
			Name:       "Test",
			DataType:   domain.CustomFieldDataTypeText,
			EntityType: domain.CustomFieldEntityContact,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/custom-fields", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("Fail - Invalid request body", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		app.Post("/custom-fields", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.Create(c)
		})

		req := httptest.NewRequest("POST", "/custom-fields", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("Fail - Validation error - missing name", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		app.Post("/custom-fields", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.Create(c)
		})

		reqBody := v1schema.CreateCustomFieldRequest{
			DataType:   domain.CustomFieldDataTypeText,
			EntityType: domain.CustomFieldEntityContact,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/custom-fields", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("Fail - Repository error", func(t *testing.T) {
		mockCustomFieldRepo := new(MockCustomFieldRepository)
		mockUserRepo := new(MockUserRepository)
		mockRoleRepo := new(MockRoleRepository)
		handler := NewHandler(mockCustomFieldRepo, mockUserRepo, mockRoleRepo)

		app := setupTestApp()

		mockRoleRepo.On("FindByUserID", mock.Anything, userID).Return([]domain.Role{}, nil).Once()
		mockCustomFieldRepo.On("Create", mock.Anything, mock.Anything).Return(nil, errors.New("database error")).Once()

		app.Post("/custom-fields", func(c fiber.Ctx) error {
			c.Locals("user_id", userID)
			return handler.Create(c)
		})

		reqBody := v1schema.CreateCustomFieldRequest{
			Name:       "Test",
			DataType:   domain.CustomFieldDataTypeText,
			EntityType: domain.CustomFieldEntityContact,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/custom-fields", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, 500, resp.StatusCode)

		mockRoleRepo.AssertExpectations(t)
		mockCustomFieldRepo.AssertExpectations(t)
	})
}
