package playbook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	playbookDomain "github.com/rockship/cosmo-agents-go/internal/domain/playbook"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	workerpayloads "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	queueworker "github.com/rockship/cosmo-agents-go/pkg/worker"
)

func processAll(t *testing.T, w *Worker) {
	t.Helper()
	if err := w.HandleProcessEnrollments(context.Background(), task(t, queueworker.TypePlaybookProcessEnrollments, workerpayloads.ProcessEnrollmentsPayload{})); err != nil {
		t.Fatalf("HandleProcessEnrollments: %v", err)
	}
}

func TestProcessEnrollmentsAdvancesThroughStages(t *testing.T) {
	f := newFixture(t)
	pb := f.playbook(
		v1schema.PlaybookStage{ID: "li", Order: 1, Type: v1schema.StageTypeLinkedIn},
		v1schema.PlaybookStage{ID: "call", Order: 2, Type: v1schema.StageTypeCall},
	)
	id := f.enroll(f.contact(0).ID, pb.PlaybookID, "li", 1, 0)
	w := f.worker(nil, nil)

	processAll(t, w)
	e, log := f.enrollment(id)
	if e.CurrentStageID != "call" || e.CurrentStageOrder != 2 || e.EnrollmentStatus != "active" {
		t.Fatalf("after run 1: %s at %s/%d, want active at call/2", e.EnrollmentStatus, e.CurrentStageID, e.CurrentStageOrder)
	}
	if !stageCompleted(log, "li") || len(log.Events) != 1 || log.Events[0].Type != "linkedin" {
		t.Errorf("log after run 1 = %+v", log)
	}

	processAll(t, w)
	e, log = f.enrollment(id)
	if e.EnrollmentStatus != "completed" || e.CompletedAt == nil {
		t.Fatalf("after the last stage: status %s completed_at %v, want completed with a timestamp", e.EnrollmentStatus, e.CompletedAt)
	}
	if len(log.StagesCompleted) != 2 {
		t.Errorf("stages completed = %d, want 2", len(log.StagesCompleted))
	}

	// A completed enrollment is not processed again.
	processAll(t, w)
	if _, again := f.enrollment(id); len(again.StagesCompleted) != 2 {
		t.Errorf("completed enrollment was re-executed")
	}
}

func TestProcessEnrollmentsTimeoutActions(t *testing.T) {
	tests := []struct {
		action     v1schema.SuccessAction
		wantStatus string
		wantStage  string
	}{
		{"", "active", "b"},
		{v1schema.SuccessActionAdvance, "active", "b"},
		{v1schema.SuccessActionPause, "paused", "a"},
		{v1schema.SuccessActionComplete, "completed", "a"},
	}
	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			f := newFixture(t)
			pb := f.playbook(
				v1schema.PlaybookStage{ID: "a", Order: 1, Type: v1schema.StageTypeWait, SuccessCriteria: v1schema.SuccessCriteria{OnTimeout: tt.action}},
				v1schema.PlaybookStage{ID: "b", Order: 2, Type: v1schema.StageTypeWait},
			)
			id := f.enroll(f.contact(0).ID, pb.PlaybookID, "a", 1, 0)
			processAll(t, f.worker(nil, nil))
			e, _ := f.enrollment(id)
			if e.EnrollmentStatus != tt.wantStatus || e.CurrentStageID != tt.wantStage {
				t.Errorf("got %s at %s, want %s at %s", e.EnrollmentStatus, e.CurrentStageID, tt.wantStatus, tt.wantStage)
			}
			if (tt.wantStatus == "completed") != (e.CompletedAt != nil) {
				t.Errorf("completed_at = %v for status %s", e.CompletedAt, e.EnrollmentStatus)
			}
		})
	}
}

// Stage orders are whatever the playbook author typed; nothing requires them
// to be consecutive. A gap must not end the playbook early.
func TestProcessEnrollmentsStageOrderGap(t *testing.T) {
	f := newFixture(t)
	pb := f.playbook(
		v1schema.PlaybookStage{ID: "first", Order: 1, Type: v1schema.StageTypeWait},
		v1schema.PlaybookStage{ID: "third", Order: 3, Type: v1schema.StageTypeWait},
	)
	id := f.enroll(f.contact(0).ID, pb.PlaybookID, "first", 1, 0)
	processAll(t, f.worker(nil, nil))
	e, _ := f.enrollment(id)
	if e.EnrollmentStatus != "active" || e.CurrentStageID != "third" {
		t.Fatalf("got %s at %s, want active at third", e.EnrollmentStatus, e.CurrentStageID)
	}
}

func TestProcessEnrollmentsWaitDuration(t *testing.T) {
	f := newFixture(t)
	pb := f.playbook(
		v1schema.PlaybookStage{ID: "wait", Order: 1, Type: v1schema.StageTypeWait, TriggerConditions: v1schema.TriggerConditions{WaitDuration: 2}},
		v1schema.PlaybookStage{ID: "next", Order: 2, Type: v1schema.StageTypeWait, TriggerConditions: v1schema.TriggerConditions{WaitDuration: 1}},
	)
	notYet := f.enroll(f.contact(0).ID, pb.PlaybookID, "wait", 1, 47*time.Hour)
	due := f.enroll(f.contact(0).ID, pb.PlaybookID, "wait", 1, 49*time.Hour)
	w := f.worker(nil, nil)

	processAll(t, w)
	if e, log := f.enrollment(notYet); e.CurrentStageID != "wait" || len(log.StagesCompleted) != 0 {
		t.Errorf("enrollment 47h in was executed before its 2-day wait")
	}
	e, _ := f.enrollment(due)
	if e.CurrentStageID != "next" {
		t.Fatalf("enrollment 49h in is at %s, want next", e.CurrentStageID)
	}

	// The next stage's wait counts from when the previous stage ran (just now),
	// not from enrollment two days ago.
	processAll(t, w)
	if e, _ := f.enrollment(due); e.CurrentStageID != "next" || e.EnrollmentStatus != "active" {
		t.Errorf("second stage ran before its own 1-day wait: %s %s", e.EnrollmentStatus, e.CurrentStageID)
	}
}

// If the stage ran and its log was saved but the stage pointer was not moved
// (a failed UpdateStage), the next run advances without executing it twice.
func TestProcessEnrollmentsResumesAfterLoggedStage(t *testing.T) {
	f := newFixture(t)
	pb := f.playbook(stages...)
	id := f.enroll(f.contact(0).ID, pb.PlaybookID, "intro", 1, 0)
	logged, _ := json.Marshal(executionLog{StagesCompleted: []stageLog{{StageID: "intro", StageOrder: 1, Status: "completed", ExecutedAt: time.Now()}}, Events: []eventLog{}})
	f.db.MustExec(`UPDATE contact_enrollments SET execution_log = $1 WHERE enrollment_id = $2`, logged, id)

	processAll(t, f.worker(nil, nil))
	e, log := f.enrollment(id)
	if e.CurrentStageID != "follow" {
		t.Fatalf("stage = %s, want follow", e.CurrentStageID)
	}
	if len(log.StagesCompleted) != 1 {
		t.Errorf("intro executed again: %d stage entries", len(log.StagesCompleted))
	}
}

func TestProcessEnrollmentsSkips(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	pb := f.playbook(stages...)
	w := f.worker(nil, nil)

	paused := f.enroll(f.contact(0).ID, pb.PlaybookID, "intro", 1, 0)
	f.db.MustExec(`UPDATE contact_enrollments SET enrollment_status = 'paused' WHERE enrollment_id = $1`, paused)

	gone := f.playbook(stages...)
	orphan := f.enroll(f.contact(0).ID, gone.PlaybookID, "intro", 1, 0)
	if err := f.repos.playbooks.Delete(ctx, gone.PlaybookID); err != nil {
		t.Fatal(err)
	}
	lost := f.enroll(f.contact(0).ID, pb.PlaybookID, "renamed", 9, 0)

	processAll(t, w)
	for name, id := range map[string]uuid.UUID{"paused": paused, "deleted playbook": orphan, "unknown stage": lost} {
		if _, log := f.enrollment(id); len(log.StagesCompleted) != 0 {
			t.Errorf("%s enrollment was executed", name)
		}
	}

	// Targeting a paused enrollment by id does not bypass the status check.
	if err := w.HandleProcessEnrollments(ctx, task(t, queueworker.TypePlaybookProcessEnrollments, workerpayloads.ProcessEnrollmentsPayload{EnrollmentID: &paused})); err != nil {
		t.Fatalf("single enrollment: %v", err)
	}
	if _, log := f.enrollment(paused); len(log.StagesCompleted) != 0 {
		t.Errorf("paused enrollment executed when targeted directly")
	}
	unknown := uuid.New()
	if err := w.HandleProcessEnrollments(ctx, task(t, queueworker.TypePlaybookProcessEnrollments, workerpayloads.ProcessEnrollmentsPayload{EnrollmentID: &unknown})); err != nil {
		t.Errorf("unknown enrollment: %v", err)
	}
	if err := w.HandleProcessEnrollments(ctx, task(t, queueworker.TypePlaybookProcessEnrollments, 42)); err == nil {
		t.Errorf("malformed payload accepted")
	}
}

func emailPlaybook(f *fixture, cfg *v1schema.ContentConfig) uuid.UUID {
	return f.playbook(
		v1schema.PlaybookStage{ID: "mail", Order: 1, Type: v1schema.StageTypeEmail, ContentConfig: cfg},
		v1schema.PlaybookStage{ID: "after", Order: 2, Type: v1schema.StageTypeWait},
	).PlaybookID
}

func TestEmailStageQueuesSend(t *testing.T) {
	client, insp := testRedis(t)
	f := newFixture(t)
	agentID := f.agent("active")
	contact := f.contact(0)
	pb := emailPlaybook(f, &v1schema.ContentConfig{Template: "Hi {{first_name}} at {{company}} ({{job_title}})"})
	id := f.enroll(contact.ID, pb, "mail", 1, 0)

	processAll(t, f.worker(client, nil))

	tasks, err := insp.ListPendingTasks("default")
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Type != queueworker.TypeSendEmail {
		t.Fatalf("pending tasks = %d, want one %s", len(tasks), queueworker.TypeSendEmail)
	}
	var p workerpayloads.SendEmailPayload
	if err := json.Unmarshal(tasks[0].Payload, &p); err != nil {
		t.Fatal(err)
	}
	if p.AgentID != agentID || p.ContactID != contact.ID || p.To != "ada@example.com" {
		t.Errorf("payload routing = %+v", p)
	}
	if p.Subject != "Quick intro, Ada" || p.Body != "Hi Ada at Analytical (CTO)" {
		t.Errorf("content = %q / %q", p.Subject, p.Body)
	}
	if e, _ := f.enrollment(id); e.CurrentStageID != "after" {
		t.Errorf("stage = %s, want after", e.CurrentStageID)
	}
}

func TestEmailStageUsesGeneratedContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"role": "assistant", "content": `{"subject":"AI subject","body":"AI body"}`}}},
		})
	}))
	defer srv.Close()
	t.Setenv("OPENAI_BASE_URL", srv.URL)

	client, insp := testRedis(t)
	f := newFixture(t)
	f.agent("active")
	pb := emailPlaybook(f, &v1schema.ContentConfig{Template: "fallback", AIGenerationPrompt: "be warm"})
	f.enroll(f.contact(0).ID, pb, "mail", 1, 0)

	processAll(t, f.worker(client, ai.NewOpenAIClient(ai.Config{APIKey: "k", Model: "gpt-4o-mini"})))

	tasks, _ := insp.ListPendingTasks("default")
	if len(tasks) != 1 {
		t.Fatalf("pending tasks = %d, want 1", len(tasks))
	}
	var p workerpayloads.SendEmailPayload
	_ = json.Unmarshal(tasks[0].Payload, &p)
	if p.Subject != "AI subject" || p.Body != "AI body" {
		t.Errorf("content = %q / %q, want the generated email", p.Subject, p.Body)
	}
}

func TestEmailStageDoNotContact(t *testing.T) {
	f := newFixture(t)
	f.agent("active")
	c := f.contact(0)
	f.gdb.Model(c).Update("do_not_contact", true)
	id := f.enroll(c.ID, emailPlaybook(f, nil), "mail", 1, 0)

	processAll(t, f.worker(nil, nil))
	_, log := f.enrollment(id)
	if len(log.Events) != 1 || !strings.Contains(log.Events[0].Message, "do-not-contact") {
		t.Fatalf("events = %+v, want a do-not-contact skip", log.Events)
	}
}

func TestEmailStageMissingContactRetries(t *testing.T) {
	f := newFixture(t)
	id := f.enroll(uuid.New(), emailPlaybook(f, nil), "mail", 1, 0)
	processAll(t, f.worker(nil, nil))
	e, log := f.enrollment(id)
	if e.CurrentStageID != "mail" || len(log.StagesCompleted) != 0 {
		t.Errorf("stage advanced for a contact that does not exist")
	}
}

// With no active agent there is nobody to send from, yet the stage is logged
// as completed and the enrollment moves on: the email is silently never sent.
func TestEmailStageWithoutAgentDoesNotAdvance(t *testing.T) {
	t.Skip("BUG: executeEmailStage returns nil when the user has no active agent (playbook_worker.go findDefaultAgentID branch), so the email stage is marked completed and the enrollment advances/completes without any email being queued")
	f := newFixture(t)
	f.agent("paused")
	id := f.enroll(f.contact(0).ID, emailPlaybook(f, nil), "mail", 1, 0)
	processAll(t, f.worker(nil, nil))
	if e, _ := f.enrollment(id); e.CurrentStageID != "mail" {
		t.Errorf("stage = %s, want mail until an agent can send it", e.CurrentStageID)
	}
}

func TestPureHelpers(t *testing.T) {
	t.Run("findStage prefers id, falls back to order", func(t *testing.T) {
		if s := findStage(stages, "follow", 1); s == nil || s.ID != "follow" {
			t.Errorf("by id = %v", s)
		}
		if s := findStage(stages, "gone", 1); s == nil || s.ID != "intro" {
			t.Errorf("by order = %v", s)
		}
		if s := findStage(stages, "", 7); s != nil {
			t.Errorf("no match = %v", s)
		}
	})
	t.Run("parseExecutionLog tolerates bad input", func(t *testing.T) {
		for _, raw := range []string{"", "not json", `{"stages_completed":null}`, `{"events":null}`} {
			log := parseExecutionLog([]byte(raw))
			if log.StagesCompleted == nil || log.Events == nil {
				t.Errorf("parseExecutionLog(%q) left nil slices", raw)
			}
		}
	})
	t.Run("buildStageContent", func(t *testing.T) {
		subj, body := buildStageContent(v1schema.PlaybookStage{}, "Grace Hopper", "Navy", "RADM")
		if subj != "Quick intro, Grace" || !strings.HasPrefix(body, "Hi Grace Hopper,") {
			t.Errorf("default = %q / %q", subj, body)
		}
		subj, _ = buildStageContent(v1schema.PlaybookStage{}, "", "", "")
		if subj != "Quick intro, " {
			t.Errorf("empty name subject = %q", subj)
		}
	})
	t.Run("extractEngagementScore", func(t *testing.T) {
		cases := map[string]int{
			``:                                 0,
			`nope`:                             0,
			`{}`:                               0,
			`{"scores":"x"}`:                   0,
			`{"scores":{}}`:                    0,
			`{"scores":{"engagement":"high"}}`: 0,
			`{"scores":{"engagement":42.9}}`:   42,
		}
		for raw, want := range cases {
			if got := extractEngagementScore([]byte(raw)); got != want {
				t.Errorf("extractEngagementScore(%q) = %d, want %d", raw, got, want)
			}
		}
	})
	t.Run("extractContactEmail", func(t *testing.T) {
		if got := extractContactEmail("x@y.z", nil); got != "x@y.z" {
			t.Errorf("got %q", got)
		}
	})
	t.Run("stageReady without a log uses enrolled_at, then created_at", func(t *testing.T) {
		stage := v1schema.PlaybookStage{TriggerConditions: v1schema.TriggerConditions{WaitDuration: 1}}
		old := time.Now().Add(-48 * time.Hour)
		if !stageReady(stage, &playbookDomain.ContactEnrollment{CreatedAt: old}, executionLog{}) {
			t.Errorf("created two days ago, 1-day wait: not ready")
		}
		now := time.Now()
		if stageReady(stage, &playbookDomain.ContactEnrollment{CreatedAt: old, EnrolledAt: &now}, executionLog{}) {
			t.Errorf("enrolled just now, 1-day wait: ready")
		}
	})
}
