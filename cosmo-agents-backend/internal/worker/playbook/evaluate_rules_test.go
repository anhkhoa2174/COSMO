package playbook

import (
	"context"
	"testing"

	"github.com/google/uuid"

	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	workerpayloads "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	queueworker "github.com/rockship/cosmo-agents-go/pkg/worker"
)

var stages = []v1schema.PlaybookStage{
	{ID: "intro", Order: 1, Type: v1schema.StageTypeWait},
	{ID: "follow", Order: 2, Type: v1schema.StageTypeWait},
}

func (f *fixture) enrolledContacts(playbookID uuid.UUID) map[uuid.UUID]*uuid.UUID {
	f.t.Helper()
	rows := []struct {
		ContactID uuid.UUID  `db:"contact_id"`
		RuleID    *uuid.UUID `db:"automation_rule_id"`
	}{}
	if err := f.db.Select(&rows, `SELECT contact_id, automation_rule_id FROM contact_enrollments WHERE playbook_id = $1`, playbookID); err != nil {
		f.t.Fatalf("read enrollments: %v", err)
	}
	out := map[uuid.UUID]*uuid.UUID{}
	for _, r := range rows {
		out[r.ContactID] = r.RuleID
	}
	return out
}

func (f *fixture) requests() map[uuid.UUID]string {
	f.t.Helper()
	rows := []struct {
		ContactID uuid.UUID `db:"contact_id"`
		Status    string    `db:"status"`
	}{}
	if err := f.db.Select(&rows, `SELECT contact_id, status FROM enrollment_approval_requests`); err != nil {
		f.t.Fatalf("read requests: %v", err)
	}
	out := map[uuid.UUID]string{}
	for _, r := range rows {
		out[r.ContactID] = r.Status
	}
	return out
}

func TestEvaluateRulesThresholds(t *testing.T) {
	ptr := func(id uuid.UUID) *uuid.UUID { return &id }

	tests := []struct {
		name     string
		criteria string
		// contacts: fit score, engagement, passes filters -> want enrolled
		contacts []struct {
			fit, engagement int
			passes, want    bool
		}
	}{
		{
			// An unset threshold means 50, not "everyone".
			name:     "default fit threshold is 50",
			criteria: `{}`,
			contacts: []struct {
				fit, engagement int
				passes, want    bool
			}{
				{49, 0, true, false},
				{50, 0, true, true},
				{95, 0, true, true},
				{95, 0, false, false}, // failed the segment's hard filters
			},
		},
		{
			name:     "explicit fit threshold",
			criteria: `{"fit_score_threshold": 70}`,
			contacts: []struct {
				fit, engagement int
				passes, want    bool
			}{
				{69, 0, true, false},
				{70, 0, true, true},
			},
		},
		{
			name:     "engagement threshold",
			criteria: `{"fit_score_threshold": 60, "engagement_score_threshold": 40}`,
			contacts: []struct {
				fit, engagement int
				passes, want    bool
			}{
				{80, 39, true, false},
				{80, 40, true, true},
				{59, 90, true, false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			f := newFixture(t)
			seg := f.segment()
			pb := f.playbook(stages...)
			ruleID := f.rule(seg, pb.PlaybookID, tt.criteria, true)

			want := map[uuid.UUID]bool{}
			for _, c := range tt.contacts {
				contact := f.contact(c.engagement)
				f.score(contact.ID, seg, c.fit, c.passes)
				want[contact.ID] = c.want
			}

			if err := f.worker(nil, nil).HandleEvaluateRules(ctx, task(t, queueworker.TypePlaybookEvaluateRules, workerpayloads.EvaluateRulesPayload{})); err != nil {
				t.Fatalf("HandleEvaluateRules: %v", err)
			}
			got := f.enrolledContacts(pb.PlaybookID)
			for id, w := range want {
				rule, enrolled := got[id]
				if enrolled != w {
					t.Errorf("contact %v enrolled = %v, want %v", id, enrolled, w)
				}
				if enrolled && (rule == nil || *rule != ruleID) {
					t.Errorf("enrollment rule = %v, want %v", rule, ptr(ruleID))
				}
			}
		})
	}
}

func TestEvaluateRulesSkipsAndScoping(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	seg := f.segment()
	pb := f.playbook(stages...)
	empty := f.playbook()
	w := f.worker(nil, nil)

	active := f.rule(seg, pb.PlaybookID, `{}`, true)
	// An inactive rule on another playbook must not fire during a full run.
	other := f.playbook(stages...)
	f.rule(seg, other.PlaybookID, `{}`, false)
	// A rule pointing at a playbook with no stages creates nothing.
	f.rule(seg, empty.PlaybookID, `{}`, true)

	already := f.contact(0)
	f.score(already.ID, seg, 90, true)
	enrolledID := f.enroll(already.ID, pb.PlaybookID, "follow", 2, 0)

	fresh := f.contact(0)
	f.score(fresh.ID, seg, 90, true)
	// A score whose contact row is gone is skipped rather than failing the run.
	f.score(uuid.New(), seg, 90, true)

	if err := w.HandleEvaluateRules(ctx, task(t, queueworker.TypePlaybookEvaluateRules, workerpayloads.EvaluateRulesPayload{})); err != nil {
		t.Fatalf("HandleEvaluateRules: %v", err)
	}

	got := f.enrolledContacts(pb.PlaybookID)
	if len(got) != 2 {
		t.Fatalf("enrollments in playbook = %d, want the existing one plus fresh", len(got))
	}
	if _, ok := got[fresh.ID]; !ok {
		t.Errorf("fresh contact was not enrolled")
	}
	// The existing enrollment is left exactly where it was.
	e, _ := f.enrollment(enrolledID)
	if e.CurrentStageID != "follow" || e.AutomationRuleID != nil {
		t.Errorf("existing enrollment was rewritten: stage %s rule %v", e.CurrentStageID, e.AutomationRuleID)
	}
	if n := len(f.enrolledContacts(other.PlaybookID)); n != 0 {
		t.Errorf("inactive rule enrolled %d contacts", n)
	}
	if n := len(f.enrolledContacts(empty.PlaybookID)); n != 0 {
		t.Errorf("stageless playbook got %d enrollments", n)
	}

	// Enrollment is idempotent: a second run changes nothing.
	if err := w.HandleEvaluateRules(ctx, task(t, queueworker.TypePlaybookEvaluateRules, workerpayloads.EvaluateRulesPayload{})); err != nil {
		t.Fatalf("second run: %v", err)
	}
	if n := len(f.enrolledContacts(pb.PlaybookID)); n != 2 {
		t.Errorf("second run changed enrollments to %d", n)
	}

	// A single-rule run for an unknown rule is a no-op, not an error.
	unknown := uuid.New()
	if err := w.HandleEvaluateRules(ctx, task(t, queueworker.TypePlaybookEvaluateRules, workerpayloads.EvaluateRulesPayload{RuleID: &unknown})); err != nil {
		t.Errorf("unknown rule: %v", err)
	}
	// And a single-rule run evaluates only that rule.
	late := f.contact(0)
	f.score(late.ID, seg, 90, true)
	if err := w.HandleEvaluateRules(ctx, task(t, queueworker.TypePlaybookEvaluateRules, workerpayloads.EvaluateRulesPayload{RuleID: &active})); err != nil {
		t.Fatalf("single rule: %v", err)
	}
	if _, ok := f.enrolledContacts(pb.PlaybookID)[late.ID]; !ok {
		t.Errorf("single-rule run did not enroll the new contact")
	}

	if err := w.HandleEvaluateRules(ctx, task(t, queueworker.TypePlaybookEvaluateRules, "not an object")); err == nil {
		t.Errorf("malformed payload accepted")
	}
}

func TestEvaluateRulesRequireApproval(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	seg := f.segment()
	pb := f.playbook(stages...)
	f.rule(seg, pb.PlaybookID, `{"fit_score_threshold": 60, "require_human_approval": true}`, true)
	w := f.worker(nil, nil)
	run := func() {
		t.Helper()
		if err := w.HandleEvaluateRules(ctx, task(t, queueworker.TypePlaybookEvaluateRules, workerpayloads.EvaluateRulesPayload{})); err != nil {
			t.Fatalf("HandleEvaluateRules: %v", err)
		}
	}

	c := f.contact(25)
	f.score(c.ID, seg, 75, true)
	low := f.contact(25)
	f.score(low.ID, seg, 40, true)

	run()
	if n := len(f.enrolledContacts(pb.PlaybookID)); n != 0 {
		t.Fatalf("approval rule enrolled %d contacts directly", n)
	}
	reqs := f.requests()
	if len(reqs) != 1 || reqs[c.ID] != "pending" {
		t.Fatalf("requests = %v, want one pending request for the qualifying contact", reqs)
	}
	pending, _ := f.repos.approvals.ListPending(ctx)
	if pending[0].FitScore != 75 || pending[0].EngagementScore != 25 {
		t.Errorf("request scores = %d/%d, want 75/25", pending[0].FitScore, pending[0].EngagementScore)
	}

	// The rule runs every five minutes; a waiting request is not duplicated.
	run()
	if n := len(f.requests()); n != 1 {
		t.Fatalf("second run left %d requests, want 1", n)
	}

	// Nor is a rejected one re-raised while the score is unchanged.
	if err := f.repos.approvals.Reject(ctx, pending[0].RequestID, uuid.New()); err != nil {
		t.Fatal(err)
	}
	run()
	var n int
	_ = f.db.Get(&n, `SELECT count(*) FROM enrollment_approval_requests`)
	if n != 1 {
		t.Errorf("rejected request re-raised: %d requests", n)
	}
}
