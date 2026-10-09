package outreach

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain/outreach"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
)

// These repositories back the outreach state machine, the daily action list
// and the feedback dashboard. Nearly every query is scoped by contact and user,
// and nearly every "last X" answer depends on ORDER BY timestamp — so the
// tests below seed rows for more than one user and out of chronological order,
// which is what a scoping or ordering mistake needs in order to show.

func pgOutreachDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&outreach.InteractionLog{},
		&outreach.OutreachState{},
		&outreach.Meeting{},
		&outreach.OutreachFeedback{},
	))
	// The model tags put the unique index on contact_id alone; migration 000039
	// declares UNIQUE(user_id, contact_id), and Upsert's ON CONFLICT names that
	// pair. Reproduce the production constraint, not the tag.
	require.NoError(t, db.Exec(`DROP INDEX IF EXISTS idx_outreach_state_unique`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX idx_outreach_states_user_contact ON outreach_states(user_id, contact_id)`).Error)
	return db
}

func addLog(t *testing.T, repo *InteractionLogRepository, userID, contactID uuid.UUID, dir outreach.InteractionDirection, at time.Time, content string) *outreach.InteractionLog {
	t.Helper()
	l := &outreach.InteractionLog{
		UserID:    userID,
		ContactID: contactID,
		Channel:   string(outreach.ChannelEmail),
		Direction: string(dir),
		Content:   content,
		Timestamp: at,
	}
	require.NoError(t, repo.Create(context.Background(), l))
	return l
}

func contents(logs []*outreach.InteractionLog) []string {
	out := make([]string, len(logs))
	for i, l := range logs {
		out[i] = l.Content
	}
	return out
}

// ============================================
// InteractionLog
// ============================================

func TestInteractionLogRepository_TimelineQueries(t *testing.T) {
	repo := NewInteractionLogRepository(pgOutreachDB(t))
	ctx := context.Background()
	me, other := uuid.New(), uuid.New()
	contact, otherContact := uuid.New(), uuid.New()
	now := time.Now()

	// Inserted out of order so that only ORDER BY timestamp can produce the
	// expected sequence.
	addLog(t, repo, me, contact, outreach.DirectionOutgoing, now.Add(-72*time.Hour), "intro")
	addLog(t, repo, me, contact, outreach.DirectionOutgoing, now.Add(-1*time.Hour), "follow-up")
	addLog(t, repo, me, contact, outreach.DirectionIncoming, now.Add(-24*time.Hour), "reply")
	addLog(t, repo, me, contact, outreach.DirectionInternal, now.Add(-30*time.Minute), "note")
	// A colleague working the same contact, and my own other contact.
	addLog(t, repo, other, contact, outreach.DirectionOutgoing, now.Add(-10*time.Minute), "colleague")
	addLog(t, repo, other, contact, outreach.DirectionIncoming, now.Add(-5*time.Minute), "reply to colleague")
	addLog(t, repo, me, otherContact, outreach.DirectionOutgoing, now, "elsewhere")

	t.Run("FindByContactID spans users, newest first, honours limit", func(t *testing.T) {
		all, err := repo.FindByContactID(ctx, contact, 0)
		require.NoError(t, err)
		assert.Equal(t, []string{"reply to colleague", "colleague", "note", "follow-up", "reply", "intro"}, contents(all))

		two, err := repo.FindByContactID(ctx, contact, 2)
		require.NoError(t, err)
		assert.Equal(t, []string{"reply to colleague", "colleague"}, contents(two))
	})

	t.Run("FindByContactIDAndUserID hides the colleague's thread", func(t *testing.T) {
		mine, err := repo.FindByContactIDAndUserID(ctx, contact, me, 0)
		require.NoError(t, err)
		assert.Equal(t, []string{"note", "follow-up", "reply", "intro"}, contents(mine))

		limited, err := repo.FindByContactIDAndUserID(ctx, contact, me, 1)
		require.NoError(t, err)
		assert.Equal(t, []string{"note"}, contents(limited))
	})

	cases := []struct {
		name string
		get  func() (*outreach.InteractionLog, error)
		want string // "" means nil
	}{
		{"last outgoing of anyone", func() (*outreach.InteractionLog, error) { return repo.GetLastOutgoing(ctx, contact) }, "colleague"},
		{"last outgoing of mine", func() (*outreach.InteractionLog, error) { return repo.GetLastOutgoingByUserID(ctx, contact, me) }, "follow-up"},
		{"last incoming of anyone", func() (*outreach.InteractionLog, error) { return repo.GetLastIncoming(ctx, contact) }, "reply to colleague"},
		{"last incoming of mine", func() (*outreach.InteractionLog, error) { return repo.GetLastIncomingByUserID(ctx, contact, me) }, "reply"},
		// An internal note counts as an interaction but is neither direction.
		{"last interaction includes notes", func() (*outreach.InteractionLog, error) { return repo.GetLastInteraction(ctx, contact) }, "reply to colleague"},
		{"no incoming at all is nil, not an error", func() (*outreach.InteractionLog, error) { return repo.GetLastIncoming(ctx, otherContact) }, ""},
		{"user with no messages is nil", func() (*outreach.InteractionLog, error) {
			return repo.GetLastOutgoingByUserID(ctx, otherContact, other)
		}, ""},
		{"unknown contact has no last interaction", func() (*outreach.InteractionLog, error) { return repo.GetLastInteraction(ctx, uuid.New()) }, ""},
		{"unknown contact has no incoming for me", func() (*outreach.InteractionLog, error) {
			return repo.GetLastIncomingByUserID(ctx, uuid.New(), me)
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.get()
			require.NoError(t, err)
			if tc.want == "" {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.want, got.Content)
		})
	}

	t.Run("counts outgoing, per contact and per user", func(t *testing.T) {
		n, err := repo.CountOutgoing(ctx, contact)
		require.NoError(t, err)
		assert.Equal(t, 3, n)

		n, err = repo.CountOutgoingByUserID(ctx, contact, me)
		require.NoError(t, err)
		assert.Equal(t, 2, n, "the colleague's message must not count towards my follow-up cap")
	})

	t.Run("HasIncomingAfter is strict and per contact", func(t *testing.T) {
		yes, err := repo.HasIncomingAfter(ctx, contact, now.Add(-48*time.Hour))
		require.NoError(t, err)
		assert.True(t, yes)

		no, err := repo.HasIncomingAfter(ctx, otherContact, now.Add(-48*time.Hour))
		require.NoError(t, err)
		assert.False(t, no)
	})

	t.Run("HasIncomingByUserAfter looks across all of a user's contacts", func(t *testing.T) {
		yes, err := repo.HasIncomingByUserAfter(ctx, me, now.Add(-48*time.Hour))
		require.NoError(t, err)
		assert.True(t, yes)

		// My only reply was 24h ago; the newer one belongs to the colleague.
		no, err := repo.HasIncomingByUserAfter(ctx, me, now.Add(-12*time.Hour))
		require.NoError(t, err)
		assert.False(t, no)
	})

	t.Run("CountByUserDirectionSince", func(t *testing.T) {
		n, err := repo.CountByUserDirectionSince(ctx, me, string(outreach.DirectionOutgoing), now.Add(-2*time.Hour))
		require.NoError(t, err)
		assert.Equal(t, int64(2), n, "follow-up and the other contact's message, not the 3-day-old intro")

		n, err = repo.CountByUserDirectionSince(ctx, me, string(outreach.DirectionIncoming), now.Add(-2*time.Hour))
		require.NoError(t, err)
		assert.Equal(t, int64(0), n)
	})
}

func TestInteractionLogRepository_Notes(t *testing.T) {
	repo := NewInteractionLogRepository(pgOutreachDB(t))
	ctx := context.Background()
	me, other, contact := uuid.New(), uuid.New(), uuid.New()
	now := time.Now()

	older := addLog(t, repo, me, contact, outreach.DirectionInternal, now.Add(-2*time.Hour), "older note")
	addLog(t, repo, me, contact, outreach.DirectionInternal, now.Add(-1*time.Hour), "newer note")
	addLog(t, repo, me, contact, outreach.DirectionOutgoing, now, "a message, not a note")
	addLog(t, repo, other, contact, outreach.DirectionInternal, now, "someone else's note")

	notes, err := repo.FindNotes(ctx, contact, me, 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"newer note", "older note"}, contents(notes))

	one, err := repo.FindNotes(ctx, contact, me, 1)
	require.NoError(t, err)
	assert.Equal(t, []string{"newer note"}, contents(one))

	found, err := repo.FindNoteByID(ctx, older.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "older note", found.Content)

	missing, err := repo.FindNoteByID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, missing)

	require.NoError(t, repo.UpdateNote(ctx, older.ID, "edited"))
	found, err = repo.FindNoteByID(ctx, older.ID)
	require.NoError(t, err)
	assert.Equal(t, "edited", found.Content)

	// Notes are hard-deleted: the row is gone, not flagged.
	require.NoError(t, repo.DeleteNote(ctx, older.ID))
	gone, err := repo.FindNoteByID(ctx, older.ID)
	require.NoError(t, err)
	assert.Nil(t, gone)
	notes, err = repo.FindNotes(ctx, contact, me, 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"newer note"}, contents(notes))
}

// ============================================
// OutreachState
// ============================================

func TestOutreachStateRepository_UpsertIsOneRowPerUserAndContact(t *testing.T) {
	db := pgOutreachDB(t)
	repo := NewOutreachStateRepository(db)
	ctx := context.Background()
	me, colleague, contact := uuid.New(), uuid.New(), uuid.New()

	require.NoError(t, repo.Upsert(ctx, &outreach.OutreachState{
		UserID: me, ContactID: contact,
		ConversationState: string(outreach.StateCold),
		NextStep:          string(outreach.NextStepSend),
	}))

	// A second upsert for the same pair — a fresh struct with a new ID, which is
	// what UpdateOutreach and GetOutreachState build — must update, not insert.
	draft := "hello"
	require.NoError(t, repo.Upsert(ctx, &outreach.OutreachState{
		UserID: me, ContactID: contact,
		ConversationState: string(outreach.StateNoReply),
		NextStep:          string(outreach.NextStepFollowUp1),
		FollowupCount:     1,
		MessageDraft:      &draft,
	}))

	// A colleague owns a separate state for the same contact.
	require.NoError(t, repo.Upsert(ctx, &outreach.OutreachState{
		UserID: colleague, ContactID: contact,
		ConversationState: string(outreach.StateReplied),
		NextStep:          string(outreach.NextStepSetMeeting),
	}))

	var count int64
	require.NoError(t, db.Model(&outreach.OutreachState{}).Count(&count).Error)
	assert.Equal(t, int64(2), count)

	mine, err := repo.FindByContactID(ctx, contact, me)
	require.NoError(t, err)
	require.NotNil(t, mine)
	assert.Equal(t, string(outreach.StateNoReply), mine.ConversationState)
	assert.Equal(t, string(outreach.NextStepFollowUp1), mine.NextStep)
	assert.Equal(t, 1, mine.FollowupCount)
	require.NotNil(t, mine.MessageDraft)
	assert.Equal(t, "hello", *mine.MessageDraft)

	theirs, err := repo.FindByContactID(ctx, contact, colleague)
	require.NoError(t, err)
	require.NotNil(t, theirs)
	assert.Equal(t, string(outreach.StateReplied), theirs.ConversationState)

	none, err := repo.FindByContactID(ctx, uuid.New(), me)
	require.NoError(t, err)
	assert.Nil(t, none)

	// Update saves the full row by primary key.
	mine.NextStep = string(outreach.NextStepWait)
	require.NoError(t, repo.Update(ctx, mine))
	again, err := repo.FindByContactID(ctx, contact, me)
	require.NoError(t, err)
	assert.Equal(t, string(outreach.NextStepWait), again.NextStep)
}

// seedState writes one state row. createdAt controls FindByColdState's order.
func seedState(t *testing.T, db *gorm.DB, userID uuid.UUID, state outreach.ConversationState, next outreach.NextStepAction, days int, createdAt time.Time) uuid.UUID {
	t.Helper()
	contactID := uuid.New()
	require.NoError(t, db.Create(&outreach.OutreachState{
		UserID:                   userID,
		ContactID:                contactID,
		ConversationState:        string(state),
		NextStep:                 string(next),
		DaysSinceLastInteraction: days,
		CreatedAt:                createdAt,
	}).Error)
	return contactID
}

func contactIDs(states []*outreach.OutreachState) []uuid.UUID {
	out := make([]uuid.UUID, len(states))
	for i, s := range states {
		out[i] = s.ContactID
	}
	return out
}

func TestOutreachStateRepository_FindByColdState(t *testing.T) {
	db := pgOutreachDB(t)
	repo := NewOutreachStateRepository(db)
	ctx := context.Background()
	me, other := uuid.New(), uuid.New()
	now := time.Now()

	oldest := seedState(t, db, me, outreach.StateCold, outreach.NextStepSend, 0, now.Add(-3*time.Hour))
	newest := seedState(t, db, me, outreach.StateCold, outreach.NextStepSend, 0, now.Add(-1*time.Hour))
	middle := seedState(t, db, me, outreach.StateCold, outreach.NextStepSend, 0, now.Add(-2*time.Hour))
	seedState(t, db, me, outreach.StateReplied, outreach.NextStepSetMeeting, 0, now)
	seedState(t, db, other, outreach.StateCold, outreach.NextStepSend, 0, now)

	all, err := repo.FindByColdState(ctx, me, 0)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{newest, middle, oldest}, contactIDs(all))

	two, err := repo.FindByColdState(ctx, me, 2)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{newest, middle}, contactIDs(two))
}

func TestOutreachStateRepository_FindByFollowUpStates(t *testing.T) {
	db := pgOutreachDB(t)
	repo := NewOutreachStateRepository(db)
	ctx := context.Background()
	me, other := uuid.New(), uuid.New()
	now := time.Now()

	// NO_REPLY rows are only ever written with FOLLOW_UP_1/FOLLOW_UP_2 (or WAIT,
	// DROP) as next step — see UpdateOutreach and DetermineConversationState.
	noReplyStale := seedState(t, db, me, outreach.StateNoReply, outreach.NextStepFollowUp2, 9, now)
	noReplyFresh := seedState(t, db, me, outreach.StateNoReply, outreach.NextStepFollowUp1, 4, now)
	replied := seedState(t, db, me, outreach.StateReplied, outreach.NextStepSetMeeting, 1, now)
	postMeeting := seedState(t, db, me, outreach.StatePostMeeting, outreach.NextStepFollowUp, 2, now)
	// Excluded: waiting, dropped, cold, and another user's replied contact.
	seedState(t, db, me, outreach.StateNoReply, outreach.NextStepWait, 30, now)
	seedState(t, db, me, outreach.StateDropped, outreach.NextStepDrop, 30, now)
	seedState(t, db, me, outreach.StateCold, outreach.NextStepSend, 30, now)
	seedState(t, db, other, outreach.StateReplied, outreach.NextStepSetMeeting, 30, now)

	got, err := repo.FindByFollowUpStates(ctx, me, 0)
	require.NoError(t, err)
	// REPLIED first, then POST_MEETING, then NO_REPLY stalest first.
	assert.Equal(t, []uuid.UUID{replied, postMeeting, noReplyStale, noReplyFresh}, contactIDs(got))

	top, err := repo.FindByFollowUpStates(ctx, me, 1)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{replied}, contactIDs(top))
}

func TestOutreachStateRepository_FindContactsNeedingOutreach(t *testing.T) {
	now := time.Now()

	// seed builds nCold cold rows and nFollow replied rows for one user.
	seed := func(t *testing.T, nCold, nFollow int) (*OutreachStateRepository, uuid.UUID) {
		db := pgOutreachDB(t)
		me := uuid.New()
		for i := 0; i < nCold; i++ {
			seedState(t, db, me, outreach.StateCold, outreach.NextStepSend, 0, now.Add(-time.Duration(i)*time.Minute))
		}
		for i := 0; i < nFollow; i++ {
			seedState(t, db, me, outreach.StateReplied, outreach.NextStepSetMeeting, i, now)
		}
		return NewOutreachStateRepository(db), me
	}

	count := func(states []*outreach.OutreachState) (cold, follow int) {
		for _, s := range states {
			if s.ConversationState == string(outreach.StateCold) {
				cold++
			} else {
				follow++
			}
		}
		return
	}

	cases := []struct {
		name              string
		nCold, nFollow    int
		kind              string
		limit             int
		wantCold, wantFol int
	}{
		{"cold only", 5, 5, "cold", 3, 3, 0},
		{"followup only", 5, 5, "followup", 3, 0, 3},
		{"mixed splits 60/40", 10, 10, "mixed", 10, 6, 4},
		// Too few cold contacts: the shortfall is filled with follow-ups.
		{"mixed tops up with follow-ups", 2, 10, "mixed", 10, 2, 8},
		// Too few follow-ups: the shortfall is filled with cold contacts.
		{"mixed tops up with cold", 10, 1, "mixed", 10, 9, 1},
		// 60% of 1 rounds down to a cold limit of 0, and a limit of 0 means
		// "no limit" to FindByColdState — asking for one contact used to
		// return every cold contact the user has.
		{"mixed with limit 1 returns one contact", 5, 5, "mixed", 1, 0, 1},
		{"unknown type returns nothing", 5, 5, "bogus", 10, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, me := seed(t, tc.nCold, tc.nFollow)
			got, err := repo.FindContactsNeedingOutreach(context.Background(), me, tc.kind, tc.limit)
			require.NoError(t, err)
			cold, follow := count(got)
			assert.Equal(t, tc.wantCold, cold, "cold")
			assert.Equal(t, tc.wantFol, follow, "follow-up")
		})
	}
}

// ============================================
// Meeting
// ============================================

func addMeeting(t *testing.T, repo *MeetingRepository, userID, contactID uuid.UUID, status outreach.MeetingStatus, at time.Time, title string) *outreach.Meeting {
	t.Helper()
	m := &outreach.Meeting{
		UserID:    userID,
		ContactID: contactID,
		Title:     &title,
		Time:      at,
		Channel:   string(outreach.MeetingChannelZoom),
		Status:    string(status),
	}
	require.NoError(t, repo.Create(context.Background(), m))
	return m
}

func titles(ms []*outreach.Meeting) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = *m.Title
	}
	return out
}

func TestMeetingRepository_ContactQueries(t *testing.T) {
	repo := NewMeetingRepository(pgOutreachDB(t))
	ctx := context.Background()
	me, colleague, contact, fresh := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Now()

	addMeeting(t, repo, me, contact, outreach.MeetingCompleted, now.Add(-10*24*time.Hour), "first call")
	addMeeting(t, repo, me, contact, outreach.MeetingCompleted, now.Add(-3*24*time.Hour), "demo")
	addMeeting(t, repo, colleague, contact, outreach.MeetingScheduled, now.Add(2*24*time.Hour), "colleague's follow-up")
	addMeeting(t, repo, me, contact, outreach.MeetingCancelled, now.Add(-1*24*time.Hour), "cancelled")

	all, err := repo.FindByContactID(ctx, contact)
	require.NoError(t, err)
	assert.Equal(t, []string{"colleague's follow-up", "cancelled", "demo", "first call"}, titles(all))

	mine, err := repo.FindByContactIDAndUserID(ctx, contact, me)
	require.NoError(t, err)
	assert.Equal(t, []string{"cancelled", "demo", "first call"}, titles(mine))

	last, err := repo.GetLastCompletedMeeting(ctx, contact)
	require.NoError(t, err)
	require.NotNil(t, last)
	assert.Equal(t, "demo", *last.Title, "a later cancelled meeting is not a completed one")

	none, err := repo.GetLastCompletedMeeting(ctx, fresh)
	require.NoError(t, err)
	assert.Nil(t, none)

	for _, tc := range []struct {
		name    string
		check   func(context.Context, uuid.UUID) (bool, error)
		contact uuid.UUID
		want    bool
	}{
		{"completed: yes", repo.HasCompletedMeeting, contact, true},
		{"completed: fresh contact", repo.HasCompletedMeeting, fresh, false},
		{"scheduled: yes (any user)", repo.HasScheduledMeeting, contact, true},
		{"scheduled: fresh contact", repo.HasScheduledMeeting, fresh, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.check(ctx, tc.contact)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMeetingRepository_CRUD(t *testing.T) {
	repo := NewMeetingRepository(pgOutreachDB(t))
	ctx := context.Background()
	me, contact := uuid.New(), uuid.New()

	m := addMeeting(t, repo, me, contact, outreach.MeetingScheduled, time.Now().Add(time.Hour), "kickoff")

	got, err := repo.FindByID(ctx, m.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "kickoff", *got.Title)
	assert.Equal(t, 30, got.DurationMinutes, "column default applies when unset")

	missing, err := repo.FindByID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, missing)

	outcome := "signed"
	got.Status = string(outreach.MeetingCompleted)
	got.Outcome = &outcome
	require.NoError(t, repo.Update(ctx, got))
	reloaded, err := repo.FindByID(ctx, m.ID)
	require.NoError(t, err)
	assert.True(t, reloaded.IsCompleted())
	require.NotNil(t, reloaded.Outcome)
	assert.Equal(t, "signed", *reloaded.Outcome)

	require.NoError(t, repo.Delete(ctx, m.ID))
	gone, err := repo.FindByID(ctx, m.ID)
	require.NoError(t, err)
	assert.Nil(t, gone)
}

func TestMeetingRepository_FindUpcoming(t *testing.T) {
	repo := NewMeetingRepository(pgOutreachDB(t))
	ctx := context.Background()
	me, other := uuid.New(), uuid.New()
	now := time.Now()

	addMeeting(t, repo, me, uuid.New(), outreach.MeetingScheduled, now.Add(72*time.Hour), "in three days")
	addMeeting(t, repo, me, uuid.New(), outreach.MeetingScheduled, now.Add(1*time.Hour), "in an hour")
	addMeeting(t, repo, me, uuid.New(), outreach.MeetingScheduled, now.Add(24*time.Hour), "tomorrow")
	// Excluded: past but never closed out, cancelled, and another user's.
	addMeeting(t, repo, me, uuid.New(), outreach.MeetingScheduled, now.Add(-1*time.Hour), "overdue")
	addMeeting(t, repo, me, uuid.New(), outreach.MeetingCancelled, now.Add(2*time.Hour), "called off")
	addMeeting(t, repo, other, uuid.New(), outreach.MeetingScheduled, now.Add(2*time.Hour), "not mine")

	all, total, err := repo.FindUpcoming(ctx, me, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Equal(t, []string{"in an hour", "tomorrow", "in three days"}, titles(all))

	// The total is the full count, not the page size.
	page, total, err := repo.FindUpcoming(ctx, me, &baseRepo.PaginationParams{Offset: 1, Limit: 1})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Equal(t, []string{"tomorrow"}, titles(page))
}

func TestMeetingRepository_CountByUserSince(t *testing.T) {
	repo := NewMeetingRepository(pgOutreachDB(t))
	ctx := context.Background()
	me, other := uuid.New(), uuid.New()
	now := time.Now()

	addMeeting(t, repo, me, uuid.New(), outreach.MeetingCompleted, now.Add(-2*24*time.Hour), "recent")
	addMeeting(t, repo, me, uuid.New(), outreach.MeetingScheduled, now.Add(24*time.Hour), "upcoming")
	addMeeting(t, repo, me, uuid.New(), outreach.MeetingCompleted, now.Add(-10*24*time.Hour), "old")
	addMeeting(t, repo, other, uuid.New(), outreach.MeetingCompleted, now.Add(-1*24*time.Hour), "not mine")

	n, err := repo.CountByUserSince(ctx, me, now.Add(-7*24*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

// ============================================
// Feedback
// ============================================

func strp(s string) *string { return &s }
func intp(i int) *int       { return &i }

func addFeedback(t *testing.T, db *gorm.DB, userID, contactID uuid.UUID, scenario string, action, outcome *string, days *int, createdAt time.Time) *outreach.OutreachFeedback {
	t.Helper()
	f := &outreach.OutreachFeedback{
		UserID:            userID,
		ContactID:         contactID,
		SuggestedScenario: scenario,
		SuggestedIntent:   string(outreach.IntentIntro),
		ActualAction:      action,
		Outcome:           outcome,
		DaysToReply:       days,
		ContextLevel:      string(outreach.ContextMedium),
		ConversationState: string(outreach.StateCold),
		CreatedAt:         createdAt,
	}
	require.NoError(t, NewFeedbackRepository(db).Create(context.Background(), f))
	return f
}

func TestFeedbackRepository_Lookups(t *testing.T) {
	db := pgOutreachDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()
	me, other, contact := uuid.New(), uuid.New(), uuid.New()
	now := time.Now()

	first := addFeedback(t, db, me, contact, "role_based", strp("used_draft"), nil, nil, now.Add(-2*time.Hour))
	latest := addFeedback(t, db, me, contact, "no_reply_followup", nil, nil, nil, now.Add(-1*time.Hour))
	addFeedback(t, db, other, contact, "role_based", strp("skipped"), nil, nil, now)
	// Actioned and already has an outcome: not pending.
	addFeedback(t, db, me, uuid.New(), "role_based", strp("wrote_own"), strp("sent"), nil, now.Add(-3*time.Hour))

	got, err := repo.FindByID(ctx, first.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "role_based", got.SuggestedScenario)

	missing, err := repo.FindByID(ctx, uuid.New())
	require.NoError(t, err)
	assert.Nil(t, missing)

	list, err := repo.FindByContactID(ctx, contact, me)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, latest.ID, list[0].ID, "newest first")
	assert.Equal(t, first.ID, list[1].ID)

	newest, err := repo.FindLatestForContact(ctx, contact, me)
	require.NoError(t, err)
	require.NotNil(t, newest)
	assert.Equal(t, latest.ID, newest.ID)

	nothing, err := repo.FindLatestForContact(ctx, uuid.New(), me)
	require.NoError(t, err)
	assert.Nil(t, nothing)

	// Pending = the BD acted on it but no outcome is recorded yet.
	pending, err := repo.FindPendingOutcome(ctx, me)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, first.ID, pending[0].ID)
}

func TestFeedbackRepository_UpdateOutcome(t *testing.T) {
	db := pgOutreachDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()
	me := uuid.New()

	cases := []struct {
		name      string
		sentiment *string
		days      *int
	}{
		{"outcome only leaves optional fields null", nil, nil},
		{"sentiment and days are written when given", strp("positive"), intp(3)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := addFeedback(t, db, me, uuid.New(), "role_based", strp("used_draft"), nil, nil, time.Now())
			require.NoError(t, repo.UpdateOutcome(ctx, f.ID, "replied", tc.sentiment, tc.days))

			got, err := repo.FindByID(ctx, f.ID)
			require.NoError(t, err)
			require.NotNil(t, got.Outcome)
			assert.Equal(t, "replied", *got.Outcome)
			assert.NotNil(t, got.OutcomeUpdatedAt)
			assert.Equal(t, tc.sentiment, got.ReplySentiment)
			assert.Equal(t, tc.days, got.DaysToReply)
		})
	}

	// Update saves the whole entry.
	f := addFeedback(t, db, me, uuid.New(), "role_based", nil, nil, nil, time.Now())
	f.ActualAction = strp("modified_draft")
	require.NoError(t, repo.Update(ctx, f))
	got, err := repo.FindByID(ctx, f.ID)
	require.NoError(t, err)
	assert.Equal(t, "modified_draft", *got.ActualAction)
}

func TestFeedbackRepository_Stats(t *testing.T) {
	db := pgOutreachDB(t)
	repo := NewFeedbackRepository(db)
	ctx := context.Background()
	me, other := uuid.New(), uuid.New()
	now := time.Now()

	// role_based: 4 suggestions — 2 used, 1 modified, 1 skipped.
	// Outcomes: sent, replied(2d), meeting_booked(4d), none.
	addFeedback(t, db, me, uuid.New(), "role_based", strp("used_draft"), strp("sent"), nil, now)
	addFeedback(t, db, me, uuid.New(), "role_based", strp("used_draft"), strp("replied"), intp(2), now)
	addFeedback(t, db, me, uuid.New(), "role_based", strp("modified_draft"), strp("meeting_booked"), intp(4), now)
	addFeedback(t, db, me, uuid.New(), "role_based", strp("skipped"), nil, nil, now)
	// re_engage: 1 suggestion, wrote own, sent, no reply.
	addFeedback(t, db, me, uuid.New(), "re_engage", strp("wrote_own"), strp("sent"), nil, now)
	// Another user's feedback must not leak into mine.
	addFeedback(t, db, other, uuid.New(), "role_based", strp("used_draft"), strp("replied"), intp(1), now)

	byScenario, err := repo.GetScenarioStats(ctx, me)
	require.NoError(t, err)
	require.Len(t, byScenario, 2)

	rb := byScenario[0]
	assert.Equal(t, "role_based", rb.Scenario, "ordered by volume")
	assert.Equal(t, string(outreach.ContextMedium), rb.ContextLevel)
	assert.Equal(t, 4, rb.TotalSuggested)
	assert.Equal(t, 2, rb.DraftsUsed)
	assert.Equal(t, 1, rb.DraftsModified)
	assert.Equal(t, 1, rb.Skipped)
	assert.Equal(t, 3, rb.TotalSent, "replied and booked also count as sent")
	assert.Equal(t, 2, rb.TotalReplied)
	assert.Equal(t, 1, rb.TotalMeetings)
	assert.InDelta(t, 66.67, rb.ReplyRate, 0.001)
	assert.InDelta(t, 50.0, rb.MeetingRate, 0.001)
	assert.InDelta(t, 75.0, rb.DraftUsageRate, 0.001)
	assert.InDelta(t, 3.0, rb.AvgDaysToReply, 0.001)

	re := byScenario[1]
	assert.Equal(t, "re_engage", re.Scenario)
	assert.Equal(t, 1, re.WroteOwn)
	assert.Equal(t, 0.0, re.ReplyRate)
	// No replies: the rate is 0, not a division-by-zero error or NULL.
	assert.Equal(t, 0.0, re.MeetingRate)

	overall, err := repo.GetOverallStats(ctx, me)
	require.NoError(t, err)
	assert.Equal(t, "overall", overall.Scenario)
	assert.Equal(t, 5, overall.TotalSuggested)
	assert.Equal(t, 4, overall.TotalSent)
	assert.Equal(t, 2, overall.TotalReplied)
	assert.InDelta(t, 50.0, overall.ReplyRate, 0.001)
	assert.InDelta(t, 60.0, overall.DraftUsageRate, 0.001)

	// A user with no feedback gets zeros, not an error.
	empty, err := repo.GetOverallStats(ctx, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, 0, empty.TotalSuggested)
	assert.Equal(t, 0.0, empty.ReplyRate)
}
