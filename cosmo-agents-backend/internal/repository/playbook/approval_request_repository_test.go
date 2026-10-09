package playbook

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"gorm.io/gorm"
)

func newApprovalRepo(t *testing.T) (*ApprovalRequestRepository, *sqlx.DB) {
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
	// The columns of migration 000035, without the foreign keys: the query
	// under test reads this one table.
	db.MustExec(`CREATE TABLE enrollment_approval_requests (
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
	)`)
	return NewApprovalRequestRepository(db), db
}

func TestGetLatestByContactRule(t *testing.T) {
	ctx := context.Background()
	repo, db := newApprovalRepo(t)
	contact, pb, rule := uuid.New(), uuid.New(), uuid.New()

	insert := func(contactID, ruleID uuid.UUID, status string, fit int, age string) {
		db.MustExec(`INSERT INTO enrollment_approval_requests
			(contact_id, playbook_id, automation_rule_id, reason, fit_score, engagement_score, status, created_at)
			VALUES ($1, $2, $3, 'r', $4, 0, $5, NOW() - $6::interval)`,
			contactID, pb, ruleID, fit, status, age)
	}

	got, err := repo.GetLatestByContactRule(ctx, contact, pb, rule)
	if err != nil || got != nil {
		t.Fatalf("no requests: got %v, %v; want nil, nil", got, err)
	}

	insert(contact, rule, "pending", 60, "2 days")
	insert(contact, rule, "rejected", 70, "1 hour")
	// Newer rows for another contact and another rule must not be picked up.
	insert(uuid.New(), rule, "pending", 99, "1 minute")
	insert(contact, uuid.New(), "pending", 99, "1 minute")

	got, err = repo.GetLatestByContactRule(ctx, contact, pb, rule)
	if err != nil {
		t.Fatalf("GetLatestByContactRule: %v", err)
	}
	if got == nil || got.Status != "rejected" || got.FitScore != 70 {
		t.Fatalf("got %+v, want the rejected request at fit 70", got)
	}
}
