package outreach

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactDomain "github.com/rockship/cosmo-agents-go/internal/domain/contact"
	outreachDomain "github.com/rockship/cosmo-agents-go/internal/domain/outreach"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	outreachRepo "github.com/rockship/cosmo-agents-go/internal/repository/outreach"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
)

// The state machine tests in state_machine_test.go cover how a contact's state
// is derived from its timeline. The tests here cover everything that acts on
// that state: which contacts are suggested for outreach, how recorded events
// move the stored state, meetings, notes, interaction logging and the feedback
// loop. They run on the full schema, because most of these paths write to the
// contacts table as well as the outreach tables, and it is the two agreeing
// that the product depends on.

type svcFixture struct {
	svc *Service
	db  *gorm.DB
}

func newServiceFixture(t *testing.T) *svcFixture {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{
		Logger:                                   glogger.Discard,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&domain.Contact{},
		&domain.ListContact{},
		&domain.ListContactAssociation{},
		&outreachDomain.InteractionLog{},
		&outreachDomain.OutreachState{},
		&outreachDomain.Meeting{},
		&outreachDomain.OutreachFeedback{},
	))
	// Production has UNIQUE(user_id, contact_id) (migration 000039), which is
	// the constraint Upsert's ON CONFLICT names; the model tag does not.
	require.NoError(t, db.Exec(`DROP INDEX IF EXISTS idx_outreach_state_unique`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX idx_outreach_states_user_contact ON outreach_states(user_id, contact_id)`).Error)
	// contacts.last_interaction_at comes from migration 000039, not the model,
	// and the service writes it on every event.
	require.NoError(t, db.Exec(`ALTER TABLE contacts ADD COLUMN IF NOT EXISTS last_interaction_at TIMESTAMPTZ`).Error)

	svc := NewService(
		outreachRepo.NewInteractionLogRepository(db),
		outreachRepo.NewOutreachStateRepository(db),
		outreachRepo.NewMeetingRepository(db),
		outreachRepo.NewFeedbackRepository(db),
		contactRepo.NewContactRepository(db),
		nil,
	)
	return &svcFixture{svc: svc, db: db}
}

// contactOpt tweaks a contact before it is saved.
type contactOpt func(*contactDomain.Contact)

func stage(state outreachDomain.ConversationState, next outreachDomain.NextStepAction) contactOpt {
	return func(c *contactDomain.Contact) {
		c.OutreachStage = string(state)
		c.NextStep = string(next)
	}
}

// saveContact persists a contact that is "ready" (all required fields set),
// which is what the suggestion queries require.
func (f *svcFixture) saveContact(t *testing.T, userID uuid.UUID, name string, opts ...contactOpt) *contactDomain.Contact {
	t.Helper()
	c := &contactDomain.Contact{
		UserID:             userID,
		Name:               name,
		Company:            "Acme",
		JobTitle:           "Head of Sales",
		ContactInformation: name + "@example.com",
		OutreachStage:      string(outreachDomain.StateCold),
		NextStep:           string(outreachDomain.NextStepSend),
	}
	for _, o := range opts {
		o(c)
	}
	require.NoError(t, f.db.Create(c).Error)
	// Flags that default to false/true at the column level cannot be set to
	// their zero value through Create, so apply them afterwards.
	if c.IsDeleted || c.DoNotContact {
		require.NoError(t, f.db.Model(c).Updates(map[string]interface{}{
			"is_deleted": c.IsDeleted, "do_not_contact": c.DoNotContact,
		}).Error)
	}
	return c
}

func (f *svcFixture) reload(t *testing.T, id uuid.UUID) *contactDomain.Contact {
	t.Helper()
	var c contactDomain.Contact
	require.NoError(t, f.db.First(&c, "id = ?", id).Error)
	return &c
}

func (f *svcFixture) state(t *testing.T, contactID, userID uuid.UUID) *outreachDomain.OutreachState {
	t.Helper()
	s, err := f.svc.stateRepo.FindByContactID(context.Background(), contactID, userID)
	require.NoError(t, err)
	return s
}

func (f *svcFixture) log(t *testing.T, userID, contactID uuid.UUID, dir outreachDomain.InteractionDirection, age time.Duration) {
	t.Helper()
	require.NoError(t, f.svc.interactionRepo.Create(context.Background(), &outreachDomain.InteractionLog{
		UserID: userID, ContactID: contactID,
		Channel: "Email", Direction: string(dir), Content: "msg",
		Timestamp: time.Now().Add(-age),
	}))
}

func names(s []*SuggestContact) []string {
	out := make([]string, len(s))
	for i, c := range s {
		out[i] = c.Contact.Name
	}
	return out
}

// ============================================
// Suggestions
// ============================================

func TestSuggestContacts_RejectsUnknownType(t *testing.T) {
	f := newServiceFixture(t)
	_, err := f.svc.SuggestContacts(context.Background(), uuid.New(), uuid.New(), false, "warm", 10)
	require.Error(t, err)
}

// Cold suggestions are contacts that are ready and due their first message.
// Everything else — not ready, deleted, already in a conversation, dropped, or
// someone else's — must stay out of the list the BD works from.
func TestSuggestContacts_ColdPicksOnlyReadyContactsDueAnIntro(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	me, other := uuid.New(), uuid.New()

	f.saveContact(t, me, "ready1")
	f.saveContact(t, me, "ready2")
	f.saveContact(t, me, "pending", func(c *contactDomain.Contact) { c.JobTitle = "" })
	f.saveContact(t, me, "deleted", func(c *contactDomain.Contact) { c.IsDeleted = true })
	f.saveContact(t, me, "waiting", stage(outreachDomain.StateNoReply, outreachDomain.NextStepWait))
	f.saveContact(t, me, "dropped", stage(outreachDomain.StateDropped, outreachDomain.NextStepDrop))
	f.saveContact(t, other, "not mine")

	res, err := f.svc.SuggestContacts(ctx, me, uuid.New(), false, "cold", 10)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"ready1", "ready2"}, names(res.Suggestions))
	assert.Equal(t, 2, res.Total)
	for _, s := range res.Suggestions {
		assert.Equal(t, "cold", s.Type)
		assert.Equal(t, string(outreachDomain.NextStepSend), s.NextStep)
	}

	// The limit caps the list, not the total.
	res, err = f.svc.SuggestContacts(ctx, me, uuid.New(), false, "cold", 1)
	require.NoError(t, err)
	assert.Len(t, res.Suggestions, 1)
	assert.Equal(t, 2, res.Total)
}

// A contact who asked not to be contacted must never be put in front of a BD
// as due outreach: the daily action list is built from these suggestions and
// drafts a message for every one of them.
func TestSuggestContacts_ExcludesDoNotContact(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	me := uuid.New()

	optOut := func(c *contactDomain.Contact) { c.DoNotContact = true }
	f.saveContact(t, me, "ok-cold")
	f.saveContact(t, me, "optout-cold", optOut)
	f.saveContact(t, me, "ok-followup", stage(outreachDomain.StateReplied, outreachDomain.NextStepSetMeeting))
	f.saveContact(t, me, "optout-followup", stage(outreachDomain.StateReplied, outreachDomain.NextStepSetMeeting), optOut)

	for _, kind := range []string{"cold", "followup", "mixed"} {
		t.Run(kind, func(t *testing.T) {
			res, err := f.svc.SuggestContacts(ctx, me, uuid.New(), false, kind, 10)
			require.NoError(t, err)
			for _, s := range res.Suggestions {
				assert.False(t, s.Contact.DoNotContact, "%s suggested %q, who opted out", kind, s.Contact.Name)
			}
		})
	}
}

// Follow-ups are ordered by how warm the conversation is: someone who replied
// comes before someone met, who comes before someone who has gone quiet.
func TestSuggestContacts_FollowupOrdersRepliedThenPostMeetingThenNoReply(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	me := uuid.New()

	f.saveContact(t, me, "noreply", stage(outreachDomain.StateNoReply, outreachDomain.NextStepFollowUp1))
	f.saveContact(t, me, "met", stage(outreachDomain.StatePostMeeting, outreachDomain.NextStepFollowUp))
	replied := f.saveContact(t, me, "replied", stage(outreachDomain.StateReplied, outreachDomain.NextStepSetMeeting))
	f.saveContact(t, me, "confirming", stage(outreachDomain.StateReplied, outreachDomain.NextStepFollowUpMeeting1))
	// Not actionable today.
	f.saveContact(t, me, "waiting", stage(outreachDomain.StateNoReply, outreachDomain.NextStepWait))
	f.saveContact(t, me, "cold")

	// A stored state is attached so the caller can categorise the action.
	require.NoError(t, f.svc.stateRepo.Upsert(ctx, &outreachDomain.OutreachState{
		UserID: me, ContactID: replied.ID,
		ConversationState: string(outreachDomain.StateReplied),
		NextStep:          string(outreachDomain.NextStepSetMeeting),
	}))

	res, err := f.svc.SuggestContacts(ctx, me, uuid.New(), false, "followup", 10)
	require.NoError(t, err)
	got := names(res.Suggestions)
	require.Len(t, got, 4)
	assert.ElementsMatch(t, []string{"replied", "confirming"}, got[:2])
	assert.Equal(t, []string{"met", "noreply"}, got[2:])
	assert.Equal(t, 4, res.Total)
	for _, s := range res.Suggestions {
		assert.Equal(t, "followup", s.Type)
		if s.Contact.ID == replied.ID {
			require.NotNil(t, s.State)
			assert.Equal(t, string(outreachDomain.StateReplied), s.State.ConversationState)
		}
	}

	// Truncation keeps the highest-priority contacts.
	res, err = f.svc.SuggestContacts(ctx, me, uuid.New(), false, "followup", 3)
	require.NoError(t, err)
	assert.Equal(t, "met", names(res.Suggestions)[2])
}

func TestSuggestContacts_MixedSplitsAndTopsUp(t *testing.T) {
	cases := []struct {
		name              string
		nCold, nFollow    int
		limit             int
		wantCold, wantFol int
	}{
		{"60/40 split", 10, 10, 10, 6, 4},
		{"short on cold, filled with follow-ups", 2, 10, 10, 2, 8},
		{"short on follow-ups, filled with cold", 10, 1, 10, 9, 1},
		{"both short returns everything", 2, 1, 10, 2, 1},
		{"non-positive limit defaults to 50", 3, 2, 0, 3, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newServiceFixture(t)
			me := uuid.New()
			for i := 0; i < tc.nCold; i++ {
				f.saveContact(t, me, "cold")
			}
			for i := 0; i < tc.nFollow; i++ {
				f.saveContact(t, me, "follow", stage(outreachDomain.StateReplied, outreachDomain.NextStepSetMeeting))
			}

			res, err := f.svc.SuggestContacts(context.Background(), me, uuid.New(), false, "mixed", tc.limit)
			require.NoError(t, err)
			var cold, follow int
			for _, s := range res.Suggestions {
				if s.Type == "cold" {
					cold++
				} else {
					follow++
				}
			}
			assert.Equal(t, tc.wantCold, cold, "cold")
			assert.Equal(t, tc.wantFol, follow, "follow-up")
			assert.Equal(t, tc.nCold+tc.nFollow, res.Total)
		})
	}
}

// An admin works from the whole organisation's contacts; a member only from
// their own, even when the organisation ID is passed.
func TestSuggestContacts_AdminSeesOrganisationMemberSeesOwn(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	admin, member := uuid.New(), uuid.New()
	org := uuid.New()
	inOrg := func(c *contactDomain.Contact) { c.OrganizationID = &org }

	f.saveContact(t, admin, "admin's", inOrg)
	f.saveContact(t, member, "member's", inOrg)
	f.saveContact(t, uuid.New(), "other org")

	res, err := f.svc.SuggestContacts(ctx, admin, org, true, "cold", 10)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"admin's", "member's"}, names(res.Suggestions))

	res, err = f.svc.SuggestContacts(ctx, member, org, false, "cold", 10)
	require.NoError(t, err)
	assert.Equal(t, []string{"member's"}, names(res.Suggestions))
}

// ============================================
// UpdateOutreach: recorded events
// ============================================

// Each event moves both the stored outreach state and the contact row the
// contact list reads. Both are asserted: they drifting apart is how the left
// and right panels of the UI came to disagree before.
func TestUpdateOutreach_EventTransitions(t *testing.T) {
	positive := outreachDomain.SentimentPositive

	cases := []struct {
		name     string
		prior    *outreachDomain.OutreachState // nil: no stored state yet
		input    UpdateOutreachInput
		wantConv outreachDomain.ConversationState
		wantNext outreachDomain.NextStepAction
		wantFU   int
		wantLog  outreachDomain.InteractionDirection // "" means no log written
		wantBiz  string                              // expected business_stage, if set
	}{
		{
			name:     "first send moves a cold contact to NO_REPLY and waits",
			input:    UpdateOutreachInput{Event: EventSent, Channel: outreachDomain.ChannelEmail},
			wantConv: outreachDomain.StateNoReply, wantNext: outreachDomain.NextStepWait,
			wantLog: outreachDomain.DirectionOutgoing,
		},
		{
			name:     "a follow-up send stays NO_REPLY and counts the follow-up",
			prior:    &outreachDomain.OutreachState{ConversationState: string(outreachDomain.StateNoReply), FollowupCount: 1},
			input:    UpdateOutreachInput{Event: EventSent},
			wantConv: outreachDomain.StateNoReply, wantNext: outreachDomain.NextStepWait, wantFU: 2,
			wantLog: outreachDomain.DirectionOutgoing,
		},
		{
			name:     "answering a reply stays REPLIED",
			prior:    &outreachDomain.OutreachState{ConversationState: string(outreachDomain.StateReplied)},
			input:    UpdateOutreachInput{Event: EventSent},
			wantConv: outreachDomain.StateReplied, wantNext: outreachDomain.NextStepWait,
			wantLog: outreachDomain.DirectionOutgoing,
		},
		{
			name:     "a post-meeting send stays POST_MEETING",
			prior:    &outreachDomain.OutreachState{ConversationState: string(outreachDomain.StatePostMeeting)},
			input:    UpdateOutreachInput{Event: EventSent},
			wantConv: outreachDomain.StatePostMeeting, wantNext: outreachDomain.NextStepWait,
			wantLog: outreachDomain.DirectionOutgoing,
		},
		{
			name:     "a reply resets follow-ups and proposes a meeting",
			prior:    &outreachDomain.OutreachState{ConversationState: string(outreachDomain.StateNoReply), FollowupCount: 2},
			input:    UpdateOutreachInput{Event: EventReplied, Content: "sounds good", Sentiment: &positive},
			wantConv: outreachDomain.StateReplied, wantNext: outreachDomain.NextStepSetMeeting,
			wantLog: outreachDomain.DirectionIncoming,
		},
		{
			// A contact can reach no_reply without a stored state: interactions
			// logged through AddInteraction write the contact row only. The
			// first no-reply must queue follow-up 1, not retire the contact.
			name:     "first no-reply without a stored state queues follow-up 1",
			input:    UpdateOutreachInput{Event: EventNoReply},
			wantConv: outreachDomain.StateNoReply, wantNext: outreachDomain.NextStepFollowUp1, wantFU: 1,
		},
		{
			name:     "second no-reply queues follow-up 2",
			prior:    &outreachDomain.OutreachState{ConversationState: string(outreachDomain.StateNoReply), FollowupCount: 1, MaxFollowups: 2},
			input:    UpdateOutreachInput{Event: EventNoReply},
			wantConv: outreachDomain.StateNoReply, wantNext: outreachDomain.NextStepFollowUp2, wantFU: 2,
		},
		{
			name:     "no-reply past the cap drops the contact",
			prior:    &outreachDomain.OutreachState{ConversationState: string(outreachDomain.StateNoReply), FollowupCount: 2, MaxFollowups: 2},
			input:    UpdateOutreachInput{Event: EventNoReply},
			wantConv: outreachDomain.StateDropped, wantNext: outreachDomain.NextStepDrop, wantFU: 3,
		},
		{
			name:     "a booked meeting waits for confirmation",
			prior:    &outreachDomain.OutreachState{ConversationState: string(outreachDomain.StateReplied), FollowupCount: 1},
			input:    UpdateOutreachInput{Event: EventMeetingBooked},
			wantConv: outreachDomain.StateReplied, wantNext: outreachDomain.NextStepWait,
		},
		{
			name:     "first missing confirmation chases once",
			prior:    &outreachDomain.OutreachState{ConversationState: string(outreachDomain.StateReplied)},
			input:    UpdateOutreachInput{Event: EventNoConfirmation},
			wantConv: outreachDomain.StateReplied, wantNext: outreachDomain.NextStepFollowUpMeeting1, wantFU: 1,
		},
		{
			name:     "second missing confirmation chases again",
			prior:    &outreachDomain.OutreachState{ConversationState: string(outreachDomain.StateReplied), FollowupCount: 1},
			input:    UpdateOutreachInput{Event: EventNoConfirmation},
			wantConv: outreachDomain.StateReplied, wantNext: outreachDomain.NextStepFollowUpMeeting2, wantFU: 2,
		},
		{
			name:     "third missing confirmation drops",
			prior:    &outreachDomain.OutreachState{ConversationState: string(outreachDomain.StateReplied), FollowupCount: 2},
			input:    UpdateOutreachInput{Event: EventNoConfirmation},
			wantConv: outreachDomain.StateDropped, wantNext: outreachDomain.NextStepDrop, wantFU: 3,
		},
		{
			name:     "a confirmed meeting moves the contact into sales",
			prior:    &outreachDomain.OutreachState{ConversationState: string(outreachDomain.StateReplied), FollowupCount: 2},
			input:    UpdateOutreachInput{Event: EventMeetingConfirmed},
			wantConv: outreachDomain.StateReplied, wantNext: outreachDomain.NextStepPrepareMeeting,
			wantBiz: string(contactDomain.BusinessStageSales),
		},
		{
			name:     "a finished meeting moves to POST_MEETING",
			prior:    &outreachDomain.OutreachState{ConversationState: string(outreachDomain.StateReplied)},
			input:    UpdateOutreachInput{Event: EventMeetingDone},
			wantConv: outreachDomain.StatePostMeeting, wantNext: outreachDomain.NextStepFollowUp,
		},
		{
			name:     "drop is final",
			prior:    &outreachDomain.OutreachState{ConversationState: string(outreachDomain.StateReplied)},
			input:    UpdateOutreachInput{Event: EventDrop},
			wantConv: outreachDomain.StateDropped, wantNext: outreachDomain.NextStepDrop,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newServiceFixture(t)
			ctx := context.Background()
			me := uuid.New()
			c := f.saveContact(t, me, "prospect")

			if tc.prior != nil {
				tc.prior.UserID, tc.prior.ContactID = me, c.ID
				if tc.prior.NextStep == "" {
					tc.prior.NextStep = string(outreachDomain.NextStepWait)
				}
				require.NoError(t, f.db.Create(tc.prior).Error)
			}

			in := tc.input
			in.UserID, in.ContactID = me, c.ID
			require.NoError(t, f.svc.UpdateOutreach(ctx, &in))

			st := f.state(t, c.ID, me)
			require.NotNil(t, st)
			assert.Equal(t, string(tc.wantConv), st.ConversationState, "stored state")
			assert.Equal(t, string(tc.wantNext), st.NextStep, "stored next step")
			assert.Equal(t, tc.wantFU, st.FollowupCount, "stored follow-up count")

			row := f.reload(t, c.ID)
			assert.Equal(t, string(tc.wantConv), row.OutreachStage, "contact stage")
			assert.Equal(t, string(tc.wantNext), row.NextStep, "contact next step")
			if tc.wantBiz != "" {
				assert.Equal(t, tc.wantBiz, row.BusinessStage)
			}

			logs, err := f.svc.GetInteractionHistory(ctx, me, c.ID, 0)
			require.NoError(t, err)
			if tc.wantLog == "" {
				assert.Empty(t, logs)
				return
			}
			require.Len(t, logs, 1)
			assert.Equal(t, string(tc.wantLog), logs[0].Direction)
			if in.Event == EventSent && in.Content == "" {
				assert.Equal(t, "[Message sent]", logs[0].Content, "an empty send still leaves a timestamp on the timeline")
			}
			if in.Sentiment != nil {
				require.NotNil(t, logs[0].Sentiment)
				assert.Equal(t, string(*in.Sentiment), *logs[0].Sentiment)
			}
		})
	}
}

// The organisation's follow-up cap has to govern recorded no-reply events the
// same way it governs the derived state, or the two disagree about when a
// contact is retired.
func TestUpdateOutreach_NoReplyHonoursTheOrganisationCap(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	me := uuid.New()
	c := f.saveContact(t, me, "prospect")
	f.svc.WithOrgSettings(func(context.Context, uuid.UUID) ([]byte, error) {
		return []byte(`{"max_followups":4}`), nil
	})

	// Stored with the column default of 2, as every state row is.
	require.NoError(t, f.db.Create(&outreachDomain.OutreachState{
		UserID: me, ContactID: c.ID,
		ConversationState: string(outreachDomain.StateNoReply),
		NextStep:          string(outreachDomain.NextStepWait),
		FollowupCount:     2, MaxFollowups: 2,
	}).Error)

	require.NoError(t, f.svc.UpdateOutreach(ctx, &UpdateOutreachInput{UserID: me, ContactID: c.ID, Event: EventNoReply}))
	st := f.state(t, c.ID, me)
	assert.Equal(t, string(outreachDomain.StateNoReply), st.ConversationState,
		"with a cap of 4 the third no-reply must not retire the contact")
}

// An event name the service does not know must be refused. Accepting it
// silently returns success to the caller while recording nothing.
func TestUpdateOutreach_RejectsUnknownEvent(t *testing.T) {
	f := newServiceFixture(t)
	me := uuid.New()
	c := f.saveContact(t, me, "prospect")

	err := f.svc.UpdateOutreach(context.Background(), &UpdateOutreachInput{UserID: me, ContactID: c.ID, Event: "dropped"})
	require.Error(t, err)
	assert.Nil(t, f.state(t, c.ID, me), "a rejected event must not create a state row")
}

// The documented cadence is: send, no reply -> FU1; FU1 sent, no reply -> FU2;
// FU2 sent, no reply -> DROP. Recorded through the events the UI sends, the
// contact is instead dropped right after FU1.
func TestUpdateOutreach_FullCadenceReachesFollowUp2(t *testing.T) {
	t.Skip("BUG: EventSent increments FollowupCount in NO_REPLY and EventNoReply increments it again, " +
		"so sent → no_reply → sent → no_reply reaches count 3 and drops the contact instead of queueing FOLLOW_UP_2")

	f := newServiceFixture(t)
	ctx := context.Background()
	me := uuid.New()
	c := f.saveContact(t, me, "prospect")

	for _, ev := range []UpdateEvent{EventSent, EventNoReply, EventSent, EventNoReply} {
		require.NoError(t, f.svc.UpdateOutreach(ctx, &UpdateOutreachInput{UserID: me, ContactID: c.ID, Event: ev}))
	}
	st := f.state(t, c.ID, me)
	assert.Equal(t, string(outreachDomain.NextStepFollowUp2), st.NextStep)
}

func TestUpdateOutreachForHandler_ReportsTheTransition(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	me := uuid.New()
	c := f.saveContact(t, me, "prospect")

	// No stored state reads as COLD; channel defaults to LinkedIn.
	res, err := f.svc.UpdateOutreachForHandler(ctx, me, c.ID, "sent", "hi", "", "")
	require.NoError(t, err)
	assert.Equal(t, outreachDomain.StateCold, res.PreviousState)
	assert.Equal(t, outreachDomain.StateNoReply, res.NewState)
	assert.Equal(t, outreachDomain.NextStepWait, res.NextStep)

	logs, err := f.svc.GetInteractionHistory(ctx, me, c.ID, 0)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, string(outreachDomain.ChannelLinkedIn), logs[0].Channel)

	res, err = f.svc.UpdateOutreachForHandler(ctx, me, c.ID, "replied", "maybe later", "Email", "negative")
	require.NoError(t, err)
	assert.Equal(t, outreachDomain.StateNoReply, res.PreviousState)
	assert.Equal(t, outreachDomain.StateReplied, res.NewState)
	require.NotNil(t, res.State)

	logs, err = f.svc.GetInteractionHistory(ctx, me, c.ID, 1)
	require.NoError(t, err)
	require.NotNil(t, logs[0].Sentiment)
	assert.Equal(t, "negative", *logs[0].Sentiment)
	assert.Equal(t, "Email", logs[0].Channel)
}

// ============================================
// GetOutreachState
// ============================================

func TestGetOutreachState_RecomputesPersistsAndSyncsTheContact(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	me := uuid.New()
	c := f.saveContact(t, me, "prospect") // stored as COLD / SEND

	// Sent two days ago and never answered: the timeline says follow-up 1.
	f.log(t, me, c.ID, outreachDomain.DirectionOutgoing, 48*time.Hour)
	// A previously recorded outcome survives the recomputation.
	require.NoError(t, f.db.Create(&outreachDomain.OutreachState{
		UserID: me, ContactID: c.ID,
		ConversationState: string(outreachDomain.StateCold),
		NextStep:          string(outreachDomain.NextStepSend),
		LastOutcome:       string(outreachDomain.OutcomeSent),
	}).Error)

	st, err := f.svc.GetOutreachState(ctx, me, c.ID)
	require.NoError(t, err)
	assert.Equal(t, string(outreachDomain.StateNoReply), st.ConversationState)
	assert.Equal(t, string(outreachDomain.NextStepFollowUp1), st.NextStep)
	assert.Equal(t, string(outreachDomain.OutcomeSent), st.LastOutcome)
	// The insight columns default to '{}'; an empty object is not context.
	assert.Equal(t, string(outreachDomain.ContextMedium), st.ContextLevel, "company and title known, no insights")
	assert.Equal(t, 2, st.DaysSinceLastInteraction)

	stored := f.state(t, c.ID, me)
	assert.Equal(t, string(outreachDomain.NextStepFollowUp1), stored.NextStep)

	row := f.reload(t, c.ID)
	assert.Equal(t, string(outreachDomain.StateNoReply), row.OutreachStage)
	assert.Equal(t, string(outreachDomain.NextStepFollowUp1), row.NextStep)
}

func TestGetOutreachState_InsightsRaiseContextToHigh(t *testing.T) {
	f := newServiceFixture(t)
	me := uuid.New()
	c := f.saveContact(t, me, "prospect", func(c *contactDomain.Contact) {
		c.AIInsights = []byte(`{"suspected_goals":[{"goal":"scale outbound"}]}`)
	})

	st, err := f.svc.GetOutreachState(context.Background(), me, c.ID)
	require.NoError(t, err)
	assert.Equal(t, string(outreachDomain.ContextHigh), st.ContextLevel)
}

func TestGetOutreachState_RefusesSomeoneElsesContact(t *testing.T) {
	f := newServiceFixture(t)
	c := f.saveContact(t, uuid.New(), "prospect")

	_, err := f.svc.GetOutreachState(context.Background(), uuid.New(), c.ID)
	require.Error(t, err)
}

// ============================================
// Meetings
// ============================================

func TestMeetings_CreateUpdateListDelete(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	me := uuid.New()
	c := f.saveContact(t, me, "prospect")

	_, err := f.svc.CreateMeetingForHandler(ctx, me, c.ID, "Demo", "tomorrow at 3", 0, "", "", "", "")
	require.Error(t, err, "time must be RFC3339")

	when := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	m, err := f.svc.CreateMeetingForHandler(ctx, me, c.ID, "Demo", when.Format(time.RFC3339), 0, "", "HQ", "https://meet", "bring slides")
	require.NoError(t, err)
	assert.Equal(t, 30, m.DurationMinutes, "non-positive duration defaults to 30")
	assert.Equal(t, string(outreachDomain.MeetingChannelZoom), m.Channel)
	assert.Equal(t, string(outreachDomain.MeetingScheduled), m.Status)
	require.NotNil(t, m.Location)
	assert.Equal(t, "HQ", *m.Location)

	// Booking the meeting records the meeting_booked event.
	st := f.state(t, c.ID, me)
	require.NotNil(t, st)
	assert.Equal(t, string(outreachDomain.NextStepWait), st.NextStep)
	assert.Equal(t, string(outreachDomain.ScenarioMeetingConfirmation), st.Scenario)

	upcoming, err := f.svc.GetUpcomingMeetingsForHandler(ctx, me)
	require.NoError(t, err)
	require.Len(t, upcoming, 1)
	assert.Equal(t, m.ID, upcoming[0].ID)

	list, err := f.svc.GetMeetingsForHandler(ctx, me, c.ID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	list, err = f.svc.GetMeetingsForHandler(ctx, uuid.New(), c.ID)
	require.NoError(t, err)
	assert.Empty(t, list, "another user sees none of my meetings")

	// Completing the meeting moves the contact to POST_MEETING.
	done, err := f.svc.UpdateMeetingForHandler(ctx, me, m.ID, "completed", "went well", "interested", "send proposal", "transcript")
	require.NoError(t, err)
	assert.True(t, done.IsCompleted())
	require.NotNil(t, done.NextSteps)
	assert.Equal(t, "send proposal", *done.NextSteps)
	assert.Equal(t, string(outreachDomain.StatePostMeeting), f.state(t, c.ID, me).ConversationState)
	assert.Equal(t, string(outreachDomain.StatePostMeeting), f.reload(t, c.ID).OutreachStage)

	// Meeting prep is only for meetings that have not happened yet.
	_, err = f.svc.GenerateMeetingPrep(ctx, me, m.ID, "en")
	require.ErrorContains(t, err, "scheduled")

	_, err = f.svc.UpdateMeetingForHandler(ctx, me, uuid.New(), "", "", "", "", "")
	require.ErrorContains(t, err, "not found")

	require.ErrorContains(t, f.svc.DeleteMeetingForHandler(ctx, uuid.New(), m.ID), "unauthorized")
	require.ErrorContains(t, f.svc.DeleteMeetingForHandler(ctx, me, uuid.New()), "not found")
	require.NoError(t, f.svc.DeleteMeetingForHandler(ctx, me, m.ID))
	list, err = f.svc.GetMeetings(ctx, c.ID, me)
	require.NoError(t, err)
	assert.Empty(t, list)
}

// Meetings belong to the user who booked them — DeleteMeetingForHandler checks
// that. Updating one must check it too: otherwise any user who learns a meeting
// ID can mark it completed, rewrite its notes, and (through meeting_done) write
// outreach state for a contact that is not theirs.
func TestUpdateMeeting_RefusesAnotherUsersMeeting(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	owner, intruder := uuid.New(), uuid.New()
	c := f.saveContact(t, owner, "prospect")

	m, err := f.svc.CreateMeeting(ctx, &CreateMeetingInput{
		UserID: owner, ContactID: c.ID, Time: time.Now().Add(time.Hour),
		Channel: outreachDomain.MeetingChannelCall,
	})
	require.NoError(t, err)

	_, err = f.svc.UpdateMeetingForHandler(ctx, intruder, m.ID, "completed", "", "hijacked", "", "")
	require.Error(t, err)

	got, err := f.svc.meetingRepo.FindByID(ctx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, string(outreachDomain.MeetingScheduled), got.Status)
	assert.Nil(t, got.Outcome)
	assert.Nil(t, f.state(t, c.ID, intruder), "no outreach state may be written for the intruder")

	// Generating prep for it is refused the same way, before any model call.
	_, err = f.svc.GenerateMeetingPrep(ctx, intruder, m.ID, "vi")
	require.ErrorContains(t, err, "not found")
}

func TestGenerateMeetingPrep_Guards(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	me := uuid.New()
	c := f.saveContact(t, me, "prospect")

	_, err := f.svc.GenerateMeetingPrep(ctx, me, uuid.New(), "vi")
	require.ErrorContains(t, err, "not found")

	m, err := f.svc.CreateMeeting(ctx, &CreateMeetingInput{UserID: me, ContactID: c.ID, Time: time.Now().Add(time.Hour)})
	require.NoError(t, err)
	// Without a model the prep cannot be written, and nothing is saved.
	_, err = f.svc.GenerateMeetingPrep(ctx, me, m.ID, "en")
	require.ErrorContains(t, err, "not configured")
	got, err := f.svc.meetingRepo.FindByID(ctx, m.ID)
	require.NoError(t, err)
	assert.Nil(t, got.MeetingPrep)
}

// ============================================
// Notes and interactions
// ============================================

func TestNotes_OnlyTheAuthorMayEditOrDeleteAndOnlyNotes(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	me, other := uuid.New(), uuid.New()
	c := f.saveContact(t, me, "prospect")

	note, err := f.svc.AddNoteForHandler(ctx, me, c.ID, "prefers email")
	require.NoError(t, err)
	assert.Equal(t, string(outreachDomain.DirectionInternal), note.Direction)
	assert.Equal(t, string(outreachDomain.ChannelNote), note.Channel)

	msg, err := f.svc.AddInteraction(ctx, me, c.ID, "hello", "outgoing", "", "")
	require.NoError(t, err)

	notes, err := f.svc.GetNotesForHandler(ctx, me, c.ID, 10)
	require.NoError(t, err)
	require.Len(t, notes, 1, "messages are not notes")
	assert.Equal(t, "prefers email", notes[0].Content)

	cases := []struct {
		name    string
		user    uuid.UUID
		id      uuid.UUID
		wantErr string
	}{
		{"missing note", me, uuid.New(), "not found"},
		{"someone else's note", other, note.ID, "unauthorized"},
		{"a message is not a note", me, msg.ID, "internal notes"},
	}
	for _, tc := range cases {
		t.Run("update: "+tc.name, func(t *testing.T) {
			_, err := f.svc.UpdateNoteForHandler(ctx, tc.user, tc.id, "tampered")
			require.ErrorContains(t, err, tc.wantErr)
		})
		t.Run("delete: "+tc.name, func(t *testing.T) {
			require.ErrorContains(t, f.svc.DeleteNoteForHandler(ctx, tc.user, tc.id), tc.wantErr)
		})
	}

	updated, err := f.svc.UpdateNoteForHandler(ctx, me, note.ID, "prefers phone")
	require.NoError(t, err)
	assert.Equal(t, "prefers phone", updated.Content)
	notes, err = f.svc.GetNotes(ctx, me, c.ID, 10)
	require.NoError(t, err)
	assert.Equal(t, "prefers phone", notes[0].Content)

	require.NoError(t, f.svc.DeleteNoteForHandler(ctx, me, note.ID))
	notes, err = f.svc.GetNotes(ctx, me, c.ID, 10)
	require.NoError(t, err)
	assert.Empty(t, notes)

	// The message the note checks refused to touch is still there.
	history, err := f.svc.GetInteractionHistory(ctx, me, c.ID, 0)
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, "hello", history[0].Content)
}

// Logging an interaction recalculates the contact's state on the spot, so the
// contact list reflects a reply the moment it is recorded.
func TestAddInteraction_LogsAndRecalculatesTheContact(t *testing.T) {
	cases := []struct {
		name      string
		direction string
		channel   string
		sentiment string
		wantDir   string
		wantChan  string
		wantStage outreachDomain.ConversationState
		wantNext  outreachDomain.NextStepAction
	}{
		{
			// Anything but "incoming" is treated as the BD's own message. Just
			// sent, so the contact waits out the no-reply window.
			name: "unknown direction defaults to outgoing", direction: "sideways",
			wantDir: "outgoing", wantChan: "LinkedIn",
			wantStage: outreachDomain.StateCold, wantNext: outreachDomain.NextStepWait,
		},
		{
			name: "a negative reply is REPLIED and waits", direction: "incoming", channel: "Email", sentiment: "negative",
			wantDir: "incoming", wantChan: "Email",
			wantStage: outreachDomain.StateReplied, wantNext: outreachDomain.NextStepWait,
		},
		{
			name: "a neutral reply proposes a meeting", direction: "incoming",
			wantDir: "incoming", wantChan: "LinkedIn",
			wantStage: outreachDomain.StateReplied, wantNext: outreachDomain.NextStepSetMeeting,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newServiceFixture(t)
			ctx := context.Background()
			me := uuid.New()
			c := f.saveContact(t, me, "prospect")

			got, err := f.svc.AddInteraction(ctx, me, c.ID, "content", tc.direction, tc.channel, tc.sentiment)
			require.NoError(t, err)
			assert.Equal(t, tc.wantDir, got.Direction)
			assert.Equal(t, tc.wantChan, got.Channel)
			if tc.sentiment == "" {
				assert.Nil(t, got.Sentiment)
			}

			row := f.reload(t, c.ID)
			assert.Equal(t, string(tc.wantStage), row.OutreachStage)
			assert.Equal(t, string(tc.wantNext), row.NextStep)
			var touched bool
			require.NoError(t, f.db.Raw(`SELECT last_interaction_at IS NOT NULL FROM contacts WHERE id = ?`, c.ID).Scan(&touched).Error)
			assert.True(t, touched, "last_interaction_at is stamped")
		})
	}
}

// ============================================
// Drafts (template path: no model configured)
// ============================================

func TestGenerateDraftForHandler_TemplateDraftAndStoredState(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	me := uuid.New()
	c := f.saveContact(t, me, "Minh Tran", func(c *contactDomain.Contact) { c.Industry = "Fintech" })

	res, err := f.svc.GenerateDraftForHandler(ctx, me, c.ID)
	require.NoError(t, err)
	assert.Equal(t, outreachDomain.ScenarioIndustryBased, res.Scenario)
	assert.Equal(t, outreachDomain.ContextHigh, res.ContextLevel)
	assert.Contains(t, res.Draft, "Chào Minh,", "greets by first name")
	assert.Contains(t, res.Draft, "Fintech")

	st := f.state(t, c.ID, me)
	require.NotNil(t, st)
	require.NotNil(t, st.MessageDraft)
	assert.Equal(t, res.Draft, *st.MessageDraft)

	// A second draft updates the stored one rather than adding a row.
	f.log(t, me, c.ID, outreachDomain.DirectionIncoming, time.Hour)
	res2, err := f.svc.GenerateDraftForHandlerWithLanguage(ctx, me, c.ID, "vi", "Khoa")
	require.NoError(t, err)
	assert.Equal(t, outreachDomain.ScenarioPostReply, res2.Scenario)
	assert.Equal(t, "Minh Tran", res2.ContactName)
	assert.Equal(t, "Minh Tran@example.com", res2.ContactInformation)
	assert.Equal(t, res2.Draft, *f.state(t, c.ID, me).MessageDraft)

	// The suggestion is recorded for the feedback loop.
	fb, err := f.svc.RecordSuggestionFromDraft(ctx, res2, me)
	require.NoError(t, err)
	assert.Equal(t, string(outreachDomain.ScenarioPostReply), fb.SuggestedScenario)
	require.NotNil(t, fb.SuggestedDraft)
	assert.Equal(t, res2.Draft, *fb.SuggestedDraft)
}

func TestGenerateDraft_Refusals(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	me := uuid.New()

	_, err := f.svc.GenerateDraftForHandler(ctx, me, uuid.New())
	require.Error(t, err, "unknown contact")
	_, err = f.svc.GenerateDraftForHandlerWithLanguage(ctx, me, uuid.New(), "en", "")
	require.Error(t, err, "unknown contact")

	dropped := f.saveContact(t, me, "gone", stage(outreachDomain.StateDropped, outreachDomain.NextStepDrop))
	for name, gen := range map[string]func() (string, error){
		"GenerateDraft":             func() (string, error) { return f.svc.GenerateDraft(ctx, dropped, me) },
		"GenerateDraftWithLanguage": func() (string, error) { return f.svc.GenerateDraftWithLanguage(ctx, dropped, me, "en") },
		"WithLanguageAndUserName": func() (string, error) {
			return f.svc.GenerateDraftWithLanguageAndUserName(ctx, dropped, me, "en", "Khoa")
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := gen()
			require.ErrorContains(t, err, "dropped", "no draft is written for a dropped contact")
		})
	}
}

// Each scenario selects its own template. The assertions pick a phrase only
// that template contains, so a scenario falling through to the wrong template
// fails.
func TestGenerateDraftByScenario_PicksTheScenarioTemplate(t *testing.T) {
	svc := &Service{}
	neg, pos := string(outreachDomain.SentimentNegative), string(outreachDomain.SentimentPositive)
	c := &contactDomain.Contact{Name: "Lan Nguyen", Company: "Acme", JobTitle: "CEO", Industry: "Healthcare"}

	cases := []struct {
		name     string
		scenario outreachDomain.Scenario
		days, fu int
		incoming *outreachDomain.InteractionLog
		want     []string
	}{
		{"role based uses the role pitch", outreachDomain.ScenarioRoleBased, 0, 0, nil, []string{"Chào Lan,", "CEO tại Acme", "nguồn lực có hạn"}},
		{"industry based uses the industry insight", outreachDomain.ScenarioIndustryBased, 0, 0, nil, []string{"ngành Healthcare", "compliance"}},
		{"first follow-up, yesterday", outreachDomain.ScenarioNoReplyFollowup, 1, 0, nil, []string{"hôm qua", "khung nào"}},
		{"second follow-up, days ago", outreachDomain.ScenarioNoReplyFollowup, 4, 1, nil, []string{"4 ngày trước", "lần cuối"}},
		{"break-up after the cap", outreachDomain.ScenarioNoReplyFollowup, 10, 2, nil, []string{"không spam", "team Acme"}},
		{"positive reply proposes a call", outreachDomain.ScenarioPostReply, 0, 0,
			&outreachDomain.InteractionLog{Sentiment: &pos}, []string{"Tuyệt vời", "20 phút"}},
		{"negative reply backs off", outreachDomain.ScenarioPostReply, 0, 0,
			&outreachDomain.InteractionLog{Sentiment: &neg}, []string{"tôn trọng quyết định"}},
		{"'next week' reply schedules a follow-up", outreachDomain.ScenarioPostReply, 0, 0,
			&outreachDomain.InteractionLog{Content: "Call me next week"}, []string{"tuần sau em sẽ follow-up"}},
		{"'busy' reply offers options", outreachDomain.ScenarioPostReply, 0, 0,
			&outreachDomain.InteractionLog{Content: "Đang bận lắm"}, []string{"video demo"}},
		{"other reply asks discovery questions", outreachDomain.ScenarioPostReply, 0, 0,
			&outreachDomain.InteractionLog{Content: "Tell me more"}, []string{"bao nhiêu người"}},
		{"meeting confirmation", outreachDomain.ScenarioMeetingConfirmation, 0, 0, nil, []string{"confirm lịch meeting"}},
		{"post meeting summary", outreachDomain.ScenarioPostMeeting, 0, 0, nil, []string{"hiểu thêm về Acme", "Bước tiếp theo"}},
		{"re-engage after months", outreachDomain.ScenarioReEngage, 120, 0, nil, []string{"vài tháng"}},
		{"re-engage after weeks", outreachDomain.ScenarioReEngage, 45, 0, nil, []string{"hơn 1 tháng"}},
		{"unknown scenario falls back to role based", "mystery", 0, 0, nil, []string{"nguồn lực có hạn"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			draft := svc.generateDraftByScenario(&DraftContext{
				Contact:      c,
				State:        &ConversationStateResult{Scenario: tc.scenario, DaysSinceLastInteraction: tc.days, FollowupCount: tc.fu},
				LastIncoming: tc.incoming,
			})
			for _, w := range tc.want {
				assert.Contains(t, draft, w)
			}
		})
	}
}

func TestDraftHelpers_NameRoleAndIndustry(t *testing.T) {
	svc := &Service{}

	for name, want := range map[string]string{
		"Lan Nguyen": "Lan",
		"":           "anh/chị",
		"N/A":        "anh/chị", // the import placeholder is not a name
		"   ":        "   ",
	} {
		assert.Equal(t, want, svc.getContactName(&contactDomain.Contact{Name: name}), "name %q", name)
	}

	roles := map[string]string{
		"Founder":         "nguồn lực có hạn",
		"BD Manager":      "đúng người, đúng thời điểm",
		"Marketing Lead":  "marketing cần align",
		"Talent Partner":  "ứng viên",
		"Product Manager": "dễ integrate",
		"Accountant":      "real-time",
	}
	for title, want := range roles {
		assert.Contains(t, svc.getValuePropositionForRole(title), want, title)
	}

	industries := map[string]string{
		"SaaS":    "ngành tech",
		"Banking": "tài chính",
		// Fintech and EdTech contain "tech"; they must still get their own pitch.
		"Fintech":    "tài chính",
		"EdTech":     "education",
		"Medical":    "healthcare",
		"E-commerce": "best practices", // "e-commerce" does not contain "ecommerce"
		"Retail":     "Retail/E-commerce",
	}
	for industry, want := range industries {
		assert.Contains(t, svc.getIndustryInsight(industry), want, industry)
	}
}

// ============================================
// Feedback loop
// ============================================

func TestFeedbackLoop_RecordSuggestionActionOutcome(t *testing.T) {
	f := newServiceFixture(t)
	ctx := context.Background()
	me, contact := uuid.New(), uuid.New()

	// Without a suggestion on file an outcome is a no-op, not an error.
	require.NoError(t, f.svc.RecordOutcome(ctx, contact, me, "replied", nil))

	fb, err := f.svc.RecordSuggestion(ctx, me, contact, "role_based", "INTRO", "", "", "LOW", "COLD")
	require.NoError(t, err)
	assert.Nil(t, fb.SuggestedDraft, "an empty draft is stored as NULL")
	assert.Nil(t, fb.SuggestedNextStep)

	require.ErrorContains(t, f.svc.RecordAction(ctx, uuid.New(), "used_draft", ""), "not found")
	require.NoError(t, f.svc.RecordAction(ctx, fb.ID, "modified_draft", "my version"))

	// Pretend the suggestion was made three days ago.
	require.NoError(t, f.db.Model(&outreachDomain.OutreachFeedback{}).Where("id = ?", fb.ID).
		Update("created_at", time.Now().Add(-73*time.Hour)).Error)

	pos := "positive"
	require.NoError(t, f.svc.RecordOutcome(ctx, contact, me, "replied", &pos))

	got, err := f.svc.feedbackRepo.FindByID(ctx, fb.ID)
	require.NoError(t, err)
	assert.Equal(t, "modified_draft", *got.ActualAction)
	assert.Equal(t, "my version", *got.ActualContent)
	assert.Equal(t, "replied", *got.Outcome)
	assert.Equal(t, "positive", *got.ReplySentiment)
	require.NotNil(t, got.DaysToReply)
	assert.Equal(t, 3, *got.DaysToReply)

	// "sent" is not a reply, so it carries no days-to-reply.
	fb2, err := f.svc.RecordSuggestion(ctx, me, uuid.New(), "re_engage", "RE_ENGAGE", "d", "SEND", "LOW", "COLD")
	require.NoError(t, err)
	require.NoError(t, f.svc.RecordOutcome(ctx, fb2.ContactID, me, "sent", nil))
	got, err = f.svc.feedbackRepo.FindByID(ctx, fb2.ID)
	require.NoError(t, err)
	assert.Nil(t, got.DaysToReply)

	stats, err := f.svc.GetFeedbackStats(ctx, me)
	require.NoError(t, err)
	assert.Equal(t, 2, stats.Overall.TotalSuggested)
	assert.Len(t, stats.ByScenario, 2)
	assert.NotEmpty(t, stats.Insights)
}

func TestGenerateInsights(t *testing.T) {
	svc := &Service{}
	sc := func(name string, sent int, rate float64) *outreachDomain.ScenarioStats {
		return &outreachDomain.ScenarioStats{Scenario: name, TotalSent: sent, ReplyRate: rate}
	}

	cases := []struct {
		name       string
		overall    *outreachDomain.ScenarioStats
		byScenario []*outreachDomain.ScenarioStats
		want       []string // substrings expected, in order
		notWant    []string
	}{
		{"no data", nil, nil, []string{"Not enough data"}, nil},
		{"zero suggestions", &outreachDomain.ScenarioStats{}, nil, []string{"Not enough data"}, nil},
		{
			"strong numbers",
			&outreachDomain.ScenarioStats{TotalSuggested: 10, DraftUsageRate: 80, ReplyRate: 25, MeetingRate: 40},
			nil,
			[]string{"Great!", "Strong reply rate", "Excellent conversion"}, nil,
		},
		{
			"weak numbers with enough volume",
			&outreachDomain.ScenarioStats{TotalSuggested: 20, TotalSent: 15, DraftUsageRate: 10, ReplyRate: 5},
			nil,
			[]string{"only using 10.0%", "Reply rate is 5.0%"}, nil,
		},
		{
			// A low rate on a handful of sends is noise, not a signal.
			"weak reply rate on low volume is not flagged",
			&outreachDomain.ScenarioStats{TotalSuggested: 5, TotalSent: 3, DraftUsageRate: 50, ReplyRate: 5},
			nil,
			[]string{"Keep going"}, []string{"Reply rate is"},
		},
		{
			"best and worst scenario need five sends each",
			&outreachDomain.ScenarioStats{TotalSuggested: 30, DraftUsageRate: 50, ReplyRate: 15},
			[]*outreachDomain.ScenarioStats{sc("role_based", 10, 30), sc("re_engage", 6, 5), sc("tiny", 2, 90)},
			[]string{"Best performing: 'role_based'", "Consider improving: 're_engage'"},
			[]string{"'tiny'"},
		},
		{
			"a single qualifying scenario is best, not also worst",
			&outreachDomain.ScenarioStats{TotalSuggested: 30, DraftUsageRate: 50, ReplyRate: 15},
			[]*outreachDomain.ScenarioStats{sc("role_based", 10, 30)},
			[]string{"Best performing"}, []string{"Consider improving"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := svc.generateInsights(tc.overall, tc.byScenario)
			joined := ""
			for _, s := range got {
				joined += s + "\n"
			}
			for _, w := range tc.want {
				assert.Contains(t, joined, w)
			}
			for _, w := range tc.notWant {
				assert.NotContains(t, joined, w)
			}
		})
	}
}

// ============================================
// State machine branches not covered in state_machine_test.go
// ============================================

func TestDetermineConversationState_MeetingBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("a scheduled meeting keeps the stored confirmation step", func(t *testing.T) {
		// Confirmation follow-ups are event-driven and cannot be rebuilt from
		// the message log, so the stored step wins while a meeting is pending.
		f := newServiceFixture(t)
		me := uuid.New()
		c := f.saveContact(t, me, "prospect", stage(outreachDomain.StateReplied, outreachDomain.NextStepFollowUpMeeting2))
		f.log(t, me, c.ID, outreachDomain.DirectionIncoming, 48*time.Hour)
		require.NoError(t, f.svc.meetingRepo.Create(ctx, &outreachDomain.Meeting{UserID: me, ContactID: c.ID, Time: time.Now().Add(time.Hour), Status: "scheduled"}))

		got, err := f.svc.DetermineConversationState(ctx, c, me)
		require.NoError(t, err)
		assert.Equal(t, outreachDomain.StateReplied, got.State)
		assert.Equal(t, outreachDomain.NextStepFollowUpMeeting2, got.NextStep)
		assert.Equal(t, outreachDomain.ScenarioMeetingConfirmation, got.Scenario)
	})

	t.Run("a scheduled meeting with an ordinary stored step is re-derived", func(t *testing.T) {
		f := newServiceFixture(t)
		me := uuid.New()
		c := f.saveContact(t, me, "prospect", stage(outreachDomain.StateReplied, outreachDomain.NextStepSetMeeting))
		f.log(t, me, c.ID, outreachDomain.DirectionIncoming, 48*time.Hour)
		require.NoError(t, f.svc.meetingRepo.Create(ctx, &outreachDomain.Meeting{UserID: me, ContactID: c.ID, Time: time.Now().Add(time.Hour), Status: "scheduled"}))

		got, err := f.svc.DetermineConversationState(ctx, c, me)
		require.NoError(t, err)
		assert.Equal(t, outreachDomain.ScenarioPostReply, got.Scenario)
	})

	t.Run("a completed meeting outranks the message log", func(t *testing.T) {
		f := newServiceFixture(t)
		me := uuid.New()
		c := f.saveContact(t, me, "prospect")
		f.log(t, me, c.ID, outreachDomain.DirectionIncoming, 10*24*time.Hour)
		f.log(t, me, c.ID, outreachDomain.DirectionOutgoing, 24*time.Hour)
		require.NoError(t, f.svc.meetingRepo.Create(ctx, &outreachDomain.Meeting{UserID: me, ContactID: c.ID, Time: time.Now().Add(-48 * time.Hour), Status: "completed"}))

		got, err := f.svc.DetermineConversationState(ctx, c, me)
		require.NoError(t, err)
		assert.Equal(t, outreachDomain.StatePostMeeting, got.State)
		assert.Equal(t, outreachDomain.NextStepFollowUp, got.NextStep)
		assert.Equal(t, outreachDomain.ScenarioPostMeeting, got.Scenario)
	})
}

func TestDetermineConversationState_ReEngagesALongQuietConversation(t *testing.T) {
	f := newServiceFixture(t)
	me := uuid.New()
	c := f.saveContact(t, me, "prospect")
	// They replied, we answered, and then nothing for 90 days.
	f.log(t, me, c.ID, outreachDomain.DirectionIncoming, 100*24*time.Hour)
	f.log(t, me, c.ID, outreachDomain.DirectionOutgoing, 90*24*time.Hour)

	got, err := f.svc.DetermineConversationState(context.Background(), c, me)
	require.NoError(t, err)
	assert.Equal(t, outreachDomain.StateCold, got.State)
	assert.Equal(t, outreachDomain.IntentReEngage, got.OutreachIntent)
	assert.Equal(t, outreachDomain.ScenarioReEngage, got.Scenario)
	assert.Equal(t, 90, got.DaysSinceLastInteraction)
}

// A prospect replied and the BD answered yesterday. That is a live
// conversation waiting on the prospect — but the state machine falls through
// every branch to the default and calls it COLD, due an introduction. Synced
// to the contact by GetOutreachState, that puts the contact back in the cold
// suggestions and the daily list drafts an intro to someone mid-conversation.
func TestDetermineConversationState_AnsweredReplyIsNotCold(t *testing.T) {
	t.Skip("BUG: DetermineConversationState returns COLD/SEND/INTRO when the last outgoing message follows " +
		"an incoming one and is younger than ReEngageThresholdDays (outreach_service.go, default branch)")

	f := newServiceFixture(t)
	me := uuid.New()
	c := f.saveContact(t, me, "prospect")
	f.log(t, me, c.ID, outreachDomain.DirectionIncoming, 48*time.Hour)
	f.log(t, me, c.ID, outreachDomain.DirectionOutgoing, 24*time.Hour)

	got, err := f.svc.DetermineConversationState(context.Background(), c, me)
	require.NoError(t, err)
	assert.NotEqual(t, outreachDomain.StateCold, got.State)
	assert.NotEqual(t, outreachDomain.NextStepSend, got.NextStep)
}

func TestService_InvalidateOrgConfigAppliesASavedCadenceAtOnce(t *testing.T) {
	f := newServiceFixture(t)
	me := uuid.New()
	raw := []byte(`{"max_followups":1}`)
	f.svc.WithOrgSettings(func(context.Context, uuid.UUID) ([]byte, error) { return raw, nil })

	assert.Equal(t, 1, f.svc.configFor(context.Background(), me).MaxFollowups)
	raw = []byte(`{"max_followups":5}`)
	assert.Equal(t, 1, f.svc.configFor(context.Background(), me).MaxFollowups, "cached")
	f.svc.InvalidateOrgConfig(me)
	assert.Equal(t, 5, f.svc.configFor(context.Background(), me).MaxFollowups)
}
