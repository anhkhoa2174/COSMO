package personalapikey

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/personal_api_key"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates a test database with workaround for timestamp issues
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the PersonalApiKey schema
	err = db.AutoMigrate(&personal_api_key.PersonalApiKey{})
	require.NoError(t, err)

	return db
}

// TestPersonalApiKeyRepository_NewPersonalApiKeyRepository tests creating a new repository
func TestPersonalApiKeyRepository_NewPersonalApiKeyRepository(t *testing.T) {
	db := setupTestDB(t)
	secret := "test-secret-key-12345"

	// Test NewPersonalApiKeyRepository
	repo := NewPersonalApiKeyRepository(db, secret)
	assert.NotNil(t, repo)
}

// TestPersonalApiKeyRepository_NewPersonalApiKeyRepository_EmptySecret tests creating repository with empty secret
func TestPersonalApiKeyRepository_NewPersonalApiKeyRepository_EmptySecret(t *testing.T) {
	db := setupTestDB(t)

	// Test NewPersonalApiKeyRepository with empty secret
	repo := NewPersonalApiKeyRepository(db, "")
	assert.NotNil(t, repo)
}

// TestPersonalApiKeyRepository_GenerateAPIKey tests generating API keys
func TestPersonalApiKeyRepository_GenerateAPIKey(t *testing.T) {
	db := setupTestDB(t)
	secret := "test-secret-key-12345"
	repo := NewPersonalApiKeyRepository(db, secret)

	// Test GenerateAPIKey
	rawKey, hashedKey, err := repo.GenerateAPIKey()
	require.NoError(t, err)
	assert.NotEmpty(t, rawKey)
	assert.NotEmpty(t, hashedKey)
	assert.NotEqual(t, rawKey, hashedKey)
	assert.Contains(t, rawKey, "p_api_key_")
	assert.NotContains(t, hashedKey, "p_api_key_") // Hashed key should not have prefix
}

// TestPersonalApiKeyRepository_GenerateAPIKey_EmptySecret tests generating API key with empty secret
func TestPersonalApiKeyRepository_GenerateAPIKey_EmptySecret(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPersonalApiKeyRepository(db, "")

	// Test GenerateAPIKey with empty secret should fail
	rawKey, hashedKey, err := repo.GenerateAPIKey()
	assert.Error(t, err)
	assert.Equal(t, ErrMissingPersonalAPIKeySecret, err)
	assert.Empty(t, rawKey)
	assert.Empty(t, hashedKey)
}

// TestPersonalApiKeyRepository_Create tests creating a personal API key
func TestPersonalApiKeyRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	secret := "test-secret-key-12345"
	repo := NewPersonalApiKeyRepository(db, secret)
	ctx := context.Background()

	// Generate API key first
	_, hashedKey, err := repo.GenerateAPIKey()
	require.NoError(t, err)

	// Test data
	userID := uuid.New()
	apiKey := &personal_api_key.PersonalApiKey{
		UserID:    userID,
		Name:      "Test API Key",
		HashedKey: hashedKey,
		Prefix:    "p_api_key_",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Test Create
	result, err := repo.Create(ctx, apiKey)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, "Test API Key", result.Name)
	assert.Equal(t, hashedKey, result.HashedKey)
}

// TestPersonalApiKeyRepository_FindByID tests finding an API key by ID
func TestPersonalApiKeyRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	secret := "test-secret-key-12345"
	repo := NewPersonalApiKeyRepository(db, secret)
	ctx := context.Background()

	// Create test API key
	_, hashedKey, err := repo.GenerateAPIKey()
	require.NoError(t, err)

	userID := uuid.New()
	apiKey := &personal_api_key.PersonalApiKey{
		UserID:    userID,
		Name:      "Find Test Key",
		HashedKey: hashedKey,
		Prefix:    "p_api_key_",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	created, err := repo.Create(ctx, apiKey)
	require.NoError(t, err)

	// Test FindByID
	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Find Test Key", found.Name)
}

// TestPersonalApiKeyRepository_FindByUserID tests finding API keys by user ID
func TestPersonalApiKeyRepository_FindByUserID(t *testing.T) {
	db := setupTestDB(t)
	secret := "test-secret-key-12345"
	repo := NewPersonalApiKeyRepository(db, secret)
	ctx := context.Background()

	// Create multiple API keys for the same user
	userID := uuid.New()
	var createdKeys []uuid.UUID

	for i := 0; i < 3; i++ {
		_, hashedKey, err := repo.GenerateAPIKey()
		require.NoError(t, err)

		apiKey := &personal_api_key.PersonalApiKey{
			UserID:    userID,
			Name:      fmt.Sprintf("API Key %d", i),
			HashedKey: hashedKey,
			Prefix:    "p_api_key_",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		created, err := repo.Create(ctx, apiKey)
		require.NoError(t, err)
		createdKeys = append(createdKeys, created.ID)
	}

	// Create API key for different user
	differentUserID := uuid.New()
	_, hashedKey2, err := repo.GenerateAPIKey()
	require.NoError(t, err)

	apiKey2 := &personal_api_key.PersonalApiKey{
		UserID:    differentUserID,
		Name:      "Different User Key",
		HashedKey: hashedKey2,
		Prefix:    "p_api_key_",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_, err = repo.Create(ctx, apiKey2)
	require.NoError(t, err)

	// Test FindByUserID
	pagination := &baseRepo.PaginationParams{Offset: 0, Limit: 10}
	result, err := repo.FindByUserID(ctx, userID, pagination)
	require.NoError(t, err)
	assert.Len(t, result.List, 3)
	assert.Equal(t, int64(3), result.Total)

	// Verify all keys belong to the user
	for _, key := range result.List {
		assert.Equal(t, userID, key.UserID)
	}
}

// TestPersonalApiKeyRepository_FindByHashedKey tests finding an API key by hashed key
func TestPersonalApiKeyRepository_FindByHashedKey(t *testing.T) {
	db := setupTestDB(t)
	secret := "test-secret-key-12345"
	repo := NewPersonalApiKeyRepository(db, secret)
	ctx := context.Background()

	// Create test API key
	_, hashedKey, err := repo.GenerateAPIKey()
	require.NoError(t, err)

	userID := uuid.New()
	apiKey := &personal_api_key.PersonalApiKey{
		UserID:    userID,
		Name:      "Hashed Key Test",
		HashedKey: hashedKey,
		Prefix:    "p_api_key_",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	created, err := repo.Create(ctx, apiKey)
	require.NoError(t, err)

	// Test FindByHashedKey
	found, err := repo.FindByHashedKey(ctx, hashedKey)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Hashed Key Test", found.Name)
	assert.Equal(t, hashedKey, found.HashedKey)
}

// TestPersonalApiKeyRepository_UpdateLastUsed tests updating the last used timestamp
func TestPersonalApiKeyRepository_UpdateLastUsed(t *testing.T) {
	db := setupTestDB(t)
	secret := "test-secret-key-12345"
	repo := NewPersonalApiKeyRepository(db, secret)
	ctx := context.Background()

	// Create test API key
	_, hashedKey, err := repo.GenerateAPIKey()
	require.NoError(t, err)

	userID := uuid.New()
	apiKey := &personal_api_key.PersonalApiKey{
		UserID:    userID,
		Name:      "Last Used Test",
		HashedKey: hashedKey,
		Prefix:    "p_api_key_",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	created, err := repo.Create(ctx, apiKey)
	require.NoError(t, err)

	// Verify LastUsedAt is initially nil
	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, found.LastUsedAt)

	// Test UpdateLastUsed
	err = repo.UpdateLastUsed(ctx, created.ID)
	require.NoError(t, err)

	// Verify LastUsedAt was updated
	updated, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, updated.LastUsedAt)
}

// TestPersonalApiKeyRepository_Delete tests deleting an API key
func TestPersonalApiKeyRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	secret := "test-secret-key-12345"
	repo := NewPersonalApiKeyRepository(db, secret)
	ctx := context.Background()

	// Create test API key
	_, hashedKey, err := repo.GenerateAPIKey()
	require.NoError(t, err)

	userID := uuid.New()
	apiKey := &personal_api_key.PersonalApiKey{
		UserID:    userID,
		Name:      "Delete Test",
		HashedKey: hashedKey,
		Prefix:    "p_api_key_",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	created, err := repo.Create(ctx, apiKey)
	require.NoError(t, err)

	// Test Delete
	err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// Verify hard deletion
	found, err := repo.FindByID(ctx, created.ID)
	assert.NoError(t, err)
	assert.Nil(t, found) // Should be nil when not found
}

// TestPersonalApiKeyRepository_ComplexKey tests creating a complex API key
func TestPersonalApiKeyRepository_ComplexKey(t *testing.T) {
	db := setupTestDB(t)
	secret := "super-secret-key-for-testing-12345"
	repo := NewPersonalApiKeyRepository(db, secret)
	ctx := context.Background()

	// Generate API key
	rawKey, hashedKey, err := repo.GenerateAPIKey()
	require.NoError(t, err)

	// Test data
	userID := uuid.New()
	expiresAt := time.Now().Add(365 * 24 * time.Hour) // 1 year

	apiKey := &personal_api_key.PersonalApiKey{
		UserID:    userID,
		Name:      "Production API Key for External Integration",
		HashedKey: hashedKey,
		Prefix:    "p_api_key_",
		ExpiresAt: expiresAt,
	}

	// Create
	result, err := repo.Create(ctx, apiKey)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)

	// Verify all fields were set correctly
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, "Production API Key for External Integration", result.Name)
	assert.Equal(t, hashedKey, result.HashedKey)
	assert.Equal(t, "p_api_key_", result.Prefix)
	assert.WithinDuration(t, expiresAt, result.ExpiresAt, time.Second)

	// Test key generation consistency
	assert.Contains(t, rawKey, "p_api_key_")
	assert.GreaterOrEqual(t, len(rawKey), len("p_api_key_")+32) // At least 32 chars base64
	assert.Len(t, hashedKey, 64)                                // 32 bytes = 64 hex chars

	// Test authentication flow
	authKey, err := repo.FindByHashedKey(ctx, hashedKey)
	require.NoError(t, err)
	assert.NotNil(t, authKey)
	assert.Equal(t, result.ID, authKey.ID)

	// Test last used update
	err = repo.UpdateLastUsed(ctx, authKey.ID)
	require.NoError(t, err)
}
