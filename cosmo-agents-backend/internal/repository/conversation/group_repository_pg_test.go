package conversation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
)

type inboxFixture struct {
	db     *gorm.DB
	convs  *ConversationRepository
	groups *GroupRepository
	agent  uuid.UUID
	owner  uuid.UUID
}

func newInboxFixture(t *testing.T) *inboxFixture {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Conversation{}, &domain.ConversationGroup{}, &domain.ConversationGroupMember{}))
	// The case-insensitive name uniqueness lives in migration 000057, not in
	// the model tags.
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX uq_conversation_groups_user_name
		ON conversation_groups (user_id, lower(btrim(name)))`).Error)
	// Only the email columns the inbox filters read.
	require.NoError(t, db.Exec(`CREATE TABLE emails (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(), conversation_id UUID,
		subject TEXT, content TEXT, from_email TEXT, to_email TEXT,
		intents TEXT[] DEFAULT '{}')`).Error)
	return &inboxFixture{
		db: db, convs: NewConversationRepository(db), groups: NewGroupRepository(db),
		agent: uuid.New(), owner: uuid.New(),
	}
}

// addConversation creates a thread on the fixture's agent with one email.
func (f *inboxFixture) addConversation(t *testing.T, subject string, intents []string, draft string, updated time.Time) uuid.UUID {
	t.Helper()
	meta := `{}`
	if draft != "" {
		meta = `{"ai_reply":{"draft_content":"` + draft + `"}}`
	}
	conv := &domain.Conversation{
		UserID: f.owner, AgentID: &f.agent, GmailThreadID: uuid.NewString(),
		Intents: pq.StringArray(intents), CMetadata: []byte(meta),
	}
	require.NoError(t, f.db.Create(conv).Error)
	require.NoError(t, f.db.Exec(`UPDATE conversations SET updated_at = ? WHERE id = ?`, updated, conv.ID).Error)
	require.NoError(t, f.db.Exec(`INSERT INTO emails (conversation_id, subject, content, from_email, to_email)
		VALUES (?, ?, 'body', 'prospect@acme.com', 'rep@us.com')`, conv.ID, subject).Error)
	return conv.ID
}

func (f *inboxFixture) search(t *testing.T, filter *ConversationSearchFilter) []uuid.UUID {
	t.Helper()
	rows, _, err := f.convs.FindByAgentIDWithFilters(context.Background(), f.agent, filter, nil, true, 0, 50)
	require.NoError(t, err)
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids
}

// The inbox's search box and filter menu were sent to this query but ignored,
// so every search returned the whole inbox.
func TestFindByAgentIDWithFilters_InboxFilters(t *testing.T) {
	f := newInboxFixture(t)
	now := time.Now()
	pricing := f.addConversation(t, "Pricing question", []string{"REQUEST_FOR_PRICING"}, "Here is our pricing", now)
	decline := f.addConversation(t, "Not now", []string{"NOT_INTERESTED"}, "", now.AddDate(0, 0, -20))
	// Stored as the display string, as the classifier pipeline writes it.
	info := f.addConversation(t, "Demo please", []string{"Request for information"}, "", now.AddDate(0, 0, -20))
	yes, no := true, false

	tests := []struct {
		name   string
		filter *ConversationSearchFilter
		want   []uuid.UUID
	}{
		{"no filter", &ConversationSearchFilter{}, []uuid.UUID{pricing, decline, info}},
		{"text matches an email subject", &ConversationSearchFilter{Q: "pricing"}, []uuid.UUID{pricing}},
		{"text that matches nothing", &ConversationSearchFilter{Q: "zzzz"}, []uuid.UUID{}},
		// % must be literal, or it matches every thread.
		{"wildcards are escaped", &ConversationSearchFilter{Q: "%"}, []uuid.UUID{}},
		{"intent", &ConversationSearchFilter{Intent: "NOT_INTERESTED"}, []uuid.UUID{decline}},
		{"intent key matches a stored display label", &ConversationSearchFilter{Intent: "REQUEST_FOR_INFORMATION"}, []uuid.UUID{info}},
		{"has a draft", &ConversationSearchFilter{HasDraft: &yes}, []uuid.UUID{pricing}},
		{"has no draft", &ConversationSearchFilter{HasDraft: &no}, []uuid.UUID{decline, info}},
		{"updated in the last week", &ConversationSearchFilter{SinceDays: 7}, []uuid.UUID{pricing}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ElementsMatch(t, tt.want, f.search(t, tt.filter))
		})
	}
}

func TestGroupRepository_PersonalGroups(t *testing.T) {
	f := newInboxFixture(t)
	ctx := context.Background()
	other := uuid.New()
	a := f.addConversation(t, "Acme renewal", nil, "", time.Now())
	b := f.addConversation(t, "Globex intro", nil, "", time.Now())

	vip := &domain.ConversationGroup{UserID: f.owner, Name: "VIP", Color: "amber"}
	require.NoError(t, f.groups.Create(ctx, vip))
	// Names are unique per user regardless of case, but another user may reuse one.
	assert.ErrorIs(t, f.groups.Create(ctx, &domain.ConversationGroup{UserID: f.owner, Name: "vip", Color: "slate"}), ErrGroupNameTaken)
	theirs := &domain.ConversationGroup{UserID: other, Name: "VIP", Color: "slate"}
	require.NoError(t, f.groups.Create(ctx, theirs))

	require.NoError(t, f.groups.AddConversation(ctx, vip.ID, a))
	require.NoError(t, f.groups.AddConversation(ctx, vip.ID, a), "adding twice is not an error")
	require.NoError(t, f.groups.AddConversation(ctx, theirs.ID, b))

	t.Run("filter by own group", func(t *testing.T) {
		assert.Equal(t, []uuid.UUID{a}, f.search(t, &ConversationSearchFilter{GroupID: &vip.ID, GroupUserID: f.owner}))
	})
	t.Run("another user's group filters to nothing", func(t *testing.T) {
		assert.Empty(t, f.search(t, &ConversationSearchFilter{GroupID: &theirs.ID, GroupUserID: f.owner}))
	})
	t.Run("only the caller's groups are reported", func(t *testing.T) {
		ids, err := f.groups.GroupIDsByConversation(ctx, f.owner, []uuid.UUID{a, b})
		require.NoError(t, err)
		assert.Equal(t, map[uuid.UUID][]uuid.UUID{a: {vip.ID}}, ids)
	})
	t.Run("list counts live conversations only", func(t *testing.T) {
		require.NoError(t, f.db.Exec(`UPDATE conversations SET is_deleted = true WHERE id = ?`, a).Error)
		defer f.db.Exec(`UPDATE conversations SET is_deleted = false WHERE id = ?`, a)
		groups, err := f.groups.ListByUser(ctx, f.owner)
		require.NoError(t, err)
		require.Len(t, groups, 1)
		assert.Equal(t, "VIP", groups[0].Name)
		assert.Equal(t, int64(0), groups[0].ConversationCount)
	})
	t.Run("owner scoping on find and delete", func(t *testing.T) {
		got, err := f.groups.FindOwned(ctx, theirs.ID, f.owner)
		require.NoError(t, err)
		assert.Nil(t, got)
		deleted, err := f.groups.Delete(ctx, theirs.ID, f.owner)
		require.NoError(t, err)
		assert.False(t, deleted)
	})
	t.Run("delete removes memberships, not conversations", func(t *testing.T) {
		deleted, err := f.groups.Delete(ctx, vip.ID, f.owner)
		require.NoError(t, err)
		assert.True(t, deleted)
		var members int64
		f.db.Table("conversation_group_members").Where("group_id = ?", vip.ID).Count(&members)
		assert.Zero(t, members)
		assert.ElementsMatch(t, []uuid.UUID{a, b}, f.search(t, &ConversationSearchFilter{}))
	})
}
