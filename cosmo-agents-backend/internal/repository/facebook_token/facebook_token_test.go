package facebooktoken

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

func setupFacebookTokenTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&domain.FacebookToken{})
	require.NoError(t, err)

	return db
}

func createTestFacebookToken(userID uuid.UUID, pageID *string, tokenType string, accessToken string) *domain.FacebookToken {
	token := &domain.FacebookToken{
		UserID:      userID,
		PageID:      pageID,
		TokenType:   domain.TokenTypeFBPage, // Default to page type, will be overridden
		AccessToken: accessToken,
	}
	// Override token type based on string
	if tokenType == "fb_user" {
		token.TokenType = domain.TokenTypeFBUser
	}
	return token
}

func TestNewFacebookTokenRepository(t *testing.T) {
	db := setupFacebookTokenTestDB(t)
	repo := NewFacebookTokenRepository(db)

	assert.NotNil(t, repo)
	assert.NotNil(t, repo.GormRepository)
	assert.Same(t, db, repo.db)
}

func TestFacebookTokenRepository_GetPageToken(t *testing.T) {
	db := setupFacebookTokenTestDB(t)
	repo := NewFacebookTokenRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	pageID := "page123"
	token := createTestFacebookToken(userID, &pageID, "fb_page", "page_access_token")
	db.Create(token)

	t.Run("success", func(t *testing.T) {
		result, err := repo.GetPageToken(ctx, "page123")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, token.ID, result.ID)
		assert.Equal(t, "page123", *result.PageID)
		assert.Equal(t, domain.TokenTypeFBPage, result.TokenType)
		assert.Equal(t, "page_access_token", result.AccessToken)
	})

	t.Run("not found", func(t *testing.T) {
		result, err := repo.GetPageToken(ctx, "nonexistent")

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestFacebookTokenRepository_GetUserToken(t *testing.T) {
	db := setupFacebookTokenTestDB(t)
	repo := NewFacebookTokenRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	token := createTestFacebookToken(userID, nil, "fb_user", "user_access_token")
	db.Create(token)

	t.Run("success", func(t *testing.T) {
		result, err := repo.GetUserToken(ctx, userID)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, token.ID, result.ID)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, domain.TokenTypeFBUser, result.TokenType)
		assert.Equal(t, "user_access_token", result.AccessToken)
	})

	t.Run("not found", func(t *testing.T) {
		nonExistentUserID := uuid.New()
		result, err := repo.GetUserToken(ctx, nonExistentUserID)

		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestFacebookTokenRepository_Create(t *testing.T) {
	db := setupFacebookTokenTestDB(t)
	repo := NewFacebookTokenRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	pageID := "new_page"
	token := createTestFacebookToken(userID, &pageID, "fb_page", "new_access_token")

	t.Run("success", func(t *testing.T) {
		result, err := repo.Create(ctx, token)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEqual(t, uuid.Nil, result.ID)
		assert.Equal(t, "new_access_token", result.AccessToken)

		// Verify it's in the database
		var found domain.FacebookToken
		err = db.Where("id = ?", result.ID).First(&found).Error
		assert.NoError(t, err)
		assert.Equal(t, "new_access_token", found.AccessToken)
	})
}

func TestFacebookTokenRepository_Update(t *testing.T) {
	db := setupFacebookTokenTestDB(t)
	repo := NewFacebookTokenRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	pageID := "update_page"
	token := createTestFacebookToken(userID, &pageID, "fb_page", "old_access_token")
	created, err := repo.Create(ctx, token)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		created.AccessToken = "new_access_token"

		err := repo.Update(ctx, created.ID, created)

		assert.NoError(t, err)

		// Verify the update
		var updated domain.FacebookToken
		err = db.Where("id = ?", created.ID).First(&updated).Error
		assert.NoError(t, err)
		assert.Equal(t, "new_access_token", updated.AccessToken)
	})
}

func TestFacebookTokenRepository_Delete(t *testing.T) {
	db := setupFacebookTokenTestDB(t)
	repo := NewFacebookTokenRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	pageID := "delete_page"
	token := createTestFacebookToken(userID, &pageID, "fb_page", "delete_access_token")
	created, err := repo.Create(ctx, token)
	require.NoError(t, err)

	t.Run("hard delete success", func(t *testing.T) {
		err := repo.Delete(ctx, created.ID)

		assert.NoError(t, err)

		// Should not be found by FindByID
		found, err := repo.FindByID(ctx, created.ID)
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}

func TestFacebookTokenRepository_FindAll(t *testing.T) {
	db := setupFacebookTokenTestDB(t)
	repo := NewFacebookTokenRepository(db)
	ctx := context.Background()

	userID1 := uuid.New()
	userID2 := uuid.New()

	// Create tokens for user 1
	pageID1 := "page1"
	pageID2 := "page2"
	tokens1 := []*domain.FacebookToken{
		createTestFacebookToken(userID1, &pageID1, "fb_page", "token1"),
		createTestFacebookToken(userID1, &pageID2, "fb_page", "token2"),
	}

	// Create token for user 2
	pageID3 := "page3"
	token2 := createTestFacebookToken(userID2, &pageID3, "fb_page", "token3")

	// Save all tokens
	for _, token := range append(tokens1, token2) {
		created, err := repo.Create(ctx, token)
		require.NoError(t, err)
		require.NotNil(t, created)
	}

	t.Run("find all with pagination", func(t *testing.T) {
		pagination := &baseRepo.PaginationParams{
			Offset: 0,
			Limit:  10,
		}
		result, err := repo.FindAll(ctx, nil, pagination)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.List, 3)
		assert.Equal(t, int64(3), result.Total)
	})

	t.Run("find with limit", func(t *testing.T) {
		pagination := &baseRepo.PaginationParams{
			Offset: 0,
			Limit:  2,
		}
		result, err := repo.FindAll(ctx, nil, pagination)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.List, 2)
		assert.Equal(t, int64(3), result.Total)
	})
}

func TestFacebookTokenRepository_SavePageToken(t *testing.T) {
	db := setupFacebookTokenTestDB(t)
	repo := NewFacebookTokenRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	fbUserID := "fb_user_123"
	pageID := "save_page"
	accessToken := "save_access_token"

	t.Run("success", func(t *testing.T) {
		result, err := repo.SavePageToken(ctx, userID, fbUserID, pageID, accessToken)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, userID, result.UserID)
		assert.Equal(t, fbUserID, result.FBUserID)
		assert.Equal(t, pageID, *result.PageID)
		assert.Equal(t, accessToken, result.AccessToken)
		assert.Equal(t, domain.TokenTypeFBPage, result.TokenType)

		// Verify it was saved
		found, err := repo.GetPageToken(ctx, pageID)
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, accessToken, found.AccessToken)
	})
}

func TestFacebookTokenRepository_ComplexScenario(t *testing.T) {
	db := setupFacebookTokenTestDB(t)
	repo := NewFacebookTokenRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	// Create a user token
	userToken := createTestFacebookToken(userID, nil, "fb_user", "user_access_token")
	userToken.FBUserID = "fb_user_123"
	_, err := repo.Create(ctx, userToken)
	require.NoError(t, err)

	// Create a page token using SavePageToken
	pageID := "complex_page"
	pageTokenName := "Complex Page Name"
	pageAccessToken := "page_access_token"
	createdPage, err := repo.SavePageToken(ctx, userID, "fb_user_123", pageID, pageAccessToken)
	require.NoError(t, err)

	// Update page name
	createdPage.PageName = &pageTokenName
	err = repo.Update(ctx, createdPage.ID, createdPage)
	require.NoError(t, err)

	t.Run("verify complex data", func(t *testing.T) {
		// Test finding user token
		foundUser, err := repo.GetUserToken(ctx, userID)
		assert.NoError(t, err)
		assert.NotNil(t, foundUser)
		assert.Equal(t, "fb_user_123", foundUser.FBUserID)
		assert.Equal(t, "user_access_token", foundUser.AccessToken)

		// Test finding page token
		foundPage, err := repo.GetPageToken(ctx, "complex_page")
		assert.NoError(t, err)
		assert.NotNil(t, foundPage)
		assert.Equal(t, "fb_user_123", foundPage.FBUserID)
		assert.Equal(t, "page_access_token", foundPage.AccessToken)
		assert.Equal(t, "Complex Page Name", *foundPage.PageName)

		// Test FindAll with filter for user ID
		filter := baseRepo.Filter{
			"user_id": userID,
		}
		pagination := &baseRepo.PaginationParams{
			Offset: 0,
			Limit:  10,
		}
		result, err := repo.FindAll(ctx, filter, pagination)
		assert.NoError(t, err)
		assert.Len(t, result.List, 2)
		assert.Equal(t, int64(2), result.Total)
	})
}
