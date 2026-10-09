// Package productivity answers one question: who on the team did what, over
// the last week or month.
//
// It reads two very different sources and keeps them apart on purpose:
//
//   - Live counts come straight from action_completion_logs and meetings, so
//     they are always available and always current.
//   - The nightly rollup in outcome_metrics carries the derived figures
//     (reply rate above all), which are too expensive to recompute per request.
//
// The rollup is allowed to be missing. A member the nightly job has not
// reached yet returns live counts with no Metrics block rather than a row of
// zeroes, because a zero reply rate and an uncomputed reply rate mean opposite
// things to whoever is reading the dashboard.
package productivity

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Period is the window a report covers. These are the only two the nightly
// rollup writes, so they are the only two that can be answered.
type Period string

const (
	Period7d  Period = "7d"
	Period30d Period = "30d"
)

// ParsePeriod maps a query parameter onto a supported window, defaulting to
// 30 days rather than rejecting the request.
func ParsePeriod(s string) Period {
	if s == string(Period7d) {
		return Period7d
	}
	return Period30d
}

// Days is the length of the window, used for the live queries.
func (p Period) Days() int {
	if p == Period7d {
		return 7
	}
	return 30
}

// Metrics is the nightly rollup for one member. Absent when the job has not
// produced a row for them.
type Metrics struct {
	EmailsSent int       `json:"emails_sent"`
	Replies    int       `json:"replies"`
	Meetings   int       `json:"meetings"`
	ReplyRate  float64   `json:"reply_rate"`
	ComputedAt time.Time `json:"computed_at"`
}

// Member is one row of the report.
type Member struct {
	UserID  uuid.UUID `json:"user_id"`
	Name    string    `json:"name"`
	Email   string    `json:"email"`
	Picture string    `json:"picture,omitempty"`
	Role    string    `json:"role"`
	IsYou   bool      `json:"is_you"`

	// Live counts over the window.
	ActionsCompleted int `json:"actions_completed"`
	ActionsSkipped   int `json:"actions_skipped"`
	MeetingsBooked   int `json:"meetings_booked"`
	ActiveDays       int `json:"active_days"`

	Metrics *Metrics `json:"metrics,omitempty"`
}

// Report is what the dashboard renders.
type Report struct {
	// Scope tells the UI which story to tell: "team" for an admin looking at
	// everyone, "self" for a member looking at their own work.
	Scope   string   `json:"scope"`
	Period  string   `json:"period"`
	Days    int      `json:"days"`
	Members []Member `json:"members"`

	// MetricsPending is true when no member has a nightly rollup yet, which is
	// the difference between "the team sent nothing" and "the job has not run".
	MetricsPending bool `json:"metrics_pending"`
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// TeamReport builds the report for one organisation.
//
// An admin sees every member; anyone else sees only themselves. The caller
// decides which by passing isAdmin — this service does not re-derive
// permissions, it only honours the decision.
func (s *Service) TeamReport(
	ctx context.Context,
	orgID, viewerID uuid.UUID,
	isAdmin bool,
	period Period,
) (*Report, error) {
	members, err := s.roster(ctx, orgID, viewerID, isAdmin)
	if err != nil {
		return nil, err
	}

	report := &Report{
		Scope:          scopeFor(isAdmin),
		Period:         string(period),
		Days:           period.Days(),
		Members:        members,
		MetricsPending: true,
	}
	if len(members) == 0 {
		return report, nil
	}

	ids := make([]uuid.UUID, len(members))
	index := make(map[uuid.UUID]int, len(members))
	for i, m := range members {
		ids[i] = m.UserID
		index[m.UserID] = i
	}

	since := time.Now().AddDate(0, 0, -period.Days())

	if err := s.applyActivity(ctx, report.Members, index, ids, since); err != nil {
		return nil, err
	}
	if err := s.applyMeetings(ctx, report.Members, index, ids, since); err != nil {
		return nil, err
	}
	if err := s.applyRollup(ctx, report.Members, index, ids, period); err != nil {
		return nil, err
	}

	report.MetricsPending = metricsPending(report.Members)
	rank(report.Members)

	return report, nil
}

// metricsPending reports whether the nightly rollup has reached nobody in the
// report. The dashboard needs the distinction: an all-zero row because the job
// has not run reads the same as a team that did nothing, and only one of those
// is worth acting on.
func metricsPending(members []Member) bool {
	for i := range members {
		if members[i].Metrics != nil {
			return false
		}
	}
	return true
}

// rank puts the busiest member first, so an admin opening the page sees the
// shape of the team without sorting anything. Ties fall back to name, which
// keeps the order stable between refreshes instead of shuffling on every load.
func rank(members []Member) {
	sort.SliceStable(members, func(a, b int) bool {
		x, y := members[a], members[b]
		if x.ActionsCompleted != y.ActionsCompleted {
			return x.ActionsCompleted > y.ActionsCompleted
		}
		return x.Name < y.Name
	})
}

func scopeFor(isAdmin bool) string {
	if isAdmin {
		return "team"
	}
	return "self"
}

// roster lists the people the viewer is allowed to see.
func (s *Service) roster(
	ctx context.Context,
	orgID, viewerID uuid.UUID,
	isAdmin bool,
) ([]Member, error) {
	type row struct {
		UserID  uuid.UUID
		Name    string
		Email   string
		Picture string
		Role    string
	}

	q := s.db.WithContext(ctx).
		Table("users").
		Select("users.id AS user_id, users.name, users.email, users.picture, roles.name AS role").
		Joins("JOIN roles ON roles.user_id = users.id").
		Where("roles.organization_id = ? AND roles.is_deleted = ? AND users.is_deleted = ?",
			orgID, false, false)

	// A member's report is their own row and nothing else. Filtering here
	// rather than after the aggregate queries also keeps their peers' numbers
	// from being fetched at all.
	if !isAdmin {
		q = q.Where("users.id = ?", viewerID)
	}

	var rows []row
	if err := q.Order("users.name").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load roster: %w", err)
	}

	members := make([]Member, 0, len(rows))
	for _, r := range rows {
		members = append(members, Member{
			UserID:  r.UserID,
			Name:    r.Name,
			Email:   r.Email,
			Picture: r.Picture,
			Role:    r.Role,
			IsYou:   r.UserID == viewerID,
		})
	}
	return members, nil
}

// applyActivity counts what each member actually did with their daily actions.
//
// mark_sent and mark_completed both mean the work was done — one is an email
// that went out, the other a call or note the rep logged — so they are counted
// together. Skips are kept separate: a skip is a decision, not a failure, and
// showing it beside the completed count is what makes a low completion count
// readable.
func (s *Service) applyActivity(
	ctx context.Context,
	members []Member,
	index map[uuid.UUID]int,
	ids []uuid.UUID,
	since time.Time,
) error {
	type row struct {
		UserID     uuid.UUID
		Completed  int
		Skipped    int
		ActiveDays int
	}

	var rows []row
	err := s.db.WithContext(ctx).
		Table("action_completion_logs").
		// `date` is qualified throughout: bare, it collides with the SQL type
		// name in some parser positions.
		Select(`user_id,
			COUNT(*) FILTER (WHERE transition IN ('mark_sent','mark_completed')) AS completed,
			COUNT(*) FILTER (WHERE transition = 'skip') AS skipped,
			COUNT(DISTINCT action_completion_logs.date)
				FILTER (WHERE transition IN ('mark_sent','mark_completed')) AS active_days`).
		Where("user_id IN ? AND action_completion_logs.date >= ?", ids, since.Format("2006-01-02")).
		Group("user_id").
		Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("load activity: %w", err)
	}

	for _, r := range rows {
		if i, ok := index[r.UserID]; ok {
			members[i].ActionsCompleted = r.Completed
			members[i].ActionsSkipped = r.Skipped
			members[i].ActiveDays = r.ActiveDays
		}
	}
	return nil
}

// applyMeetings counts meetings booked inside the window.
//
// It filters on created_at, not on the meeting time: the question is how much
// a rep booked this month, and a meeting booked today for next quarter is
// still this month's work. Cancelled meetings are excluded — a booking that
// fell through is not productivity.
func (s *Service) applyMeetings(
	ctx context.Context,
	members []Member,
	index map[uuid.UUID]int,
	ids []uuid.UUID,
	since time.Time,
) error {
	type row struct {
		UserID uuid.UUID
		Booked int
	}

	var rows []row
	err := s.db.WithContext(ctx).
		Table("meetings").
		Select("user_id, COUNT(*) AS booked").
		Where("user_id IN ? AND created_at >= ? AND status <> ?", ids, since, "cancelled").
		Group("user_id").
		Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("load meetings: %w", err)
	}

	for _, r := range rows {
		if i, ok := index[r.UserID]; ok {
			members[i].MeetingsBooked = r.Booked
		}
	}
	return nil
}

// applyRollup attaches the nightly figures to the members that have them.
func (s *Service) applyRollup(
	ctx context.Context,
	members []Member,
	index map[uuid.UUID]int,
	ids []uuid.UUID,
	period Period,
) error {
	type row struct {
		UserID           uuid.UUID
		TotalSent        int
		TotalReplied     int
		TotalMeetings    int
		ReplyRateOverall float64
		ComputedAt       time.Time
	}

	var rows []row
	err := s.db.WithContext(ctx).
		Table("outcome_metrics").
		Select("user_id, total_sent, total_replied, total_meetings, reply_rate_overall, computed_at").
		Where("user_id IN ? AND period = ?", ids, string(period)).
		Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("load outcome metrics: %w", err)
	}

	for _, r := range rows {
		if i, ok := index[r.UserID]; ok {
			members[i].Metrics = &Metrics{
				EmailsSent: r.TotalSent,
				Replies:    r.TotalReplied,
				Meetings:   r.TotalMeetings,
				ReplyRate:  r.ReplyRateOverall,
				ComputedAt: r.ComputedAt,
			}
		}
	}
	return nil
}
