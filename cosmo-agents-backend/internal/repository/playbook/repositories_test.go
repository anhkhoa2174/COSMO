package playbook

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/playbook"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
)

// newPlaybookDB creates the playbooks, automation_rules and
// contact_enrollments tables of migration 000035, without foreign keys, plus
// the updated_at trigger so that repository methods returning updated_at are
// checked against the column the database maintains.
func newPlaybookDB(t *testing.T) *sqlx.DB {
	t.Helper()
	gdb, err := pgtest.Open(t, &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	db := sqlx.NewDb(sqlDB, "postgres")
	db.MustExec(`
	CREATE TABLE playbooks (
		playbook_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(200) NOT NULL,
		description TEXT,
		playbook_type VARCHAR(50) NOT NULL,
		config JSONB NOT NULL DEFAULT '{}'::jsonb,
		performance JSONB NOT NULL DEFAULT '{}'::jsonb,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE TABLE automation_rules (
		automation_rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(200) NOT NULL,
		segment_id UUID NOT NULL,
		playbook_id UUID NOT NULL,
		enrollment_criteria JSONB NOT NULL DEFAULT '{}'::jsonb,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE TABLE contact_enrollments (
		enrollment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		contact_id UUID NOT NULL,
		playbook_id UUID NOT NULL,
		automation_rule_id UUID,
		enrollment_status VARCHAR(50) NOT NULL DEFAULT 'pending_approval',
		current_stage_order INTEGER NOT NULL DEFAULT 0,
		current_stage_id VARCHAR(100),
		enrolled_at TIMESTAMP,
		completed_at TIMESTAMP,
		execution_log JSONB NOT NULL DEFAULT '{"stages_completed": [], "events": []}'::jsonb,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
		UNIQUE(contact_id, playbook_id)
	);
	CREATE FUNCTION touch_updated_at() RETURNS trigger AS $$
	BEGIN NEW.updated_at = clock_timestamp(); RETURN NEW; END; $$ LANGUAGE plpgsql;
	CREATE TRIGGER playbooks_touch BEFORE UPDATE ON playbooks FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
	`)
	return db
}

func TestPlaybookRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(newPlaybookDB(t))

	p := &playbook.Playbook{Name: "A", Description: "d", PlaybookType: "nurture", Config: base.JSONB(`{"stages":[]}`), Performance: base.JSONB(`{}`), IsActive: true}
	if err := repo.Create(ctx, p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.PlaybookID == uuid.Nil || p.CreatedAt.IsZero() {
		t.Fatalf("Create did not return generated columns: %+v", p)
	}
	created := p.UpdatedAt

	p.Name = "B"
	if err := repo.Update(ctx, p); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !p.UpdatedAt.After(created) {
		t.Errorf("Update did not return the new updated_at")
	}
	if err := repo.UpdatePerformance(ctx, p.PlaybookID, []byte(`{"total_sent": 3}`)); err != nil {
		t.Fatalf("UpdatePerformance: %v", err)
	}
	got, err := repo.GetByID(ctx, p.PlaybookID)
	if err != nil || got == nil {
		t.Fatalf("GetByID: %v, %v", got, err)
	}
	if got.Name != "B" || string(got.Performance) != `{"total_sent": 3}` {
		t.Errorf("stored = %s / %s", got.Name, got.Performance)
	}

	other := &playbook.Playbook{Name: "C", PlaybookType: "upsell", Config: base.JSONB(`{}`), Performance: base.JSONB(`{}`), IsActive: true}
	if err := repo.Create(ctx, other); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, other.PlaybookID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	list, err := repo.List(ctx)
	if err != nil || len(list) != 1 || list[0].PlaybookID != p.PlaybookID {
		t.Fatalf("List = %d items (%v), want only the active playbook", len(list), err)
	}
	if got, err := repo.GetByID(ctx, other.PlaybookID); err != nil || got != nil {
		t.Errorf("GetByID(deleted) = %v, %v; want nil, nil", got, err)
	}
	if err := repo.Delete(ctx, uuid.New()); err == nil {
		t.Errorf("Delete(unknown) succeeded")
	}
}

func TestAutomationRuleRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewAutomationRuleRepository(newPlaybookDB(t))

	rule := &playbook.AutomationRule{Name: "r", SegmentID: uuid.New(), PlaybookID: uuid.New(), EnrollmentCriteria: base.JSONB(`{"fit_score_threshold":60}`), IsActive: true}
	if err := repo.Create(ctx, rule); err != nil {
		t.Fatalf("Create: %v", err)
	}
	inactive := &playbook.AutomationRule{Name: "off", SegmentID: uuid.New(), PlaybookID: uuid.New(), EnrollmentCriteria: base.JSONB(`{}`)}
	if err := repo.Create(ctx, inactive); err != nil {
		t.Fatal(err)
	}

	all, _ := repo.List(ctx)
	active, _ := repo.ListActive(ctx)
	if len(all) != 2 || len(active) != 1 || active[0].AutomationRuleID != rule.AutomationRuleID {
		t.Fatalf("List = %d, ListActive = %d; want 2 and the active rule", len(all), len(active))
	}

	if err := repo.Toggle(ctx, rule.AutomationRuleID); err != nil {
		t.Fatalf("Toggle: %v", err)
	}
	got, _ := repo.GetByID(ctx, rule.AutomationRuleID)
	if got.IsActive {
		t.Errorf("Toggle left the rule active")
	}
	if err := repo.UpdateStatus(ctx, rule.AutomationRuleID, true); err != nil {
		t.Fatal(err)
	}
	if got, _ = repo.GetByID(ctx, rule.AutomationRuleID); !got.IsActive {
		t.Errorf("UpdateStatus(true) left the rule inactive")
	}

	if err := repo.Delete(ctx, rule.AutomationRuleID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, err := repo.GetByID(ctx, rule.AutomationRuleID); err != nil || got != nil {
		t.Errorf("GetByID(deleted) = %v, %v", got, err)
	}
}

func TestEnrollmentRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewEnrollmentRepository(newPlaybookDB(t))

	enrolledAt := time.Now().Add(-72 * time.Hour).Truncate(time.Second)
	e := &playbook.ContactEnrollment{
		ContactID: uuid.New(), PlaybookID: uuid.New(), EnrollmentStatus: "active",
		CurrentStageOrder: 1, CurrentStageID: "a", EnrolledAt: &enrolledAt,
		ExecutionLog: base.JSONB(`{"stages_completed":[],"events":[]}`),
	}
	if err := repo.Create(ctx, e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, e.EnrollmentID)
	if err != nil || got == nil {
		t.Fatalf("GetByID: %v, %v", got, err)
	}
	// The worker measures a first stage's wait from enrolled_at; Create used to
	// drop it, so every enrollment read back with enrolled_at NULL.
	if got.EnrolledAt == nil || !got.EnrolledAt.Equal(enrolledAt) {
		t.Errorf("enrolled_at = %v, want %v", got.EnrolledAt, enrolledAt)
	}

	dup := *e
	if err := repo.Create(ctx, &dup); err == nil {
		t.Errorf("duplicate contact/playbook enrollment accepted")
	}

	if err := repo.UpdateStage(ctx, e.EnrollmentID, 2, "b"); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateExecutionLog(ctx, e.EnrollmentID, []byte(`{"stages_completed":[{"stage_id":"a"}],"events":[]}`)); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateStatus(ctx, e.EnrollmentID, "completed"); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateCompletedAt(ctx, e.EnrollmentID); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.GetByContactAndPlaybook(ctx, e.ContactID, e.PlaybookID)
	if got.CurrentStageID != "b" || got.CurrentStageOrder != 2 || got.EnrollmentStatus != "completed" || got.CompletedAt == nil {
		t.Errorf("after updates: %+v", got)
	}

	active, _ := repo.ListByStatus(ctx, "active")
	done, _ := repo.ListByStatus(ctx, "completed")
	if len(active) != 0 || len(done) != 1 {
		t.Errorf("ListByStatus active=%d completed=%d, want 0 and 1", len(active), len(done))
	}
	if got, err := repo.GetByContactAndPlaybook(ctx, uuid.New(), e.PlaybookID); err != nil || got != nil {
		t.Errorf("GetByContactAndPlaybook(unknown) = %v, %v", got, err)
	}
	if got, err := repo.GetByID(ctx, uuid.New()); err != nil || got != nil {
		t.Errorf("GetByID(unknown) = %v, %v", got, err)
	}
}

func TestApprovalRequestDecisions(t *testing.T) {
	ctx := context.Background()
	repo, _ := newApprovalRepo(t)

	req := &playbook.EnrollmentApprovalRequest{ContactID: uuid.New(), PlaybookID: uuid.New(), AutomationRuleID: uuid.New(), Reason: "r", FitScore: 70, Status: "pending"}
	if err := repo.Create(ctx, req); err != nil {
		t.Fatalf("Create: %v", err)
	}
	other := &playbook.EnrollmentApprovalRequest{ContactID: uuid.New(), PlaybookID: uuid.New(), AutomationRuleID: uuid.New(), Reason: "r", Status: "pending"}
	if err := repo.Create(ctx, other); err != nil {
		t.Fatal(err)
	}

	reviewer := uuid.New()
	if err := repo.Approve(ctx, req.RequestID, reviewer); err != nil {
		t.Fatal(err)
	}
	if err := repo.Reject(ctx, other.RequestID, reviewer); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetByID(ctx, req.RequestID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != "approved" || got.ReviewedBy == nil || *got.ReviewedBy != reviewer || got.ReviewedAt == nil {
		t.Errorf("approved request = %+v", got)
	}
	if got, _ := repo.GetByID(ctx, other.RequestID); got.Status != "rejected" {
		t.Errorf("rejected request status = %s", got.Status)
	}
	if pending, _ := repo.ListPending(ctx); len(pending) != 0 {
		t.Errorf("decided requests still pending: %d", len(pending))
	}
	if _, err := repo.GetByID(ctx, uuid.New()); err == nil {
		t.Errorf("GetByID(unknown) returned no error")
	}
}
