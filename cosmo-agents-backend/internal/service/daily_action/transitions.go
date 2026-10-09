package daily_action

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	v1 "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// TransitionResult holds the outcome of an action state transition.
type TransitionResult struct {
	Action           *domain.DailyAction
	ContactChange    *v1.ContactStateChange
	CompletionLog    *domain.ActionCompletionLog
}

// ExecuteTransition dispatches to the appropriate transition handler.
func (s *Service) ExecuteTransition(
	ctx context.Context,
	action *domain.DailyAction,
	transition domain.Transition,
	content *string,
	channel *string,
	skipReason *string,
	snoozeUntil *time.Time,
	feedbackAction *string,
) (*TransitionResult, error) {
	if !domain.IsValidTransition(action.Status, transition) {
		logger.Logger.Warn().
			Str("action_id", action.ID.String()).
			Str("current_status", string(action.Status)).
			Str("transition", string(transition)).
			Msg("Invalid transition attempted")
		return nil, fmt.Errorf("invalid transition %s from status %s", transition, action.Status)
	}

	logger.Logger.Info().
		Str("action_id", action.ID.String()).
		Str("user_id", action.UserID.String()).
		Str("transition", string(transition)).
		Str("from_status", string(action.Status)).
		Msg("Executing action transition")

	// Record transition metric
	TransitionsTotal.WithLabelValues(string(transition)).Inc()

	switch transition {
	case domain.TransitionMarkSent:
		return s.executeMarkSent(ctx, action, content, channel, feedbackAction)
	case domain.TransitionSkip:
		return s.executeSkip(ctx, action, skipReason)
	case domain.TransitionSnooze:
		return s.executeSnooze(ctx, action, nil)
	case domain.TransitionSnoozeCustom:
		return s.executeSnooze(ctx, action, snoozeUntil)
	case domain.TransitionMarkCompleted:
		return s.executeMarkCompleted(ctx, action)
	case domain.TransitionReopen:
		return s.executeReopen(ctx, action)
	default:
		return nil, fmt.Errorf("unsupported transition: %s", transition)
	}
}

// executeMarkSent handles the mark_sent transition: log interaction, advance contact state.
func (s *Service) executeMarkSent(
	ctx context.Context,
	action *domain.DailyAction,
	content *string,
	channel *string,
	feedbackAction *string,
) (*TransitionResult, error) {
	// Update action status to completed
	if err := s.actionRepo.UpdateStatus(ctx, action.ID, domain.ActionStatusCompleted, nil); err != nil {
		return nil, fmt.Errorf("update action status: %w", err)
	}
	action.Status = domain.ActionStatusCompleted

	// Log interaction via outreach service
	var contactChange *v1.ContactStateChange
	event := "sent"
	contentStr := ""
	channelStr := "LinkedIn"
	if content != nil {
		contentStr = *content
	}
	if channel != nil {
		channelStr = *channel
	}

	result, err := s.outreachSvc.UpdateOutreachForHandler(ctx, action.UserID, action.ContactID, event, contentStr, channelStr, "")
	if err != nil {
		logger.Error(err).Str("contact_id", action.ContactID.String()).Msg("failed to update outreach state")
	} else if result != nil {
		contactChange = &v1.ContactStateChange{
			PreviousState: string(result.PreviousState),
			NewState:      string(result.NewState),
			NewNextStep:   string(result.NextStep),
		}
	}

	// Create completion log
	log := s.createCompletionLog(action, string(domain.TransitionMarkSent), content, channel, nil, feedbackAction)
	if _, err := s.completionLogRepo.Create(ctx, log); err != nil {
		logger.Error(err).Msg("failed to create completion log")
	}

	return &TransitionResult{
		Action:        action,
		ContactChange: contactChange,
		CompletionLog: log,
	}, nil
}

// executeSkip handles the skip transition.
func (s *Service) executeSkip(ctx context.Context, action *domain.DailyAction, reason *string) (*TransitionResult, error) {
	if err := s.actionRepo.UpdateStatus(ctx, action.ID, domain.ActionStatusSkipped, nil); err != nil {
		return nil, fmt.Errorf("update action status: %w", err)
	}
	action.Status = domain.ActionStatusSkipped

	log := s.createCompletionLog(action, string(domain.TransitionSkip), nil, nil, reason, nil)
	if _, err := s.completionLogRepo.Create(ctx, log); err != nil {
		logger.Error(err).Msg("failed to create completion log")
	}

	return &TransitionResult{Action: action, CompletionLog: log}, nil
}

// executeSnooze handles snooze and snooze_custom transitions.
func (s *Service) executeSnooze(ctx context.Context, action *domain.DailyAction, until *time.Time) (*TransitionResult, error) {
	// Default snooze: 5pm today (or tomorrow if past 5pm)
	if until == nil {
		now := time.Now()
		snoozeTime := time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, now.Location())
		if now.After(snoozeTime) {
			snoozeTime = snoozeTime.Add(24 * time.Hour)
		}
		until = &snoozeTime
	}

	// Create snooze record
	snooze := &domain.ActionSnooze{
		UserID:      action.UserID,
		ActionID:    action.ID,
		SnoozeUntil: *until,
	}
	if _, err := s.snoozeRepo.Create(ctx, snooze); err != nil {
		return nil, fmt.Errorf("create snooze record: %w", err)
	}

	// Update action status
	if err := s.actionRepo.UpdateStatus(ctx, action.ID, domain.ActionStatusSnoozed, map[string]interface{}{
		"snooze_until": until,
	}); err != nil {
		return nil, fmt.Errorf("update action status: %w", err)
	}
	action.Status = domain.ActionStatusSnoozed
	action.SnoozeUntil = until

	log := s.createCompletionLog(action, string(domain.TransitionSnooze), nil, nil, nil, nil)
	if _, err := s.completionLogRepo.Create(ctx, log); err != nil {
		logger.Error(err).Msg("failed to create completion log")
	}

	return &TransitionResult{Action: action, CompletionLog: log}, nil
}

// executeMarkCompleted handles the mark_completed transition.
func (s *Service) executeMarkCompleted(ctx context.Context, action *domain.DailyAction) (*TransitionResult, error) {
	if err := s.actionRepo.UpdateStatus(ctx, action.ID, domain.ActionStatusCompleted, nil); err != nil {
		return nil, fmt.Errorf("update action status: %w", err)
	}
	action.Status = domain.ActionStatusCompleted

	log := s.createCompletionLog(action, string(domain.TransitionMarkCompleted), nil, nil, nil, nil)
	if _, err := s.completionLogRepo.Create(ctx, log); err != nil {
		logger.Error(err).Msg("failed to create completion log")
	}

	return &TransitionResult{Action: action, CompletionLog: log}, nil
}

// executeReopen handles the reopen transition.
func (s *Service) executeReopen(ctx context.Context, action *domain.DailyAction) (*TransitionResult, error) {
	// Clear snooze if applicable
	if action.Status == domain.ActionStatusSnoozed {
		if err := s.snoozeRepo.ClearByActionID(ctx, action.ID); err != nil {
			logger.Error(err).Msg("failed to clear snooze")
		}
	}

	if err := s.actionRepo.UpdateStatus(ctx, action.ID, domain.ActionStatusSuggested, map[string]interface{}{
		"snooze_until": nil,
	}); err != nil {
		return nil, fmt.Errorf("update action status: %w", err)
	}
	action.Status = domain.ActionStatusSuggested
	action.SnoozeUntil = nil

	log := s.createCompletionLog(action, string(domain.TransitionReopen), nil, nil, nil, nil)
	if _, err := s.completionLogRepo.Create(ctx, log); err != nil {
		logger.Error(err).Msg("failed to create completion log")
	}

	return &TransitionResult{Action: action, CompletionLog: log}, nil
}

// createCompletionLog builds an ActionCompletionLog entry.
func (s *Service) createCompletionLog(
	action *domain.DailyAction,
	transition string,
	content *string,
	channel *string,
	skipReason *string,
	feedback *string,
) *domain.ActionCompletionLog {
	// Extract contact name from snapshot
	var contactName string
	var snapshot domain.ContactSnapshot
	if err := action.ContactSnapshot.Unmarshal(&snapshot); err == nil {
		contactName = snapshot.Name
	}

	return &domain.ActionCompletionLog{
		UserID:      action.UserID,
		ActionID:    action.ID,
		ActionType:  action.Type,
		Transition:  transition,
		ContactID:   action.ContactID,
		ContactName: contactName,
		Content:     content,
		Channel:     channel,
		SkipReason:  skipReason,
		Feedback:    feedback,
		Date:        time.Now().Format("2006-01-02"),
	}
}

// FindActionByIDAndUser finds an action by ID scoped to user. Returns nil if not found.
func (s *Service) FindActionByIDAndUser(ctx context.Context, userID, actionID uuid.UUID) (*domain.DailyAction, error) {
	return s.actionRepo.FindByIDAndUserID(ctx, userID, actionID)
}
