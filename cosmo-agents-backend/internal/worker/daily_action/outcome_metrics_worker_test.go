package daily_action

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"gorm.io/gorm"
)

// newMetricsDB creates the tables the rollup reads, with only the columns its
// queries touch, plus the real outcome_metrics table from the domain model.
func newMetricsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	for _, stmt := range []string{
		`CREATE TABLE action_completion_logs (user_id UUID, contact_id UUID, transition TEXT, channel TEXT, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE TABLE emails (user_id UUID, conversation_id UUID, from_email TEXT, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE TABLE conversations (user_id UUID, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE TABLE agents (email TEXT)`,
		`CREATE TABLE meetings (user_id UUID, status TEXT DEFAULT 'scheduled', created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE TABLE interaction_logs (id UUID, contact_id UUID, direction TEXT, "timestamp" TIMESTAMPTZ)`,
		`CREATE TABLE contacts (id UUID, industry TEXT)`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
	if err := db.AutoMigrate(&domain.OutcomeMetrics{}); err != nil {
		t.Fatalf("migrate outcome_metrics: %v", err)
	}
	return db
}

func metricsFor(t *testing.T, db *gorm.DB, userID uuid.UUID, period string) domain.OutcomeMetrics {
	t.Helper()
	var m domain.OutcomeMetrics
	if err := db.Where("user_id = ? AND period = ?", userID, period).First(&m).Error; err != nil {
		t.Fatalf("load %s metrics: %v", period, err)
	}
	return m
}

// A user who has gone quiet has no recent outreach, so they are not "active".
// Their row must still be recomputed, or last month's figures stay on the
// dashboard as if they described the current window.
func TestHandleComputeMetrics_RecomputesQuietUsers(t *testing.T) {
	db := newMetricsDB(t)
	w := NewOutcomeMetricsWorker(db)
	ctx := context.Background()

	quiet := uuid.New()
	stale := time.Now().AddDate(0, 0, -20)
	for _, period := range []string{"7d", "30d"} {
		row := domain.OutcomeMetrics{
			UserID: quiet, Period: period, ComputedAt: stale,
			TotalSent: 4, TotalReplied: 3, ReplyRateOverall: 0.75,
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("seed stale row: %v", err)
		}
	}
	// Their last outreach is older than either window.
	db.Exec(`INSERT INTO agents (email) VALUES ('rep@acme.com')`)
	db.Exec(`INSERT INTO emails (user_id, from_email, created_at) VALUES (?, 'rep@acme.com', NOW() - INTERVAL '35 days')`, quiet)

	if err := w.HandleComputeMetrics(ctx, nil); err != nil {
		t.Fatalf("HandleComputeMetrics: %v", err)
	}

	for _, period := range []string{"7d", "30d"} {
		m := metricsFor(t, db, quiet, period)
		if m.TotalSent != 0 || m.TotalReplied != 0 || m.ReplyRateOverall != 0 {
			t.Errorf("%s: got sent=%d replied=%d rate=%.2f, want all zero for a window with no outreach",
				period, m.TotalSent, m.TotalReplied, m.ReplyRateOverall)
		}
		if !m.ComputedAt.After(stale.Add(time.Hour)) {
			t.Errorf("%s: computed_at %v was not refreshed", period, m.ComputedAt)
		}
	}
}

// The ordinary case still works: recent sends and replies are counted.
// Reply rate is the share of threads we wrote to that got an answer.
func TestHandleComputeMetrics_CountsRecentOutreach(t *testing.T) {
	db := newMetricsDB(t)
	w := NewOutcomeMetricsWorker(db)
	ctx := context.Background()

	user := uuid.New()
	threads := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	db.Exec(`INSERT INTO agents (email) VALUES ('rep@acme.com')`)
	for _, conv := range threads {
		db.Exec(`INSERT INTO emails (user_id, conversation_id, from_email, created_at) VALUES (?, ?, 'rep@acme.com', NOW() - INTERVAL '2 days')`, user, conv)
	}
	// One of the four threads gets an answer.
	db.Exec(`INSERT INTO emails (user_id, conversation_id, from_email, created_at) VALUES (?, ?, 'prospect@x.com', NOW() - INTERVAL '1 day')`, user, threads[0])
	// A booked meeting counts; a cancelled one does not.
	db.Exec(`INSERT INTO meetings (user_id, status, created_at) VALUES (?, 'scheduled', NOW() - INTERVAL '1 day'), (?, 'cancelled', NOW() - INTERVAL '1 day')`, user, user)

	if err := w.HandleComputeMetrics(ctx, nil); err != nil {
		t.Fatalf("HandleComputeMetrics: %v", err)
	}

	m := metricsFor(t, db, user, "7d")
	if m.TotalSent != 4 || m.TotalReplied != 1 {
		t.Fatalf("7d: got sent=%d replied=%d, want 4 and 1", m.TotalSent, m.TotalReplied)
	}
	if m.ReplyRateOverall != 0.25 {
		t.Fatalf("7d: reply rate %.2f, want 0.25 (1 of 4 threads)", m.ReplyRateOverall)
	}
	if m.TotalMeetings != 1 {
		t.Fatalf("7d: meetings %d, want 1 (the cancelled one excluded)", m.TotalMeetings)
	}
}

// Follow-ups are more emails in the same thread, not more prospects: one
// prospect emailed four times who answered is a 100% reply rate, not 25%.
func TestHandleComputeMetrics_FollowUpsDoNotDiluteReplyRate(t *testing.T) {
	db := newMetricsDB(t)
	w := NewOutcomeMetricsWorker(db)
	ctx := context.Background()

	user, conv := uuid.New(), uuid.New()
	db.Exec(`INSERT INTO agents (email) VALUES ('rep@acme.com')`)
	for i := 0; i < 4; i++ {
		db.Exec(`INSERT INTO emails (user_id, conversation_id, from_email, created_at) VALUES (?, ?, 'rep@acme.com', NOW() - INTERVAL '3 days')`, user, conv)
	}
	db.Exec(`INSERT INTO emails (user_id, conversation_id, from_email, created_at) VALUES (?, ?, 'prospect@x.com', NOW() - INTERVAL '1 day')`, user, conv)

	if err := w.HandleComputeMetrics(ctx, nil); err != nil {
		t.Fatalf("HandleComputeMetrics: %v", err)
	}
	m := metricsFor(t, db, user, "7d")
	if m.TotalSent != 4 {
		t.Fatalf("emails sent %d, want 4", m.TotalSent)
	}
	if m.ReplyRateOverall != 1 {
		t.Fatalf("reply rate %.2f, want 1.00", m.ReplyRateOverall)
	}
}
