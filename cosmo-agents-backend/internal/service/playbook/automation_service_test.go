package playbook

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

func TestCreateAutomationRule(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	svc := f.automationService()
	seg := f.seedSegment(t)
	pb := f.seedPlaybook(t, twoStages())
	engagement := 30

	tests := []struct {
		name    string
		segment uuid.UUID
		pb      uuid.UUID
		wantErr string
	}{
		{"unknown segment", uuid.New(), pb.PlaybookID, "segment not found"},
		{"unknown playbook", seg.ID, uuid.New(), "playbook not found"},
		{"valid rule", seg.ID, pb.PlaybookID, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.CreateAutomationRule(ctx, &v1schema.AutomationRuleRequest{
				Name:       "High fit",
				SegmentID:  tt.segment,
				PlaybookID: tt.pb,
				EnrollmentCriteria: v1schema.EnrollmentCriteria{
					FitScoreThreshold:        70,
					EngagementScoreThreshold: &engagement,
					RequireHumanApproval:     true,
				},
				IsActive: true,
			})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateAutomationRule: %v", err)
			}
			if got.SegmentName != "ICP" || got.PlaybookName != "Nurture" {
				t.Errorf("names = %q / %q", got.SegmentName, got.PlaybookName)
			}
			c := got.EnrollmentCriteria
			if c.FitScoreThreshold != 70 || c.EngagementScoreThreshold == nil || *c.EngagementScoreThreshold != 30 || !c.RequireHumanApproval {
				t.Errorf("criteria did not round-trip: %+v", c)
			}
		})
	}

	var n int
	if err := f.db.Get(&n, `SELECT count(*) FROM automation_rules`); err != nil || n != 1 {
		t.Fatalf("automation_rules = %d (%v), want only the valid rule stored", n, err)
	}
}

func TestAutomationRuleLifecycle(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	svc := f.automationService()
	seg := f.seedSegment(t)
	pb := f.seedPlaybook(t, twoStages())

	rule, err := svc.CreateAutomationRule(ctx, &v1schema.AutomationRuleRequest{
		Name: "r", SegmentID: seg.ID, PlaybookID: pb.PlaybookID, IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateAutomationRule: %v", err)
	}
	// A rule whose playbook was deleted is dropped from the list, not an error.
	orphan := f.seedPlaybook(t, twoStages())
	orphanRule, err := svc.CreateAutomationRule(ctx, &v1schema.AutomationRuleRequest{
		Name: "orphan", SegmentID: seg.ID, PlaybookID: orphan.PlaybookID, IsActive: true,
	})
	if err != nil {
		t.Fatalf("CreateAutomationRule: %v", err)
	}
	if err := f.playbooks.Delete(ctx, orphan.PlaybookID); err != nil {
		t.Fatalf("delete playbook: %v", err)
	}
	// So is one whose segment disappeared.
	f.db.MustExec(`INSERT INTO automation_rules (name, segment_id, playbook_id) VALUES ('lost', $1, $2)`, uuid.New(), pb.PlaybookID)

	list, err := svc.ListAutomationRules(ctx)
	if err != nil {
		t.Fatalf("ListAutomationRules: %v", err)
	}
	if len(list) != 1 || list[0].AutomationRuleID != rule.AutomationRuleID {
		t.Fatalf("ListAutomationRules = %d rules, want only the intact one", len(list))
	}

	if _, err := svc.GetAutomationRule(ctx, orphanRule.AutomationRuleID); err == nil || !strings.Contains(err.Error(), "playbook not found") {
		t.Errorf("GetAutomationRule(orphan) err = %v", err)
	}
	if _, err := svc.GetAutomationRule(ctx, uuid.New()); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("GetAutomationRule(unknown) err = %v", err)
	}

	if err := svc.ToggleAutomationRule(ctx, rule.AutomationRuleID, false); err != nil {
		t.Fatalf("ToggleAutomationRule: %v", err)
	}
	got, err := svc.GetAutomationRule(ctx, rule.AutomationRuleID)
	if err != nil || got.IsActive {
		t.Fatalf("after toggle off: %+v, %v; want inactive", got, err)
	}
	active, _ := f.rules.ListActive(ctx)
	for _, r := range active {
		if r.AutomationRuleID == rule.AutomationRuleID {
			t.Errorf("toggled-off rule still listed as active")
		}
	}

	if err := svc.DeleteAutomationRule(ctx, rule.AutomationRuleID); err != nil {
		t.Fatalf("DeleteAutomationRule: %v", err)
	}
	if _, err := svc.GetAutomationRule(ctx, rule.AutomationRuleID); err == nil {
		t.Errorf("deleted rule still readable")
	}
}

// A lost segment on a single rule is reported, not skipped.
func TestGetAutomationRuleMissingSegment(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	pb := f.seedPlaybook(t, twoStages())
	var id uuid.UUID
	if err := f.db.Get(&id, `INSERT INTO automation_rules (name, segment_id, playbook_id) VALUES ('lost', $1, $2) RETURNING automation_rule_id`, uuid.New(), pb.PlaybookID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.automationService().GetAutomationRule(ctx, id); err == nil || !strings.Contains(err.Error(), "segment not found") {
		t.Errorf("err = %v, want segment not found", err)
	}
}
