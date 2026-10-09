package outreach

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	contactDomain "github.com/rockship/cosmo-agents-go/internal/domain/contact"
	outreachDomain "github.com/rockship/cosmo-agents-go/internal/domain/outreach"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	outreachRepo "github.com/rockship/cosmo-agents-go/internal/repository/outreach"
)

// DetermineConversationState is the outreach state machine: it decides, for one
// contact, which state the conversation is in and what should happen next.
// Everything downstream — the daily action list, which prompt is used, whether
// a contact is retired — reads its answer, so a wrong transition here is not a
// cosmetic bug.
//
// The service takes concrete repositories rather than interfaces, so these
// tests drive it through a real in-memory database, the same way the
// repository tests in this codebase do. That also means the queries are
// exercised, not just the branching.
//
// These cover the transitions specified as TC-20, TC-21 and TC-22.

// The outreach domain models carry Postgres-only defaults
// (gen_random_uuid(), now()) and a GIN index, so GORM's AutoMigrate cannot
// build them on SQLite the way the simpler repository tests do. The two tables
// this function needs are therefore declared directly.
//
// Only those two are needed: DetermineConversationState receives the contact
// as an argument and never loads it, and it touches neither the state nor the
// feedback repository.

func newStateMachineFixture(t *testing.T) (*Service, *outreachRepo.InteractionLogRepository) {
	t.Helper()

	db, err := pgtest.Open(t, &gorm.Config{Logger: glogger.Discard})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.InteractionLog{}, &domain.Meeting{}))

	interactions := outreachRepo.NewInteractionLogRepository(db)
	svc := NewServiceWithAI(
		interactions,
		outreachRepo.NewOutreachStateRepository(db),
		outreachRepo.NewMeetingRepository(db),
		outreachRepo.NewFeedbackRepository(db),
		contactRepo.NewContactRepository(db),
		nil, // no LLM: the state machine is pure decision logic
		nil, // default cadence
	)
	return svc, interactions
}

// logMessage records one message on the timeline. Direction and age are what
// the state machine actually reads, so they are the only knobs the tests need.
func logMessage(
	t *testing.T,
	repo *outreachRepo.InteractionLogRepository,
	userID, contactID uuid.UUID,
	direction string,
	age time.Duration,
	sentiment *string,
) {
	t.Helper()
	entry := &outreachDomain.InteractionLog{
		ID:        uuid.New(),
		UserID:    userID,
		ContactID: contactID,
		Channel:   "Email",
		Direction: direction,
		Content:   "message",
		Sentiment: sentiment,
		Timestamp: time.Now().Add(-age),
	}
	require.NoError(t, repo.Create(context.Background(), entry))
}

// newContact builds the contact the state machine is asked about. It is never
// persisted: the function under test takes it by argument.
func newContact(userID uuid.UUID) *contactDomain.Contact {
	return &contactDomain.Contact{
		Base:               base.Base{ID: uuid.New()},
		UserID:             userID,
		Name:               "Prospect",
		ContactInformation: "prospect@example.com",
	}
}

// TC-20: COLD -> NO_REPLY.
//
// The transition is driven by the no-reply window, not by the send itself. A
// message sent an hour ago is still COLD and must not produce a follow-up;
// once the window has passed the same contact becomes NO_REPLY with follow-up
// one queued. Both halves are asserted, because a state machine that always
// answers NO_REPLY would pass the second alone.
func TestDetermineConversationState_ColdToNoReplyOnlyAfterTheWindow(t *testing.T) {
	cfg := DefaultConfig()
	ctx := context.Background()

	t.Run("inside the window it stays COLD and waits", func(t *testing.T) {
		svc, logs := newStateMachineFixture(t)
		userID := uuid.New()
		c := newContact(userID)

		logMessage(t, logs, userID, c.ID, "outgoing", 1*time.Hour, nil)

		got, err := svc.DetermineConversationState(ctx, c, userID)
		require.NoError(t, err)
		require.Equal(t, outreachDomain.StateCold, got.State)
		require.Equal(t, outreachDomain.NextStepWait, got.NextStep,
			"a contact still inside the no-reply window must not be chased")
	})

	t.Run("past the window it becomes NO_REPLY with follow-up 1", func(t *testing.T) {
		svc, logs := newStateMachineFixture(t)
		userID := uuid.New()
		c := newContact(userID)

		past := time.Duration(cfg.NoReplyHours+1) * time.Hour
		logMessage(t, logs, userID, c.ID, "outgoing", past, nil)

		got, err := svc.DetermineConversationState(ctx, c, userID)
		require.NoError(t, err)
		require.Equal(t, outreachDomain.StateNoReply, got.State)
		require.Equal(t, outreachDomain.NextStepFollowUp1, got.NextStep)
		require.Equal(t, 0, got.FollowupCount,
			"the initial message is not itself a follow-up")
	})
}

// TC-21: NO_REPLY -> REPLIED.
//
// A reply that arrives after our last message flips the state regardless of
// how many follow-ups preceded it, and the next step comes from the reply's
// sentiment: a positive reply earns a meeting invitation, a negative one earns
// silence. Proposing a meeting to someone who just pushed back is the failure
// this pins down.
func TestDetermineConversationState_ReplyFlipsStateAndSentimentPicksTheNextStep(t *testing.T) {
	positive := string(outreachDomain.SentimentPositive)
	negative := string(outreachDomain.SentimentNegative)

	cases := []struct {
		name      string
		sentiment *string
		wantStep  outreachDomain.NextStepAction
	}{
		{"no sentiment recorded falls back to proposing a meeting", nil,
			outreachDomain.NextStepSetMeeting},
		{"a positive reply proposes a meeting", &positive,
			outreachDomain.NextStepSetMeeting},
		{"a negative reply waits instead of pushing", &negative,
			outreachDomain.NextStepWait},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, logs := newStateMachineFixture(t)
			userID := uuid.New()
			c := newContact(userID)

			// Two follow-ups already sent, then the prospect answers.
			logMessage(t, logs, userID, c.ID, "outgoing", 10*24*time.Hour, nil)
			logMessage(t, logs, userID, c.ID, "outgoing", 5*24*time.Hour, nil)
			logMessage(t, logs, userID, c.ID, "incoming", 2*time.Hour, tc.sentiment)

			got, err := svc.DetermineConversationState(context.Background(), c, userID)
			require.NoError(t, err)
			require.Equal(t, outreachDomain.StateReplied, got.State,
				"an inbound message newer than our last one means they replied")
			require.Equal(t, tc.wantStep, got.NextStep)
			require.NotNil(t, got.LastIncoming)
		})
	}
}

// TC-22: max follow-ups -> DROPPED.
//
// The cap counts follow-ups, not messages, so the initial send does not count
// towards it. With the default cap of two, a contact who received the intro
// plus two follow-ups and never answered is retired; one who received the
// intro plus a single follow-up is not, and is due follow-up two instead.
func TestDetermineConversationState_DropsOnlyAfterTheFollowUpCapIsSpent(t *testing.T) {
	cfg := DefaultConfig()
	ctx := context.Background()
	past := time.Duration(cfg.NoReplyHours+1) * time.Hour

	t.Run("one follow-up short of the cap is still chased", func(t *testing.T) {
		svc, logs := newStateMachineFixture(t)
		userID := uuid.New()
		c := newContact(userID)

		logMessage(t, logs, userID, c.ID, "outgoing", 12*24*time.Hour, nil) // intro
		logMessage(t, logs, userID, c.ID, "outgoing", past, nil)            // FU 1

		got, err := svc.DetermineConversationState(ctx, c, userID)
		require.NoError(t, err)
		require.Equal(t, outreachDomain.StateNoReply, got.State)
		require.Equal(t, outreachDomain.NextStepFollowUp2, got.NextStep)
		require.Equal(t, 1, got.FollowupCount)
	})

	t.Run("at the cap with no reply the contact is dropped", func(t *testing.T) {
		svc, logs := newStateMachineFixture(t)
		userID := uuid.New()
		c := newContact(userID)

		logMessage(t, logs, userID, c.ID, "outgoing", 20*24*time.Hour, nil) // intro
		logMessage(t, logs, userID, c.ID, "outgoing", 15*24*time.Hour, nil) // FU 1
		logMessage(t, logs, userID, c.ID, "outgoing", past, nil)            // FU 2

		got, err := svc.DetermineConversationState(ctx, c, userID)
		require.NoError(t, err)
		require.Equal(t, outreachDomain.StateDropped, got.State)
		require.Equal(t, outreachDomain.NextStepDrop, got.NextStep)
		require.Equal(t, cfg.MaxFollowups, got.FollowupCount)
	})

	t.Run("the cap applies even inside the no-reply window", func(t *testing.T) {
		// Once the cap is spent there is nothing left to wait for, so the drop
		// must not be deferred until the no-reply window expires. This is the
		// only case that reaches the cap check ahead of the follow-up branch —
		// without it, disabling that check changes no test result.
		svc, logs := newStateMachineFixture(t)
		userID := uuid.New()
		c := newContact(userID)

		logMessage(t, logs, userID, c.ID, "outgoing", 20*24*time.Hour, nil) // intro
		logMessage(t, logs, userID, c.ID, "outgoing", 15*24*time.Hour, nil) // FU 1
		logMessage(t, logs, userID, c.ID, "outgoing", 1*time.Hour, nil)     // FU 2, fresh

		got, err := svc.DetermineConversationState(ctx, c, userID)
		require.NoError(t, err)
		require.Equal(t, outreachDomain.StateDropped, got.State,
			"a contact past the follow-up cap is retired, not left waiting")
		require.Equal(t, outreachDomain.NextStepDrop, got.NextStep)
	})

	t.Run("a reply rescues a contact that hit the cap", func(t *testing.T) {
		// The cap is only allowed to retire silent contacts. Dropping someone
		// who answered on the last follow-up would lose a live conversation.
		svc, logs := newStateMachineFixture(t)
		userID := uuid.New()
		c := newContact(userID)

		logMessage(t, logs, userID, c.ID, "outgoing", 20*24*time.Hour, nil)
		logMessage(t, logs, userID, c.ID, "outgoing", 15*24*time.Hour, nil)
		logMessage(t, logs, userID, c.ID, "outgoing", 10*24*time.Hour, nil)
		logMessage(t, logs, userID, c.ID, "incoming", 1*time.Hour, nil)

		got, err := svc.DetermineConversationState(ctx, c, userID)
		require.NoError(t, err)
		require.Equal(t, outreachDomain.StateReplied, got.State,
			"the follow-up cap must not retire a contact who answered")
	})
}

// A contact already stored as DROPPED stays dropped without the timeline being
// consulted at all. Manual drops and opt-outs are recorded that way, and
// re-deriving the state from messages would silently resurrect them.
func TestDetermineConversationState_StoredDropIsFinal(t *testing.T) {
	svc, logs := newStateMachineFixture(t)
	userID := uuid.New()
	c := newContact(userID)
	c.OutreachStage = string(outreachDomain.StateDropped)

	// A fresh inbound message would otherwise read as REPLIED.
	logMessage(t, logs, userID, c.ID, "incoming", 1*time.Hour, nil)

	got, err := svc.DetermineConversationState(context.Background(), c, userID)
	require.NoError(t, err)
	require.Equal(t, outreachDomain.StateDropped, got.State)
	require.Equal(t, outreachDomain.NextStepDrop, got.NextStep)
}

// A contact with no history at all is COLD and due an intro. The scenario
// switches on whether the contact's industry is known, which is what selects
// the prompt used to write that intro.
func TestDetermineConversationState_UntouchedContactIsColdAndDueAnIntro(t *testing.T) {
	ctx := context.Background()

	t.Run("industry known", func(t *testing.T) {
		svc, _ := newStateMachineFixture(t)
		userID := uuid.New()
		c := newContact(userID)
		c.Industry = "Logistics"

		got, err := svc.DetermineConversationState(ctx, c, userID)
		require.NoError(t, err)
		require.Equal(t, outreachDomain.StateCold, got.State)
		require.Equal(t, outreachDomain.NextStepSend, got.NextStep)
		require.Equal(t, outreachDomain.ScenarioIndustryBased, got.Scenario)
	})

	t.Run("industry unknown", func(t *testing.T) {
		svc, _ := newStateMachineFixture(t)
		userID := uuid.New()
		c := newContact(userID)

		got, err := svc.DetermineConversationState(ctx, c, userID)
		require.NoError(t, err)
		require.Equal(t, outreachDomain.StateCold, got.State)
		require.Equal(t, outreachDomain.ScenarioRoleBased, got.Scenario)
	})
}

// The cadence an admin configures has to reach the state machine, not just the
// settings table. With a 48-hour window a contact messaged 12 hours ago is
// still COLD, where the 8-hour default would already have moved it on.
func TestDetermineConversationState_HonoursTheOrganisationCadence(t *testing.T) {
	svc, logs := newStateMachineFixture(t)
	userID := uuid.New()
	c := newContact(userID)

	logMessage(t, logs, userID, c.ID, "outgoing", 12*time.Hour, nil)

	// Default cadence: 12 hours is past the 8-hour window.
	got, err := svc.DetermineConversationState(context.Background(), c, userID)
	require.NoError(t, err)
	require.Equal(t, outreachDomain.StateNoReply, got.State)

	svc.WithOrgSettings(func(context.Context, uuid.UUID) ([]byte, error) {
		return []byte(`{"no_reply_hours":48}`), nil
	})

	got, err = svc.DetermineConversationState(context.Background(), c, userID)
	require.NoError(t, err)
	require.Equal(t, outreachDomain.StateCold, got.State,
		"a 48-hour window must keep a 12-hour-old send in COLD")
	require.Equal(t, outreachDomain.NextStepWait, got.NextStep)
}
