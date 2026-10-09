package conversation

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the Conversation schema
	err = db.AutoMigrate(&domain.Conversation{})
	require.NoError(t, err)

	return db
}

// TestConversationRepository_Create tests creating a new conversation
func TestConversationRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Test data
	userID := uuid.New()
	agentID := uuid.New()
	conversation := &domain.Conversation{
		UserID:        userID,
		GmailThreadID: "thread_123",
		Labels:        pq.StringArray{"inbox", "important"},
		Replied:       false,
		Status:        domain.ConversationStatusUnread,
		AssigneeID:    nil,
		Intents:       pq.StringArray{"sales_inquiry"},
		CMetadata:     base.JSONB([]byte(`{"source": "gmail"}`)),
		AgentID:       &agentID,
	}

	// Test Create
	result, err := repo.Create(ctx, conversation)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, "thread_123", result.GmailThreadID)
	assert.Equal(t, pq.StringArray{"inbox", "important"}, result.Labels)
	assert.False(t, result.Replied)
	assert.Equal(t, domain.ConversationStatusUnread, result.Status)
	assert.Equal(t, &agentID, result.AgentID)
}

// TestConversationRepository_FindByID tests finding a conversation by ID
func TestConversationRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Create test conversation
	userID := uuid.New()
	conversation := &domain.Conversation{
		UserID:        userID,
		GmailThreadID: "thread_find_test",
		Status:        domain.ConversationStatusUnread,
	}
	created, err := repo.Create(ctx, conversation)
	require.NoError(t, err)

	// Test FindByID
	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "thread_find_test", found.GmailThreadID)
	assert.Equal(t, userID, found.UserID)
}

// TestConversationRepository_FindByGmailThreadID tests finding a conversation by Gmail thread ID
func TestConversationRepository_FindByGmailThreadID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Create test conversation
	userID := uuid.New()
	conversation := &domain.Conversation{
		UserID:        userID,
		GmailThreadID: "gmail_thread_456",
		Status:        domain.ConversationStatusUnread,
	}
	created, err := repo.Create(ctx, conversation)
	require.NoError(t, err)

	// Test FindByGmailThreadID
	found, err := repo.FindByGmailThreadID(ctx, "gmail_thread_456")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "gmail_thread_456", found.GmailThreadID)

	// Test with non-existent thread ID
	notFound, err := repo.FindByGmailThreadID(ctx, "non_existent")
	assert.Error(t, err)
	assert.Nil(t, notFound)
}

// TestConversationRepository_FindByCampaignID tests finding conversations by campaign ID
func TestConversationRepository_FindByCampaignID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Create test conversations
	userID := uuid.New()
	campaignID := uuid.New()

	conversation1 := &domain.Conversation{
		UserID:        userID,
		CampaignID:    &campaignID,
		GmailThreadID: "thread_1_" + uuid.New().String(),
		Status:        domain.ConversationStatusUnread,
	}
	_, err := repo.Create(ctx, conversation1)
	require.NoError(t, err)

	conversation2 := &domain.Conversation{
		UserID:        userID,
		CampaignID:    &campaignID,
		GmailThreadID: "thread_2_" + uuid.New().String(),
		Status:        domain.ConversationStatusRead,
	}
	_, err = repo.Create(ctx, conversation2)
	require.NoError(t, err)

	// Create a conversation for a different campaign
	differentCampaignID := uuid.New()
	conversation3 := &domain.Conversation{
		UserID:        userID,
		CampaignID:    &differentCampaignID,
		GmailThreadID: "thread_3_" + uuid.New().String(),
		Status:        domain.ConversationStatusUnread,
	}
	_, err = repo.Create(ctx, conversation3)
	require.NoError(t, err)

	// Test FindByCampaignID
	conversations, total, err := repo.FindByCampaignID(ctx, campaignID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, conversations, 2)
	assert.Equal(t, int64(2), total)

	// Verify the correct conversations are returned
	for _, conv := range conversations {
		assert.Equal(t, &campaignID, conv.CampaignID)
	}
}

// TestConversationRepository_FindByUserID tests finding conversations by user ID
func TestConversationRepository_FindByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Create test conversations
	userID := uuid.New()
	for i := 0; i < 3; i++ {
		conversation := &domain.Conversation{
			UserID: userID,
			// string(rune(i)) turned 0 into a NUL byte, not "0". SQLite stored it;
			// PostgreSQL rejects NUL in text, which is how the typo surfaced.
			GmailThreadID: "thread_" + string(rune('0'+i)),
			Status:        domain.ConversationStatusUnread,
		}
		_, err := repo.Create(ctx, conversation)
		require.NoError(t, err)
	}

	// Create a conversation for a different user
	differentUserID := uuid.New()
	conversation := &domain.Conversation{
		UserID: differentUserID,
		Status: domain.ConversationStatusUnread,
	}
	_, err := repo.Create(ctx, conversation)
	require.NoError(t, err)

	// Test FindByUserID
	conversations, total, err := repo.FindByUserID(ctx, userID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, conversations, 3)
	assert.Equal(t, int64(3), total)

	// Verify the correct conversations are returned
	for _, conv := range conversations {
		assert.Equal(t, userID, conv.UserID)
	}
}

// TestConversationRepository_FindUnreadByUserID tests finding unread conversations by user ID
func TestConversationRepository_FindUnreadByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Create test conversations
	userID := uuid.New()

	// Unread conversations
	for i := 0; i < 2; i++ {
		conversation := &domain.Conversation{
			UserID:        userID,
			GmailThreadID: fmt.Sprintf("unread_thread_%d_%s", i, uuid.New().String()),
			Status:        domain.ConversationStatusUnread,
		}
		_, err := repo.Create(ctx, conversation)
		require.NoError(t, err)
	}

	// Read conversation
	conversation := &domain.Conversation{
		UserID:        userID,
		GmailThreadID: fmt.Sprintf("read_thread_%s", uuid.New().String()),
		Status:        domain.ConversationStatusRead,
	}
	_, err := repo.Create(ctx, conversation)
	require.NoError(t, err)

	// Test FindUnreadByUserID
	conversations, total, err := repo.FindUnreadByUserID(ctx, userID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, conversations, 2)
	assert.Equal(t, int64(2), total)

	// Verify only unread conversations are returned
	for _, conv := range conversations {
		assert.Equal(t, domain.ConversationStatusUnread, conv.Status)
	}
}

// TestConversationRepository_SetReplied tests setting the replied flag
func TestConversationRepository_SetReplied(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Create test conversation
	conversation := &domain.Conversation{
		UserID:  uuid.New(),
		Replied: false,
		Status:  domain.ConversationStatusUnread,
	}
	created, err := repo.Create(ctx, conversation)
	require.NoError(t, err)

	// Test SetReplied to true
	err = repo.SetReplied(ctx, created.ID, true)
	require.NoError(t, err)

	// Verify the update
	updated, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.True(t, updated.Replied)

	// Test SetReplied to false
	err = repo.SetReplied(ctx, created.ID, false)
	require.NoError(t, err)

	// Verify the update
	updated, err = repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.False(t, updated.Replied)
}

// TestConversationRepository_FindByAssigneeID tests finding conversations by assignee ID
func TestConversationRepository_FindByAssigneeID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Create test conversations
	userID := uuid.New()
	assigneeID := uuid.New()

	for i := 0; i < 3; i++ {
		conversation := &domain.Conversation{
			UserID:        userID,
			AssigneeID:    &assigneeID,
			GmailThreadID: fmt.Sprintf("assignee_thread_%d_%s", i, uuid.New().String()),
			Status:        domain.ConversationStatusUnread,
		}
		_, err := repo.Create(ctx, conversation)
		require.NoError(t, err)
	}

	// Create a conversation with a different assignee
	differentAssigneeID := uuid.New()
	conversation := &domain.Conversation{
		UserID:        userID,
		AssigneeID:    &differentAssigneeID,
		GmailThreadID: fmt.Sprintf("different_assignee_thread_%s", uuid.New().String()),
		Status:        domain.ConversationStatusUnread,
	}
	_, err := repo.Create(ctx, conversation)
	require.NoError(t, err)

	// Test FindByAssigneeID
	conversations, total, err := repo.FindByAssigneeID(ctx, assigneeID, 0, 10)
	require.NoError(t, err)
	assert.Len(t, conversations, 3)
	assert.Equal(t, int64(3), total)

	// Verify the correct conversations are returned
	for _, conv := range conversations {
		assert.Equal(t, &assigneeID, conv.AssigneeID)
	}
}

// TestConversationRepository_SoftDelete tests soft deleting a conversation
func TestConversationRepository_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Create test conversation
	conversation := &domain.Conversation{
		UserID: uuid.New(),
		Status: domain.ConversationStatusUnread,
	}
	created, err := repo.Create(ctx, conversation)
	require.NoError(t, err)

	// Test SoftDelete
	err = repo.SoftDelete(ctx, created.ID)
	require.NoError(t, err)

	// Verify soft deletion - should not be found with normal FindByID
	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, found) // Soft deleted records should not be found
}

// TestConversationRepository_FindByGmailThreadIDs tests finding conversations by multiple Gmail thread IDs
func TestConversationRepository_FindByGmailThreadIDs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Create test conversations
	threadIDs := []string{"thread_1", "thread_2", "thread_3"}
	userID := uuid.New()

	for _, threadID := range threadIDs {
		conversation := &domain.Conversation{
			UserID:        userID,
			GmailThreadID: threadID,
			Status:        domain.ConversationStatusUnread,
		}
		_, err := repo.Create(ctx, conversation)
		require.NoError(t, err)
	}

	// Test FindByGmailThreadIDs
	result, err := repo.FindByGmailThreadIDs(ctx, threadIDs)
	require.NoError(t, err)
	assert.Len(t, result, 3)

	// Verify all thread IDs are present
	for _, threadID := range threadIDs {
		conversation, exists := result[threadID]
		assert.True(t, exists)
		assert.Equal(t, threadID, conversation.GmailThreadID)
	}

	// Test with empty thread IDs
	emptyResult, err := repo.FindByGmailThreadIDs(ctx, []string{})
	require.NoError(t, err)
	assert.Len(t, emptyResult, 0)
}

// TestConversationRepository_WithAgentUserID tests context with agent user ID
func TestConversationRepository_WithAgentUserID(t *testing.T) {
	ctx := context.Background()
	agentUserID := uuid.New()

	// Test WithAgentUserID
	newCtx := WithAgentUserID(ctx, agentUserID)

	// Test agentUserIDFromContext
	retrievedID, err := agentUserIDFromContext(newCtx)
	require.NoError(t, err)
	assert.Equal(t, agentUserID, retrievedID)
}

// TestConversationRepository_SearchConversations tests searching conversations with filters
func TestConversationRepository_SearchConversations(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Create test conversations
	userID := uuid.New()

	// Unread conversation
	unreadConv := &domain.Conversation{
		UserID:        userID,
		GmailThreadID: fmt.Sprintf("search_unread_%s", uuid.New().String()),
		Status:        domain.ConversationStatusUnread,
		Replied:       false,
	}
	_, err := repo.Create(ctx, unreadConv)
	require.NoError(t, err)

	// Read conversation
	readConv := &domain.Conversation{
		UserID:        userID,
		GmailThreadID: fmt.Sprintf("search_read_%s", uuid.New().String()),
		Status:        domain.ConversationStatusRead,
		Replied:       true,
	}
	_, err = repo.Create(ctx, readConv)
	require.NoError(t, err)

	// Test search without filters
	conversations, total, err := repo.SearchConversations(ctx, userID, "", 0, 10, nil)
	require.NoError(t, err)
	assert.Len(t, conversations, 2)
	assert.Equal(t, int64(2), total)

	// Test search with status filter
	filter := map[string]interface{}{
		"status": domain.ConversationStatusUnread,
	}
	conversations, total, err = repo.SearchConversations(ctx, userID, "", 0, 10, filter)
	require.NoError(t, err)
	assert.Len(t, conversations, 1)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, domain.ConversationStatusUnread, conversations[0].Status)
}

// TestConversationRepository_ComplexConversation tests a complex conversation scenario
func TestConversationRepository_ComplexConversation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	// Create a complex conversation
	userID := uuid.New()
	agentID := uuid.New()
	campaignID := uuid.New()
	assigneeID := uuid.New()

	conversation := &domain.Conversation{
		UserID:        userID,
		GmailThreadID: "complex_thread_789",
		Labels:        pq.StringArray{"inbox", "important", "sales"},
		Replied:       false,
		Status:        domain.ConversationStatusUnread,
		CampaignID:    &campaignID,
		AssigneeID:    &assigneeID,
		Intents:       pq.StringArray{"sales_inquiry", "product_question"},
		CMetadata:     base.JSONB([]byte(`{"source": "gmail", "priority": "high", "tags": ["hot_lead"]}`)),
		AgentID:       &agentID,
	}

	// Create the conversation
	created, err := repo.Create(ctx, conversation)
	require.NoError(t, err)
	assert.NotNil(t, created)

	// Verify all fields were set correctly
	assert.Equal(t, userID, created.UserID)
	assert.Equal(t, "complex_thread_789", created.GmailThreadID)
	assert.Equal(t, pq.StringArray{"inbox", "important", "sales"}, created.Labels)
	assert.False(t, created.Replied)
	assert.Equal(t, domain.ConversationStatusUnread, created.Status)
	assert.Equal(t, &campaignID, created.CampaignID)
	assert.Equal(t, &assigneeID, created.AssigneeID)
	assert.Equal(t, pq.StringArray{"sales_inquiry", "product_question"}, created.Intents)
	assert.Equal(t, &agentID, created.AgentID)

	// Update the conversation
	err = repo.SetReplied(ctx, created.ID, true)
	require.NoError(t, err)

	// Mark as read
	updated, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	updated.Status = domain.ConversationStatusRead
	err = repo.Update(ctx, updated.ID, updated)
	require.NoError(t, err)

	// Verify updates
	final, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.True(t, final.Replied)
	assert.Equal(t, domain.ConversationStatusRead, final.Status)
}

func TestConversationRepository_UpdateRepliedIfTrue(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	convTrue := &domain.Conversation{UserID: uuid.New(), Replied: true, GmailThreadID: uuid.NewString()}
	convFalse := &domain.Conversation{UserID: uuid.New(), Replied: false, GmailThreadID: uuid.NewString()}
	require.NoError(t, db.Create([]*domain.Conversation{convTrue, convFalse}).Error)

	require.NoError(t, repo.UpdateRepliedIfTrue(ctx, convTrue.ID))
	refreshed, err := repo.FindByID(ctx, convTrue.ID)
	require.NoError(t, err)
	assert.False(t, refreshed.Replied)

	require.NoError(t, repo.UpdateRepliedIfTrue(ctx, convFalse.ID))
	refreshedFalse, err := repo.FindByID(ctx, convFalse.ID)
	require.NoError(t, err)
	assert.False(t, refreshedFalse.Replied)
}

func TestConversationRepository_FindByAgentIDWithFilters_Human(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	agentID := uuid.New()
	assignee := uuid.New()
	user := uuid.New()

	match := &domain.Conversation{
		UserID:        user,
		AgentID:       &agentID,
		AssigneeID:    &assignee,
		Status:        domain.ConversationStatusUnread,
		Replied:       false,
		GmailThreadID: uuid.NewString(),
	}
	other := &domain.Conversation{
		UserID:        user,
		AgentID:       &agentID,
		AssigneeID:    nil,
		Status:        domain.ConversationStatusRead,
		Replied:       true,
		GmailThreadID: uuid.NewString(),
	}
	require.NoError(t, db.Create([]*domain.Conversation{match, other}).Error)

	status := string(domain.ConversationStatusUnread)
	replied := false
	convType := domain.ConversationTypeHuman

	results, total, err := repo.FindByAgentIDWithFilters(ctx, agentID, &ConversationSearchFilter{
		Status:  &status,
		Replied: &replied,
	}, &convType, true, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, results, 1)
	assert.Equal(t, match.ID, results[0].ID)
}

func TestConversationRepository_GetDistinctAssignees(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	user := uuid.New()
	a1 := uuid.New()
	a2 := uuid.New()

	convs := []*domain.Conversation{
		{UserID: user, AssigneeID: &a1, GmailThreadID: uuid.NewString()},
		{UserID: user, AssigneeID: &a1, GmailThreadID: uuid.NewString()},
		{UserID: user, AssigneeID: &a2, GmailThreadID: uuid.NewString()},
		{UserID: user, AssigneeID: nil, GmailThreadID: uuid.NewString()},
	}
	require.NoError(t, db.Create(convs).Error)

	assignees, total, err := repo.GetDistinctAssignees(ctx, user, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.ElementsMatch(t, []uuid.UUID{a1, a2}, assignees)
}

func TestAgentUserIDFromContext_Errors(t *testing.T) {
	_, err := agentUserIDFromContext(nil)
	require.Error(t, err)

	_, err = agentUserIDFromContext(context.Background())
	require.Error(t, err)
}

func TestConversationRepository_SearchConversations_InvalidType(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)
	ctx := context.Background()

	_, _, err := repo.SearchConversations(ctx, uuid.New(), "invalid_type", 0, 10, nil)
	require.Error(t, err)
}

func TestConversationRepository_applySafeFiltersValidation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewConversationRepository(db)

	q := db.Model(&domain.Conversation{})
	err := repo.applySafeFilters(q, map[string]interface{}{
		"unknown": true,
	})
	require.Error(t, err)

	err = repo.applySafeFilters(q, map[string]interface{}{
		"is_deleted": map[string]interface{}{"$in": []interface{}{true, false}},
		"replied":    map[string]interface{}{"$eq": true},
	})
	require.NoError(t, err)
}
