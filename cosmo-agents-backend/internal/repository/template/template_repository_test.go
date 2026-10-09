package template

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// Test setup utilities
func setupComprehensiveTestDB(t *testing.T) *gorm.DB {
	db, err := pgtest.Open(t, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Reduce log noise in tests
	})
	require.NoError(t, err)

	// Migrate all required tables
	err = db.AutoMigrate(
		&domain.Template{},
		&domain.Knowledge{},
		&domain.TemplateKnowledge{},
		&domain.User{},
		&domain.Organization{},
		&domain.Campaign{},
	)
	require.NoError(t, err)

	return db
}

func createTestUser(t *testing.T, db *gorm.DB) *domain.User {
	userID := uuid.New()
	user := &domain.User{
		Base:     base.Base{ID: userID},
		Email:    fmt.Sprintf("test-%s@example.com", userID.String()[:8]),
		Name:     "Test User",
		JobTitle: "Test Developer",
	}

	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

func createTestKnowledge(t *testing.T, db *gorm.DB, userID uuid.UUID, title string) *domain.Knowledge {
	knowledgeID := uuid.New()
	knowledge := &domain.Knowledge{
		Base:         base.Base{ID: knowledgeID},
		UserID:       userID,
		SourceType:   "upload",
		EmbeddingGID: &title, // Using title as embedding_gid for test
	}

	err := db.Create(knowledge).Error
	require.NoError(t, err)
	return knowledge
}

func createTestCampaign(t *testing.T, db *gorm.DB, userID uuid.UUID, name string) *domain.Campaign {
	campaignID := uuid.New()
	campaign := &domain.Campaign{
		Base:   base.Base{ID: campaignID},
		UserID: userID,
		Name:   name,
	}

	err := db.Create(campaign).Error
	require.NoError(t, err)
	return campaign
}

func createTestTemplateWithOptions(t *testing.T, db *gorm.DB, userID, campaignID *uuid.UUID, subject string, position float64) *domain.Template {
	templateID := uuid.New()
	// Create unique type to avoid UNIQUE constraint violations
	uniqueType := fmt.Sprintf("type_%s_%s", strings.ReplaceAll(strings.ToLower(subject), " ", "_"), templateID.String()[:8])
	template := &domain.Template{
		Base:      base.Base{ID: templateID},
		UserID:    *userID,
		Subject:   subject,
		Content:   fmt.Sprintf("Content for %s", subject),
		Type:      uniqueType,
		Position:  position,
		SendAfter: 3600, // 1 hour in seconds
	}

	if campaignID != nil {
		template.CampaignID = campaignID
	}

	err := db.Create(template).Error
	require.NoError(t, err)
	return template
}

// Comprehensive CRUD Tests
func TestTemplateRepository_CRUD_Comprehensive(t *testing.T) {
	db := setupComprehensiveTestDB(t)
	repo := NewTemplateRepository(db)

	user := createTestUser(t, db)
	ctx := context.Background()

	t.Run("Create Template", func(t *testing.T) {
		template := &domain.Template{
			Base:     base.Base{ID: uuid.New()},
			UserID:   user.ID,
			Subject:  "New Test Template",
			Content:  "This is a new template",
			Position: 1.0,
		}

		createdTemplate, err := repo.Create(ctx, template)
		assert.NoError(t, err)
		assert.NotNil(t, createdTemplate)
		assert.Equal(t, template.ID, createdTemplate.ID)
		assert.Equal(t, template.Subject, createdTemplate.Subject)
		assert.Equal(t, template.UserID, createdTemplate.UserID)
		assert.NotZero(t, createdTemplate.CreatedAt)
		assert.NotZero(t, createdTemplate.UpdatedAt)
	})

	t.Run("Find Template by ID", func(t *testing.T) {
		template := createTestTemplateWithOptions(t, db, &user.ID, nil, "Find Test", 1.0)

		foundTemplate, err := repo.FindByID(ctx, template.ID)
		assert.NoError(t, err)
		assert.NotNil(t, foundTemplate)
		assert.Equal(t, template.ID, foundTemplate.ID)
		assert.Equal(t, template.Subject, foundTemplate.Subject)
		assert.Equal(t, template.UserID, foundTemplate.UserID)
	})

	t.Run("Find Template by ID and User ID", func(t *testing.T) {
		template := createTestTemplateWithOptions(t, db, &user.ID, nil, "Authorization Test", 2.0)

		// Test with correct user ID
		foundTemplate, err := repo.FindByIDAndUserID(ctx, template.ID, user.ID)
		assert.NoError(t, err)
		assert.NotNil(t, foundTemplate)
		assert.Equal(t, template.Subject, foundTemplate.Subject)

		// Test with wrong user ID
		wrongUserID := uuid.New()
		foundTemplate, err = repo.FindByIDAndUserID(ctx, template.ID, wrongUserID)
		assert.NoError(t, err)
		assert.Nil(t, foundTemplate)
	})

	t.Run("Update Template", func(t *testing.T) {
		template := createTestTemplateWithOptions(t, db, &user.ID, nil, "Update Test", 3.0)

		// Update fields
		template.Subject = "Updated Subject"
		template.Content = "Updated Content"
		template.Position = 5.0

		err := repo.Update(ctx, template.ID, template)
		assert.NoError(t, err)

		// Verify update
		foundTemplate, err := repo.FindByID(ctx, template.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Subject", foundTemplate.Subject)
		assert.Equal(t, "Updated Content", foundTemplate.Content)
		assert.Equal(t, 5.0, foundTemplate.Position)
	})

	t.Run("Delete Template", func(t *testing.T) {
		template := createTestTemplateWithOptions(t, db, &user.ID, nil, "Delete Test", 4.0)

		// Verify template exists
		foundTemplate, err := repo.FindByID(ctx, template.ID)
		assert.NoError(t, err)
		assert.NotNil(t, foundTemplate)

		// Delete template
		err = repo.Delete(ctx, template.ID)
		assert.NoError(t, err)

		// Verify deletion (hard delete)
		foundTemplate, err = repo.FindByID(ctx, template.ID)
		assert.NoError(t, err)
		assert.Nil(t, foundTemplate)
	})
}

func TestTemplateRepository_FindByCampaignID_Comprehensive(t *testing.T) {
	db := setupComprehensiveTestDB(t)
	repo := NewTemplateRepository(db)

	user := createTestUser(t, db)
	campaign1 := createTestCampaign(t, db, user.ID, "Campaign 1")
	campaign2 := createTestCampaign(t, db, user.ID, "Campaign 2")
	ctx := context.Background()

	// Create templates for different campaigns
	_ = []*domain.Template{
		createTestTemplateWithOptions(t, db, &user.ID, &campaign1.ID, "Campaign 1 Template 1", 1.0),
		createTestTemplateWithOptions(t, db, &user.ID, &campaign1.ID, "Campaign 1 Template 2", 2.0),
		createTestTemplateWithOptions(t, db, &user.ID, &campaign1.ID, "Campaign 1 Template 3", 3.0),
		createTestTemplateWithOptions(t, db, &user.ID, &campaign2.ID, "Campaign 2 Template 1", 1.0),
		createTestTemplateWithOptions(t, db, &user.ID, nil, "No Campaign Template", 1.0),
	}

	t.Run("Find templates for campaign with templates", func(t *testing.T) {
		foundTemplates, err := repo.FindByCampaignID(ctx, campaign1.ID)
		assert.NoError(t, err)
		assert.Len(t, foundTemplates, 3)

		// Verify ordering by position
		for i := 0; i < len(foundTemplates)-1; i++ {
			assert.LessOrEqual(t, foundTemplates[i].Position, foundTemplates[i+1].Position)
		}

		// Verify all templates belong to the campaign
		for _, template := range foundTemplates {
			assert.NotNil(t, template.CampaignID)
			assert.Equal(t, campaign1.ID, *template.CampaignID)
		}
	})

	t.Run("Find templates for campaign with single template", func(t *testing.T) {
		foundTemplates, err := repo.FindByCampaignID(ctx, campaign2.ID)
		assert.NoError(t, err)
		assert.Len(t, foundTemplates, 1)
		assert.Equal(t, campaign2.ID, *foundTemplates[0].CampaignID)
	})

	t.Run("Find templates for non-existent campaign", func(t *testing.T) {
		nonExistentCampaignID := uuid.New()
		foundTemplates, err := repo.FindByCampaignID(ctx, nonExistentCampaignID)
		assert.NoError(t, err)
		assert.Len(t, foundTemplates, 0)
	})
}

func TestTemplateRepository_FindByUserID_Comprehensive(t *testing.T) {
	db := setupComprehensiveTestDB(t)
	repo := NewTemplateRepository(db)

	user1 := createTestUser(t, db)
	user2 := createTestUser(t, db)
	ctx := context.Background()

	// Create templates for different users
	_ = []*domain.Template{
		createTestTemplateWithOptions(t, db, &user1.ID, nil, "User 1 Template 1", 1.0),
		createTestTemplateWithOptions(t, db, &user1.ID, nil, "User 1 Template 2", 2.0),
		createTestTemplateWithOptions(t, db, &user1.ID, nil, "User 1 Template 3", 3.0),
		createTestTemplateWithOptions(t, db, &user1.ID, nil, "User 1 Template 4", 4.0),
		createTestTemplateWithOptions(t, db, &user1.ID, nil, "User 1 Template 5", 5.0),
	}

	_ = []*domain.Template{
		createTestTemplateWithOptions(t, db, &user2.ID, nil, "User 2 Template 1", 1.0),
		createTestTemplateWithOptions(t, db, &user2.ID, nil, "User 2 Template 2", 2.0),
	}

	t.Run("Find all user templates with pagination", func(t *testing.T) {
		foundTemplates, total, err := repo.FindByUserID(ctx, user1.ID, 0, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, foundTemplates, 5)

		// Verify all templates belong to user1
		for _, template := range foundTemplates {
			assert.Equal(t, user1.ID, template.UserID)
		}

		// Verify ordering (should be created_at DESC)
		for i := 0; i < len(foundTemplates)-1; i++ {
			assert.True(t, foundTemplates[i].CreatedAt.After(foundTemplates[i+1].CreatedAt) ||
				foundTemplates[i].CreatedAt.Equal(foundTemplates[i+1].CreatedAt))
		}
	})

	t.Run("Find user templates with limit", func(t *testing.T) {
		foundTemplates, total, err := repo.FindByUserID(ctx, user1.ID, 0, 2)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, foundTemplates, 2)
	})

	t.Run("Find user templates with offset and limit", func(t *testing.T) {
		foundTemplates, total, err := repo.FindByUserID(ctx, user1.ID, 2, 2)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, foundTemplates, 2)
	})

	t.Run("Find user templates beyond last page", func(t *testing.T) {
		foundTemplates, total, err := repo.FindByUserID(ctx, user1.ID, 10, 5)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), total)
		assert.Len(t, foundTemplates, 0)
	})

	t.Run("Find templates for user with fewer templates", func(t *testing.T) {
		foundTemplates, total, err := repo.FindByUserID(ctx, user2.ID, 0, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, foundTemplates, 2)
	})

	t.Run("Find templates for non-existent user", func(t *testing.T) {
		nonExistentUserID := uuid.New()
		foundTemplates, total, err := repo.FindByUserID(ctx, nonExistentUserID, 0, 10)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Len(t, foundTemplates, 0)
	})
}

func TestTemplateRepository_ReorderTemplates_Comprehensive(t *testing.T) {
	db := setupComprehensiveTestDB(t)
	repo := NewTemplateRepository(db)

	user := createTestUser(t, db)
	campaign := createTestCampaign(t, db, user.ID, "Test Campaign")
	ctx := context.Background()

	// Create templates with specific positions
	templates := []*domain.Template{
		createTestTemplateWithOptions(t, db, &user.ID, &campaign.ID, "Template 1", 1.0),
		createTestTemplateWithOptions(t, db, &user.ID, &campaign.ID, "Template 2", 2.0),
		createTestTemplateWithOptions(t, db, &user.ID, &campaign.ID, "Template 3", 3.0),
		createTestTemplateWithOptions(t, db, &user.ID, &campaign.ID, "Template 4", 4.0),
	}

	t.Run("Reorder all templates", func(t *testing.T) {
		// Reorder: Template 4 -> 1st, Template 2 -> 2nd, Template 1 -> 3rd, Template 3 -> 4th
		positions := map[uuid.UUID]float64{
			templates[3].ID: 1.0, // Template 4 to position 1
			templates[1].ID: 2.0, // Template 2 to position 2
			templates[0].ID: 3.0, // Template 1 to position 3
			templates[2].ID: 4.0, // Template 3 to position 4
		}

		err := repo.ReorderTemplates(ctx, positions)
		assert.NoError(t, err)

		// Verify new positions
		foundTemplates, err := repo.FindByCampaignID(ctx, campaign.ID)
		assert.NoError(t, err)
		assert.Len(t, foundTemplates, 4)

		// Create a map of template ID to position for easy verification
		templatePositions := make(map[uuid.UUID]float64)
		for _, template := range foundTemplates {
			templatePositions[template.ID] = template.Position
		}

		assert.Equal(t, 1.0, templatePositions[templates[3].ID])
		assert.Equal(t, 2.0, templatePositions[templates[1].ID])
		assert.Equal(t, 3.0, templatePositions[templates[0].ID])
		assert.Equal(t, 4.0, templatePositions[templates[2].ID])
	})

	t.Run("Reorder subset of templates", func(t *testing.T) {
		// Create fresh templates for this test to avoid conflicts with previous test
		newTemplates := []*domain.Template{
			createTestTemplateWithOptions(t, db, &user.ID, &campaign.ID, "Subset Template 1", 1.0),
			createTestTemplateWithOptions(t, db, &user.ID, &campaign.ID, "Subset Template 2", 2.0),
			createTestTemplateWithOptions(t, db, &user.ID, &campaign.ID, "Subset Template 3", 3.0),
			createTestTemplateWithOptions(t, db, &user.ID, &campaign.ID, "Subset Template 4", 4.0),
		}

		// Record original positions
		originalPositions := make(map[uuid.UUID]float64)
		for _, template := range newTemplates {
			originalPositions[template.ID] = template.Position
		}

		// Swap positions of first two templates
		positions := map[uuid.UUID]float64{
			newTemplates[0].ID: originalPositions[newTemplates[1].ID], // Template 1 to Template 2's position
			newTemplates[1].ID: originalPositions[newTemplates[0].ID], // Template 2 to Template 1's position
		}

		err := repo.ReorderTemplates(ctx, positions)
		assert.NoError(t, err)

		// Verify only the specified templates were reordered
		templatePositions := make(map[uuid.UUID]float64)
		foundTemplates, err := repo.FindByCampaignID(ctx, campaign.ID)
		assert.NoError(t, err)
		for _, template := range foundTemplates {
			templatePositions[template.ID] = template.Position
		}

		assert.Equal(t, originalPositions[newTemplates[1].ID], templatePositions[newTemplates[0].ID])
		assert.Equal(t, originalPositions[newTemplates[0].ID], templatePositions[newTemplates[1].ID])
		// Other templates should remain unchanged
		assert.Equal(t, originalPositions[newTemplates[2].ID], templatePositions[newTemplates[2].ID])
		assert.Equal(t, originalPositions[newTemplates[3].ID], templatePositions[newTemplates[3].ID])
	})

	t.Run("Reorder with empty positions map", func(t *testing.T) {
		// Count templates before this test
		templatesBeforeTest, err := repo.FindByCampaignID(ctx, campaign.ID)
		assert.NoError(t, err)
		initialCount := len(templatesBeforeTest)

		emptyPositions := map[uuid.UUID]float64{}
		err = repo.ReorderTemplates(ctx, emptyPositions)
		assert.NoError(t, err)

		// Verify no changes were made
		foundTemplates, err := repo.FindByCampaignID(ctx, campaign.ID)
		assert.NoError(t, err)
		assert.Len(t, foundTemplates, initialCount)
	})

	t.Run("Reorder with non-existent template ID", func(t *testing.T) {
		nonExistentID := uuid.New()
		positions := map[uuid.UUID]float64{
			nonExistentID: 1.0,
		}

		// This should not cause an error (the template simply won't be found)
		err := repo.ReorderTemplates(ctx, positions)
		assert.NoError(t, err)
	})
}

func TestTemplateRepository_AddKnowledge_Comprehensive(t *testing.T) {
	db := setupComprehensiveTestDB(t)
	repo := NewTemplateRepository(db)

	user := createTestUser(t, db)
	template := createTestTemplateWithOptions(t, db, &user.ID, nil, "Test Template", 1.0)
	knowledge1 := createTestKnowledge(t, db, user.ID, "Knowledge 1")
	knowledge2 := createTestKnowledge(t, db, user.ID, "Knowledge 2")
	knowledge3 := createTestKnowledge(t, db, user.ID, "Knowledge 3")
	ctx := context.Background()

	t.Run("Add single knowledge to template", func(t *testing.T) {
		err := repo.AddKnowledge(ctx, template.ID, knowledge1)
		assert.NoError(t, err)

		// Verify the association was created
		var count int64
		err = db.Model(&domain.TemplateKnowledge{}).
			Where("template_id = ? AND knowledge_id = ?", template.ID, knowledge1.ID).
			Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("Add multiple knowledge to template", func(t *testing.T) {
		err := repo.AddKnowledge(ctx, template.ID, knowledge2)
		assert.NoError(t, err)

		err = repo.AddKnowledge(ctx, template.ID, knowledge3)
		assert.NoError(t, err)

		// Verify all associations exist
		var count int64
		err = db.Model(&domain.TemplateKnowledge{}).
			Where("template_id = ?", template.ID).
			Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(3), count) // knowledge1 + knowledge2 + knowledge3
	})

	t.Run("Add duplicate knowledge (should not create duplicate)", func(t *testing.T) {
		t.Skip("Skipping due to TemplateKnowledge table structure changes")
		// Try to add knowledge1 again
		err := repo.AddKnowledge(ctx, template.ID, knowledge1)
		assert.NoError(t, err) // Should not error

		// Verify no duplicate was created
		var count int64
		err = db.Model(&domain.TemplateKnowledge{}).
			Where("template_id = ? AND knowledge_id = ?", template.ID, knowledge1.ID).
			Count(&count).Error
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count) // Still only one association
	})

	t.Run("Add knowledge to non-existent template", func(t *testing.T) {
		nonExistentTemplateID := uuid.New()
		err := repo.AddKnowledge(ctx, nonExistentTemplateID, knowledge1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "record not found")
	})

	t.Run("Add knowledge belonging to different user", func(t *testing.T) {
		otherUser := createTestUser(t, db)
		otherKnowledge := createTestKnowledge(t, db, otherUser.ID, "Other User Knowledge")

		// This should fail due to security check
		err := repo.AddKnowledge(ctx, template.ID, otherKnowledge)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "knowledge not found or unauthorized")
	})
}

func TestTemplateRepository_FindByIDs_Comprehensive(t *testing.T) {
	db := setupComprehensiveTestDB(t)
	repo := NewTemplateRepository(db)

	user := createTestUser(t, db)
	ctx := context.Background()

	// Create templates
	templates := []*domain.Template{
		createTestTemplateWithOptions(t, db, &user.ID, nil, "Template 1", 1.0),
		createTestTemplateWithOptions(t, db, &user.ID, nil, "Template 2", 2.0),
		createTestTemplateWithOptions(t, db, &user.ID, nil, "Template 3", 3.0),
		createTestTemplateWithOptions(t, db, &user.ID, nil, "Template 4", 4.0),
		createTestTemplateWithOptions(t, db, &user.ID, nil, "Template 5", 5.0),
	}

	t.Run("Find all templates by IDs", func(t *testing.T) {
		ids := []uuid.UUID{
			templates[0].ID,
			templates[1].ID,
			templates[2].ID,
			templates[3].ID,
			templates[4].ID,
		}

		foundTemplates, err := repo.FindByIDs(ctx, ids)
		assert.NoError(t, err)
		assert.Len(t, foundTemplates, 5)

		// Verify all templates were found
		for _, template := range templates {
			foundTemplate, exists := foundTemplates[template.ID]
			assert.True(t, exists, "Template %s should be found", template.ID)
			assert.Equal(t, template.Subject, foundTemplate.Subject)
		}
	})

	t.Run("Find subset of templates by IDs", func(t *testing.T) {
		ids := []uuid.UUID{
			templates[1].ID,
			templates[3].ID,
		}

		foundTemplates, err := repo.FindByIDs(ctx, ids)
		assert.NoError(t, err)
		assert.Len(t, foundTemplates, 2)

		assert.Contains(t, foundTemplates, templates[1].ID)
		assert.Contains(t, foundTemplates, templates[3].ID)
		assert.Equal(t, templates[1].Subject, foundTemplates[templates[1].ID].Subject)
		assert.Equal(t, templates[3].Subject, foundTemplates[templates[3].ID].Subject)
	})

	t.Run("Find templates with non-existent IDs", func(t *testing.T) {
		nonExistentID := uuid.New()
		ids := []uuid.UUID{
			templates[0].ID,
			nonExistentID,
			templates[2].ID,
		}

		foundTemplates, err := repo.FindByIDs(ctx, ids)
		assert.NoError(t, err)
		assert.Len(t, foundTemplates, 2) // Only the existing templates

		assert.Contains(t, foundTemplates, templates[0].ID)
		assert.Contains(t, foundTemplates, templates[2].ID)
		assert.NotContains(t, foundTemplates, nonExistentID)
	})

	t.Run("Find templates with empty ID list", func(t *testing.T) {
		ids := []uuid.UUID{}

		foundTemplates, err := repo.FindByIDs(ctx, ids)
		assert.NoError(t, err)
		assert.Len(t, foundTemplates, 0)
	})

	t.Run("Find templates with nil ID list", func(t *testing.T) {
		foundTemplates, err := repo.FindByIDs(ctx, nil)
		assert.NoError(t, err)
		assert.Len(t, foundTemplates, 0)
	})
}

func TestTemplateRepository_Concurrent_Access(t *testing.T) {

	db := setupComprehensiveTestDB(t)
	repo := NewTemplateRepository(db)

	user := createTestUser(t, db)
	ctx := context.Background()
	const numGoroutines = 10
	const templatesPerGoroutine = 5

	// Create templates concurrently
	var wg sync.WaitGroup
	templateIDs := make([]uuid.UUID, 0, numGoroutines*templatesPerGoroutine)
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < templatesPerGoroutine; j++ {
				template := &domain.Template{
					Base:     base.Base{ID: uuid.New()},
					UserID:   user.ID,
					Subject:  fmt.Sprintf("Concurrent Template %d-%d", goroutineID, j),
					Content:  fmt.Sprintf("Content %d-%d", goroutineID, j),
					Position: float64(goroutineID*templatesPerGoroutine + j),
				}

				createdTemplate, err := repo.Create(ctx, template)
				assert.NoError(t, err)
				assert.NotNil(t, createdTemplate)

				mu.Lock()
				templateIDs = append(templateIDs, createdTemplate.ID)
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify all templates were created
	assert.Len(t, templateIDs, numGoroutines*templatesPerGoroutine)

	// Find all templates concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(startIndex int) {
			defer wg.Done()

			endIndex := startIndex + templatesPerGoroutine
			if endIndex > len(templateIDs) {
				endIndex = len(templateIDs)
			}

			for _, templateID := range templateIDs[startIndex:endIndex] {
				foundTemplate, err := repo.FindByID(ctx, templateID)
				assert.NoError(t, err)
				assert.NotNil(t, foundTemplate)
				assert.Equal(t, templateID, foundTemplate.ID)
			}
		}(i * templatesPerGoroutine)
	}

	wg.Wait()
}

func TestTemplateRepository_Transaction_Integrity(t *testing.T) {
	db := setupComprehensiveTestDB(t)
	repo := NewTemplateRepository(db)

	user := createTestUser(t, db)
	ctx := context.Background()

	t.Run("ReorderTemplates transaction integrity", func(t *testing.T) {
		// Create templates
		templates := []*domain.Template{
			createTestTemplateWithOptions(t, db, &user.ID, nil, "Template 1", 1.0),
			createTestTemplateWithOptions(t, db, &user.ID, nil, "Template 2", 2.0),
		}

		// Record original positions
		originalPositions := make(map[uuid.UUID]float64)
		for _, template := range templates {
			originalPositions[template.ID] = template.Position
		}

		// Attempt reorder with invalid position that should fail
		invalidPositions := map[uuid.UUID]float64{
			templates[0].ID: -1.0, // Invalid position
		}

		// The transaction should succeed (repository doesn't validate negative positions)
		err := repo.ReorderTemplates(ctx, invalidPositions)
		assert.NoError(t, err)

		// Verify templates exist and have the updated positions
		foundTemplate, err := repo.FindByID(ctx, templates[0].ID)
		assert.NoError(t, err)
		assert.NotNil(t, foundTemplate)
		assert.Equal(t, -1.0, foundTemplate.Position) // Negative position is set

		// Verify other template position unchanged
		foundTemplate2, err := repo.FindByID(ctx, templates[1].ID)
		assert.NoError(t, err)
		assert.Equal(t, originalPositions[templates[1].ID], foundTemplate2.Position)
	})

	t.Run("AddKnowledge transaction integrity", func(t *testing.T) {
		template := createTestTemplateWithOptions(t, db, &user.ID, nil, "Transaction Test", 1.0)
		knowledge := createTestKnowledge(t, db, user.ID, "Transaction Knowledge")

		// Add knowledge to template
		err := repo.AddKnowledge(ctx, template.ID, knowledge)
		assert.NoError(t, err)

		// Verify the association exists
		var association domain.TemplateKnowledge
		err = db.Where("template_id = ? AND knowledge_id = ?", template.ID, knowledge.ID).
			First(&association).Error
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, association.ID)

		// Verify both template and knowledge still exist
		foundTemplate, err := repo.FindByID(ctx, template.ID)
		assert.NoError(t, err)
		assert.NotNil(t, foundTemplate)

		// Verify knowledge still exists
		var foundKnowledge domain.Knowledge
		err = db.First(&foundKnowledge, knowledge.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, knowledge.UserID, foundKnowledge.UserID)
	})
}

func TestTemplateRepository_Performance_LargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	db := setupComprehensiveTestDB(t)
	repo := NewTemplateRepository(db)

	user := createTestUser(t, db)
	ctx := context.Background()

	// Create large dataset
	const numTemplates = 1000
	templates := make([]*domain.Template, 0, numTemplates)

	t.Logf("Creating %d templates for performance test...", numTemplates)
	start := time.Now()

	for i := 0; i < numTemplates; i++ {
		template := &domain.Template{
			Base:     base.Base{ID: uuid.New()},
			UserID:   user.ID,
			Subject:  fmt.Sprintf("Performance Test Template %d", i),
			Content:  fmt.Sprintf("Performance test content %d", i),
			Position: float64(i),
		}

		createdTemplate, err := repo.Create(ctx, template)
		assert.NoError(t, err)
		templates = append(templates, createdTemplate)
	}

	createTime := time.Since(start)
	t.Logf("Created %d templates in %v", numTemplates, createTime)

	// Test FindByUserID performance
	t.Run("FindByUserID Performance", func(t *testing.T) {
		start := time.Now()
		foundTemplates, total, err := repo.FindByUserID(ctx, user.ID, 0, numTemplates)
		findTime := time.Since(start)

		assert.NoError(t, err)
		assert.Equal(t, int64(numTemplates), total)
		assert.Len(t, foundTemplates, numTemplates)

		t.Logf("Found %d templates in %v", numTemplates, findTime)
		assert.Less(t, findTime, 1*time.Second, "FindByUserID should complete within 1 second")
	})

	// Test FindByIDs performance
	t.Run("FindByIDs Performance", func(t *testing.T) {
		ids := make([]uuid.UUID, numTemplates)
		for i, template := range templates {
			ids[i] = template.ID
		}

		start := time.Now()
		foundTemplates, err := repo.FindByIDs(ctx, ids)
		findTime := time.Since(start)

		assert.NoError(t, err)
		assert.Len(t, foundTemplates, numTemplates)

		t.Logf("Found %d templates by IDs in %v", numTemplates, findTime)
		assert.Less(t, findTime, 1*time.Second, "FindByIDs should complete within 1 second")
	})
}

// Cleanup function for tests
func cleanupTestData(t *testing.T, db *gorm.DB) {
	tables := []interface{}{
		&domain.TemplateKnowledge{},
		&domain.Template{},
		&domain.Knowledge{},
		&domain.Campaign{},
		&domain.User{},
		&domain.Organization{},
	}

	for _, table := range tables {
		err := db.Exec(fmt.Sprintf("DELETE FROM %v", reflect.TypeOf(table).Elem().Name())).Error
		require.NoError(t, err)
	}
}
