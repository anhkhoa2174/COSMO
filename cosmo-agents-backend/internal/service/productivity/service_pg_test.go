package productivity

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"gorm.io/gorm"
)

// newReportDB creates the tables TeamReport reads, with only the columns its
// queries touch.
func newReportDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	for _, stmt := range []string{
		`CREATE TABLE users (id UUID PRIMARY KEY, name TEXT, email TEXT, picture TEXT, is_deleted BOOLEAN DEFAULT FALSE)`,
		`CREATE TABLE roles (user_id UUID, organization_id UUID, name TEXT, is_deleted BOOLEAN DEFAULT FALSE)`,
		`CREATE TABLE action_completion_logs (user_id UUID, transition TEXT, date DATE)`,
		`CREATE TABLE meetings (user_id UUID, status TEXT, created_at TIMESTAMPTZ DEFAULT NOW())`,
		`CREATE TABLE outcome_metrics (user_id UUID, period TEXT, total_sent INT, total_replied INT, total_meetings INT, reply_rate_overall DOUBLE PRECISION, computed_at TIMESTAMPTZ)`,
	} {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create table: %v", err)
		}
	}
	return db
}

func exec(t *testing.T, db *gorm.DB, sql string, args ...any) {
	t.Helper()
	if err := db.Exec(sql, args...).Error; err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func addUser(t *testing.T, db *gorm.DB, org uuid.UUID, name, role string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	exec(t, db, `INSERT INTO users (id, name, email) VALUES (?, ?, ?)`, id, name, name+"@acme.test")
	exec(t, db, `INSERT INTO roles (user_id, organization_id, name) VALUES (?, ?, ?)`, id, org, role)
	return id
}

func day(offset int) string {
	return time.Now().AddDate(0, 0, offset).Format("2006-01-02")
}

func byID(r *Report) map[uuid.UUID]Member {
	out := make(map[uuid.UUID]Member, len(r.Members))
	for _, m := range r.Members {
		out[m.UserID] = m
	}
	return out
}

// The scoping rule is the security property of this endpoint: a member must
// never receive a peer's row, and a user from another organisation must never
// appear in anybody's report, admin or not.
func TestTeamReport_Scoping(t *testing.T) {
	db := newReportDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org, other := uuid.New(), uuid.New()
	admin := addUser(t, db, org, "Admin", "admin")
	alice := addUser(t, db, org, "Alice", "member")
	bob := addUser(t, db, org, "Bob", "member")
	outsider := addUser(t, db, other, "Outsider", "admin")

	// A deleted membership and a deleted user are both gone from the team.
	gone := addUser(t, db, org, "Gone", "member")
	exec(t, db, `UPDATE roles SET is_deleted = TRUE WHERE user_id = ?`, gone)
	ghost := addUser(t, db, org, "Ghost", "member")
	exec(t, db, `UPDATE users SET is_deleted = TRUE WHERE id = ?`, ghost)

	// Everyone has activity so that a leak in the aggregate queries would
	// surface as numbers, not just as roster rows.
	for _, u := range []uuid.UUID{admin, alice, bob, outsider, gone, ghost} {
		exec(t, db, `INSERT INTO action_completion_logs (user_id, transition, date) VALUES (?, 'mark_sent', ?)`, u, day(0))
	}

	tests := []struct {
		name    string
		viewer  uuid.UUID
		org     uuid.UUID
		isAdmin bool
		scope   string
		want    []uuid.UUID
	}{
		{"admin sees whole team", admin, org, true, "team", []uuid.UUID{admin, alice, bob}},
		{"member sees only self", alice, org, false, "self", []uuid.UUID{alice}},
		// A member of org A asking about org B is not in its roster, so they
		// get nothing — not a stranger's row.
		{"member of other org sees nothing", alice, other, false, "self", nil},
		// The isAdmin flag is scoped by the caller to one org; even so, an
		// admin view of org B must only list org B's members.
		{"admin of other org sees only that org", outsider, other, true, "team", []uuid.UUID{outsider}},
		// Non-admins asking about a peer's org they belong to still only see
		// themselves even if the viewer is the admin user flagged false.
		{"admin flagged as member sees only self", admin, org, false, "self", []uuid.UUID{admin}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := svc.TeamReport(ctx, tc.org, tc.viewer, tc.isAdmin, Period30d)
			if err != nil {
				t.Fatalf("TeamReport: %v", err)
			}
			if r.Scope != tc.scope {
				t.Errorf("scope = %q, want %q", r.Scope, tc.scope)
			}
			got := byID(r)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d members, want %d: %+v", len(got), len(tc.want), r.Members)
			}
			for _, id := range tc.want {
				m, ok := got[id]
				if !ok {
					t.Fatalf("missing member %s", id)
				}
				if m.IsYou != (id == tc.viewer) {
					t.Errorf("%s IsYou = %v", m.Name, m.IsYou)
				}
				if m.ActionsCompleted != 1 {
					t.Errorf("%s completed = %d, want 1", m.Name, m.ActionsCompleted)
				}
			}
		})
	}
}

func TestTeamReport_CountsAndWindow(t *testing.T) {
	db := newReportDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := uuid.New()
	rep := addUser(t, db, org, "Rep", "member")

	// Two done on the same day (one active day), one done on another day, a
	// skip, an unrelated transition, and old work outside the 7d window.
	for _, row := range []struct {
		tr  string
		day int
	}{
		{"mark_sent", 0}, {"mark_completed", 0}, {"mark_sent", -2},
		{"skip", -1}, {"snooze", 0},
		{"mark_sent", -20}, {"skip", -20},
	} {
		exec(t, db, `INSERT INTO action_completion_logs (user_id, transition, date) VALUES (?, ?, ?)`, rep, row.tr, day(row.day))
	}
	// Cancelled bookings are not work; a booking made long ago is outside the
	// window regardless of when it takes place.
	exec(t, db, `INSERT INTO meetings (user_id, status) VALUES (?, 'scheduled'), (?, 'completed'), (?, 'cancelled')`, rep, rep, rep)
	exec(t, db, `INSERT INTO meetings (user_id, status, created_at) VALUES (?, 'scheduled', NOW() - INTERVAL '20 days')`, rep)

	tests := []struct {
		period                               Period
		completed, skipped, active, meetings int
	}{
		{Period7d, 3, 1, 2, 2},
		{Period30d, 4, 2, 3, 3},
	}
	for _, tc := range tests {
		t.Run(string(tc.period), func(t *testing.T) {
			r, err := svc.TeamReport(ctx, org, rep, false, tc.period)
			if err != nil {
				t.Fatalf("TeamReport: %v", err)
			}
			if r.Days != tc.period.Days() || r.Period != string(tc.period) {
				t.Errorf("period/days = %s/%d", r.Period, r.Days)
			}
			m := r.Members[0]
			if m.ActionsCompleted != tc.completed || m.ActionsSkipped != tc.skipped ||
				m.ActiveDays != tc.active || m.MeetingsBooked != tc.meetings {
				t.Errorf("got completed=%d skipped=%d active=%d meetings=%d, want %d/%d/%d/%d",
					m.ActionsCompleted, m.ActionsSkipped, m.ActiveDays, m.MeetingsBooked,
					tc.completed, tc.skipped, tc.active, tc.meetings)
			}
		})
	}
}

// The rollup is optional per member and per period: a missing row must stay
// nil (not zeroes), a row for the other period must not be borrowed, and
// MetricsPending flips as soon as one member has a row.
func TestTeamReport_Rollup(t *testing.T) {
	db := newReportDB(t)
	svc := NewService(db)
	ctx := context.Background()

	org := uuid.New()
	admin := addUser(t, db, org, "Admin", "admin")
	alice := addUser(t, db, org, "Alice", "member")

	// Before any rollup: pending.
	r, err := svc.TeamReport(ctx, org, admin, true, Period7d)
	if err != nil {
		t.Fatalf("TeamReport: %v", err)
	}
	if !r.MetricsPending {
		t.Fatal("no rollup rows should read as pending")
	}

	computed := time.Now().Add(-time.Hour).UTC().Truncate(time.Second)
	exec(t, db, `INSERT INTO outcome_metrics VALUES (?, '30d', 10, 4, 2, 0.4, ?)`, alice, computed)

	// Only a 30d row exists, so 7d must still be pending.
	r, err = svc.TeamReport(ctx, org, admin, true, Period7d)
	if err != nil {
		t.Fatalf("TeamReport: %v", err)
	}
	if !r.MetricsPending || byID(r)[alice].Metrics != nil {
		t.Fatal("a 30d row must not answer a 7d report")
	}

	r, err = svc.TeamReport(ctx, org, admin, true, Period30d)
	if err != nil {
		t.Fatalf("TeamReport: %v", err)
	}
	if r.MetricsPending {
		t.Fatal("one computed member means the job has run")
	}
	got := byID(r)
	if got[admin].Metrics != nil {
		t.Error("admin has no rollup row, Metrics should be nil")
	}
	m := got[alice].Metrics
	if m == nil || m.EmailsSent != 10 || m.Replies != 4 || m.Meetings != 2 || m.ReplyRate != 0.4 ||
		!m.ComputedAt.Equal(computed) {
		t.Fatalf("alice metrics = %+v", m)
	}
}

func TestTeamReport_EmptyRosterAndRanking(t *testing.T) {
	db := newReportDB(t)
	svc := NewService(db)
	ctx := context.Background()

	// An org with nobody in it is a valid, empty report.
	r, err := svc.TeamReport(ctx, uuid.New(), uuid.New(), true, Period30d)
	if err != nil {
		t.Fatalf("TeamReport: %v", err)
	}
	if len(r.Members) != 0 || !r.MetricsPending {
		t.Fatalf("empty org: %+v", r)
	}

	org := uuid.New()
	a := addUser(t, db, org, "An", "member")
	b := addUser(t, db, org, "Binh", "admin")
	for i := 0; i < 3; i++ {
		exec(t, db, `INSERT INTO action_completion_logs (user_id, transition, date) VALUES (?, 'mark_sent', ?)`, b, day(0))
	}
	exec(t, db, `INSERT INTO action_completion_logs (user_id, transition, date) VALUES (?, 'mark_sent', ?)`, a, day(0))

	r, err = svc.TeamReport(ctx, org, b, true, Period30d)
	if err != nil {
		t.Fatalf("TeamReport: %v", err)
	}
	if r.Members[0].UserID != b || r.Members[0].Role != "admin" || r.Members[0].Email != "Binh@acme.test" {
		t.Fatalf("busiest member should rank first with role/email: %+v", r.Members)
	}
}

func TestTeamReport_DatabaseErrorsSurface(t *testing.T) {
	// Each dependent table missing in turn must fail the report rather than
	// silently return partial numbers.
	for _, table := range []string{"users", "action_completion_logs", "meetings", "outcome_metrics"} {
		t.Run(table, func(t *testing.T) {
			db := newReportDB(t)
			org := uuid.New()
			u := addUser(t, db, org, "Rep", "member")
			exec(t, db, `DROP TABLE `+table+` CASCADE`)
			if _, err := NewService(db).TeamReport(context.Background(), org, u, false, Period7d); err == nil {
				t.Fatalf("expected error with %s missing", table)
			}
		})
	}
}
