package outreach

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	contactdomain "github.com/rockship/cosmo-agents-go/internal/domain/contact"
	outreachdomain "github.com/rockship/cosmo-agents-go/internal/domain/outreach"
	contactrepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	outreachrepo "github.com/rockship/cosmo-agents-go/internal/repository/outreach"
	outreachservice "github.com/rockship/cosmo-agents-go/internal/service/outreach"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

const defaultBatchSize = 200

// RecalculateWorker recalculates outreach next_step for contacts periodically.
// This worker runs on a schedule to update contacts whose next_step may have changed
// due to time passing (e.g., WAIT -> FOLLOW_UP_1 after configured interval).
type RecalculateWorker struct {
	db              *gorm.DB
	contactRepo     *contactrepo.ContactRepository
	interactionRepo *outreachrepo.InteractionLogRepository
	meetingRepo     *outreachrepo.MeetingRepository
	stateRepo       *outreachrepo.OutreachStateRepository
	feedbackRepo    *outreachrepo.FeedbackRepository
	outreachConfig  *outreachservice.Config
}

// NewRecalculateWorker creates a new outreach recalculate worker
func NewRecalculateWorker(
	db *gorm.DB,
	contactRepo *contactrepo.ContactRepository,
	interactionRepo *outreachrepo.InteractionLogRepository,
	meetingRepo *outreachrepo.MeetingRepository,
	stateRepo *outreachrepo.OutreachStateRepository,
	feedbackRepo *outreachrepo.FeedbackRepository,
	config *outreachservice.Config,
) *RecalculateWorker {
	cfg := outreachservice.DefaultConfig()
	if config != nil {
		cfg = *config
	}

	return &RecalculateWorker{
		db:              db,
		contactRepo:     contactRepo,
		interactionRepo: interactionRepo,
		meetingRepo:     meetingRepo,
		stateRepo:       stateRepo,
		feedbackRepo:    feedbackRepo,
		outreachConfig:  &cfg,
	}
}

// HandleRecalculateNextStep recalculates next_step for all contacts in WAIT state.
// This is triggered periodically by the scheduler.
func (w *RecalculateWorker) HandleRecalculateNextStep(ctx context.Context, _ *asynq.Task) error {
	logger.Logger.Info().Msg("outreach: recalculate next_step started")

	// Get all distinct user IDs from contacts
	userIDs, err := w.listActiveUsers(ctx)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("outreach: failed to list users")
		return err
	}

	logger.Logger.Info().
		Int("users", len(userIDs)).
		Msg("outreach: processing users for next_step recalculation")

	totalUpdated := 0
	for _, userID := range userIDs {
		updated, err := w.recalculateForUser(ctx, userID)
		if err != nil {
			logger.Logger.Error().
				Err(err).
				Str("user_id", userID.String()).
				Msg("outreach: recalculation failed for user")
			continue
		}
		totalUpdated += updated
	}

	logger.Logger.Info().
		Int("total_updated", totalUpdated).
		Msg("outreach: recalculate next_step finished")

	return nil
}

// recalculateForUser recalculates next_step for all contacts of a user that are in WAIT state
func (w *RecalculateWorker) recalculateForUser(ctx context.Context, userID uuid.UUID) (int, error) {
	// Create outreach service for state determination
	outreachSvc := outreachservice.NewService(
		w.interactionRepo,
		w.stateRepo,
		w.meetingRepo,
		w.feedbackRepo,
		w.contactRepo,
		w.outreachConfig,
	)

	// Find contacts with next_step = WAIT (they need time-based recalculation)
	contacts, err := w.findWaitingContacts(ctx, userID)
	if err != nil {
		return 0, err
	}

	if len(contacts) == 0 {
		logger.Logger.Debug().
			Str("user_id", userID.String()).
			Msg("outreach: no waiting contacts to recalculate")
		return 0, nil
	}

	logger.Logger.Debug().
		Str("user_id", userID.String()).
		Int("contacts", len(contacts)).
		Msg("outreach: recalculating contacts")

	updatedCount := 0
	for _, contact := range contacts {
		// DEBUG: Log the contact being processed
		logger.Logger.Info().
			Str("contact_id", contact.ID.String()).
			Str("contact_user_id", contact.UserID.String()).
			Str("worker_user_id", userID.String()).
			Bool("user_id_match", contact.UserID == userID).
			Msg("outreach: processing contact")

		// Recalculate state
		result, err := outreachSvc.DetermineConversationState(ctx, contact, userID)
		if err != nil {
			logger.Logger.Error().
				Err(err).
				Str("contact_id", contact.ID.String()).
				Msg("outreach: failed to determine state")
			continue
		}

		// DEBUG: Log the calculated state details
		logger.Logger.Info().
			Str("contact_id", contact.ID.String()).
			Str("current_next_step", contact.NextStep).
			Str("calculated_next_step", string(result.NextStep)).
			Str("calculated_state", string(result.State)).
			Int("minutes_since_interaction", result.DaysSinceLastInteraction).
			Int("followup_count", result.FollowupCount).
			Bool("has_last_outgoing", result.LastOutgoing != nil).
			Bool("has_last_incoming", result.LastIncoming != nil).
			Msg("outreach: recalculate debug info")

		// Check if next_step changed
		newNextStep := string(result.NextStep)
		if contact.NextStep != newNextStep {
			// Update contact's next_step
			if err := w.updateContactNextStep(ctx, contact.ID, newNextStep, result); err != nil {
				logger.Logger.Error().
					Err(err).
					Str("contact_id", contact.ID.String()).
					Msg("outreach: failed to update next_step")
				continue
			}

			logger.Logger.Info().
				Str("contact_id", contact.ID.String()).
				Str("old_next_step", contact.NextStep).
				Str("new_next_step", newNextStep).
				Msg("outreach: next_step updated")

			updatedCount++
		}
	}

	return updatedCount, nil
}

// findWaitingContacts finds all contacts with next_step = WAIT for a user
func (w *RecalculateWorker) findWaitingContacts(ctx context.Context, userID uuid.UUID) ([]*contactdomain.Contact, error) {
	var contacts []*contactdomain.Contact

	err := w.db.WithContext(ctx).
		Where("user_id = ? AND next_step = ? AND is_deleted = ?", userID, "WAIT", false).
		Find(&contacts).Error

	if err != nil {
		return nil, err
	}
	return contacts, nil
}

// updateContactNextStep updates the contact's next_step and related fields
func (w *RecalculateWorker) updateContactNextStep(ctx context.Context, contactID uuid.UUID, newNextStep string, result *outreachservice.ConversationStateResult) error {
	updates := map[string]interface{}{
		"next_step":       newNextStep,
		"outreach_stage":  string(result.State),
		"scenario":        string(result.Scenario),
		"followup_count":  result.FollowupCount,
		"updated_at":      time.Now(),
	}

	return w.db.WithContext(ctx).
		Model(&contactdomain.Contact{}).
		Where("id = ?", contactID).
		Updates(updates).Error
}

// listActiveUsers returns all distinct user IDs that have contacts
func (w *RecalculateWorker) listActiveUsers(ctx context.Context) ([]uuid.UUID, error) {
	type row struct {
		UserID uuid.UUID `gorm:"column:user_id"`
	}
	var rows []row

	err := w.db.WithContext(ctx).
		Table("contacts").
		Select("DISTINCT user_id").
		Where("is_deleted = ? AND next_step = ?", false, "WAIT").
		Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	out := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.UserID)
	}
	return out, nil
}

// Also recalculate contacts in NO_REPLY state to check timing transitions
func (w *RecalculateWorker) findContactsNeedingRecalculation(ctx context.Context, userID uuid.UUID) ([]*contactdomain.Contact, error) {
	var contacts []*contactdomain.Contact

	// Find contacts that are:
	// 1. In WAIT state (time may have passed)
	// 2. In NO_REPLY state with WAIT next_step (time-based follow-up transitions)
	err := w.db.WithContext(ctx).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Where("next_step = ? OR outreach_stage = ?", "WAIT", string(outreachdomain.StateNoReply)).
		Find(&contacts).Error

	if err != nil {
		return nil, err
	}
	return contacts, nil
}
