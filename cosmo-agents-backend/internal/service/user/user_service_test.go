package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/notification"
	"github.com/rockship/cosmo-agents-go/internal/domain/relations"
	helperRepo "github.com/rockship/cosmo-agents-go/internal/repository/helper"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Migrate all required tables
	err = db.AutoMigrate(
		&domain.User{},
		&domain.Organization{},
		&domain.Role{},
	)
	require.NoError(t, err)

	return db
}

// createTestUser creates a test user
func createTestUser(t *testing.T, db *gorm.DB) *domain.User {
	user := &domain.User{
		Email:    "test@example.com",
		Name:     "Test User",
		JobTitle: "Developer",
	}

	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

// TestNewUserWithRelationsService tests creating a new user service
func TestNewUserWithRelationsService(t *testing.T) {
	db := setupTestDB(t)
	userRepo := user.NewUserRepository(db)
	userRelationsRepo := helperRepo.NewUserWithRelationsRepository(db)

	service := NewUserWithRelationsService(userRepo, userRelationsRepo)
	assert.NotNil(t, service)
	assert.NotNil(t, service.userRepo)
	assert.NotNil(t, service.userRelationsRepo)
}

// TestUserWithRelationsService_GetUserWithOrganizationsForResponse tests getting user with organizations for response
func TestUserWithRelationsService_GetUserWithOrganizationsForResponse(t *testing.T) {
	db := setupTestDB(t)
	userRepo := user.NewUserRepository(db)
	userRelationsRepo := helperRepo.NewUserWithRelationsRepository(db)
	service := NewUserWithRelationsService(userRepo, userRelationsRepo)
	ctx := context.Background()

	// Create a test user
	testUser := createTestUser(t, db)

	// Test getting user with organizations (fallback to basic user since no organizations)
	response, err := service.GetUserWithOrganizationsForResponse(ctx, testUser.ID)
	require.NoError(t, err)
	assert.Equal(t, testUser.ID, response.ID)
	assert.Equal(t, testUser.Email, response.Email)

	// Test getting non-existing user
	nonExistingID := uuid.New()
	_, err = service.GetUserWithOrganizationsForResponse(ctx, nonExistingID)
	assert.Error(t, err)
}

// TestUserWithRelationsService_GetUserWithFullRelationsForResponse tests getting user with full relations for response
func TestUserWithRelationsService_GetUserWithFullRelationsForResponse(t *testing.T) {
	db := setupTestDB(t)
	userRepo := user.NewUserRepository(db)
	userRelationsRepo := helperRepo.NewUserWithRelationsRepository(db)
	service := NewUserWithRelationsService(userRepo, userRelationsRepo)
	ctx := context.Background()

	// Create a test user
	testUser := createTestUser(t, db)

	// Test getting user with full relations (fallback to basic user since no relations)
	response, err := service.GetUserWithFullRelationsForResponse(ctx, testUser.ID)
	require.NoError(t, err)
	assert.Equal(t, testUser.ID, response.ID)
	assert.Equal(t, testUser.Email, response.Email)

	// Test getting non-existing user
	nonExistingID := uuid.New()
	_, err = service.GetUserWithFullRelationsForResponse(ctx, nonExistingID)
	assert.Error(t, err)
}

func TestUserWithRelationsService_ConvertHelpers(t *testing.T) {
	db := setupTestDB(t)
	userRepo := user.NewUserRepository(db)
	userRelationsRepo := helperRepo.NewUserWithRelationsRepository(db)
	service := NewUserWithRelationsService(userRepo, userRelationsRepo)

	userID := uuid.New()
	withOrgs := &relations.UserWithOrganizations{
		User: domain.User{
			Base:  domain.Base{ID: userID},
			Email: "org@example.com",
		},
		Organizations: []domain.Organization{
			{Base: domain.Base{ID: uuid.New()}, Name: "Org1"},
			{Base: domain.Base{ID: uuid.New()}, Name: "Org2"},
		},
	}
	resp := service.convertUserWithOrgsToResponse(withOrgs)
	assert.Equal(t, "org@example.com", resp.Email)
	assert.Len(t, resp.Organizations, 2)

	withFull := &relations.UserWithFullRelations{
		User: domain.User{
			Base:  domain.Base{ID: userID},
			Email: "full@example.com",
		},
		Organizations: []domain.Organization{{Base: domain.Base{ID: uuid.New()}, Name: "FullOrg"}},
		Roles: []domain.Role{{
			Base:           domain.Base{ID: uuid.New()},
			UserID:         userID,
			OrganizationID: uuid.New(),
			Name:           domain.RoleNameAdmin,
			Status:         domain.RoleStatusActive,
		}},
		Notifications: []notification.Notification{{
			Base:       domain.Base{ID: uuid.New()},
			UserID:     userID,
			CampaignID: uuid.New(),
		}},
	}
	fullResp := service.convertUserWithFullRelationsToResponse(withFull)
	assert.Equal(t, "full@example.com", fullResp.Email)
	assert.Len(t, fullResp.Roles, 1)
	assert.Len(t, fullResp.Notifications, 1)
	assert.Len(t, fullResp.Organizations, 1)

	assert.Equal(t, string(domain.RoleNameAdmin), fullResp.Roles[0].Name)
	assert.Equal(t, withFull.Roles[0].OrganizationID, fullResp.Roles[0].OrganizationID)
	assert.Equal(t, withFull.Notifications[0].CampaignID, fullResp.Notifications[0].CampaignID)

	assert.IsType(t, v1schema.UserResponse{}, resp)
}
