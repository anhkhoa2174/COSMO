package notification

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the Notification schema
	err = db.AutoMigrate(&domain.Notification{})
	require.NoError(t, err)

	return db
}

// TestNotificationRepository_Create tests creating a new notification
func TestNotificationRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	notificationRepo := NewNotificationRepository(db)
	ctx := context.Background()

	// Test data
	userID := uuid.New()
	campaignID := uuid.New()
	notification := &domain.Notification{
		UserID:     userID,
		CampaignID: campaignID,
	}

	// Test Create
	result, err := notificationRepo.Create(ctx, notification)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, campaignID, result.CampaignID)
}

// TestNotificationRepository_FindByID tests finding a notification by ID
func TestNotificationRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	notificationRepo := NewNotificationRepository(db)
	ctx := context.Background()

	// Create test notification
	notification := &domain.Notification{
		UserID:     uuid.New(),
		CampaignID: uuid.New(),
	}
	created, err := notificationRepo.Create(ctx, notification)
	require.NoError(t, err)

	// Test FindByID
	found, err := notificationRepo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.UserID, found.UserID)
	assert.Equal(t, created.CampaignID, found.CampaignID)
}

// TestNotificationRepository_UpsertMany tests creating or updating multiple notifications
func TestNotificationRepository_UpsertMany(t *testing.T) {
	db := setupTestDB(t)
	notificationRepo := NewNotificationRepository(db)
	ctx := context.Background()

	// Test data
	userID := uuid.New()
	campaignID := uuid.New()
	notifications := []domain.Notification{
		{
			UserID:     userID,
			CampaignID: campaignID,
		},
		{
			UserID:     userID,
			CampaignID: uuid.New(),
		},
	}

	// Test UpsertMany
	result, err := notificationRepo.UpsertMany(ctx, notifications)
	require.NoError(t, err)
	assert.Len(t, result, 2)

	// Verify notifications were created
	for i, notification := range result {
		assert.NotEqual(t, uuid.Nil, notification.ID)
		assert.Equal(t, notifications[i].UserID, notification.UserID)
		assert.Equal(t, notifications[i].CampaignID, notification.CampaignID)
	}
}

// TestNotificationRepository_FindByCampaignID tests finding notifications by campaign ID
func TestNotificationRepository_FindByCampaignID(t *testing.T) {
	db := setupTestDB(t)
	notificationRepo := NewNotificationRepository(db)
	ctx := context.Background()

	// Create test notifications
	campaignID := uuid.New()
	for i := 0; i < 3; i++ {
		userID := uuid.New() // Each notification needs a different user ID
		notification := &domain.Notification{
			UserID:     userID,
			CampaignID: campaignID,
		}
		_, err := notificationRepo.Create(ctx, notification)
		require.NoError(t, err)
	}

	// Test FindByCampaignID
	foundNotifications, err := notificationRepo.FindByCampaignID(ctx, campaignID)
	require.NoError(t, err)
	assert.Len(t, foundNotifications, 3)

	for _, notification := range foundNotifications {
		assert.Equal(t, campaignID, notification.CampaignID)
	}
}

// TestNotificationRepository_DeleteByIDs tests deleting notifications by IDs
func TestNotificationRepository_DeleteByIDs(t *testing.T) {
	db := setupTestDB(t)
	notificationRepo := NewNotificationRepository(db)
	ctx := context.Background()

	// Create test notifications
	campaignID := uuid.New()
	var notificationIDs []uuid.UUID
	for i := 0; i < 3; i++ {
		notification := &domain.Notification{
			UserID:     uuid.New(),
			CampaignID: campaignID,
		}
		created, err := notificationRepo.Create(ctx, notification)
		require.NoError(t, err)
		notificationIDs = append(notificationIDs, created.ID)
	}

	// Delete first two notifications
	err := notificationRepo.DeleteByIDs(ctx, campaignID, notificationIDs[:2])
	require.NoError(t, err)

	// Verify deletions
	remainingNotifications, err := notificationRepo.FindByCampaignID(ctx, campaignID)
	require.NoError(t, err)
	assert.Len(t, remainingNotifications, 1)
	assert.Equal(t, notificationIDs[2], remainingNotifications[0].ID)
}

// TestNotificationRepository_Delete tests deleting a notification
func TestNotificationRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	notificationRepo := NewNotificationRepository(db)
	ctx := context.Background()

	// Create test notification
	notification := &domain.Notification{
		UserID:     uuid.New(),
		CampaignID: uuid.New(),
	}
	created, err := notificationRepo.Create(ctx, notification)
	require.NoError(t, err)

	// Delete notification
	err = notificationRepo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// Verify deletion
	found, err := notificationRepo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, found)
}
