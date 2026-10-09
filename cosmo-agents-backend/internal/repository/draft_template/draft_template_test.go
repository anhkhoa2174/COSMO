package drafttemplate

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	db = db.Session(&gorm.Session{AllowGlobalUpdate: true})

	// Auto-migrate the required schemas
	err = db.AutoMigrate(&domain.DraftTemplate{}, &domain.Template{})
	require.NoError(t, err)

	return db
}

// createTestTemplate creates a test template using GormRepository
func createTestTemplate(t *testing.T, db *gorm.DB, userID uuid.UUID, campaignID *uuid.UUID, templateType string) *domain.Template {
	templateRepo := gormpkg.NewGormRepository[domain.Template](db)
	ctx := context.Background()

	template := &domain.Template{
		UserID:     userID,
		CampaignID: campaignID,
		Type:       templateType,
		Category:   domain.TemplateCategoryDraft,
		Subject:    "Test Subject",
		Content:    "Test Content",
		Position:   1000.0,
	}

	created, err := templateRepo.Create(ctx, template)
	require.NoError(t, err)
	return created
}

// TestDraftTemplateRepository_Create tests creating a new draft template
func TestDraftTemplateRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDraftTemplateRepository(db)
	ctx := context.Background()

	// Create a template first
	userID := uuid.New()
	template := createTestTemplate(t, db, userID, nil, "welcome_template")

	// Test data
	draftTemplate := &domain.DraftTemplate{
		Intent:     "welcome_email",
		CampaignID: uuid.New(),
		TemplateID: template.ID,
	}

	// Test Create
	result, err := repo.Create(ctx, draftTemplate)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.Equal(t, "welcome_email", result.Intent)
	assert.Equal(t, draftTemplate.CampaignID, result.CampaignID)
	assert.Equal(t, template.ID, result.TemplateID)
}

// TestDraftTemplateRepository_FindByID tests finding a draft template by ID
func TestDraftTemplateRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDraftTemplateRepository(db)
	ctx := context.Background()

	// Create a template first
	userID := uuid.New()
	template := createTestTemplate(t, db, userID, nil, "follow_up_template")

	// Create test draft template
	draftTemplate := &domain.DraftTemplate{
		Intent:     "follow_up_email",
		CampaignID: uuid.New(),
		TemplateID: template.ID,
	}
	created, err := repo.Create(ctx, draftTemplate)
	require.NoError(t, err)

	// Test FindByID
	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "follow_up_email", found.Intent)
	assert.Equal(t, draftTemplate.CampaignID, found.CampaignID)
	assert.Equal(t, template.ID, found.TemplateID)
}

// TestDraftTemplateRepository_FindByIDWithTemplate tests finding a draft template with its associated template
func TestDraftTemplateRepository_FindByIDWithTemplate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDraftTemplateRepository(db)
	ctx := context.Background()

	// Create a template first
	userID := uuid.New()
	template := createTestTemplate(t, db, userID, nil, "welcome_message_template")

	// Create test draft template
	draftTemplate := &domain.DraftTemplate{
		Intent:     "welcome_message",
		CampaignID: uuid.New(),
		TemplateID: template.ID,
	}
	created, err := repo.Create(ctx, draftTemplate)
	require.NoError(t, err)

	// Test FindByIDWithTemplate
	found, err := repo.FindByIDWithTemplate(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "welcome_message", found.Intent)
	assert.NotNil(t, found.Template)
	assert.Equal(t, template.ID, found.Template.ID)
	assert.Equal(t, "Test Subject", found.Template.Subject)
}

// TestDraftTemplateRepository_FindByCampaignID tests finding draft templates by campaign ID
func TestDraftTemplateRepository_FindByCampaignID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDraftTemplateRepository(db)
	ctx := context.Background()

	// Create templates
	userID := uuid.New()
	template1 := createTestTemplate(t, db, userID, nil, "template1")
	template2 := createTestTemplate(t, db, userID, nil, "template2")

	// Create test draft templates for the same campaign
	campaignID := uuid.New()
	draftTemplate1 := &domain.DraftTemplate{
		Intent:     "welcome",
		CampaignID: campaignID,
		TemplateID: template1.ID,
	}
	_, err := repo.Create(ctx, draftTemplate1)
	require.NoError(t, err)

	draftTemplate2 := &domain.DraftTemplate{
		Intent:     "follow_up",
		CampaignID: campaignID,
		TemplateID: template2.ID,
	}
	_, err = repo.Create(ctx, draftTemplate2)
	require.NoError(t, err)

	// Create a draft template for a different campaign
	draftTemplate3 := &domain.DraftTemplate{
		Intent:     "welcome",
		CampaignID: uuid.New(),
		TemplateID: template1.ID,
	}
	_, err = repo.Create(ctx, draftTemplate3)
	require.NoError(t, err)

	// Test FindByCampaignID
	found, err := repo.FindByCampaignID(ctx, campaignID)
	require.NoError(t, err)
	assert.Len(t, found, 2)

	// Verify the correct draft templates are returned
	intents := []string{found[0].Intent, found[1].Intent}
	assert.Contains(t, intents, "welcome")
	assert.Contains(t, intents, "follow_up")

	// Verify templates are preloaded
	for _, dt := range found {
		assert.NotNil(t, dt.Template)
	}
}

// TestDraftTemplateRepository_FindByCampaignAndIntent tests finding a draft template by campaign ID and intent
func TestDraftTemplateRepository_FindByCampaignAndIntent(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDraftTemplateRepository(db)
	ctx := context.Background()

	// Create a template first
	userID := uuid.New()
	template := createTestTemplate(t, db, userID, nil, "appointment_template")

	// Create test draft template
	campaignID := uuid.New()
	draftTemplate := &domain.DraftTemplate{
		Intent:     "appointment_reminder",
		CampaignID: campaignID,
		TemplateID: template.ID,
	}
	created, err := repo.Create(ctx, draftTemplate)
	require.NoError(t, err)

	// Create another draft template with different intent
	draftTemplate2 := &domain.DraftTemplate{
		Intent:     "welcome",
		CampaignID: campaignID,
		TemplateID: template.ID,
	}
	_, err = repo.Create(ctx, draftTemplate2)
	require.NoError(t, err)

	// Test FindByCampaignAndIntent
	found, err := repo.FindByCampaignAndIntent(ctx, campaignID, "appointment_reminder")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "appointment_reminder", found.Intent)
	assert.Equal(t, campaignID, found.CampaignID)

	// Verify template is preloaded
	assert.NotNil(t, found.Template)
	assert.Equal(t, template.ID, found.Template.ID)

	// Test with non-existent intent
	notFound, err := repo.FindByCampaignAndIntent(ctx, campaignID, "non_existent")
	assert.Error(t, err)
	assert.Nil(t, notFound)
}

// TestDraftTemplateRepository_Update tests updating a draft template
func TestDraftTemplateRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDraftTemplateRepository(db)
	ctx := context.Background()

	// Create templates
	userID := uuid.New()
	template := createTestTemplate(t, db, userID, nil, "original_template")
	newTemplate := createTestTemplate(t, db, userID, nil, "new_template")

	// Create test draft template
	draftTemplate := &domain.DraftTemplate{
		Intent:     "original_intent",
		CampaignID: uuid.New(),
		TemplateID: template.ID,
	}
	created, err := repo.Create(ctx, draftTemplate)
	require.NoError(t, err)

	// Update the draft template
	created.Intent = "updated_intent"
	created.TemplateID = newTemplate.ID

	// Test Update
	err = repo.Update(ctx, created.ID, created)
	require.NoError(t, err)

	// Verify the update
	updated, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated_intent", updated.Intent)
	assert.Equal(t, newTemplate.ID, updated.TemplateID)
}

// TestDraftTemplateRepository_Delete tests deleting a draft template (soft delete)
func TestDraftTemplateRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDraftTemplateRepository(db)
	ctx := context.Background()

	// Create a template first
	userID := uuid.New()
	template := createTestTemplate(t, db, userID, nil, "delete_template")

	// Create test draft template
	draftTemplate := &domain.DraftTemplate{
		Intent:     "delete_intent",
		CampaignID: uuid.New(),
		TemplateID: template.ID,
	}
	created, err := repo.Create(ctx, draftTemplate)
	require.NoError(t, err)

	// Delete draft template
	err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// Verify soft deletion - should not be found with normal FindByID
	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, found) // Soft deleted records should not be found
}

func TestDraftTemplateRepository_FindAllAndNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewDraftTemplateRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	templateA := createTestTemplate(t, db, userID, nil, "type-a")
	templateB := createTestTemplate(t, db, userID, nil, "type-b")

	campaignID := uuid.New()
	_, err := repo.Create(ctx, &domain.DraftTemplate{
		Intent:     "intent-a",
		CampaignID: campaignID,
		TemplateID: templateA.ID,
	})
	require.NoError(t, err)
	second, err := repo.Create(ctx, &domain.DraftTemplate{
		Intent:     "intent-b",
		CampaignID: campaignID,
		TemplateID: templateB.ID,
	})
	require.NoError(t, err)

	// Filtering and pagination
	filter := baseRepo.Filter{"campaign_id": campaignID}
	pagination := &baseRepo.PaginationParams{Limit: 1, Offset: 1}
	result, err := repo.FindAll(ctx, filter, pagination)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
	assert.Len(t, result.List, 1)
	assert.Equal(t, second.ID, result.List[0].ID)
	assert.Equal(t, 1, result.Offset)
	assert.Equal(t, 1, result.Limit)

	// Missing ID should return nil, nil
	missing, err := repo.FindByID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, missing)
}
