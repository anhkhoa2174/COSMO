package customfield

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the CustomField schema
	err = db.AutoMigrate(&domain.CustomField{})
	require.NoError(t, err)

	return db
}

// TestCustomFieldRepository_Create tests creating a new custom field
func TestCustomFieldRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomFieldRepository(db)
	ctx := context.Background()

	// Test data
	userID := uuid.New()
	orgID := uuid.New()
	customField := &domain.CustomField{
		UserID:         userID,
		OrganizationID: &orgID,
		Name:           "Test Field",
		DataType:       domain.CustomFieldDataTypeText,
		EntityType:     domain.CustomFieldEntityContact,
		IsRequired:     true,
		Options:        pq.StringArray{"option1", "option2"},
		SampleData:     stringPtr("sample data"),
		FallbackValue:  stringPtr("fallback"),
	}

	// Test Create
	result, err := repo.Create(ctx, customField)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, &orgID, result.OrganizationID)
	assert.Equal(t, "Test Field", result.Name)
	assert.Equal(t, "test_field", result.NormalizedName)
	assert.Equal(t, domain.CustomFieldDataTypeText, result.DataType)
	assert.Equal(t, domain.CustomFieldEntityContact, result.EntityType)
	assert.True(t, result.IsRequired)
	assert.Equal(t, pq.StringArray{"option1", "option2"}, result.Options)
	assert.Equal(t, "sample data", *result.SampleData)
	assert.Equal(t, "fallback", *result.FallbackValue)
}

// TestCustomFieldRepository_FindByID tests finding a custom field by ID
func TestCustomFieldRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomFieldRepository(db)
	ctx := context.Background()

	// Create test custom field
	customField := &domain.CustomField{
		UserID:     uuid.New(),
		Name:       "Find Test Field",
		DataType:   domain.CustomFieldDataTypeNumber,
		EntityType: domain.CustomFieldEntityCompany,
	}
	created, err := repo.Create(ctx, customField)
	require.NoError(t, err)

	// Test FindByID
	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Find Test Field", found.Name)
	assert.Equal(t, "find_test_field", found.NormalizedName)
}

// TestCustomFieldRepository_FindByNormalizedNames tests finding custom fields by normalized names
func TestCustomFieldRepository_FindByNormalizedNames(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomFieldRepository(db)
	ctx := context.Background()

	// Create test custom fields
	userID := uuid.New()
	fields := []*domain.CustomField{
		{
			UserID:     userID,
			Name:       "First Name",
			DataType:   domain.CustomFieldDataTypeText,
			EntityType: domain.CustomFieldEntityContact,
		},
		{
			UserID:     userID,
			Name:       "Last Name",
			DataType:   domain.CustomFieldDataTypeText,
			EntityType: domain.CustomFieldEntityContact,
		},
		{
			UserID:     userID,
			Name:       "Age",
			DataType:   domain.CustomFieldDataTypeNumber,
			EntityType: domain.CustomFieldEntityContact,
		},
	}

	for _, field := range fields {
		_, err := repo.Create(ctx, field)
		require.NoError(t, err)
	}

	// Test FindByNormalizedNames
	result, err := repo.FindByNormalizedNames(ctx, userID, []string{"first_name", "age", "nonexistent"})
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Verify first_name field
	firstNameField, exists := result["first_name"]
	assert.True(t, exists)
	assert.Equal(t, "First Name", firstNameField.Name)

	// Verify age field
	ageField, exists := result["age"]
	assert.True(t, exists)
	assert.Equal(t, "Age", ageField.Name)
}

// TestCustomFieldRepository_FindByOrganizationID tests finding custom fields by organization ID
func TestCustomFieldRepository_FindByOrganizationID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomFieldRepository(db)
	ctx := context.Background()

	// Create test custom fields
	orgID := uuid.New()
	userID := uuid.New()
	fields := []*domain.CustomField{
		{
			UserID:         userID,
			OrganizationID: &orgID,
			Name:           "Company Field 1",
			DataType:       domain.CustomFieldDataTypeText,
			EntityType:     domain.CustomFieldEntityCompany,
		},
		{
			UserID:         userID,
			OrganizationID: &orgID,
			Name:           "Company Field 2",
			DataType:       domain.CustomFieldDataTypeEmail,
			EntityType:     domain.CustomFieldEntityCompany,
		},
		{
			UserID:         userID,
			OrganizationID: nil, // Different org
			Name:           "Personal Field",
			DataType:       domain.CustomFieldDataTypeText,
			EntityType:     domain.CustomFieldEntityContact,
		},
	}

	for _, field := range fields {
		_, err := repo.Create(ctx, field)
		require.NoError(t, err)
	}

	// Test FindByOrganizationID
	result, err := repo.FindByOrganizationID(ctx, orgID)
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Verify only org fields are returned
	for _, field := range result {
		assert.Equal(t, &orgID, field.OrganizationID)
	}
}

// TestCustomFieldRepository_UpdateFields tests updating specific fields of a custom field
func TestCustomFieldRepository_UpdateFields(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomFieldRepository(db)
	ctx := context.Background()

	// Create test custom field
	customField := &domain.CustomField{
		UserID:     uuid.New(),
		Name:       "Original Name",
		DataType:   domain.CustomFieldDataTypeText,
		EntityType: domain.CustomFieldEntityContact,
		IsRequired: false,
	}
	created, err := repo.Create(ctx, customField)
	require.NoError(t, err)

	// Test UpdateFields
	updates := map[string]interface{}{
		"name":        "Updated Name",
		"data_type":   domain.CustomFieldDataTypeSelect,
		"is_required": true,
		"options":     pq.StringArray{"option1", "option2"},
	}
	err = repo.UpdateFields(ctx, created.ID, updates)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	// Renaming keeps the storage key so values already on contacts stay reachable.
	assert.Equal(t, "original_name", updated.NormalizedName)
	assert.Equal(t, domain.CustomFieldDataTypeSelect, updated.DataType)
	assert.True(t, updated.IsRequired)
	assert.Equal(t, pq.StringArray{"option1", "option2"}, updated.Options)
}

// TestCustomFieldRepository_UpdateFields_InvalidDataType tests validation on data type updates
func TestCustomFieldRepository_UpdateFields_InvalidDataType(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomFieldRepository(db)
	ctx := context.Background()

	// Create test custom field
	customField := &domain.CustomField{
		UserID:     uuid.New(),
		Name:       "Test Field",
		DataType:   domain.CustomFieldDataTypeText,
		EntityType: domain.CustomFieldEntityContact,
	}
	created, err := repo.Create(ctx, customField)
	require.NoError(t, err)

	// Test invalid data type
	updates := map[string]interface{}{
		"data_type": "invalid_type",
	}
	err = repo.UpdateFields(ctx, created.ID, updates)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid data_type for custom field")
}

// TestCustomFieldRepository_UpdateFields_InvalidEntityType tests validation on entity type updates
func TestCustomFieldRepository_UpdateFields_InvalidEntityType(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomFieldRepository(db)
	ctx := context.Background()

	// Create test custom field
	customField := &domain.CustomField{
		UserID:     uuid.New(),
		Name:       "Test Field",
		DataType:   domain.CustomFieldDataTypeText,
		EntityType: domain.CustomFieldEntityContact,
	}
	created, err := repo.Create(ctx, customField)
	require.NoError(t, err)

	// Test invalid entity type
	updates := map[string]interface{}{
		"entity_type": "invalid_entity",
	}
	err = repo.UpdateFields(ctx, created.ID, updates)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid entity_type for custom field")
}

// TestCustomFieldRepository_Delete tests deleting a custom field
func TestCustomFieldRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewCustomFieldRepository(db)
	ctx := context.Background()

	// Create test custom field
	customField := &domain.CustomField{
		UserID:     uuid.New(),
		Name:       "To Delete",
		DataType:   domain.CustomFieldDataTypeText,
		EntityType: domain.CustomFieldEntityContact,
	}
	created, err := repo.Create(ctx, customField)
	require.NoError(t, err)

	// Delete custom field
	err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// Verify deletion
	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, found)
}

// stringPtr is a helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}
