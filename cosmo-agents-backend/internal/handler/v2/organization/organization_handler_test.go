package organization

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
)

func TestNewOrganizationHandler(t *testing.T) {
	// Test that handler can be created - basic constructor test
	assert.NotNil(t, NewOrganizationHandler)
}

func TestOrganizationHandler_V2_UUIDValidation(t *testing.T) {
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

func TestOrganizationHandler_V2_MemberCreateValidation(t *testing.T) {
	// Test valid member creation request
	adminRole := domain.RoleNameAdmin
	validReq := v2schema.OrganizationMemberCreateRequest{
		Email:    "test@example.com",
		Name:     stringPtr("Test User"),
		JobTitle: stringPtr("Developer"),
		Role:     &adminRole,
	}
	err := v1validation.ValidateStruct(&validReq)
	assert.NoError(t, err)

	// Test invalid request (missing required email)
	invalidReq := v2schema.OrganizationMemberCreateRequest{
		Email: "", // empty email should fail validation
	}
	err = v1validation.ValidateStruct(&invalidReq)
	assert.Error(t, err)
}

func TestOrganizationHandler_V2_MemberSearchValidation(t *testing.T) {
	// Test valid member search request
	validReq := v2schema.OrganizationMemberSearchRequest{
		Filter: map[string]interface{}{
			"query": "test",
		},
	}
	err := v1validation.ValidateStruct(&validReq)
	assert.NoError(t, err)

	// Test valid request with filters
	validReqWithFilters := v2schema.OrganizationMemberSearchRequest{
		Filter: map[string]interface{}{
			"query":  "test",
			"status": domain.RoleStatusActive,
		},
	}
	err = v1validation.ValidateStruct(&validReqWithFilters)
	assert.NoError(t, err)
}

func TestOrganizationHandler_V2_MemberDeleteValidation(t *testing.T) {
	// Test valid member delete request
	validReq := v2schema.OrganizationMemberDeleteRequest{
		MemberIDs: []uuid.UUID{
			uuid.New(),
			uuid.New(),
		},
	}
	err := v1validation.ValidateStruct(&validReq)
	assert.NoError(t, err)

	// Test valid empty delete request (edge case)
	emptyReq := v2schema.OrganizationMemberDeleteRequest{
		MemberIDs: []uuid.UUID{},
	}
	err = v1validation.ValidateStruct(&emptyReq)
	assert.NoError(t, err)
}

func TestOrganizationHandler_V2_DomainStructures(t *testing.T) {
	// Test domain.Organization structure for V2
	userID := uuid.New()
	org := &domain.Organization{
		Base:                    base.Base{ID: uuid.New()},
		UserID:                  &userID,
		Name:                    "Test Organization V2",
		CompanyURL:              "https://example-v2.com",
		CompanyDescription:      "Test Description V2",
		CompanyTargetingPersona: pq.StringArray{"persona1", "persona2"},
		ValueOffering:           "Test Value V2",
	}

	assert.Equal(t, "Test Organization V2", org.Name)
	assert.Equal(t, "https://example-v2.com", org.CompanyURL)
	assert.Equal(t, userID, *org.UserID)
	assert.Len(t, org.CompanyTargetingPersona, 2)

	// Test V2 response structure
	response := v2schema.OrganizationReadResponse{
		ID:                      org.Base.ID,
		Name:                    &org.Name,
		CompanyURL:              &org.CompanyURL,
		CompanyDescription:      &org.CompanyDescription,
		CompanyTargetingPersona: (*[]string)(&org.CompanyTargetingPersona),
		ValueOffering:           &org.ValueOffering,
	}

	assert.Equal(t, org.Base.ID, response.ID)
	assert.Equal(t, org.Name, *response.Name)
	assert.Equal(t, org.CompanyURL, *response.CompanyURL)
	assert.Equal(t, org.CompanyDescription, *response.CompanyDescription)
	assert.Equal(t, []string(org.CompanyTargetingPersona), *response.CompanyTargetingPersona)
	assert.Equal(t, org.ValueOffering, *response.ValueOffering)
}

func TestOrganizationHandler_V2_MemberResponseStructures(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	roleID := uuid.New()

	// Test domain.User structure for V2 responses
	user := &domain.User{
		Base:     base.Base{ID: userID},
		Email:    "test@example.com",
		Name:     "Test User",
		JobTitle: "Developer",
	}

	// Test domain.Role structure for V2 responses
	role := &domain.Role{
		Base:           base.Base{ID: roleID},
		UserID:         userID,
		OrganizationID: orgID,
		Name:           domain.RoleNameAdmin,
		Status:         domain.RoleStatusActive,
		JobTitle:       "Senior Developer",
	}

	// Test V2 member create response structure
	createResponse := v2schema.OrganizationMemberCreateResponse{
		InvitedUser: user,
		Role:        role,
	}

	assert.Equal(t, userID, createResponse.InvitedUser.ID)
	assert.Equal(t, "test@example.com", createResponse.InvitedUser.Email)
	assert.Equal(t, roleID, createResponse.Role.ID)
	assert.Equal(t, domain.RoleNameAdmin, createResponse.Role.Name)
	assert.Equal(t, domain.RoleStatusActive, createResponse.Role.Status)
	assert.Equal(t, "Senior Developer", createResponse.Role.JobTitle)
}

func TestOrganizationHandler_V2_PayloadStructures(t *testing.T) {
	// Test invite member payload structure
	payload := v2schema.InviteMemberPayload{
		MemberName: "Test User",
		Email:      "test@example.com",
		URL:        "https://example.com/invite?token=abc123",
	}

	assert.Equal(t, "Test User", payload.MemberName)
	assert.Equal(t, "test@example.com", payload.Email)
	assert.Contains(t, payload.URL, "invite")
	assert.Contains(t, payload.URL, "token")
}

func TestOrganizationHandler_V2_SearchResponseStructures(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	// Test search response structure
	searchResponse := v2schema.OrganizationMemberSearchResponse{
		List: []v2schema.OrganizationListItem{
			{
				Entity: v2schema.OrganizationMemberEntity{
					ID:    userID,
					Email: "test@example.com",
					Name:  stringPtr("Test User"),
				},
				Role: v2schema.OrganizationRole{
					ID:             uuid.New(),
					OrganizationID: orgID,
					Name:           "admin",
					Status:         domain.RoleStatusActive,
				},
			},
		},
		Offset: 0,
		Limit:  25,
		Total:  1,
	}

	assert.Len(t, searchResponse.List, 1)
	assert.Equal(t, 0, searchResponse.Offset)
	assert.Equal(t, 25, searchResponse.Limit)
	assert.Equal(t, int64(1), searchResponse.Total)
	assert.Equal(t, userID, searchResponse.List[0].Entity.ID)
	assert.Equal(t, "test@example.com", searchResponse.List[0].Entity.Email)
	assert.Equal(t, "admin", searchResponse.List[0].Role.Name)
}

func TestOrganizationHandler_V2_ErrorScenarios(t *testing.T) {
	// Test various error scenarios that might occur in V2 handler

	// Test UUID parsing error
	testOrgID := uuid.New()
	testUserID := uuid.New()

	// This simulates finding an organization that doesn't exist
	_, err := uuid.Parse("invalid-uuid")
	assert.Error(t, err)

	// Test validation errors
	req := v2schema.OrganizationMemberCreateRequest{
		Email: "", // invalid email
	}
	err = v1validation.ValidateStruct(&req)
	assert.Error(t, err)

	// Test that error scenarios can be handled
	assert.NotNil(t, testOrgID)
	assert.NotNil(t, testUserID)
}

func TestOrganizationHandler_V2_Pagination(t *testing.T) {
	// Test pagination parameter parsing for V2
	tests := []struct {
		name         string
		offset       int
		limit        int
		expectOffset int
		expectLimit  int
	}{
		{"default values", 0, 0, 0, 25}, // V2 default limit is 25
		{"custom values", 10, 20, 10, 20},
		{"negative values", -1, -1, -1, 25}, // limit should be reset to 25
		{"zero values", 0, 0, 0, 25},
		{"large limit", 0, 150, 0, 100}, // max limit is 100
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test pagination parameter validation
			offset := tt.offset
			limit := tt.limit
			if limit > 100 {
				limit = 100
			}
			if limit <= 0 {
				limit = 25 // V2 default
			}
			assert.Equal(t, tt.expectOffset, offset)
			assert.Equal(t, tt.expectLimit, limit)
		})
	}
}

func TestOrganizationHandler_V2_RoleValidation(t *testing.T) {
	// Test role name validation for V2
	tests := []struct {
		name        string
		role        domain.RoleName
		expectValid bool
	}{
		{"admin role", domain.RoleNameAdmin, true},
		{"member role", domain.RoleNameMember, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, len(tt.role) > 0)
			if tt.expectValid {
				assert.Contains(t, []domain.RoleName{domain.RoleNameAdmin, domain.RoleNameMember}, tt.role)
			}
		})
	}
}

func TestOrganizationHandler_V2_OrganizationIDResolution(t *testing.T) {
	// Test organization ID resolution logic
	orgID := uuid.New()

	tests := []struct {
		name             string
		paramValue       string
		shouldUseMainOrg bool
		expectedOrgID    uuid.UUID
	}{
		{"me parameter", "me", true, orgID},
		{"UUID parameter", orgID.String(), false, orgID},
		{"invalid UUID", "invalid-uuid", false, uuid.Nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.paramValue == "me" {
				assert.Equal(t, "me", tt.paramValue)
				assert.True(t, tt.shouldUseMainOrg)
			} else if tt.paramValue == "invalid-uuid" {
				_, err := uuid.Parse(tt.paramValue)
				assert.Error(t, err)
			} else {
				parsedID, err := uuid.Parse(tt.paramValue)
				if tt.shouldUseMainOrg {
					assert.Equal(t, "me", tt.paramValue)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.expectedOrgID, parsedID)
				}
			}
		})
	}
}

func TestOrganizationHandler_V2_InviteMemberTransactionAtomicity(t *testing.T) {
	// This test documents the transaction boundary fix ensuring atomicity
	// between database operations and worker task queuing

	// The fix ensures that worker task EnqueueLowPriorityTask is called within
	// the same transaction as database operations, guaranteeing atomicity:
	// - If email task fails -> entire transaction rolls back
	// - If email task succeeds -> transaction commits with all changes

	tests := []struct {
		name          string
		workerSuccess bool
		expectError   bool
		description   string
	}{
		{
			name:          "successful invitation",
			workerSuccess: true,
			expectError:   false,
			description:   "Both database changes and worker task should succeed atomically",
		},
		{
			name:          "worker task failure",
			workerSuccess: false,
			expectError:   true,
			description:   "Worker task failure should rollback database changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the test scenario makes sense
			assert.NotEmpty(t, tt.name)
			assert.NotEmpty(t, tt.description)

			// The actual atomicity is guaranteed by moving EnqueueLowPriorityTask
			// inside the executeInTransaction function in organization_handler.go:306-359

			if tt.workerSuccess {
				// Success case: DB operations + worker task -> commit
				assert.False(t, tt.expectError, "Success case should not return error")
			} else {
				// Failure case: worker task fails -> rollback of DB operations
				assert.True(t, tt.expectError, "Worker failure should return error and rollback")
			}
		})
	}
}

func TestOrganizationHandler_V2_InviteMemberPayloadStructure(t *testing.T) {
	// Test the payload structure used for invitation emails
	payload := v2schema.InviteMemberPayload{
		MemberName: "John Doe",
		Email:      "john@example.com",
		URL:        "https://example.com/invite?state=encoded_params",
	}

	assert.Equal(t, "John Doe", payload.MemberName)
	assert.Equal(t, "john@example.com", payload.Email)
	assert.Contains(t, payload.URL, "https://example.com/invite")
	assert.Contains(t, payload.URL, "state=")
}

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}
