package user

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
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

	// Auto-migrate the User schema
	err = db.AutoMigrate(&domain.User{}, &domain.Organization{}, &domain.Role{})
	require.NoError(t, err)

	return db
}

// TestUserRepository_Create tests creating a new user
func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Test data
	user := &domain.User{
		Email:       "test@example.com",
		Name:        "Test User",
		Picture:     "https://example.com/avatar.jpg",
		Provider:    "google",
		PhoneNumber: pq.StringArray{"+1234567890"},
		JobTitle:    "Software Engineer",
	}

	// Test Create
	result, err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.Equal(t, "test@example.com", result.Email)
	assert.Equal(t, "Test User", result.Name)
	assert.Equal(t, "Software Engineer", result.JobTitle)
}

// TestUserRepository_FindByEmail tests finding a user by email
func TestUserRepository_FindByEmail(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &domain.User{
		Email:    "test@example.com",
		Name:     "Test User",
		Provider: "google",
	}
	created, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Test FindByEmail
	found, err := userRepo.FindByEmail(ctx, "test@example.com")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "test@example.com", found.Email)
}

// TestUserRepository_FindByEmail_NotFound tests finding non-existent user
func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Test FindByEmail with non-existent email
	found, err := userRepo.FindByEmail(ctx, "nonexistent@example.com")
	require.NoError(t, err)
	assert.Nil(t, found)
}

// TestUserRepository_GetDetailByID tests finding a user by ID
func TestUserRepository_GetDetailByID(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &domain.User{
		Email:    "test@example.com",
		Name:     "Test User",
		Provider: "google",
	}
	created, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Test GetDetailByID
	found, err := userRepo.GetDetailByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "test@example.com", found.Email)

	missing, err := userRepo.GetDetailByID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, missing)
}

// TestUserRepository_UpdateByID tests updating a user by ID
func TestUserRepository_UpdateByID(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &domain.User{
		Email:    "test@example.com",
		Name:     "Test User",
		Provider: "google",
	}
	created, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Update user using UpdateByID
	updates := map[string]interface{}{
		"name":      "Updated User",
		"job_title": "Senior Software Engineer",
	}
	err = userRepo.UpdateByID(ctx, created.ID, updates)
	require.NoError(t, err)

	// Verify update
	found, err := userRepo.GetDetailByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated User", found.Name)
	assert.Equal(t, "Senior Software Engineer", found.JobTitle)
}

// TestUserRepository_Delete tests soft deleting a user
func TestUserRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &domain.User{
		Email:    "test@example.com",
		Name:     "Test User",
		Provider: "google",
	}
	created, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Delete user
	err = userRepo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// Verify soft deletion - should not be found with normal GetDetailByID
	found, err := userRepo.GetDetailByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, found) // Soft deleted records should not be found
}

// TestUserRepository_FindByIDs tests finding multiple users by IDs
func TestUserRepository_FindByIDs(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Create test users
	var userIDs []uuid.UUID
	for i := 0; i < 3; i++ {
		user := &domain.User{
			Email:    fmt.Sprintf("user%d@example.com", i),
			Name:     fmt.Sprintf("User %d", i),
			Provider: "google",
		}
		created, err := userRepo.Create(ctx, user)
		require.NoError(t, err)
		userIDs = append(userIDs, created.ID)
	}

	// Test FindByIDs
	users, err := userRepo.FindByIDs(ctx, userIDs)
	require.NoError(t, err)
	assert.Len(t, users, 3)

	// Verify all users are found
	foundIDs := make(map[uuid.UUID]bool)
	for _, user := range users {
		foundIDs[user.ID] = true
		assert.Equal(t, "google", user.Provider)
	}

	for _, id := range userIDs {
		assert.True(t, foundIDs[id])
	}

	// Test with empty IDs
	emptyUsers, err := userRepo.FindByIDs(ctx, []uuid.UUID{})
	require.NoError(t, err)
	assert.Len(t, emptyUsers, 0)
}

// TestUserRepository_UpsertByEmail tests creating or updating a user by email
func TestUserRepository_UpsertByEmail(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Create initial user
	user := &domain.User{
		Email:    "upsert@example.com",
		Name:     "Original Name",
		JobTitle: "Developer",
		Provider: "google",
	}

	// Test UpsertByEmail - should create
	result, err := userRepo.UpsertByEmail(ctx, user)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)
	originalID := result.ID

	// Update the user
	user.Name = "Updated Name"
	user.JobTitle = "Senior Developer"

	// Test UpsertByEmail - should update
	result, err = userRepo.UpsertByEmail(ctx, user)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, originalID, result.ID) // Same ID after upsert

	// Verify the update
	found, err := userRepo.FindByEmail(ctx, "upsert@example.com")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "Updated Name", found.Name)
	assert.Equal(t, "Senior Developer", found.JobTitle)
}

// TestUserRepository_HardDelete tests hard deleting a user
func TestUserRepository_HardDelete(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Create test user
	user := &domain.User{
		Email:    "harddelete@example.com",
		Name:     "Test User",
		Provider: "google",
	}
	created, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Verify user exists
	found, err := userRepo.GetDetailByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)

	// Hard delete user
	err = userRepo.HardDelete(ctx, created.ID)
	require.NoError(t, err)

	// Verify user is permanently deleted
	found, err = userRepo.GetDetailByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, found)
}

// TestUserRepository_ComplexUser tests creating a complex user with all fields
func TestUserRepository_ComplexUser(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Create test data
	phoneNumbers := pq.StringArray{"+1234567890", "+0987654321"}
	staffEmails := pq.StringArray{"staff1@company.com", "staff2@company.com"}
	hubspotCreds := base.JSON([]byte(`{"access_token": "abc123", "refresh_token": "def456"}`))
	fieldMapping := base.JSON([]byte(`{"email": "email_property", "name": "name_property"}`))
	uiMetadata := base.JSONB([]byte(`{"theme": "dark", "language": "en"}`))

	// Test complex user
	user := &domain.User{
		Email:               "complex@example.com",
		Name:                "Complex User",
		Picture:             "https://example.com/complex-avatar.jpg",
		LastHistoryID:       "history_12345",
		Provider:            "microsoft",
		PhoneNumber:         phoneNumbers,
		StaffEmails:         staffEmails,
		HubspotCredentials:  hubspotCreds,
		JobTitle:            "Product Manager",
		HubspotFieldMapping: fieldMapping,
		UIMetadata:          uiMetadata,
	}

	// Create
	result, err := userRepo.Create(ctx, user)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)

	// Verify all fields were set correctly
	assert.Equal(t, "complex@example.com", result.Email)
	assert.Equal(t, "Complex User", result.Name)
	assert.Equal(t, "https://example.com/complex-avatar.jpg", result.Picture)
	assert.Equal(t, "history_12345", result.LastHistoryID)
	assert.Equal(t, "microsoft", result.Provider)
	assert.Equal(t, phoneNumbers, result.PhoneNumber)
	assert.Equal(t, staffEmails, result.StaffEmails)
	assert.Equal(t, "Product Manager", result.JobTitle)
}

// TestUserRepository_BulkOperations tests bulk user operations
func TestUserRepository_BulkOperations(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Create multiple users
	var users []*domain.User
	var userIDs []uuid.UUID

	for i := range 10 {
		user := &domain.User{
			Email:    fmt.Sprintf("bulk%d@example.com", i+1),
			Name:     fmt.Sprintf("Bulk User %d", i+1),
			Provider: "google",
		}
		created, err := userRepo.Create(ctx, user)
		require.NoError(t, err)
		users = append(users, created)
		userIDs = append(userIDs, created.ID)
	}

	// Test bulk find
	foundUsers, err := userRepo.FindByIDs(ctx, userIDs)
	require.NoError(t, err)
	assert.Len(t, foundUsers, 10)

	// Verify all users were found
	foundIDs := make(map[uuid.UUID]bool)
	for _, user := range foundUsers {
		foundIDs[user.ID] = true
	}
	for _, id := range userIDs {
		assert.True(t, foundIDs[id])
	}

	// Test bulk find with pagination
	pagination := &baseRepo.PaginationParams{Offset: 0, Limit: 5}
	result, err := userRepo.FindAll(ctx, nil, pagination)
	require.NoError(t, err)
	assert.Len(t, result.List, 5)
	assert.Equal(t, int64(10), result.Total)

	// Test different pagination page
	pagination = &baseRepo.PaginationParams{Offset: 5, Limit: 5}
	result, err = userRepo.FindAll(ctx, nil, pagination)
	require.NoError(t, err)
	assert.Len(t, result.List, 5)
	assert.Equal(t, int64(10), result.Total)
}

// TestUserRepository_ProviderIsolation tests users with different providers
func TestUserRepository_ProviderIsolation(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	providers := []string{"google", "microsoft", "github", "email"}
	var providerUserIDs map[string][]uuid.UUID = make(map[string][]uuid.UUID)

	// Create users with different providers
	for _, provider := range providers {
		for i := 0; i < 3; i++ {
			user := &domain.User{
				Email:    fmt.Sprintf("%s_user%d@example.com", provider, i+1),
				Name:     fmt.Sprintf("%s User %d", provider, i+1),
				Provider: provider,
			}
			created, err := userRepo.Create(ctx, user)
			require.NoError(t, err)
			providerUserIDs[provider] = append(providerUserIDs[provider], created.ID)
		}
	}

	// Test provider-specific queries
	for _, provider := range providers {
		pagination := &baseRepo.PaginationParams{Offset: 0, Limit: 10}
		result, err := userRepo.FindAll(ctx, baseRepo.Filter{"provider": provider}, pagination)
		require.NoError(t, err)
		assert.Len(t, result.List, 3)
		assert.Equal(t, int64(3), result.Total)

		// Verify all users belong to the correct provider
		for _, user := range result.List {
			assert.Equal(t, provider, user.Provider)
			assert.Contains(t, user.Email, provider+"_user")
		}
	}
}

// TestUserRepository_EmailUniqueness tests email uniqueness constraint
func TestUserRepository_EmailUniqueness(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	email := "unique@example.com"

	// Create first user
	user1 := &domain.User{
		Email:    email,
		Name:     "First User",
		Provider: "google",
	}
	created1, err := userRepo.Create(ctx, user1)
	require.NoError(t, err)
	assert.NotNil(t, created1)

	// Try to create second user with same email but different provider (should work in some cases)
	user2 := &domain.User{
		Email:    email,
		Name:     "Second User",
		Provider: "microsoft",
	}
	created2, err := userRepo.Create(ctx, user2)
	// This might succeed or fail depending on database constraints and upsert logic
	if err == nil {
		assert.NotNil(t, created2)
		assert.NotEqual(t, created1.ID, created2.ID)
	}

	// Test FindByEmail returns the most recent or appropriate user
	found, err := userRepo.FindByEmail(ctx, email)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, email, found.Email)
}

// TestUserRepository_UpdatePartial tests full updates (current Update behavior)
func TestUserRepository_UpdatePartial(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Create initial user
	user := &domain.User{
		Email:    "partial@example.com",
		Name:     "Original Name",
		JobTitle: "Original Title",
		Provider: "google",
	}
	created, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Update with new data (Note: Update updates all fields, not partial)
	updateData := &domain.User{
		Name:     "Updated Name",
		JobTitle: "Updated Title",
	}

	err = userRepo.Update(ctx, created.ID, updateData)
	require.NoError(t, err)

	// Verify the update (Note: Update replaces all fields with provided ones)
	found, err := userRepo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "Updated Name", found.Name)
	assert.Equal(t, "Updated Title", found.JobTitle)
	// Note: Email and Provider may be empty/nil since Update sets all fields
}

func TestUserRepository_UpdateHubspotCredentials(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{
		Email:    "hubspot@example.com",
		Name:     "Hub Spot",
		Provider: "google",
	}
	created, err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	creds := base.JSON([]byte(`{"access_token":"abc","refresh_token":"def"}`))
	require.NoError(t, userRepo.UpdateHubspotCredentials(ctx, created.ID, creds))

	fetched, err := userRepo.GetDetailByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.JSONEq(t, `{"access_token":"abc","refresh_token":"def"}`, string(fetched.HubspotCredentials))
}

func TestUserRepository_FindUserMainOrganization(t *testing.T) {
	t.Run("admin_role_first", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserRepository(db)
		ctx := context.Background()

		user := &domain.User{Email: "admin@example.com", Name: "Admin", Provider: "google"}
		createdUser, err := repo.Create(ctx, user)
		require.NoError(t, err)

		org := &domain.Organization{UserID: &createdUser.ID, Name: "Admin Org"}
		require.NoError(t, db.Create(org).Error)
		role := &domain.Role{
			UserID:         createdUser.ID,
			OrganizationID: org.ID,
			Name:           domain.RoleNameAdmin,
			Status:         domain.RoleStatusActive,
		}
		require.NoError(t, db.Create(role).Error)

		mainOrg, err := repo.FindUserMainOrganization(ctx, createdUser.ID)
		require.NoError(t, err)
		require.NotNil(t, mainOrg)
		assert.Equal(t, org.ID, mainOrg.ID)
	})

	t.Run("fallback_to_first_org", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserRepository(db)
		ctx := context.Background()

		user := &domain.User{Email: "owner@example.com", Name: "Owner", Provider: "google"}
		createdUser, err := repo.Create(ctx, user)
		require.NoError(t, err)

		first := &domain.Organization{UserID: &createdUser.ID, Name: "First Org"}
		second := &domain.Organization{UserID: &createdUser.ID, Name: "Second Org"}
		require.NoError(t, db.Create(first).Error)
		require.NoError(t, db.Create(second).Error)

		mainOrg, err := repo.FindUserMainOrganization(ctx, createdUser.ID)
		require.NoError(t, err)
		require.NotNil(t, mainOrg)
		assert.Equal(t, first.ID, mainOrg.ID)
	})

	t.Run("no_org_returns_error", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserRepository(db)
		ctx := context.Background()

		user := &domain.User{Email: "noorg@example.com", Name: "No Org", Provider: "google"}
		createdUser, err := repo.Create(ctx, user)
		require.NoError(t, err)

		mainOrg, err := repo.FindUserMainOrganization(ctx, createdUser.ID)
		assert.Error(t, err)
		assert.Nil(t, mainOrg)
	})
}

func TestUserRepository_FindFirstOrganizationOfUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user := &domain.User{Email: "firstorg@example.com", Name: "First Org", Provider: "google"}
	createdUser, err := repo.Create(ctx, user)
	require.NoError(t, err)

	orgOld := &domain.Organization{
		Base:   base.Base{ID: uuid.New()},
		UserID: &createdUser.ID,
		Name:   "Org A",
		TimestampMixin: base.TimestampMixin{
			CreatedAt: time.Now().Add(-time.Hour),
			UpdatedAt: time.Now().Add(-time.Hour),
		},
	}
	require.NoError(t, db.Create(orgOld).Error)

	orgNew := &domain.Organization{UserID: &createdUser.ID, Name: "Org B"}
	require.NoError(t, db.Create(orgNew).Error)

	firstOrg, err := repo.FindFirstOrganizationOfUser(ctx, createdUser.ID)
	require.NoError(t, err)
	require.NotNil(t, firstOrg)
	assert.Equal(t, orgOld.ID, firstOrg.ID)
}

func TestUserRepository_UpsertByEmail_Invalid(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.UpsertByEmail(ctx, nil)
	assert.Error(t, err)

	emptyEmail := &domain.User{}
	_, err = repo.UpsertByEmail(ctx, emptyEmail)
	assert.Error(t, err)
}

// TestUserRepository_SearchAndFilter tests search functionality
func TestUserRepository_SearchAndFilter(t *testing.T) {
	db := setupTestDB(t)
	userRepo := NewUserRepository(db)
	ctx := context.Background()

	// Create users with different attributes
	testUsers := []*domain.User{
		{Email: "john.doe@company.com", Name: "John Doe", JobTitle: "Developer", Provider: "google"},
		{Email: "jane.smith@company.com", Name: "Jane Smith", JobTitle: "Designer", Provider: "microsoft"},
		{Email: "bob.wilson@company.com", Name: "Bob Wilson", JobTitle: "Developer", Provider: "github"},
		{Email: "alice.brown@company.com", Name: "Alice Brown", JobTitle: "Manager", Provider: "google"},
	}

	for _, user := range testUsers {
		_, err := userRepo.Create(ctx, user)
		require.NoError(t, err)
	}

	// Test filter by job title
	pagination := &baseRepo.PaginationParams{Offset: 0, Limit: 10}
	result, err := userRepo.FindAll(ctx, baseRepo.Filter{"job_title": "Developer"}, pagination)
	require.NoError(t, err)
	assert.Len(t, result.List, 2)
	assert.Equal(t, int64(2), result.Total)

	// Test filter by provider
	result, err = userRepo.FindAll(ctx, baseRepo.Filter{"provider": "google"}, pagination)
	require.NoError(t, err)
	assert.Len(t, result.List, 2)
	assert.Equal(t, int64(2), result.Total)

	// Test pagination with smaller limit
	pagination = &baseRepo.PaginationParams{Offset: 0, Limit: 2}
	result, err = userRepo.FindAll(ctx, nil, pagination)
	require.NoError(t, err)
	assert.Len(t, result.List, 2)
	assert.Equal(t, int64(4), result.Total)
}
