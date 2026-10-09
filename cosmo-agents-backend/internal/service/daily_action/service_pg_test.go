package daily_action

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	rootdomain "github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	contact "github.com/rockship/cosmo-agents-go/internal/domain/contact"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	outreach "github.com/rockship/cosmo-agents-go/internal/domain/outreach"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	dailyActionRepo "github.com/rockship/cosmo-agents-go/internal/repository/daily_action"
	outreachRepo "github.com/rockship/cosmo-agents-go/internal/repository/outreach"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	outreachSvc "github.com/rockship/cosmo-agents-go/internal/service/outreach"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"gorm.io/gorm"
)

// newServiceDB builds a Service over a fresh schema with the real daily-action
// and outreach tables. users/organizations/roles are minimal: only the
// columns the user repository reads.
func newServiceDB(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(
		&domain.DailyActionGeneration{}, &domain.DailyAction{}, &domain.ActionSnooze{},
		&domain.ActionCompletionLog{}, &domain.OutcomeMetrics{},
		&outreach.InteractionLog{}, &outreach.Meeting{}, &outreach.OutreachState{},
		&rootdomain.Contact{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, stmt := range []string{
		`CREATE TABLE users (id UUID PRIMARY KEY, name TEXT, email TEXT, is_deleted BOOLEAN DEFAULT FALSE)`,
		// Migration 000039 declares UNIQUE(user_id, contact_id), which the
		// state upsert conflicts on; the GORM tags do not carry it.
		`CREATE UNIQUE INDEX outreach_states_user_contact ON outreach_states (user_id, contact_id)`,
		// Migration 000048: one live generation per user and day.
		`CREATE UNIQUE INDEX gen_user_date_live ON daily_action_generations (user_id, date) WHERE is_deleted = FALSE AND replaced_by_id IS NULL`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	interactions := outreachRepo.NewInteractionLogRepository(db)
	meetings := outreachRepo.NewMeetingRepository(db)
	out := outreachSvc.NewService(interactions, outreachRepo.NewOutreachStateRepository(db), meetings,
		outreachRepo.NewFeedbackRepository(db), contactRepo.NewContactRepository(db), nil)

	svc := NewService(out,
		dailyActionRepo.NewGenerationRepository(db),
		dailyActionRepo.NewActionRepository(db),
		dailyActionRepo.NewSnoozeRepository(db),
		dailyActionRepo.NewCompletionLogRepository(db),
		meetings, interactions,
		userRepo.NewUserRepository(db),
	)
	return svc, db
}

func mustExec(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

// seedGeneration writes a ready generation with one action per status given.
func seedGeneration(t *testing.T, db *gorm.DB, userID uuid.UUID, specs ...domain.DailyAction) (*domain.DailyActionGeneration, []domain.DailyAction) {
	t.Helper()
	now := time.Now()
	gen := &domain.DailyActionGeneration{UserID: userID, Date: now.Format("2006-01-02"), Status: domain.GenerationStatusReady, GeneratedAt: &now}
	if err := db.Create(gen).Error; err != nil {
		t.Fatalf("create gen: %v", err)
	}
	out := make([]domain.DailyAction, 0, len(specs))
	for i, a := range specs {
		a.UserID = userID
		a.GenerationID = gen.ID
		if a.ContactID == uuid.Nil {
			a.ContactID = uuid.New()
		}
		if a.Type == "" {
			a.Type = domain.ActionTypeFollowup
		}
		if a.CategoryID == "" {
			a.CategoryID = domain.CategoryFollowup
		}
		if a.Status == "" {
			a.Status = domain.ActionStatusSuggested
		}
		if len(a.ContactSnapshot) == 0 {
			snap, _ := json.Marshal(domain.ContactSnapshot{Name: "Contact " + string(rune('A'+i))})
			a.ContactSnapshot = base.JSONB(snap)
		}
		if err := db.Create(&a).Error; err != nil {
			t.Fatalf("create action: %v", err)
		}
		out = append(out, a)
	}
	return gen, out
}

func reload(t *testing.T, db *gorm.DB, id uuid.UUID) domain.DailyAction {
	t.Helper()
	var a domain.DailyAction
	if err := db.First(&a, "id = ?", id).Error; err != nil {
		t.Fatalf("reload action: %v", err)
	}
	return a
}

func TestExecuteTransition_StateMachine(t *testing.T) {
	svc, db := newServiceDB(t)
	ctx := context.Background()
	custom := time.Now().Add(72 * time.Hour).Truncate(time.Second)
	reason := "not a fit"

	tests := []struct {
		name       string
		from       domain.ActionStatus
		transition domain.Transition
		wantStatus domain.ActionStatus
		wantErr    bool
	}{
		{"skip", domain.ActionStatusSuggested, domain.TransitionSkip, domain.ActionStatusSkipped, false},
		{"snooze", domain.ActionStatusSuggested, domain.TransitionSnooze, domain.ActionStatusSnoozed, false},
		{"snooze custom", domain.ActionStatusSuggested, domain.TransitionSnoozeCustom, domain.ActionStatusSnoozed, false},
		{"mark completed", domain.ActionStatusSuggested, domain.TransitionMarkCompleted, domain.ActionStatusCompleted, false},
		{"in progress completes", domain.ActionStatusInProgress, domain.TransitionMarkCompleted, domain.ActionStatusCompleted, false},
		{"reopen skipped", domain.ActionStatusSkipped, domain.TransitionReopen, domain.ActionStatusSuggested, false},
		{"reopen completed", domain.ActionStatusCompleted, domain.TransitionReopen, domain.ActionStatusSuggested, false},
		// A finished action cannot be finished twice, and a live one cannot be
		// "reopened": both would write a second completion log and inflate the
		// productivity counts.
		{"double complete", domain.ActionStatusCompleted, domain.TransitionMarkCompleted, domain.ActionStatusCompleted, true},
		{"skip completed", domain.ActionStatusCompleted, domain.TransitionSkip, domain.ActionStatusCompleted, true},
		{"reopen suggested", domain.ActionStatusSuggested, domain.TransitionReopen, domain.ActionStatusSuggested, true},
		{"snooze in progress", domain.ActionStatusInProgress, domain.TransitionSnooze, domain.ActionStatusInProgress, true},
		{"unknown status", "archived", domain.TransitionReopen, "archived", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, actions := seedGeneration(t, db, uuid.New(), domain.DailyAction{Status: tc.from})
			a := actions[0]

			res, err := svc.ExecuteTransition(ctx, &a, tc.transition, nil, nil, &reason, &custom, nil)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got := reload(t, db, a.ID).Status; got != tc.wantStatus {
				t.Fatalf("stored status = %s, want %s", got, tc.wantStatus)
			}
			var logs int64
			db.Model(&domain.ActionCompletionLog{}).Where("action_id = ?", a.ID).Count(&logs)
			if tc.wantErr {
				if logs != 0 {
					t.Fatalf("rejected transition wrote %d logs", logs)
				}
				return
			}
			if logs != 1 || res.CompletionLog == nil || res.CompletionLog.ContactName != "Contact A" {
				t.Fatalf("logs=%d log=%+v", logs, res.CompletionLog)
			}
			if tc.transition == domain.TransitionSkip && (res.CompletionLog.SkipReason == nil || *res.CompletionLog.SkipReason != reason) {
				t.Fatal("skip reason must be kept on the log")
			}
		})
	}
}

func TestExecuteTransition_SnoozeUntil(t *testing.T) {
	svc, db := newServiceDB(t)
	ctx := context.Background()
	user := uuid.New()

	// Default snooze lands at 17:00 today, or tomorrow once that has passed —
	// never in the past, or the action would reappear immediately.
	_, actions := seedGeneration(t, db, user, domain.DailyAction{}, domain.DailyAction{})
	a := actions[0]
	res, err := svc.ExecuteTransition(ctx, &a, domain.TransitionSnooze, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("snooze: %v", err)
	}
	until := *res.Action.SnoozeUntil
	if !until.After(time.Now()) || until.Hour() != 17 || until.Sub(time.Now()) > 24*time.Hour {
		t.Fatalf("default snooze until %v", until)
	}
	var sz domain.ActionSnooze
	if err := db.First(&sz, "action_id = ?", a.ID).Error; err != nil || sz.ClearedAt != nil {
		t.Fatalf("snooze record: %v %+v", err, sz)
	}

	// Reopening a snoozed action clears both the record and the column, so the
	// expiry worker does not flip it again later.
	res, err = svc.ExecuteTransition(ctx, res.Action, domain.TransitionReopen, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	stored := reload(t, db, a.ID)
	if stored.Status != domain.ActionStatusSuggested || stored.SnoozeUntil != nil || res.Action.SnoozeUntil != nil {
		t.Fatalf("after reopen: %+v", stored)
	}
	db.First(&sz, "action_id = ?", a.ID)
	if sz.ClearedAt == nil {
		t.Fatal("snooze record not cleared on reopen")
	}

	// Custom snooze honours the caller's time.
	b := actions[1]
	custom := time.Now().Add(50 * time.Hour).Truncate(time.Second)
	if _, err := svc.ExecuteTransition(ctx, &b, domain.TransitionSnoozeCustom, nil, nil, nil, &custom, nil); err != nil {
		t.Fatalf("snooze custom: %v", err)
	}
	if got := reload(t, db, b.ID).SnoozeUntil; got == nil || !got.Equal(custom) {
		t.Fatalf("custom snooze = %v, want %v", got, custom)
	}
}

func TestExecuteTransition_MarkSentAdvancesOutreach(t *testing.T) {
	svc, db := newServiceDB(t)
	ctx := context.Background()
	user := uuid.New()

	_, actions := seedGeneration(t, db, user, domain.DailyAction{Type: domain.ActionTypeOutreach})
	a := actions[0]
	mustExec(t, db, `INSERT INTO contacts (id, user_id, name, source, source_id, is_deleted) VALUES (?, ?, 'A', 'csv', ?, FALSE)`,
		a.ContactID, user, uuid.NewString())
	content, channel, feedback := "Hi there", "Email", "edited"

	res, err := svc.ExecuteTransition(ctx, &a, domain.TransitionMarkSent, &content, &channel, nil, nil, &feedback)
	if err != nil {
		t.Fatalf("mark_sent: %v", err)
	}
	if reload(t, db, a.ID).Status != domain.ActionStatusCompleted {
		t.Fatal("mark_sent should complete the action")
	}
	// The send is recorded on the contact's timeline, which is what moves the
	// outreach state machine forward.
	var logs []outreach.InteractionLog
	db.Where("contact_id = ? AND user_id = ?", a.ContactID, user).Find(&logs)
	if len(logs) != 1 || logs[0].Direction != "outgoing" || logs[0].Channel != "Email" || logs[0].Content != content {
		t.Fatalf("interaction logs = %+v", logs)
	}
	if res.ContactChange == nil || res.ContactChange.PreviousState != "COLD" || res.ContactChange.NewState != "NO_REPLY" {
		t.Fatalf("contact change = %+v", res.ContactChange)
	}
	cl := res.CompletionLog
	if cl.Transition != "mark_sent" || *cl.Channel != "Email" || *cl.Feedback != "edited" || *cl.Content != content {
		t.Fatalf("completion log = %+v", cl)
	}
}

func TestExecuteTransition_RepositoryFailuresSurface(t *testing.T) {
	// With the actions table gone every transition must fail rather than
	// report success for a write that never happened.
	for _, tr := range []domain.Transition{
		domain.TransitionSkip, domain.TransitionSnooze, domain.TransitionMarkCompleted, domain.TransitionMarkSent,
	} {
		t.Run(string(tr), func(t *testing.T) {
			svc, db := newServiceDB(t)
			a := domain.DailyAction{UserID: uuid.New(), Status: domain.ActionStatusSuggested}
			a.ID = uuid.New()
			mustExec(t, db, `DROP TABLE daily_actions`)
			if _, err := svc.ExecuteTransition(context.Background(), &a, tr, nil, nil, nil, nil, nil); err == nil {
				t.Fatal("expected error")
			}
		})
	}
	t.Run("reopen", func(t *testing.T) {
		svc, db := newServiceDB(t)
		a := domain.DailyAction{UserID: uuid.New(), Status: domain.ActionStatusSnoozed}
		a.ID = uuid.New()
		mustExec(t, db, `DROP TABLE daily_actions`)
		if _, err := svc.ExecuteTransition(context.Background(), &a, domain.TransitionReopen, nil, nil, nil, nil, nil); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("snooze record", func(t *testing.T) {
		svc, db := newServiceDB(t)
		a := domain.DailyAction{UserID: uuid.New(), Status: domain.ActionStatusSuggested}
		mustExec(t, db, `DROP TABLE action_snoozes`)
		if _, err := svc.ExecuteTransition(context.Background(), &a, domain.TransitionSnooze, nil, nil, nil, nil, nil); err == nil {
			t.Fatal("expected error")
		}
	})
}

// Actions are scoped to their owner: guessing another rep's action id must
// return nothing, not their action.
func TestFindActionByIDAndUser_Scoped(t *testing.T) {
	svc, db := newServiceDB(t)
	owner, other := uuid.New(), uuid.New()
	_, actions := seedGeneration(t, db, owner, domain.DailyAction{})

	got, err := svc.FindActionByIDAndUser(context.Background(), owner, actions[0].ID)
	if err != nil || got == nil {
		t.Fatalf("owner lookup: %v %v", got, err)
	}
	got, err = svc.FindActionByIDAndUser(context.Background(), other, actions[0].ID)
	if err != nil || got != nil {
		t.Fatalf("other user must not see the action: %v %v", got, err)
	}
}

func TestGetOutcomeMetrics(t *testing.T) {
	svc, db := newServiceDB(t)
	ctx := context.Background()
	user, other := uuid.New(), uuid.New()

	if err := db.Create(&domain.OutcomeMetrics{UserID: user, Period: "7d", TotalSent: 5, TotalReplied: 2, ReplyRateOverall: 0.4}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	db.Create(&domain.OutcomeMetrics{UserID: other, Period: "30d", TotalSent: 99})

	m, err := svc.GetOutcomeMetrics(ctx, user, "7d")
	if err != nil || m.TotalSent != 5 || m.ReplyRateOverall != 0.4 {
		t.Fatalf("7d: %+v %v", m, err)
	}
	// A missing row is an error, which the handler turns into "not computed"
	// rather than a confident zero. It must not fall back to another period or
	// another user's row.
	for _, tc := range []struct {
		user   uuid.UUID
		period string
	}{{user, "30d"}, {user, "90d"}, {uuid.New(), "30d"}} {
		if m, err := svc.GetOutcomeMetrics(ctx, tc.user, tc.period); err == nil {
			t.Errorf("%s/%s: expected not-found, got %+v", tc.user, tc.period, m)
		}
	}
}

func TestCheckStaleness(t *testing.T) {
	svc, db := newServiceDB(t)
	ctx := context.Background()
	user := uuid.New()

	gen, _ := seedGeneration(t, db, user)
	generated := *gen.GeneratedAt

	if stale, err := svc.CheckStaleness(ctx, user, nil); stale || err != nil {
		t.Fatal("nil generation is never stale")
	}
	if stale, _ := svc.CheckStaleness(ctx, user, &domain.DailyActionGeneration{}); stale {
		t.Fatal("a generation that never finished is not stale")
	}
	if stale, _ := svc.CheckStaleness(ctx, user, &domain.DailyActionGeneration{Status: domain.GenerationStatusStale, GeneratedAt: &generated}); !stale {
		t.Fatal("already stale stays stale")
	}

	// An outgoing message, an older reply, or another user's reply do not
	// invalidate today's list.
	add := func(u uuid.UUID, dir string, at time.Time) {
		if err := db.Create(&outreach.InteractionLog{UserID: u, ContactID: uuid.New(), Direction: dir, Timestamp: at}).Error; err != nil {
			t.Fatalf("seed interaction: %v", err)
		}
	}
	add(user, "outgoing", generated.Add(time.Minute))
	add(user, "incoming", generated.Add(-time.Hour))
	add(uuid.New(), "incoming", generated.Add(time.Minute))
	if stale, err := svc.CheckStaleness(ctx, user, gen); stale || err != nil {
		t.Fatalf("stale=%v err=%v, want fresh", stale, err)
	}

	add(user, "incoming", generated.Add(time.Minute))
	if stale, err := svc.CheckStaleness(ctx, user, gen); !stale || err != nil {
		t.Fatalf("stale=%v err=%v, want stale after a new reply", stale, err)
	}
	var stored domain.DailyActionGeneration
	db.First(&stored, "id = ?", gen.ID)
	if stored.Status != domain.GenerationStatusStale {
		t.Fatalf("stored status = %s, want stale persisted", stored.Status)
	}

	mustExec(t, db, `DROP TABLE interaction_logs`)
	if _, err := svc.CheckStaleness(ctx, user, gen); err == nil {
		t.Fatal("expected a database error to surface")
	}
}

func TestBuildPipelineSummary(t *testing.T) {
	svc, db := newServiceDB(t)
	ctx := context.Background()
	user := uuid.New()

	snap := func(stage, life string) base.JSONB {
		b, _ := json.Marshal(domain.ContactSnapshot{OutreachStage: stage, LifecycleStage: life})
		return b
	}
	actions := []domain.DailyAction{
		{ContactSnapshot: snap("COLD", "LEAD")},
		{ContactSnapshot: snap("COLD", "OPPORTUNITY")},
		{ContactSnapshot: snap("", "")},
		{ContactSnapshot: base.JSONB(`not json`)},
	}

	empty := svc.BuildPipelineSummary(ctx, user, actions)
	if empty.TotalActiveContacts != 4 || empty.ContactsByStage["COLD"] != 2 || empty.ContactsByLifecycle["LEAD"] != 1 ||
		len(empty.ContactsByLifecycle) != 2 {
		t.Fatalf("stage counts = %+v", empty)
	}
	// No outgoing traffic: rate and response time stay zero instead of NaN/Inf.
	if empty.ResponseRate7d != 0 || empty.AvgResponseTimeHours != 0 {
		t.Fatalf("no traffic should read zero, got %+v", empty)
	}

	for i := 0; i < 4; i++ {
		db.Create(&outreach.InteractionLog{UserID: user, ContactID: uuid.New(), Direction: "outgoing", Timestamp: time.Now().Add(-time.Hour)})
	}
	db.Create(&outreach.InteractionLog{UserID: user, ContactID: uuid.New(), Direction: "incoming", Timestamp: time.Now().Add(-time.Hour)})
	// Outside the window.
	db.Create(&outreach.InteractionLog{UserID: user, ContactID: uuid.New(), Direction: "incoming", Timestamp: time.Now().AddDate(0, 0, -10)})
	db.Create(&outreach.Meeting{UserID: user, ContactID: uuid.New(), Time: time.Now().Add(-24 * time.Hour), Status: "completed"})

	s := svc.BuildPipelineSummary(ctx, user, nil)
	if s.ResponseRate7d != 0.25 || s.MeetingsBooked7d != 1 || s.AvgResponseTimeHours != 168 {
		t.Fatalf("summary = %+v", s)
	}
}

func TestCreateFromAgentRecommendations(t *testing.T) {
	svc, db := newServiceDB(t)
	user := uuid.New()
	c1, c2 := uuid.New(), uuid.New()

	gen, err := svc.CreateFromAgentRecommendations(context.Background(), user, &domain.CreateFromAgentRequest{
		Language:      "en",
		StrategicPlan: "Reply to warm leads first",
		Recommendations: []domain.AgentRecommendation{
			{ContactID: c1, PriorityRank: 1, ActionType: "respond", CategoryID: "replied", DraftMessage: "Thanks!", ConfidenceLevel: "high"},
			{ContactID: c2, PriorityRank: 2, ActionType: "outreach", CategoryID: "new_outreach"},
		},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if gen.Status != domain.GenerationStatusReady || gen.ActionCount != 2 || gen.GeneratedAt == nil {
		t.Fatalf("gen = %+v", gen)
	}
	var brief domain.AgentBriefingData
	_ = json.Unmarshal(gen.AgentBriefing, &brief)
	if brief.StrategicReasoning != "Reply to warm leads first" {
		t.Fatalf("briefing = %+v", brief)
	}

	var actions []domain.DailyAction
	db.Where("generation_id = ?", gen.ID).Order("priority").Find(&actions)
	if len(actions) != 2 || actions[0].ContactID != c1 || actions[0].Type != domain.ActionTypeRespond ||
		actions[0].Status != domain.ActionStatusSuggested {
		t.Fatalf("actions = %+v", actions)
	}
	var od map[string]any
	_ = json.Unmarshal(actions[0].OutreachData, &od)
	if od["draft_message"] != "Thanks!" || od["confidence_level"] != "high" {
		t.Fatalf("outreach data = %v", od)
	}

	mustExec(t, db, `DROP TABLE daily_actions`)
	if _, err := svc.CreateFromAgentRecommendations(context.Background(), user, &domain.CreateFromAgentRequest{
		Recommendations: []domain.AgentRecommendation{{ContactID: c1}},
	}); err == nil {
		t.Fatal("expected batch create failure to surface")
	}
}

func TestChatResponses(t *testing.T) {
	svc, db := newServiceDB(t)
	ctx := context.Background()
	user := uuid.New()

	later := time.Now().Add(5 * time.Hour)
	gen, _ := seedGeneration(t, db, user,
		domain.DailyAction{Priority: 30, Status: domain.ActionStatusCompleted, Reasoning: "done"},
		domain.DailyAction{Priority: 10, Status: domain.ActionStatusSkipped, Reasoning: "skipped"},
		domain.DailyAction{Priority: 5, Status: domain.ActionStatusSnoozed, SnoozeUntil: &later, Reasoning: "snoozed"},
		domain.DailyAction{Priority: 40, CategoryID: domain.CategoryReplied, Reasoning: "reply to D"},
		domain.DailyAction{Priority: 20, CategoryID: domain.CategoryNewOutreach, Reasoning: "reach E"},
	)

	tests := []struct {
		msg  string
		gen  *domain.DailyActionGeneration
		want string
	}{
		// Skipped actions are decided, not remaining — the same rule the
		// summary uses. Two open actions plus one snoozed remain.
		{"How many actions today?", gen, "You have 5 actions today. 1 completed, 3 remaining."},
		{"give me a summary", gen, "Today's summary: 5 total actions. 1 completed, 1 skipped, 1 snoozed, 3 remaining."},
		{"show my progress", gen, "Progress: 1/5 (20%) completed."},
		{"help", gen, "I can help you"},
		{"xin chào", gen, `I received your message: "xin chào"`},
		{"how many", nil, "No actions generated for today yet"},
		{"summary", nil, "No daily actions have been generated yet today."},
		{"priority", nil, "No actions generated yet."},
		{"progress", nil, "No actions generated for today."},
	}
	for _, tc := range tests {
		if got := svc.GenerateChatResponse(ctx, user, tc.msg, tc.gen); !strings.HasPrefix(got, tc.want) {
			t.Errorf("%q: got %q, want prefix %q", tc.msg, got, tc.want)
		}
	}

	// Top priorities span every category and leave out what is finished or
	// still snoozed. Asking for them used to filter on an empty category and
	// always claim everything was done.
	got := svc.GenerateChatResponse(ctx, user, "what is my top priority?", gen)
	want := "Your top priorities:\n1. Contact E — reach E (priority: 20)\n2. Contact D — reply to D (priority: 40)"
	if got != want {
		t.Fatalf("priority answer:\n got %q\nwant %q", got, want)
	}

	empty, _ := seedGeneration(t, db, uuid.New(), domain.DailyAction{Status: domain.ActionStatusCompleted})
	if got := svc.GenerateChatResponse(ctx, user, "urgent?", empty); got != "All actions are completed! Great work." {
		t.Fatalf("all done: %q", got)
	}
	none, _ := seedGeneration(t, db, uuid.New())
	if got := svc.GenerateChatResponse(ctx, user, "progress", none); got != "No actions today." {
		t.Fatalf("empty generation: %q", got)
	}
}

// buildAction must pick the nearest upcoming meeting. Meetings come back
// newest-first, so taking the first scheduled one picked the furthest away
// and hid a call two hours out behind one next month.
func TestBuildAction_NearestMeetingDrivesPrep(t *testing.T) {
	svc, db := newServiceDB(t)
	ctx := context.Background()
	user := uuid.New()
	c := completeContact("LEAD")
	c.ID = uuid.New()
	draft := "ready draft"

	soon := time.Now().Add(2 * time.Hour)
	for _, m := range []outreach.Meeting{
		{UserID: user, ContactID: c.ID, Time: time.Now().Add(-48 * time.Hour), Status: "completed"},
		{UserID: user, ContactID: c.ID, Time: soon, Status: "scheduled"},
		{UserID: user, ContactID: c.ID, Time: time.Now().Add(30 * 24 * time.Hour), Status: "scheduled"},
		{UserID: user, ContactID: c.ID, Time: time.Now().Add(time.Hour), Status: "cancelled"},
	} {
		m := m
		if err := db.Create(&m).Error; err != nil {
			t.Fatalf("seed meeting: %v", err)
		}
	}

	sc := &outreachSvc.SuggestContact{Contact: c, State: &outreach.OutreachState{
		ConversationState: "NO_REPLY", NextStep: "FOLLOW_UP_1", DaysSinceLastInteraction: 4,
		MaxFollowups: 2, MessageDraft: &draft, UpdatedAt: time.Now(),
	}}
	a, err := svc.buildAction(ctx, uuid.New(), user, sc, "Rep", "en")
	if err != nil {
		t.Fatalf("buildAction: %v", err)
	}
	if a.Type != domain.ActionTypeMeetingPrep || a.CategoryID != domain.CategoryMeetingPrep {
		t.Fatalf("type = %s/%s, want meeting_prep", a.Type, a.CategoryID)
	}
	var md domain.MeetingActionData
	_ = json.Unmarshal(a.MeetingData, &md)
	if md.HoursUntilMeeting > 2.1 {
		t.Fatalf("prep points at meeting %.1fh away, want the one in 2h", md.HoursUntilMeeting)
	}

	// A reply outranks prep: the contact is waiting on us now.
	sc.State.ConversationState = "REPLIED"
	a, _ = svc.buildAction(ctx, uuid.New(), user, sc, "Rep", "en")
	if a.Type != domain.ActionTypeRespond || len(a.RespondData) == 0 {
		t.Fatalf("replied contact should stay respond, got %s", a.Type)
	}
}

func TestBuildAction_TypeSpecificPayloads(t *testing.T) {
	svc, _ := newServiceDB(t)
	ctx := context.Background()
	user := uuid.New()
	draft := "stored"

	pending := &contact.Contact{Name: "P", Status: "pending"}
	pending.ID = uuid.New()
	a, err := svc.buildAction(ctx, uuid.New(), user, &outreachSvc.SuggestContact{Contact: pending}, "Rep", "en")
	if err != nil || a.Type != domain.ActionTypeEnrich || len(a.EnrichmentData) == 0 {
		t.Fatalf("pending contact: %+v %v", a, err)
	}

	ready := completeContact("LEAD")
	ready.ID = uuid.New()
	a, err = svc.buildAction(ctx, uuid.New(), user, &outreachSvc.SuggestContact{Contact: ready, State: &outreach.OutreachState{
		ConversationState: "COLD", NextStep: "SEND", MaxFollowups: 2, MessageDraft: &draft, UpdatedAt: time.Now(),
	}}, "Rep", "en")
	if err != nil || a.Type != domain.ActionTypeOutreach {
		t.Fatalf("cold contact: %+v %v", a, err)
	}
	var od domain.OutreachActionData
	_ = json.Unmarshal(a.OutreachData, &od)
	if od.DraftMessage != "stored" {
		t.Fatalf("outreach data = %+v", od)
	}
	var snap domain.ContactSnapshot
	_ = json.Unmarshal(a.ContactSnapshot, &snap)
	if snap.ID != ready.ID || snap.LifecycleStage != "LEAD" || a.Priority < 1 || a.Priority > 100 {
		t.Fatalf("snapshot=%+v priority=%d", snap, a.Priority)
	}
}

// A generation that fails part-way must not be left behind as "in progress".
// FindActiveByUserAndDate treats started/generating as a run already under
// way, so a leftover row made every later non-forced request — including the
// queue's own retry — return the dead generation as a success, and the UI
// showed "generating" for the rest of the day.
func TestGenerateActions_FailureDoesNotLeaveZombieGeneration(t *testing.T) {
	svc, db := newServiceDB(t)
	ctx := context.Background()
	user := uuid.New() // no users row: the pipeline fails loading the user

	if _, err := svc.GenerateActions(ctx, user, "en", false); err == nil {
		t.Fatal("expected failure for a user that does not exist")
	}

	gen, err := svc.GenerateActions(ctx, user, "en", false)
	if err == nil {
		t.Fatalf("retry returned the failed generation %s (status %s) as a success", gen.ID, gen.Status)
	}

	var live int64
	db.Model(&domain.DailyActionGeneration{}).
		Where("user_id = ? AND is_deleted = ? AND status IN ?", user, false, []string{"started", "generating"}).
		Count(&live)
	if live != 0 {
		t.Fatalf("%d in-progress generations left behind", live)
	}
}

func TestGenerateActions_ReturnsActiveRun(t *testing.T) {
	svc, db := newServiceDB(t)
	user := uuid.New()

	// A run already in progress is returned as-is so a double click does not
	// start a second pipeline.
	running := &domain.DailyActionGeneration{UserID: user, Date: time.Now().Format("2006-01-02"), Status: domain.GenerationStatusGenerating}
	if err := db.Create(running).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, err := svc.GenerateActions(context.Background(), user, "en", false)
	if err != nil || got.ID != running.ID {
		t.Fatalf("want the running generation back, got %+v %v", got, err)
	}
}

// Force refresh retires today's generation and links it to its replacement.
// The link used to be written after the soft-delete, by a query that only
// touches live rows, so replaced_by_id was never set.
func TestGenerateActions_ForceRefreshLinksReplacedGeneration(t *testing.T) {
	svc, db := newServiceDB(t)
	if err := db.AutoMigrate(&rootdomain.Organization{}, &rootdomain.Role{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	ctx := context.Background()
	user := uuid.New()
	mustExec(t, db, `INSERT INTO users (id, name) VALUES (?, 'Rep')`, user)
	prev, prevActions := seedGeneration(t, db, user, domain.DailyAction{})

	gen, err := svc.GenerateActions(ctx, user, "en", true)
	if err != nil {
		t.Fatalf("force refresh: %v", err)
	}
	var stored domain.DailyActionGeneration
	db.First(&stored, "id = ?", prev.ID)
	if !stored.IsDeleted || stored.ReplacedByID == nil || *stored.ReplacedByID != gen.ID {
		t.Fatalf("previous generation should be retired and point at %s: deleted=%v replaced_by=%v",
			gen.ID, stored.IsDeleted, stored.ReplacedByID)
	}
	if !reload(t, db, prevActions[0].ID).IsDeleted {
		t.Fatal("previous generation's actions should be retired with it")
	}
}

// The full pipeline: suggestions become persisted actions in the model's order,
// and the generation ends ready with its briefing and summary attached.
func TestGenerateActions_HappyPath(t *testing.T) {
	svc, db := newServiceDB(t)
	if err := db.AutoMigrate(&rootdomain.Organization{}, &rootdomain.Role{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	ctx := context.Background()
	user, org := uuid.New(), uuid.New()
	mustExec(t, db, `INSERT INTO users (id, name) VALUES (?, 'Rep')`, user)
	mustExec(t, db, `INSERT INTO organizations (id, name, is_deleted) VALUES (?, 'Acme', FALSE)`, org)
	mustExec(t, db, `INSERT INTO roles (id, user_id, organization_id, name, is_deleted) VALUES (?, ?, ?, 'admin', FALSE)`, uuid.New(), user, org)
	for _, name := range []string{"Cold One", "Cold Two"} {
		mustExec(t, db, `INSERT INTO contacts (id, user_id, name, company, job_title, source, source_id, contact_information,
			status, next_step, is_deleted) VALUES (?, ?, ?, 'Acme', 'CTO', 'csv', ?, 'x@acme.test', 'ready', 'SEND', FALSE)`,
			uuid.New(), user, name, uuid.NewString())
	}

	// The model reverses whatever order the heuristic produced.
	client := &stubChatClient{reply: `[{"id":"a1","reason":"warmer"},{"id":"a0","reason":"colder"}]`}
	svc.WithAIPrioritizer(client)

	gen, err := svc.GenerateActions(ctx, user, "en", false)
	if err != nil {
		t.Fatalf("GenerateActions: %v", err)
	}
	if gen.Status != domain.GenerationStatusReady || gen.ActionCount != 2 || client.calls != 1 {
		t.Fatalf("gen=%+v calls=%d", gen, client.calls)
	}
	var stored domain.DailyActionGeneration
	db.First(&stored, "id = ?", gen.ID)
	if stored.Status != domain.GenerationStatusReady || stored.ActionCount != 2 || len(stored.AgentBriefing) < 10 {
		t.Fatalf("stored gen = %+v", stored)
	}

	var actions []domain.DailyAction
	db.Where("generation_id = ?", gen.ID).Order("priority").Find(&actions)
	if len(actions) != 2 || actions[0].Priority != 1 || actions[1].Priority != 2 {
		t.Fatalf("actions = %+v", actions)
	}
	var factors []domain.PriorityFactor
	_ = json.Unmarshal(actions[0].PriorityFactors, &factors)
	if last := factors[len(factors)-1]; last.Factor != "ai_ranking" || last.Description != "warmer" {
		t.Fatalf("top action should carry the model's reason, got %+v", factors)
	}
}

// A plain member of someone else's organisation has no "main" organisation
// (they neither administer nor created one). The org lookup's not-found used
// to abort the whole pipeline, so members could never get a daily list.
func TestGenerateActions_MemberWithoutOwnOrganization(t *testing.T) {
	svc, db := newServiceDB(t)
	if err := db.AutoMigrate(&rootdomain.Organization{}, &rootdomain.Role{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	user, org := uuid.New(), uuid.New()
	mustExec(t, db, `INSERT INTO users (id, name) VALUES (?, 'Member')`, user)
	mustExec(t, db, `INSERT INTO organizations (id, name, user_id, is_deleted) VALUES (?, 'Acme', ?, FALSE)`, org, uuid.New())
	mustExec(t, db, `INSERT INTO roles (id, user_id, organization_id, name, is_deleted) VALUES (?, ?, ?, 'member', FALSE)`, uuid.New(), user, org)

	gen, err := svc.GenerateActions(context.Background(), user, "vi", false)
	if err != nil {
		t.Fatalf("member generation failed: %v", err)
	}
	if gen.Status != domain.GenerationStatusReady {
		t.Fatalf("status = %s, want ready", gen.Status)
	}
}
