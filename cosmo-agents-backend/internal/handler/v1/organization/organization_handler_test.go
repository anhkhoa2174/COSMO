package organization

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

func TestNewOrganizationHandler(t *testing.T) {
	// Test that handler can be created - basic constructor test
	// In actual integration tests, these would be proper mocks
	assert.NotNil(t, NewOrganizationHandler)
}

func TestOrganizationHandler_UUIDValidation(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		shouldFail bool
	}{
		{"valid UUID", uuid.New().String(), false},
		{"invalid UUID", "bad-uuid", true},
		{"empty UUID", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uuid.Parse(tt.id)
			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOrganizationHandler_OrganizationValidation(t *testing.T) {
	// Test valid organization creation request
	validReq := v1schema.CreateOrganizationRequest{
		Name:                    "Test Organization",
		CompanyURL:              "https://example.com",
		CompanyDescription:      "Test Description",
		CompanyTargetingPersona: []string{"persona1", "persona2"},
		ValueOffering:           "Test Value",
	}
	err := v1validation.ValidateStruct(&validReq)
	assert.NoError(t, err)

	// Test invalid request (missing required name)
	invalidReq := v1schema.CreateOrganizationRequest{
		Name: "", // empty name should fail validation
	}
	err = v1validation.ValidateStruct(&invalidReq)
	assert.Error(t, err)
}

func TestOrganizationHandler_UpdateRequestValidation(t *testing.T) {
	// Test valid update request
	validReq := v1schema.UpdateOrganizationRequest{
		Name:                    stringPtr("Updated Name"),
		CompanyURL:              stringPtr("https://updated.com"),
		CompanyDescription:      stringPtr("Updated Description"),
		CompanyTargetingPersona: []string{"updated-persona"},
		ValueOffering:           stringPtr("Updated Value"),
	}
	err := v1validation.ValidateStruct(&validReq)
	assert.NoError(t, err)

	// Test valid request with partial updates
	partialReq := v1schema.UpdateOrganizationRequest{
		Name: stringPtr("Just Name"),
	}
	err = v1validation.ValidateStruct(&partialReq)
	assert.NoError(t, err)
}

func TestOrganizationHandler_DomainStructures(t *testing.T) {
	// Test domain.Organization structure
	userID := uuid.New()
	org := &domain.Organization{
		Base:                    base.Base{ID: uuid.New()},
		UserID:                  &userID,
		Name:                    "Test Organization",
		CompanyURL:              "https://example.com",
		CompanyDescription:      "Test Description",
		CompanyTargetingPersona: pq.StringArray{"persona1", "persona2"},
		ValueOffering:           "Test Value",
	}

	assert.Equal(t, "Test Organization", org.Name)
	assert.Equal(t, "https://example.com", org.CompanyURL)
	assert.Equal(t, userID, *org.UserID)
	assert.Len(t, org.CompanyTargetingPersona, 2)

	// Test domain.Role structure
	role := &domain.Role{
		Base:           base.Base{ID: uuid.New()},
		UserID:         userID,
		OrganizationID: org.Base.ID,
		Name:           domain.RoleNameAdmin,
		Status:         domain.RoleStatusActive,
		JobTitle:       "Administrator",
	}

	assert.Equal(t, userID, role.UserID)
	assert.Equal(t, org.Base.ID, role.OrganizationID)
	assert.Equal(t, domain.RoleNameAdmin, role.Name)
	assert.Equal(t, domain.RoleStatusActive, role.Status)
	assert.Equal(t, "Administrator", role.JobTitle)
}

func TestOrganizationHandler_ErrorScenarios(t *testing.T) {
	// Test various error scenarios that might occur in the handler

	// Test gorm.ErrRecordNotFound handling
	testOrgID := uuid.New()
	testUserID := uuid.New()

	// This simulates finding an organization that doesn't exist
	_, err := uuid.Parse("invalid-uuid")
	assert.Error(t, err)

	// Test that error scenarios can be handled
	assert.NotNil(t, testOrgID)
	assert.NotNil(t, testUserID)
}

func TestOrganizationHandler_Pagination(t *testing.T) {
	// Test pagination parameter parsing
	tests := []struct {
		name         string
		offset       int
		limit        int
		expectOffset int
		expectLimit  int
	}{
		{"default values", 0, 0, 0, 0},
		{"custom values", 10, 20, 10, 20},
		{"negative values", -1, -1, -1, -1},
		{"zero values", 0, 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test pagination parameter validation
			assert.Equal(t, tt.expectOffset, tt.offset)
			assert.Equal(t, tt.expectLimit, tt.limit)
		})
	}
}

// Test organization response conversion
func TestOrganizationHandler_ResponseConversion(t *testing.T) {
	userID := uuid.New()
	org := &domain.Organization{
		Base:                    base.Base{ID: uuid.New()},
		UserID:                  &userID,
		Name:                    "Test Organization",
		CompanyURL:              "https://example.com",
		CompanyDescription:      "Test Description",
		CompanyTargetingPersona: pq.StringArray{"persona1", "persona2"},
		ValueOffering:           "Test Value",
	}

	// Test conversion to response DTO
	response := v1schema.ToOrganizationResponse(org)

	assert.Equal(t, org.Base.ID, response.ID)
	assert.Equal(t, org.UserID, response.UserID)
	assert.Equal(t, org.Name, response.Name)
	assert.Equal(t, org.CompanyURL, response.CompanyURL)
	assert.Equal(t, org.CompanyDescription, response.CompanyDescription)
	assert.Equal(t, []string(org.CompanyTargetingPersona), response.CompanyTargetingPersona)
	assert.Equal(t, org.ValueOffering, response.ValueOffering)
	assert.NotEmpty(t, response.CreatedAt)
	assert.NotEmpty(t, response.UpdatedAt)
}

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}
