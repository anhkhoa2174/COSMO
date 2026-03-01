package inbound_lead_form

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the required schemas
	err = db.AutoMigrate(
		&domain.InboundLeadForm{},
		&domain.FormField{},
		&domain.CustomField{},
		&domain.InboundLeadFormListContactAssociation{},
	)
	require.NoError(t, err)

	return db
}

// TestInboundLeadFormRepository_NewInboundLeadFormRepository tests creating a new repository
func TestInboundLeadFormRepository_NewInboundLeadFormRepository(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboundLeadFormRepository(db)
	assert.NotNil(t, repo)
}

// TestInboundLeadFormRepository_CreateWithListContact tests creating a form with list contacts
func TestInboundLeadFormRepository_CreateWithListContact(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboundLeadFormRepository(db)
	ctx := context.Background()

	// Test data
	userID := uuid.New()
	form := &domain.InboundLeadForm{
		UserID: &userID,
		Name:   "Contact Form",
		Slug:   "contact-form",
	}
	listContactIDs := []uuid.UUID{uuid.New(), uuid.New()}

	// Test CreateWithListContact
	err := repo.CreateWithListContact(ctx, form, listContactIDs)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, form.ID)
	assert.Equal(t, userID, *form.UserID)
	assert.Equal(t, "Contact Form", form.Name)
	assert.Equal(t, "contact-form", form.Slug)
}

// TestInboundLeadFormRepository_CreateWithFields tests creating a form with fields
func TestInboundLeadFormRepository_CreateWithFields(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboundLeadFormRepository(db)
	ctx := context.Background()

	// Test data
	userID := uuid.New()
	form := &domain.InboundLeadForm{
		UserID: &userID,
		Name:   "Registration Form",
		Slug:   "registration-form",
	}

	customFieldID := uuid.New()
	fields := []domain.FormField{
		{
			DisplayName:   "First Name",
			IsRequired:    true,
			CustomFieldID: &customFieldID,
			UIMetadata:    base.JSONB([]byte(`{"placeholder": "Enter your first name"}`)),
		},
		{
			DisplayName:     "Email",
			IsRequired:      true,
			IsSystemField:   true,
			SystemFieldName: func() *string { s := "email"; return &s }(),
		},
		{
			DisplayName: "Phone",
			IsRequired:  false,
			UIMetadata:  base.JSONB([]byte(`{"type": "tel", "placeholder": "Phone number"}`)),
		},
	}

	// Test CreateWithFields
	err := repo.CreateWithFields(ctx, form, fields)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, form.ID)
	assert.Equal(t, userID, *form.UserID)
	assert.Equal(t, "Registration Form", form.Name)
	assert.Equal(t, "registration-form", form.Slug)

	// Verify fields were created with correct form IDs
	for _, field := range fields {
		assert.Equal(t, form.ID, field.FormID)
	}
}

// TestInboundLeadFormRepository_CreateWithFieldsTx tests creating a form with fields using a transaction
func TestInboundLeadFormRepository_CreateWithFieldsTx(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboundLeadFormRepository(db)
	ctx := context.Background()

	// Begin transaction
	tx := db.Begin()
	defer tx.Rollback()

	// Test data
	form := &domain.InboundLeadForm{
		Name: "Transaction Test Form",
		Slug: "transaction-test-form",
	}

	fields := []domain.FormField{
		{
			DisplayName: "Test Field",
			IsRequired:  false,
		},
	}

	// Test CreateWithFieldsTx
	err := repo.CreateWithFieldsTx(ctx, tx, form, fields)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, form.ID)

	// Verify fields were created
	for _, field := range fields {
		assert.Equal(t, form.ID, field.FormID)
	}
}

// TestInboundLeadFormRepository_FindBySlug tests finding a form by slug
func TestInboundLeadFormRepository_FindBySlug(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboundLeadFormRepository(db)
	ctx := context.Background()

	// Create test form
	form := &domain.InboundLeadForm{
		Name: "Find Test Form",
		Slug: "find-test-form",
	}
	_, err := repo.Create(ctx, form)
	require.NoError(t, err)

	// Test FindBySlug
	found, err := repo.FindBySlug(ctx, "find-test-form")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "Find Test Form", found.Name)
	assert.Equal(t, "find-test-form", found.Slug)

	// Test with non-existent slug
	notFound, err := repo.FindBySlug(ctx, "non-existent")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

// TestInboundLeadFormRepository_GetByIdentifier tests finding a form by ID or slug
func TestInboundLeadFormRepository_GetByIdentifier(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboundLeadFormRepository(db)
	ctx := context.Background()

	// Create test form with fields
	userID := uuid.New()
	form := &domain.InboundLeadForm{
		UserID: &userID, // Set UserID
		Name:   "Identifier Test Form",
		Slug:   "identifier-test-form",
	}
	_, err := repo.Create(ctx, form)
	require.NoError(t, err)

	// Create some fields
	customFieldID := uuid.New()
	fields := []domain.FormField{
		{
			FormID:        form.ID, // Set FormID before creation
			DisplayName:   "Test Field",
			IsRequired:    true,
			CustomFieldID: &customFieldID,
		},
	}

	err = repo.CreateFormFields(ctx, fields)
	require.NoError(t, err)

	// Test GetByIdentifier with slug
	foundBySlug, err := repo.GetByIdentifier(ctx, "identifier-test-form")
	require.NoError(t, err)
	assert.NotNil(t, foundBySlug)
	assert.Equal(t, "Identifier Test Form", foundBySlug.Name)
	assert.Len(t, foundBySlug.Fields, 1)

	// Test GetByIdentifier with ID
	foundByID, err := repo.GetByIdentifier(ctx, form.ID.String())
	require.NoError(t, err)
	assert.NotNil(t, foundByID)
	assert.Equal(t, "Identifier Test Form", foundByID.Name)

	// Test with non-existent identifier
	notFound, err := repo.GetByIdentifier(ctx, "non-existent")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

// TestInboundLeadFormRepository_CreateFormFields tests creating form fields
func TestInboundLeadFormRepository_CreateFormFields(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboundLeadFormRepository(db)
	ctx := context.Background()

	// Create test form first
	form := &domain.InboundLeadForm{
		Name: "Fields Test Form",
		Slug: "fields-test-form",
	}
	created, err := repo.Create(ctx, form)
	require.NoError(t, err)

	// Test data
	customFieldID := uuid.New()
	fields := []domain.FormField{
		{
			FormID:        created.ID,
			DisplayName:   "Field 1",
			IsRequired:    true,
			CustomFieldID: &customFieldID,
		},
		{
			FormID:        created.ID,
			DisplayName:   "Field 2",
			IsRequired:    false,
			IsSystemField: true,
		},
	}

	// Test CreateFormFields
	err = repo.CreateFormFields(ctx, fields)
	require.NoError(t, err)

	// Verify fields were created
	assert.Equal(t, created.ID, fields[0].FormID)
	assert.Equal(t, created.ID, fields[1].FormID)
}

// TestInboundLeadFormRepository_DeleteFormFields tests deleting form fields
func TestInboundLeadFormRepository_DeleteFormFields(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboundLeadFormRepository(db)
	ctx := context.Background()

	// Create test form
	form := &domain.InboundLeadForm{
		Name: "Delete Fields Form",
		Slug: "delete-fields-form",
	}
	created, err := repo.Create(ctx, form)
	require.NoError(t, err)

	// Create some fields
	fields := []domain.FormField{
		{FormID: created.ID, DisplayName: "Field 1"},
		{FormID: created.ID, DisplayName: "Field 2"},
	}
	err = repo.CreateFormFields(ctx, fields)
	require.NoError(t, err)

	// Test DeleteFormFields
	err = repo.DeleteFormFields(ctx, created.ID)
	require.NoError(t, err)

	// Verify fields were deleted
	remainingFields := []domain.FormField{}
	err = db.WithContext(ctx).Where("form_id = ?", created.ID).Find(&remainingFields).Error
	require.NoError(t, err)
	assert.Len(t, remainingFields, 0)
}

// TestInboundLeadFormRepository_UpdateForm tests updating a form
func TestInboundLeadFormRepository_UpdateForm(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboundLeadFormRepository(db)
	ctx := context.Background()

	// Create test form
	form := &domain.InboundLeadForm{
		Name: "Update Test Form",
		Slug: "update-test-form",
	}
	created, err := repo.Create(ctx, form)
	require.NoError(t, err)

	// Test UpdateForm
	updates := map[string]interface{}{
		"name": "Updated Form Name",
		"slug": "updated-form-slug",
	}
	err = repo.UpdateForm(ctx, created.ID, updates)
	require.NoError(t, err)

	// Verify update
	updated, err := repo.FindBySlug(ctx, "updated-form-slug")
	require.NoError(t, err)
	assert.NotNil(t, updated)
	assert.Equal(t, "Updated Form Name", updated.Name)
	assert.Equal(t, created.ID, updated.ID)
}

// TestInboundLeadFormRepository_ListByUser tests listing forms by user
func TestInboundLeadFormRepository_ListByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboundLeadFormRepository(db)
	ctx := context.Background()

	// Create test forms for user
	userID := uuid.New()
	for i := 0; i < 3; i++ {
		form := &domain.InboundLeadForm{
			UserID: &userID,
			Name:   fmt.Sprintf("Form %d", i),
			Slug:   fmt.Sprintf("form-%d", i),
		}
		_, err := repo.Create(ctx, form)
		require.NoError(t, err)
	}

	// Create form for different user
	differentUserID := uuid.New()
	form := &domain.InboundLeadForm{
		UserID: &differentUserID,
		Name:   "Different User Form",
		Slug:   "different-user-form",
	}
	_, err := repo.Create(ctx, form)
	require.NoError(t, err)

	// Test ListByUser
	forms, err := repo.ListByUser(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, forms, 3)

	// Verify all forms belong to the user
	for _, form := range forms {
		assert.Equal(t, &userID, form.UserID)
	}

	// Verify they are ordered by created_at DESC
	for i := 1; i < len(forms); i++ {
		assert.True(t, forms[i-1].CreatedAt.After(forms[i].CreatedAt) ||
			forms[i-1].CreatedAt.Equal(forms[i].CreatedAt))
	}
}

// TestInboundLeadFormRepository_ComplexForm tests creating a complex form with all features
func TestInboundLeadFormRepository_ComplexForm(t *testing.T) {
	db := setupTestDB(t)
	repo := NewInboundLeadFormRepository(db)
	ctx := context.Background()

	// Test data
	userID := uuid.New()
	form := &domain.InboundLeadForm{
		UserID: &userID,
		Name:   "Complex Lead Capture Form",
		Slug:   "complex-lead-capture-form",
		UIMetadata: base.JSONB([]byte(`{
			"description": "Advanced lead form for enterprise customers",
			"theme": "professional",
			"redirect_url": "/thank-you",
			"analytics": true
		}`)),
	}

	// Create custom fields
	customFieldID1 := uuid.New()
	customFieldID2 := uuid.New()

	fields := []domain.FormField{
		{
			DisplayName:   "Company Name",
			IsRequired:    true,
			CustomFieldID: &customFieldID1,
			UIMetadata:    base.JSONB([]byte(`{"placeholder": "Enter your company name", "maxlength": 100}`)),
		},
		{
			DisplayName:     "Email",
			IsRequired:      true,
			IsSystemField:   true,
			SystemFieldName: func() *string { s := "email"; return &s }(),
			UIMetadata:      base.JSONB([]byte(`{"type": "email", "placeholder": "your@email.com"}`)),
		},
		{
			DisplayName:   "Phone Number",
			IsRequired:    false,
			CustomFieldID: &customFieldID2,
			UIMetadata:    base.JSONB([]byte(`{"type": "tel", "pattern": "[0-9]{3}-[0-9]{3}-[0-9]{4}"}`)),
		},
		{
			DisplayName: "Message",
			IsRequired:  true,
			UIMetadata:  base.JSONB([]byte(`{"type": "textarea", "rows": 4, "placeholder": "Tell us about your needs..."}`)),
		},
	}

	// Test CreateWithFields
	err := repo.CreateWithFields(ctx, form, fields)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, form.ID)

	// Test CreateWithListContact with a new form
	newForm := &domain.InboundLeadForm{
		UserID:     &userID,
		Name:       "Contact Form",
		Slug:       "contact-form-new",
		UIMetadata: base.JSONB([]byte(`{"type": "contact"}`)),
	}
	listContactIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	err = repo.CreateWithListContact(ctx, newForm, listContactIDs)
	require.NoError(t, err)

	// Use the original form for the remaining tests
	form.Slug = "complex-lead-capture-form"

	// Verify GetByIdentifier with all relations
	found, err := repo.GetByIdentifier(ctx, form.Slug)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "Complex Lead Capture Form", found.Name)
	assert.Equal(t, "complex-lead-capture-form", found.Slug)
	assert.Len(t, found.Fields, 4)

	// Verify field properties
	for _, field := range found.Fields {
		assert.Equal(t, form.ID, field.FormID)
		assert.NotEmpty(t, field.DisplayName)
		assert.NotNil(t, field.UIMetadata)
	}
}
