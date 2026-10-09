package playbook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	playbookDomain "github.com/rockship/cosmo-agents-go/internal/domain/playbook"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	playbookRepo "github.com/rockship/cosmo-agents-go/internal/repository/playbook"
	segRepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
)

// playbookTables are the four tables of migration 000035 without their
// foreign keys: contacts, segmentations and users come from GORM models here,
// and the services never rely on a cascade. The UNIQUE(contact_id,
// playbook_id) constraint is kept because duplicate enrollment is behaviour
// under test.
const playbookTables = `
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
CREATE TABLE enrollment_approval_requests (
	request_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	contact_id UUID NOT NULL,
	playbook_id UUID NOT NULL,
	automation_rule_id UUID NOT NULL,
	reason TEXT NOT NULL,
	fit_score INTEGER NOT NULL,
	engagement_score INTEGER NOT NULL,
	status VARCHAR(50) NOT NULL DEFAULT 'pending',
	reviewed_by UUID,
	reviewed_at TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT NOW()
);`

// fixture holds the repositories every service in this package is built from,
// all backed by one fresh schema.
type fixture struct {
	gdb         *gorm.DB
	db          *sqlx.DB
	playbooks   *playbookRepo.Repository
	rules       *playbookRepo.AutomationRuleRepository
	enrollments *playbookRepo.EnrollmentRepository
	approvals   *playbookRepo.ApprovalRequestRepository
	contacts    *contactRepo.ContactRepository
	segments    *segRepo.SegmentationRepository
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	gdb, err := pgtest.Open(t, &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(&domain.Contact{}, &domain.Segmentation{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	db := sqlx.NewDb(sqlDB, "postgres")
	db.MustExec(playbookTables)
	return &fixture{
		gdb:         gdb,
		db:          db,
		playbooks:   playbookRepo.NewRepository(db),
		rules:       playbookRepo.NewAutomationRuleRepository(db),
		enrollments: playbookRepo.NewEnrollmentRepository(db),
		approvals:   playbookRepo.NewApprovalRequestRepository(db),
		contacts:    contactRepo.NewContactRepository(gdb),
		segments:    segRepo.NewSegmentationRepository(gdb),
	}
}

func (f *fixture) enrollmentService() *EnrollmentService {
	return NewEnrollmentService(f.enrollments, f.approvals, f.playbooks, f.contacts)
}

func (f *fixture) automationService() *AutomationService {
	return NewAutomationService(f.rules, f.playbooks, f.segments, f.enrollments)
}

// twoStages is the smallest config with a stage to advance to.
func twoStages() []v1schema.PlaybookStage {
	return []v1schema.PlaybookStage{
		{ID: "intro", Order: 1, Name: "Intro", Type: v1schema.StageTypeEmail},
		{ID: "follow", Order: 2, Name: "Follow up", Type: v1schema.StageTypeWait},
	}
}

func (f *fixture) seedPlaybook(t *testing.T, stages []v1schema.PlaybookStage) *playbookDomain.Playbook {
	t.Helper()
	cfg, _ := json.Marshal(v1schema.PlaybookConfig{Stages: stages})
	pb := &playbookDomain.Playbook{
		Name:         "Nurture",
		PlaybookType: "nurture",
		Config:       base.JSONB(cfg),
		Performance:  base.JSONB(`{}`),
		IsActive:     true,
	}
	if err := f.playbooks.Create(context.Background(), pb); err != nil {
		t.Fatalf("seed playbook: %v", err)
	}
	return pb
}

func (f *fixture) seedContact(t *testing.T, profile string) *domain.Contact {
	t.Helper()
	c := &domain.Contact{
		UserID:   uuid.New(),
		SourceID: uuid.NewString(),
		Source:   "manual",
		Name:     "Ada Lovelace",
		Company:  "Analytical",
		JobTitle: "CTO",
		Profile:  base.JSONB(profile),
	}
	if err := f.gdb.Create(c).Error; err != nil {
		t.Fatalf("seed contact: %v", err)
	}
	return c
}

func (f *fixture) seedSegment(t *testing.T) *domain.Segmentation {
	t.Helper()
	s := &domain.Segmentation{UserID: uuid.New(), Name: "ICP"}
	if err := f.gdb.Create(s).Error; err != nil {
		t.Fatalf("seed segment: %v", err)
	}
	return s
}

// seedRequest writes a pending approval request for contact into pb.
func (f *fixture) seedRequest(t *testing.T, contactID, playbookID uuid.UUID) *playbookDomain.EnrollmentApprovalRequest {
	t.Helper()
	req := &playbookDomain.EnrollmentApprovalRequest{
		ContactID:        contactID,
		PlaybookID:       playbookID,
		AutomationRuleID: uuid.New(),
		Reason:           "rule",
		FitScore:         80,
		EngagementScore:  10,
		Status:           "pending",
	}
	if err := f.approvals.Create(context.Background(), req); err != nil {
		t.Fatalf("seed request: %v", err)
	}
	return req
}

func (f *fixture) requestStatus(t *testing.T, id uuid.UUID) string {
	t.Helper()
	var status string
	if err := f.db.Get(&status, `SELECT status FROM enrollment_approval_requests WHERE request_id = $1`, id); err != nil {
		t.Fatalf("read request status: %v", err)
	}
	return status
}

func (f *fixture) countEnrollments(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.db.Get(&n, `SELECT count(*) FROM contact_enrollments`); err != nil {
		t.Fatalf("count enrollments: %v", err)
	}
	return n
}
