package playbook

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jmoiron/sqlx"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	playbookDomain "github.com/rockship/cosmo-agents-go/internal/domain/playbook"
	agentrepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	contactrepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	playbookRepo "github.com/rockship/cosmo-agents-go/internal/repository/playbook"
	segrepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	queueworker "github.com/rockship/cosmo-agents-go/pkg/worker"
)

// playbookTables are the tables of migration 000035 without their foreign
// keys; contacts, segments, scores and agents come from their GORM models.
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

type fixture struct {
	t     *testing.T
	gdb   *gorm.DB
	db    *sqlx.DB
	repos struct {
		playbooks   *playbookRepo.Repository
		rules       *playbookRepo.AutomationRuleRepository
		enrollments *playbookRepo.EnrollmentRepository
		approvals   *playbookRepo.ApprovalRequestRepository
	}
	userID uuid.UUID
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	gdb, err := pgtest.Open(t, &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := gdb.AutoMigrate(&domain.Contact{}, &domain.Segmentation{}, &domain.SegmentationScore{}, &domain.Agent{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	f := &fixture{t: t, gdb: gdb, db: sqlx.NewDb(sqlDB, "postgres"), userID: uuid.New()}
	f.db.MustExec(playbookTables)
	f.repos.playbooks = playbookRepo.NewRepository(f.db)
	f.repos.rules = playbookRepo.NewAutomationRuleRepository(f.db)
	f.repos.enrollments = playbookRepo.NewEnrollmentRepository(f.db)
	f.repos.approvals = playbookRepo.NewApprovalRequestRepository(f.db)
	return f
}

func (f *fixture) worker(client *queueworker.Client, openAI *ai.OpenAIClient) *Worker {
	return New(
		f.repos.playbooks, f.repos.rules, f.repos.enrollments, f.repos.approvals,
		contactrepo.NewContactRepository(f.gdb),
		segrepo.NewSegmentationRepository(f.gdb),
		segrepo.NewScoreRepository(f.gdb),
		agentrepo.NewAgentRepository(f.gdb),
		client, openAI,
	)
}

func (f *fixture) playbook(stages ...v1schema.PlaybookStage) *playbookDomain.Playbook {
	f.t.Helper()
	cfg, _ := json.Marshal(v1schema.PlaybookConfig{Stages: stages})
	pb := &playbookDomain.Playbook{Name: "P", PlaybookType: "nurture", Config: base.JSONB(cfg), Performance: base.JSONB(`{}`), IsActive: true}
	if err := f.repos.playbooks.Create(context.Background(), pb); err != nil {
		f.t.Fatalf("seed playbook: %v", err)
	}
	return pb
}

func (f *fixture) segment() uuid.UUID {
	f.t.Helper()
	s := &domain.Segmentation{UserID: f.userID, Name: "ICP"}
	if err := f.gdb.Create(s).Error; err != nil {
		f.t.Fatalf("seed segment: %v", err)
	}
	return s.ID
}

// contact seeds a contact whose profile carries the given engagement score.
func (f *fixture) contact(engagement int) *domain.Contact {
	f.t.Helper()
	profile, _ := json.Marshal(map[string]any{"scores": map[string]any{"engagement": engagement}})
	c := &domain.Contact{
		UserID: f.userID, SourceID: uuid.NewString(), Source: "manual",
		Name: "Ada Lovelace", Company: "Analytical", JobTitle: "CTO",
		ContactInformation: "ada@example.com",
		Profile:            base.JSONB(profile),
	}
	if err := f.gdb.Create(c).Error; err != nil {
		f.t.Fatalf("seed contact: %v", err)
	}
	return c
}

func (f *fixture) score(contactID, segmentID uuid.UUID, fit int, passes bool) {
	f.t.Helper()
	s := &domain.SegmentationScore{ContactID: contactID, SegmentationID: segmentID, FitScore: fit, PassesFilters: passes}
	if err := f.gdb.Create(s).Error; err != nil {
		f.t.Fatalf("seed score: %v", err)
	}
	// GORM skips a false bool on insert and the column defaults to true.
	if !passes {
		f.gdb.Model(s).Update("passes_filters", false)
	}
}

func (f *fixture) rule(segmentID, playbookID uuid.UUID, criteria string, active bool) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	if err := f.db.Get(&id, `INSERT INTO automation_rules (name, segment_id, playbook_id, enrollment_criteria, is_active)
		VALUES ('rule', $1, $2, $3, $4) RETURNING automation_rule_id`, segmentID, playbookID, criteria, active); err != nil {
		f.t.Fatalf("seed rule: %v", err)
	}
	return id
}

func (f *fixture) agent(status string) uuid.UUID {
	f.t.Helper()
	a := &domain.Agent{UserID: f.userID, Name: "Rep", Email: uuid.NewString() + "@cosmo.test", Status: domain.AgentStatus(status)}
	if err := f.gdb.Create(a).Error; err != nil {
		f.t.Fatalf("seed agent: %v", err)
	}
	return a.ID
}

// enroll writes an active enrollment at the given stage, enrolled `ago` in the past.
func (f *fixture) enroll(contactID, playbookID uuid.UUID, stageID string, order int, ago time.Duration) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	if err := f.db.Get(&id, `INSERT INTO contact_enrollments
		(contact_id, playbook_id, enrollment_status, current_stage_order, current_stage_id, enrolled_at)
		VALUES ($1, $2, 'active', $3, $4, NOW() - $5::interval) RETURNING enrollment_id`,
		contactID, playbookID, order, stageID, ago.String()); err != nil {
		f.t.Fatalf("seed enrollment: %v", err)
	}
	return id
}

func (f *fixture) enrollment(id uuid.UUID) (*playbookDomain.ContactEnrollment, executionLog) {
	f.t.Helper()
	e, err := f.repos.enrollments.GetByID(context.Background(), id)
	if err != nil || e == nil {
		f.t.Fatalf("read enrollment: %v, %v", e, err)
	}
	return e, parseExecutionLog(e.ExecutionLog)
}

func task(t *testing.T, typ string, payload any) *asynq.Task {
	t.Helper()
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return asynq.NewTask(typ, b)
}

// testRedis returns a queue client on a Redis database of its own, plus an
// inspector over that database, or skips when no Redis is reachable. The
// worker holds the concrete client type, so there is no seam for a fake.
func testRedis(t *testing.T) (*queueworker.Client, *asynq.Inspector) {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6381"
	}
	conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		t.Skipf("no Redis at %s (set TEST_REDIS_ADDR): %v", addr, err)
	}
	_ = conn.Close()
	const db = 9
	client := queueworker.NewClient(queueworker.Config{RedisAddr: addr, RedisDB: db})
	insp := asynq.NewInspector(asynq.RedisClientOpt{Addr: addr, DB: db})
	_, _ = insp.DeleteAllPendingTasks("default")
	t.Cleanup(func() {
		_, _ = insp.DeleteAllPendingTasks("default")
		_ = insp.Close()
		_ = client.Close()
	})
	return client, insp
}
