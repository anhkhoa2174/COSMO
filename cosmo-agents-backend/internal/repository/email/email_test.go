package email

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *gorm.DB {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	db = db.Session(&gorm.Session{AllowGlobalUpdate: true})

	// Auto-migrate the Email schema
	err = db.AutoMigrate(&domain.Email{})
	require.NoError(t, err)

	return db
}

// TestEmailRepository_Create tests creating a new email
func TestEmailRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := NewEmailRepository(db)
	ctx := context.Background()

	// Test data
	email := &domain.Email{
		FromEmail:      "sender@example.com",
		ToEmail:        "recipient@example.com",
		Subject:        "Test Email",
		Content:        "This is a test email",
		GmailMessageID: "msg123",
		UserID:         uuid.New(),
		Status:         domain.EmailStatusSending,
	}

	// Test Create
	result, err := emailRepo.Create(ctx, email)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.Equal(t, "sender@example.com", result.FromEmail)
	assert.Equal(t, "recipient@example.com", result.ToEmail)
	assert.Equal(t, "Test Email", result.Subject)
}

// TestEmailRepository_FindByID tests finding an email by ID
func TestEmailRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := NewEmailRepository(db)
	ctx := context.Background()

	// Create test email
	email := &domain.Email{
		FromEmail: "sender@example.com",
		ToEmail:   "recipient@example.com",
		Subject:   "Test Email",
		UserID:    uuid.New(),
	}
	created, err := emailRepo.Create(ctx, email)
	require.NoError(t, err)

	// Test FindByID
	found, err := emailRepo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Test Email", found.Subject)
}

// TestEmailRepository_FindByConversationID tests finding emails by conversation ID
func TestEmailRepository_FindByConversationID(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := NewEmailRepository(db)
	ctx := context.Background()

	// Create test emails in same conversation
	conversationID := uuid.New()
	userID := uuid.New()

	email1 := &domain.Email{
		FromEmail:      "sender@example.com",
		ToEmail:        "recipient@example.com",
		Subject:        "Email 1",
		ConversationID: &conversationID,
		UserID:         userID,
	}
	email2 := &domain.Email{
		FromEmail:      "recipient@example.com",
		ToEmail:        "sender@example.com",
		Subject:        "Email 2",
		ConversationID: &conversationID,
		UserID:         userID,
	}

	_, err := emailRepo.Create(ctx, email1)
	require.NoError(t, err)
	_, err = emailRepo.Create(ctx, email2)
	require.NoError(t, err)

	// Test FindByConversationID
	emails, err := emailRepo.FindByConversationID(ctx, conversationID)
	require.NoError(t, err)
	assert.Len(t, emails, 2)

	// Verify they are ordered by created_at ASC
	for i, email := range emails {
		assert.Equal(t, &conversationID, email.ConversationID)
		if i == 0 {
			assert.Equal(t, "Email 1", email.Subject)
		} else {
			assert.Equal(t, "Email 2", email.Subject)
		}
	}
}

// TestEmailRepository_FindByUserID tests finding emails by user ID
func TestEmailRepository_FindByUserID(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := NewEmailRepository(db)
	ctx := context.Background()

	// Create test emails
	userID := uuid.New()
	for i := 0; i < 3; i++ {
		email := &domain.Email{
			FromEmail: fmt.Sprintf("sender%d@example.com", i),
			ToEmail:   "recipient@example.com",
			Subject:   fmt.Sprintf("Email %d", i),
			UserID:    userID,
		}
		_, err := emailRepo.Create(ctx, email)
		require.NoError(t, err)
	}

	// Create email for different user
	differentUserID := uuid.New()
	email := &domain.Email{
		FromEmail: "different@example.com",
		ToEmail:   "recipient@example.com",
		Subject:   "Different User Email",
		UserID:    differentUserID,
	}
	_, err := emailRepo.Create(ctx, email)
	require.NoError(t, err)

	// Test FindByUserID
	emails, total, err := emailRepo.FindByUserID(ctx, userID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, emails, 3)
	assert.Equal(t, int64(3), total)

	// Verify correct user's emails are returned
	for _, email := range emails {
		assert.Equal(t, userID, email.UserID)
	}

	// Test pagination
	emails, total, err = emailRepo.FindByUserID(ctx, userID, 0, 2)
	require.NoError(t, err)
	assert.Len(t, emails, 2)
	assert.Equal(t, int64(3), total)
}

// TestEmailRepository_FindByGmailMessageID tests finding an email by Gmail message ID
func TestEmailRepository_FindByGmailMessageID(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := NewEmailRepository(db)
	ctx := context.Background()

	// Create test email
	email := &domain.Email{
		FromEmail:      "sender@example.com",
		ToEmail:        "recipient@example.com",
		Subject:        "Test Email",
		GmailMessageID: "gmail_msg_12345",
		UserID:         uuid.New(),
	}
	created, err := emailRepo.Create(ctx, email)
	require.NoError(t, err)

	// Test FindByGmailMessageID
	found, err := emailRepo.FindByGmailMessageID(ctx, "gmail_msg_12345")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "gmail_msg_12345", found.GmailMessageID)

	// Test with non-existent message ID
	notFound, err := emailRepo.FindByGmailMessageID(ctx, "non_existent")
	require.Error(t, err)
	assert.Nil(t, notFound)
}

// TestEmailRepository_FindByStatus tests finding emails by status
func TestEmailRepository_FindByStatus(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := NewEmailRepository(db)
	ctx := context.Background()

	// Create test emails with different statuses
	userID := uuid.New()

	sentEmail := &domain.Email{
		FromEmail: "sender@example.com",
		ToEmail:   "recipient@example.com",
		Subject:   "Sent Email",
		UserID:    userID,
		Status:    domain.EmailStatusSent,
	}
	_, err := emailRepo.Create(ctx, sentEmail)
	require.NoError(t, err)

	inboxEmail := &domain.Email{
		FromEmail: "someone@example.com",
		ToEmail:   "recipient@example.com",
		Subject:   "Inbox Email",
		UserID:    userID,
		Status:    domain.EmailStatusInbox,
	}
	_, err = emailRepo.Create(ctx, inboxEmail)
	require.NoError(t, err)

	// Test FindByStatus for sent emails
	emails, total, err := emailRepo.FindByStatus(ctx, userID, domain.EmailStatusSent, 0, 10)
	require.NoError(t, err)
	assert.Len(t, emails, 1)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "Sent Email", emails[0].Subject)

	// Test FindByStatus for inbox emails
	emails, total, err = emailRepo.FindByStatus(ctx, userID, domain.EmailStatusInbox, 0, 10)
	require.NoError(t, err)
	assert.Len(t, emails, 1)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "Inbox Email", emails[0].Subject)
}

// TestEmailRepository_FindLatestByConversationID tests finding the latest email in a conversation
func TestEmailRepository_FindLatestByConversationID(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := NewEmailRepository(db)
	ctx := context.Background()

	// Create test emails
	conversationID := uuid.New()
	userID := uuid.New()

	email1 := &domain.Email{
		FromEmail:      "sender@example.com",
		ToEmail:        "recipient@example.com",
		Subject:        "First Email",
		ConversationID: &conversationID,
		UserID:         userID,
	}
	created1, err := emailRepo.Create(ctx, email1)
	require.NoError(t, err)

	// Wait a moment to ensure different timestamps
	// In real tests, you might need to mock time
	email2 := &domain.Email{
		FromEmail:      "recipient@example.com",
		ToEmail:        "sender@example.com",
		Subject:        "Second Email (Latest)",
		ConversationID: &conversationID,
		UserID:         userID,
	}
	_, err = emailRepo.Create(ctx, email2)
	require.NoError(t, err)

	// Test FindLatestByConversationID
	latest, err := emailRepo.FindLatestByConversationID(ctx, conversationID)
	require.NoError(t, err)
	assert.NotNil(t, latest)
	assert.Equal(t, &conversationID, latest.ConversationID)
	assert.Equal(t, "Second Email (Latest)", latest.Subject)
	assert.NotEqual(t, created1.ID, latest.ID) // Should be the second email

	// Missing conversation should return error
	none, err := emailRepo.FindLatestByConversationID(ctx, uuid.New())
	assert.Error(t, err)
	assert.Nil(t, none)
}

func TestEmailRepository_FindByConversationAndUser(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := NewEmailRepository(db)
	ctx := context.Background()

	conversationID := uuid.New()
	userID := uuid.New()

	first := &domain.Email{
		ConversationID: &conversationID,
		UserID:         userID,
		Subject:        "First",
	}
	second := &domain.Email{
		ConversationID: &conversationID,
		UserID:         userID,
		Subject:        "Second",
	}
	otherUser := &domain.Email{
		ConversationID: &conversationID,
		UserID:         uuid.New(),
		Subject:        "Other",
	}

	_, err := emailRepo.Create(ctx, first)
	require.NoError(t, err)
	require.NoError(t, db.Model(&domain.Email{}).Where("subject = ?", "First").Update("created_at", time.Now().Add(-2*time.Minute)).Error)
	_, err = emailRepo.Create(ctx, second)
	require.NoError(t, err)
	_, err = emailRepo.Create(ctx, otherUser)
	require.NoError(t, err)

	limited, err := emailRepo.FindByConversationAndUser(ctx, conversationID, userID, 1)
	require.NoError(t, err)
	require.Len(t, limited, 1)
	assert.Equal(t, "Second", limited[0].Subject, "should return latest first")

	all, err := emailRepo.FindByConversationAndUser(ctx, conversationID, userID, 0)
	require.NoError(t, err)
	assert.Len(t, all, 2)
	for _, e := range all {
		assert.Equal(t, userID, e.UserID)
	}
}

// TestEmailRepository_FindByGmailMessageIDs tests finding emails by multiple Gmail message IDs
func TestEmailRepository_FindByGmailMessageIDs(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := NewEmailRepository(db)
	ctx := context.Background()

	// Create test emails
	messageIDs := []string{"gmail_msg_1", "gmail_msg_2", "gmail_msg_3"}
	userID := uuid.New()

	for _, msgID := range messageIDs {
		email := &domain.Email{
			FromEmail:      "sender@example.com",
			ToEmail:        "recipient@example.com",
			Subject:        fmt.Sprintf("Email %s", msgID),
			GmailMessageID: msgID,
			UserID:         userID,
		}
		_, err := emailRepo.Create(ctx, email)
		require.NoError(t, err)
	}

	// Test FindByGmailMessageIDs
	result, err := emailRepo.FindByGmailMessageIDs(ctx, messageIDs)
	require.NoError(t, err)
	assert.Len(t, result, 3)

	// Verify all message IDs are present
	for _, msgID := range messageIDs {
		email, exists := result[msgID]
		assert.True(t, exists)
		assert.Equal(t, msgID, email.GmailMessageID)
	}

	// Test with empty message IDs
	emptyResult, err := emailRepo.FindByGmailMessageIDs(ctx, []string{})
	require.NoError(t, err)
	assert.Len(t, emptyResult, 0)
}

func TestEmailRepository_FindByCampaignAndLatestBatch(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := NewEmailRepository(db)
	ctx := context.Background()

	campaignID := uuid.New()
	otherCampaignID := uuid.New()
	conversationA := uuid.New()
	conversationB := uuid.New()
	conversationC := uuid.New()

	emailA := &domain.Email{CampaignID: &campaignID, ConversationID: &conversationA, Subject: "A", UserID: uuid.New()}
	emailB := &domain.Email{CampaignID: &campaignID, ConversationID: &conversationB, Subject: "B", UserID: uuid.New()}
	emailC := &domain.Email{CampaignID: &otherCampaignID, ConversationID: &conversationC, Subject: "C", UserID: uuid.New()}

	_, err := emailRepo.Create(ctx, emailA)
	require.NoError(t, err)
	require.NoError(t, db.Model(&domain.Email{}).Where("subject = ?", "A").Update("created_at", time.Now().Add(-1*time.Hour)).Error)
	_, err = emailRepo.Create(ctx, emailB)
	require.NoError(t, err)
	_, err = emailRepo.Create(ctx, emailC)
	require.NoError(t, err)

	emails, total, err := emailRepo.FindByCampaignID(ctx, campaignID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, emails, 2)
	assert.Equal(t, "B", emails[0].Subject, "newest first")

	latestMap, err := emailRepo.FindLatestByConversationIDs(ctx, []uuid.UUID{conversationA, conversationB, conversationC})
	require.NoError(t, err)
	assert.Len(t, latestMap, 3)
	assert.Equal(t, "B", latestMap[conversationB].Subject)
	assert.Equal(t, "A", latestMap[conversationA].Subject)
	assert.Equal(t, "C", latestMap[conversationC].Subject)

	empty, err := emailRepo.FindLatestByConversationIDs(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

// TestEmailRepository_Update tests updating an email
func TestEmailRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := NewEmailRepository(db)
	ctx := context.Background()

	// Create test email
	email := &domain.Email{
		FromEmail: "sender@example.com",
		ToEmail:   "recipient@example.com",
		Subject:   "Test Email",
		Content:   "Original body",
		UserID:    uuid.New(),
	}
	created, err := emailRepo.Create(ctx, email)
	require.NoError(t, err)

	// Update email
	created.Content = "Updated body"
	created.Status = domain.EmailStatusSent
	err = emailRepo.Update(ctx, created.ID, created)
	require.NoError(t, err)

	// Verify update
	found, err := emailRepo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated body", found.Content)
	assert.Equal(t, domain.EmailStatusSent, found.Status)
}

// TestEmailRepository_Delete tests deleting an email (soft delete)
func TestEmailRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := NewEmailRepository(db)
	ctx := context.Background()

	// Create test email
	email := &domain.Email{
		FromEmail: "sender@example.com",
		ToEmail:   "recipient@example.com",
		Subject:   "Test Email",
		UserID:    uuid.New(),
	}
	created, err := emailRepo.Create(ctx, email)
	require.NoError(t, err)

	// Delete email
	err = emailRepo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// Verify soft deletion - should not be found with normal FindByID
	found, err := emailRepo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, found) // Soft deleted records should not be found
}

// TestEmailRepository_ComplexEmail tests creating a complex email with all fields
func TestEmailRepository_ComplexEmail(t *testing.T) {
	db := setupTestDB(t)
	emailRepo := NewEmailRepository(db)
	ctx := context.Background()

	// Create test data
	userID := uuid.New()
	conversationID := uuid.New()
	campaignID := uuid.New()
	attachments := pq.StringArray{uuid.New().String(), uuid.New().String()}
	labels := pq.StringArray{"inbox", "important", "work"}
	intents := pq.StringArray{"sales_inquiry", "follow_up_required"}

	// Test complex email
	email := &domain.Email{
		UserID:         userID,
		ConversationID: &conversationID,
		GmailMessageID: "gmail_msg_12345",
		FromEmail:      "sales@company.com",
		ToEmail:        "client@example.com",
		Subject:        "Follow-up on our discussion",
		Content:        "Hi, I wanted to follow up on our previous conversation...",
		Attachments:    attachments,
		Labels:         labels,
		Status:         domain.EmailStatusSent,
		CampaignID:     &campaignID,
		Intents:        intents,
	}

	// Create
	result, err := emailRepo.Create(ctx, email)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)

	// Verify all fields were set correctly
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, &conversationID, result.ConversationID)
	assert.Equal(t, "gmail_msg_12345", result.GmailMessageID)
	assert.Equal(t, "sales@company.com", result.FromEmail)
	assert.Equal(t, "client@example.com", result.ToEmail)
	assert.Equal(t, "Follow-up on our discussion", result.Subject)
	assert.Equal(t, attachments, result.Attachments)
	assert.Equal(t, labels, result.Labels)
	assert.Equal(t, domain.EmailStatusSent, result.Status)
	assert.Equal(t, &campaignID, result.CampaignID)
	assert.Equal(t, intents, result.Intents)
}
