package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/integration"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

func setupIntegrationTestDB(t *testing.T) *gorm.DB {
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&domain.Integration{})
	require.NoError(t, err)

	return db
}

func TestIntegrationRepository_Create(t *testing.T) {
	db := setupIntegrationTestDB(t)
	repo := NewIntegrationRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	source := integration.SourceIntegrationHubspot

	// Test Create
	integration := &domain.Integration{
		ID:         uuid.New().String(),
		UserID:     userID,
		Source:     source,
		Config:     base.JSONB(`{"test": "config"}`),
		Credential: base.JSONB(`{"token": "secret"}`),
	}

	_, err := repo.Create(ctx, integration)
	require.NoError(t, err)
	assert.NotEmpty(t, integration.ID)
}

func TestIntegrationRepository_FindByUserIDAndSource(t *testing.T) {
	db := setupIntegrationTestDB(t)
	repo := NewIntegrationRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	source := integration.SourceIntegrationHubspot

	// Create test integration
	testIntegration := &domain.Integration{
		ID:         uuid.New().String(),
		UserID:     userID,
		Source:     source,
		Config:     base.JSONB(`{"api_key": "test"}`),
		Credential: base.JSONB(`{"token": "test_token"}`),
	}

	_, err := repo.Create(ctx, testIntegration)
	require.NoError(t, err)

	// Test FindByUserIDAndSource - existing integration
	found, err := repo.FindByUserIDAndSource(ctx, userID, source)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, testIntegration.ID, found.ID)

	// Test FindByUserIDAndSource - non-existing integration
	found, err = repo.FindByUserIDAndSource(ctx, uuid.New(), integration.SourceIntegrationHubspot)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestIntegrationRepository_UpdateConfig(t *testing.T) {
	db := setupIntegrationTestDB(t)
	repo := NewIntegrationRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	testIntegration := &domain.Integration{
		ID:     uuid.New().String(),
		UserID: userID,
		Source: integration.SourceIntegrationHubspot,
		Config: base.JSONB(`{"old": "config"}`),
	}

	_, err := repo.Create(ctx, testIntegration)
	require.NoError(t, err)

	// Test UpdateConfig
	newConfig := map[string]interface{}{
		"new":    "value",
		"number": 123,
	}

	err = repo.UpdateConfig(ctx, testIntegration.ID, newConfig)
	require.NoError(t, err)

	// Verify update by querying directly
	var updated domain.Integration
	err = db.Where("id = ?", testIntegration.ID).First(&updated).Error
	require.NoError(t, err)
	assert.Contains(t, string(updated.Config), "new")
	assert.Contains(t, string(updated.Config), "123")

	// Test UpdateConfig with invalid ID
	err = repo.UpdateConfig(ctx, "invalid-uuid", newConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid integration ID format")
}

func TestIntegrationRepository_UpdateCredential(t *testing.T) {
	db := setupIntegrationTestDB(t)
	repo := NewIntegrationRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	testIntegration := &domain.Integration{
		ID:         uuid.New().String(),
		UserID:     userID,
		Source:     integration.SourceIntegrationHubspot,
		Credential: base.JSONB(`{"old": "credential"}`),
	}

	_, err := repo.Create(ctx, testIntegration)
	require.NoError(t, err)

	// Test UpdateCredential
	newCredential := map[string]interface{}{
		"access_token":  "new_token",
		"refresh_token": "refresh",
	}

	err = repo.UpdateCredential(ctx, testIntegration.ID, newCredential)
	require.NoError(t, err)

	// Verify update by querying directly
	var updated domain.Integration
	err = db.Where("id = ?", testIntegration.ID).First(&updated).Error
	require.NoError(t, err)
	assert.Contains(t, string(updated.Credential), "new_token")
	assert.Contains(t, string(updated.Credential), "refresh")

	// Test UpdateCredential with invalid ID
	err = repo.UpdateCredential(ctx, "not-a-uuid", newCredential)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid integration ID format")
}

func TestIntegrationRepository_FindByUserIDAndSourceForUpdate(t *testing.T) {
	db := setupIntegrationTestDB(t)
	repo := NewIntegrationRepository(db)
	ctx := context.Background()

	userID := uuid.New()
	source := integration.SourceIntegrationHubspot

	// Create test integration
	testIntegration := &domain.Integration{
		ID:         uuid.New().String(),
		UserID:     userID,
		Source:     source,
		Config:     base.JSONB(`{"test": "config"}`),
		Credential: base.JSONB(`{"token": "secret"}`),
	}

	_, err := repo.Create(ctx, testIntegration)
	require.NoError(t, err)

	// Test FindByUserIDAndSourceForUpdate
	// Note: SQLite doesn't support FOR UPDATE NOWAIT, but we can test the basic functionality
	found, err := repo.FindByUserIDAndSourceForUpdate(ctx, userID, source)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, testIntegration.ID, found.ID)

	// Test with non-existing integration
	found, err = repo.FindByUserIDAndSourceForUpdate(ctx, uuid.New(), integration.SourceIntegrationHubspot)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestIntegrationRepository_FindAll(t *testing.T) {
	db := setupIntegrationTestDB(t)
	repo := NewIntegrationRepository(db)
	ctx := context.Background()

	userID := uuid.New()

	// Create multiple integrations
	integrations := []domain.Integration{
		{ID: uuid.New().String(), UserID: userID, Source: integration.SourceIntegrationHubspot},
		{ID: uuid.New().String(), UserID: userID, Source: integration.SourceIntegrationHubspot},
		{ID: uuid.New().String(), UserID: uuid.New(), Source: integration.SourceIntegrationHubspot},
	}

	for i := range integrations {
		err := db.Create(&integrations[i]).Error
		require.NoError(t, err)
	}

	// Test FindAll without filter
	all, err := repo.FindAll(ctx, map[string]interface{}{}, &baseRepo.PaginationParams{Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 3, len(all.List))

	// Test FindAll with filter by user_id
	filter := map[string]interface{}{"user_id": userID}
	result, err := repo.FindAll(ctx, filter, &baseRepo.PaginationParams{Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 2, len(result.List))

	for _, integration := range result.List {
		assert.Equal(t, userID, integration.UserID)
	}
}

func TestIntegrationRepository_Delete(t *testing.T) {
	// Skip this test as Integration model doesn't support soft delete
	t.Skip("Integration model doesn't have is_deleted field")
}
