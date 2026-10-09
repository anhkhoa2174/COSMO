package daily_action

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	"gorm.io/gorm"
)

// OutcomeMetricsWorker computes and stores aggregated outcome metrics daily.
type OutcomeMetricsWorker struct {
	db *gorm.DB
}

// NewOutcomeMetricsWorker creates a new outcome metrics worker.
func NewOutcomeMetricsWorker(db *gorm.DB) *OutcomeMetricsWorker {
	return &OutcomeMetricsWorker{db: db}
}

// HandleComputeMetrics computes outcome metrics for all active users.
func (w *OutcomeMetricsWorker) HandleComputeMetrics(ctx context.Context, _ *asynq.Task) error {
	slog.Info("Starting outcome metrics computation")

	// Anyone with recent outreach, not only people who ticked a daily action.
	// Campaign sends never touch action_completion_logs, so keying off that
	// table alone left every campaign-only user without metrics.
	//
	// Anyone who already has a row is recomputed too. A user who went quiet
	// has no recent outreach, and skipping them left last month's figures in
	// place: the dashboard went on showing a 30-day reply rate for a window in
	// which nothing was sent. Recomputing brings the row down to zero.
	since30 := time.Now().AddDate(0, 0, -30)
	var userIDs []uuid.UUID
	err := w.db.WithContext(ctx).Raw(`
		SELECT DISTINCT user_id FROM (
			SELECT user_id FROM action_completion_logs WHERE created_at > ?
			UNION
			SELECT user_id FROM emails WHERE created_at > ?
			UNION
			SELECT user_id FROM conversations WHERE created_at > ?
			UNION
			SELECT user_id FROM outcome_metrics
		) AS active_users
		WHERE user_id IS NOT NULL
	`, since30, since30, since30).Scan(&userIDs).Error
	if err != nil {
		slog.Error("Failed to list users for metrics", "error", err)
		return err
	}

	slog.Info("Computing metrics for users", "count", len(userIDs))

	for _, userID := range userIDs {
		for _, period := range []string{"7d", "30d"} {
			if err := w.computeForUser(ctx, userID, period); err != nil {
				slog.Error("Failed to compute metrics", "user_id", userID, "period", period, "error", err)
				continue
			}
		}
	}

	slog.Info("Outcome metrics computation complete", "users", len(userIDs))
	return nil
}

func (w *OutcomeMetricsWorker) computeForUser(ctx context.Context, userID uuid.UUID, period string) error {
	days := 30
	if period == "7d" {
		days = 7
	}
	since := time.Now().AddDate(0, 0, -days)

	// Outgoing mail actually recorded, plus daily actions marked sent by hand.
	// The card is labelled "Emails Sent", so counting only the manual
	// transitions reported 0 while real campaign mail sat in the emails table.
	var totalSent int64
	w.db.WithContext(ctx).Raw(`
		SELECT
			(SELECT COUNT(*) FROM emails e
			   JOIN agents a ON lower(a.email) = lower(e.from_email)
			  WHERE e.user_id = ? AND e.created_at > ?)
			+
			(SELECT COUNT(*) FROM action_completion_logs
			  WHERE user_id = ? AND transition = 'mark_sent' AND created_at > ?)
	`, userID, since, userID, since).Scan(&totalSent)

	// Conversations that received an inbound message. interaction_logs is only
	// written by the outreach flow, so campaign replies were invisible to it.
	var totalReplied int64
	w.db.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT e.conversation_id)
		FROM emails e
		WHERE e.user_id = ?
		  AND e.created_at > ?
		  AND e.conversation_id IS NOT NULL
		  AND NOT EXISTS (
		        SELECT 1 FROM agents a
		         WHERE lower(a.email) = lower(e.from_email)
		      )
	`, userID, since).Scan(&totalReplied)

	// Count meetings booked. Cancelled ones are left out, as the team
	// productivity view leaves them out: a booking that fell through is not a
	// meeting booked, and the two screens must agree for the same person.
	var totalMeetings int64
	w.db.WithContext(ctx).
		Table("meetings").
		Where("user_id = ? AND created_at > ? AND status <> ?", userID, since, "cancelled").
		Count(&totalMeetings)

	// Threads we wrote to in the window: the denominator of the reply rate.
	// Dividing replied threads by every email sent counted each follow-up
	// as another chance to reply, so a prospect who answered the fourth email
	// read as a 25% reply rate.
	// The numerator is those same threads that also received an inbound
	// message in the window, so the rate stays within 0..1.
	var rate struct {
		Emailed int64
		Replied int64
	}
	w.db.WithContext(ctx).Raw(`
		WITH emailed AS (
			SELECT DISTINCT e.conversation_id
			FROM emails e
			JOIN agents a ON lower(a.email) = lower(e.from_email)
			WHERE e.user_id = ? AND e.created_at > ? AND e.conversation_id IS NOT NULL
		)
		SELECT
			(SELECT COUNT(*) FROM emailed) AS emailed,
			(SELECT COUNT(*) FROM emailed t WHERE EXISTS (
				SELECT 1 FROM emails r
				WHERE r.conversation_id = t.conversation_id
				  AND r.user_id = ? AND r.created_at > ?
				  AND NOT EXISTS (SELECT 1 FROM agents a WHERE lower(a.email) = lower(r.from_email))
			)) AS replied
	`, userID, since, userID, since).Scan(&rate)

	// Reply rate by channel
	type channelRate struct {
		Channel string `gorm:"column:channel"`
		Sent    int64  `gorm:"column:sent"`
		Replied int64  `gorm:"column:replied"`
	}
	var channelRates []channelRate
	w.db.WithContext(ctx).Raw(`
		SELECT acl.channel,
			COUNT(*) as sent,
			COUNT(DISTINCT CASE WHEN il_in.id IS NOT NULL THEN acl.contact_id END) as replied
		FROM action_completion_logs acl
		LEFT JOIN interaction_logs il_in ON il_in.contact_id = acl.contact_id
			AND il_in.direction = 'incoming'
			AND il_in.timestamp > acl.created_at
		WHERE acl.user_id = ?
			AND acl.transition = 'mark_sent'
			AND acl.created_at > ?
			AND acl.channel IS NOT NULL
			AND acl.channel != ''
		GROUP BY acl.channel
	`, userID, since).Scan(&channelRates)

	replyByChannel := make(map[string]float64)
	for _, cr := range channelRates {
		if cr.Sent > 0 {
			replyByChannel[cr.Channel] = float64(cr.Replied) / float64(cr.Sent)
		}
	}

	// Reply rate by industry (join contacts)
	type industryRate struct {
		Industry string `gorm:"column:industry"`
		Sent     int64  `gorm:"column:sent"`
		Replied  int64  `gorm:"column:replied"`
	}
	var industryRates []industryRate
	w.db.WithContext(ctx).Raw(`
		SELECT c.industry,
			COUNT(*) as sent,
			COUNT(DISTINCT CASE WHEN il_in.id IS NOT NULL THEN acl.contact_id END) as replied
		FROM action_completion_logs acl
		JOIN contacts c ON c.id = acl.contact_id
		LEFT JOIN interaction_logs il_in ON il_in.contact_id = acl.contact_id
			AND il_in.direction = 'incoming'
			AND il_in.timestamp > acl.created_at
		WHERE acl.user_id = ?
			AND acl.transition = 'mark_sent'
			AND acl.created_at > ?
			AND c.industry != ''
		GROUP BY c.industry
	`, userID, since).Scan(&industryRates)

	replyByIndustry := make(map[string]float64)
	for _, ir := range industryRates {
		if ir.Sent > 0 {
			replyByIndustry[ir.Industry] = float64(ir.Replied) / float64(ir.Sent)
		}
	}

	// Overall reply rate: the share of threads we wrote to that got an answer.
	replyRate := float64(0)
	if rate.Emailed > 0 {
		replyRate = float64(rate.Replied) / float64(rate.Emailed)
	}

	// Marshal JSONB fields
	channelJSON, _ := json.Marshal(replyByChannel)
	industryJSON, _ := json.Marshal(replyByIndustry)
	emptyJSON := []byte("{}")

	metrics := domain.OutcomeMetrics{
		UserID:           userID,
		ComputedAt:       time.Now(),
		Period:           period,
		TotalSent:        int(totalSent),
		TotalReplied:     int(totalReplied),
		TotalMeetings:    int(totalMeetings),
		ReplyRateOverall: replyRate,
	}
	_ = metrics.ReplyRateByChannel.Scan(channelJSON)
	_ = metrics.ReplyRateByIndustry.Scan(industryJSON)
	_ = metrics.ReplyRateByStrategy.Scan(emptyJSON)
	_ = metrics.ReplyRateByTimeOfDay.Scan(emptyJSON)
	_ = metrics.TopPerformingStrategies.Scan([]byte("[]"))

	// Upsert
	result := w.db.WithContext(ctx).
		Where("user_id = ? AND period = ?", userID, period).
		First(&domain.OutcomeMetrics{})

	if result.Error == gorm.ErrRecordNotFound {
		metrics.ID = uuid.New()
		return w.db.WithContext(ctx).Create(&metrics).Error
	}

	return w.db.WithContext(ctx).
		Model(&domain.OutcomeMetrics{}).
		Where("user_id = ? AND period = ?", userID, period).
		Updates(map[string]interface{}{
			"computed_at":               time.Now(),
			"total_sent":                metrics.TotalSent,
			"total_replied":             metrics.TotalReplied,
			"total_meetings":            metrics.TotalMeetings,
			"reply_rate_overall":        metrics.ReplyRateOverall,
			"reply_rate_by_channel":     channelJSON,
			"reply_rate_by_industry":    industryJSON,
			"reply_rate_by_strategy":    emptyJSON,
			"reply_rate_by_time_of_day": emptyJSON,
		}).Error
}
