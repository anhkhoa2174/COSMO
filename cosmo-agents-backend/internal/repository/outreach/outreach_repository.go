package outreach

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/outreach"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ============================================
// InteractionLog Repository
// ============================================

// InteractionLogRepository handles InteractionLog entity operations
type InteractionLogRepository struct {
	*gormpkg.GormRepository[outreach.InteractionLog]
	db *gorm.DB
}

// NewInteractionLogRepository creates a new interaction log repository
func NewInteractionLogRepository(db *gorm.DB) *InteractionLogRepository {
	return &InteractionLogRepository{
		GormRepository: gormpkg.NewGormRepository[outreach.InteractionLog](db),
		db:             db,
	}
}

// Create creates a new interaction log
func (r *InteractionLogRepository) Create(ctx context.Context, log *outreach.InteractionLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// FindByContactID finds all interaction logs for a contact
func (r *InteractionLogRepository) FindByContactID(ctx context.Context, contactID uuid.UUID, limit int) ([]*outreach.InteractionLog, error) {
	var logs []*outreach.InteractionLog
	query := r.db.WithContext(ctx).
		Where("contact_id = ?", contactID).
		Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// FindByContactIDAndUserID finds interaction logs for a contact belonging to a user
func (r *InteractionLogRepository) FindByContactIDAndUserID(ctx context.Context, contactID, userID uuid.UUID, limit int) ([]*outreach.InteractionLog, error) {
	var logs []*outreach.InteractionLog
	query := r.db.WithContext(ctx).
		Where("contact_id = ? AND user_id = ?", contactID, userID).
		Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// GetLastOutgoing gets the last outgoing message for a contact
func (r *InteractionLogRepository) GetLastOutgoing(ctx context.Context, contactID uuid.UUID) (*outreach.InteractionLog, error) {
	var log outreach.InteractionLog
	err := r.db.WithContext(ctx).
		Where("contact_id = ? AND direction = ?", contactID, outreach.DirectionOutgoing).
		Order("timestamp DESC").
		First(&log).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetLastOutgoingByUserID gets the last outgoing message for a contact by a specific user
func (r *InteractionLogRepository) GetLastOutgoingByUserID(ctx context.Context, contactID, userID uuid.UUID) (*outreach.InteractionLog, error) {
	var log outreach.InteractionLog
	err := r.db.WithContext(ctx).
		Where("contact_id = ? AND user_id = ? AND direction = ?", contactID, userID, outreach.DirectionOutgoing).
		Order("timestamp DESC").
		First(&log).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetLastIncoming gets the last incoming message for a contact
func (r *InteractionLogRepository) GetLastIncoming(ctx context.Context, contactID uuid.UUID) (*outreach.InteractionLog, error) {
	var log outreach.InteractionLog
	err := r.db.WithContext(ctx).
		Where("contact_id = ? AND direction = ?", contactID, outreach.DirectionIncoming).
		Order("timestamp DESC").
		First(&log).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetLastIncomingByUserID gets the last incoming message for a contact by a specific user
func (r *InteractionLogRepository) GetLastIncomingByUserID(ctx context.Context, contactID, userID uuid.UUID) (*outreach.InteractionLog, error) {
	var log outreach.InteractionLog
	err := r.db.WithContext(ctx).
		Where("contact_id = ? AND user_id = ? AND direction = ?", contactID, userID, outreach.DirectionIncoming).
		Order("timestamp DESC").
		First(&log).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetLastInteraction gets the most recent interaction for a contact
func (r *InteractionLogRepository) GetLastInteraction(ctx context.Context, contactID uuid.UUID) (*outreach.InteractionLog, error) {
	var log outreach.InteractionLog
	err := r.db.WithContext(ctx).
		Where("contact_id = ?", contactID).
		Order("timestamp DESC").
		First(&log).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// HasIncomingAfter checks if there's an incoming message after a given timestamp
func (r *InteractionLogRepository) HasIncomingAfter(ctx context.Context, contactID uuid.UUID, after time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&outreach.InteractionLog{}).
		Where("contact_id = ? AND direction = ? AND timestamp > ?", contactID, outreach.DirectionIncoming, after).
		Count(&count).Error

	return count > 0, err
}

// CountOutgoing counts outgoing messages for a contact
func (r *InteractionLogRepository) CountOutgoing(ctx context.Context, contactID uuid.UUID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&outreach.InteractionLog{}).
		Where("contact_id = ? AND direction = ?", contactID, outreach.DirectionOutgoing).
		Count(&count).Error

	return int(count), err
}

// CountOutgoingByUserID counts outgoing messages for a contact by a specific user
func (r *InteractionLogRepository) CountOutgoingByUserID(ctx context.Context, contactID, userID uuid.UUID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&outreach.InteractionLog{}).
		Where("contact_id = ? AND user_id = ? AND direction = ?", contactID, userID, outreach.DirectionOutgoing).
		Count(&count).Error

	return int(count), err
}

// FindNotes finds all internal notes for a contact
func (r *InteractionLogRepository) FindNotes(ctx context.Context, contactID, userID uuid.UUID, limit int) ([]*outreach.InteractionLog, error) {
	var logs []*outreach.InteractionLog
	query := r.db.WithContext(ctx).
		Where("contact_id = ? AND user_id = ? AND direction = ?", contactID, userID, outreach.DirectionInternal).
		Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// FindNoteByID finds a note by ID (without is_deleted check)
func (r *InteractionLogRepository) FindNoteByID(ctx context.Context, noteID uuid.UUID) (*outreach.InteractionLog, error) {
	var log outreach.InteractionLog
	err := r.db.WithContext(ctx).
		Where("id = ?", noteID).
		First(&log).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// UpdateNote updates a note (without is_deleted check)
func (r *InteractionLogRepository) UpdateNote(ctx context.Context, noteID uuid.UUID, content string) error {
	return r.db.WithContext(ctx).
		Model(&outreach.InteractionLog{}).
		Where("id = ?", noteID).
		Update("content", content).Error
}

// DeleteNote permanently deletes a note (hard delete)
func (r *InteractionLogRepository) DeleteNote(ctx context.Context, noteID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", noteID).
		Delete(&outreach.InteractionLog{}).Error
}

// ============================================
// OutreachState Repository
// ============================================

// OutreachStateRepository handles OutreachState entity operations
type OutreachStateRepository struct {
	*gormpkg.GormRepository[outreach.OutreachState]
	db *gorm.DB
}

// NewOutreachStateRepository creates a new outreach state repository
func NewOutreachStateRepository(db *gorm.DB) *OutreachStateRepository {
	return &OutreachStateRepository{
		GormRepository: gormpkg.NewGormRepository[outreach.OutreachState](db),
		db:             db,
	}
}

// FindByContactID finds outreach state for a contact
func (r *OutreachStateRepository) FindByContactID(ctx context.Context, contactID, userID uuid.UUID) (*outreach.OutreachState, error) {
	var state outreach.OutreachState
	err := r.db.WithContext(ctx).
		Where("contact_id = ? AND user_id = ?", contactID, userID).
		First(&state).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// Upsert creates or updates outreach state
func (r *OutreachStateRepository) Upsert(ctx context.Context, state *outreach.OutreachState) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "contact_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"conversation_state", "context_level", "outreach_intent", "scenario",
				"message_draft", "last_outcome", "next_step", "last_interaction_at",
				"days_since_last_interaction", "followup_count", "updated_at",
			}),
		}).Create(state).Error
}

// Update updates outreach state
func (r *OutreachStateRepository) Update(ctx context.Context, state *outreach.OutreachState) error {
	return r.db.WithContext(ctx).Save(state).Error
}

// FindByColdState finds contacts in COLD state for a user
func (r *OutreachStateRepository) FindByColdState(ctx context.Context, userID uuid.UUID, limit int) ([]*outreach.OutreachState, error) {
	var states []*outreach.OutreachState
	query := r.db.WithContext(ctx).
		Where("user_id = ? AND conversation_state = ?", userID, outreach.StateCold).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&states).Error; err != nil {
		return nil, err
	}
	return states, nil
}

// FindByFollowUpStates finds contacts needing follow-up
func (r *OutreachStateRepository) FindByFollowUpStates(ctx context.Context, userID uuid.UUID, limit int) ([]*outreach.OutreachState, error) {
	var states []*outreach.OutreachState
	query := r.db.WithContext(ctx).
		Where("user_id = ? AND conversation_state IN ?", userID, []string{
			string(outreach.StateNoReply),
			string(outreach.StateReplied),
			string(outreach.StatePostMeeting),
		}).
		Where("next_step IN ?", []string{
			string(outreach.NextStepFollowUp),
			string(outreach.NextStepSetMeeting),
			string(outreach.NextStepSend),
		}).
		Order(`
			CASE conversation_state
				WHEN 'REPLIED' THEN 1
				WHEN 'POST_MEETING' THEN 2
				WHEN 'NO_REPLY' THEN 3
				ELSE 4
			END,
			days_since_last_interaction DESC
		`)

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&states).Error; err != nil {
		return nil, err
	}
	return states, nil
}

// FindContactsNeedingOutreach finds contacts that need outreach with combined logic
func (r *OutreachStateRepository) FindContactsNeedingOutreach(ctx context.Context, userID uuid.UUID, outreachType string, limit int) ([]*outreach.OutreachState, error) {
	switch outreachType {
	case "cold":
		return r.FindByColdState(ctx, userID, limit)
	case "followup":
		return r.FindByFollowUpStates(ctx, userID, limit)
	case "mixed":
		// 60% cold, 40% follow-up
		coldLimit := int(float64(limit) * 0.6)
		followupLimit := limit - coldLimit

		coldStates, err := r.FindByColdState(ctx, userID, coldLimit)
		if err != nil {
			return nil, err
		}

		followupStates, err := r.FindByFollowUpStates(ctx, userID, followupLimit)
		if err != nil {
			return nil, err
		}

		// Fill up if one type is short
		totalFound := len(coldStates) + len(followupStates)
		if totalFound < limit {
			remaining := limit - totalFound
			if len(coldStates) < coldLimit {
				// Try to fill with more follow-ups
				moreFollowup, _ := r.FindByFollowUpStates(ctx, userID, remaining+followupLimit)
				followupStates = moreFollowup
			} else if len(followupStates) < followupLimit {
				// Try to fill with more cold
				moreCold, _ := r.FindByColdState(ctx, userID, remaining+coldLimit)
				coldStates = moreCold
			}
		}

		result := make([]*outreach.OutreachState, 0, len(coldStates)+len(followupStates))
		result = append(result, coldStates...)
		result = append(result, followupStates...)
		return result, nil
	default:
		return nil, nil
	}
}

// ============================================
// Meeting Repository
// ============================================

// MeetingRepository handles Meeting entity operations
type MeetingRepository struct {
	*gormpkg.GormRepository[outreach.Meeting]
	db *gorm.DB
}

// NewMeetingRepository creates a new meeting repository
func NewMeetingRepository(db *gorm.DB) *MeetingRepository {
	return &MeetingRepository{
		GormRepository: gormpkg.NewGormRepository[outreach.Meeting](db),
		db:             db,
	}
}

// Create creates a new meeting
func (r *MeetingRepository) Create(ctx context.Context, meeting *outreach.Meeting) error {
	return r.db.WithContext(ctx).Create(meeting).Error
}

// FindByID finds a meeting by ID
func (r *MeetingRepository) FindByID(ctx context.Context, id uuid.UUID) (*outreach.Meeting, error) {
	var meeting outreach.Meeting
	err := r.db.WithContext(ctx).First(&meeting, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &meeting, nil
}

// FindByContactID finds meetings for a contact
func (r *MeetingRepository) FindByContactID(ctx context.Context, contactID uuid.UUID) ([]*outreach.Meeting, error) {
	var meetings []*outreach.Meeting
	err := r.db.WithContext(ctx).
		Where("contact_id = ?", contactID).
		Order("time DESC").
		Find(&meetings).Error

	if err != nil {
		return nil, err
	}
	return meetings, nil
}

// FindByContactIDAndUserID finds meetings for a contact belonging to a user
func (r *MeetingRepository) FindByContactIDAndUserID(ctx context.Context, contactID, userID uuid.UUID) ([]*outreach.Meeting, error) {
	var meetings []*outreach.Meeting
	err := r.db.WithContext(ctx).
		Where("contact_id = ? AND user_id = ?", contactID, userID).
		Order("time DESC").
		Find(&meetings).Error

	if err != nil {
		return nil, err
	}
	return meetings, nil
}

// GetLastCompletedMeeting gets the last completed meeting for a contact
func (r *MeetingRepository) GetLastCompletedMeeting(ctx context.Context, contactID uuid.UUID) (*outreach.Meeting, error) {
	var meeting outreach.Meeting
	err := r.db.WithContext(ctx).
		Where("contact_id = ? AND status = ?", contactID, outreach.MeetingCompleted).
		Order("time DESC").
		First(&meeting).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &meeting, nil
}

// HasCompletedMeeting checks if contact has any completed meeting
func (r *MeetingRepository) HasCompletedMeeting(ctx context.Context, contactID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&outreach.Meeting{}).
		Where("contact_id = ? AND status = ?", contactID, outreach.MeetingCompleted).
		Count(&count).Error

	return count > 0, err
}

// Update updates a meeting
func (r *MeetingRepository) Update(ctx context.Context, meeting *outreach.Meeting) error {
	return r.db.WithContext(ctx).Save(meeting).Error
}

// Delete deletes a meeting by ID
func (r *MeetingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&outreach.Meeting{}, id).Error
}

// FindUpcoming finds upcoming meetings for a user
func (r *MeetingRepository) FindUpcoming(ctx context.Context, userID uuid.UUID, pagination *baseRepo.PaginationParams) ([]*outreach.Meeting, int, error) {
	var meetings []*outreach.Meeting
	var total int64

	if pagination == nil {
		pagination = baseRepo.DefaultPagination()
	}
	pagination.Validate()

	query := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ? AND time > ?", userID, outreach.MeetingScheduled, time.Now())

	if err := query.Model(&outreach.Meeting{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Offset(pagination.Offset).
		Limit(pagination.Limit).
		Order("time ASC").
		Find(&meetings).Error; err != nil {
		return nil, 0, err
	}

	return meetings, int(total), nil
}

// ============================================
// Feedback Repository (Task 8)
// ============================================

// FeedbackRepository handles OutreachFeedback entity operations
type FeedbackRepository struct {
	*gormpkg.GormRepository[outreach.OutreachFeedback]
	db *gorm.DB
}

// NewFeedbackRepository creates a new feedback repository
func NewFeedbackRepository(db *gorm.DB) *FeedbackRepository {
	return &FeedbackRepository{
		GormRepository: gormpkg.NewGormRepository[outreach.OutreachFeedback](db),
		db:             db,
	}
}

// Create creates a new feedback entry
func (r *FeedbackRepository) Create(ctx context.Context, feedback *outreach.OutreachFeedback) error {
	return r.db.WithContext(ctx).Create(feedback).Error
}

// FindByID finds feedback by ID
func (r *FeedbackRepository) FindByID(ctx context.Context, id uuid.UUID) (*outreach.OutreachFeedback, error) {
	var feedback outreach.OutreachFeedback
	err := r.db.WithContext(ctx).First(&feedback, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

// FindByContactID finds feedback entries for a contact
func (r *FeedbackRepository) FindByContactID(ctx context.Context, contactID, userID uuid.UUID) ([]*outreach.OutreachFeedback, error) {
	var feedbacks []*outreach.OutreachFeedback
	err := r.db.WithContext(ctx).
		Where("contact_id = ? AND user_id = ?", contactID, userID).
		Order("created_at DESC").
		Find(&feedbacks).Error

	if err != nil {
		return nil, err
	}
	return feedbacks, nil
}

// FindLatestForContact finds the most recent feedback for a contact
func (r *FeedbackRepository) FindLatestForContact(ctx context.Context, contactID, userID uuid.UUID) (*outreach.OutreachFeedback, error) {
	var feedback outreach.OutreachFeedback
	err := r.db.WithContext(ctx).
		Where("contact_id = ? AND user_id = ?", contactID, userID).
		Order("created_at DESC").
		First(&feedback).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

// FindPendingOutcome finds feedback entries that need outcome update
func (r *FeedbackRepository) FindPendingOutcome(ctx context.Context, userID uuid.UUID) ([]*outreach.OutreachFeedback, error) {
	var feedbacks []*outreach.OutreachFeedback
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND actual_action IS NOT NULL AND outcome IS NULL", userID).
		Order("created_at ASC").
		Find(&feedbacks).Error

	if err != nil {
		return nil, err
	}
	return feedbacks, nil
}

// Update updates a feedback entry
func (r *FeedbackRepository) Update(ctx context.Context, feedback *outreach.OutreachFeedback) error {
	return r.db.WithContext(ctx).Save(feedback).Error
}

// UpdateOutcome updates the outcome of a feedback entry
func (r *FeedbackRepository) UpdateOutcome(ctx context.Context, id uuid.UUID, outcome string, sentiment *string, daysToReply *int) error {
	now := time.Now()
	updates := map[string]interface{}{
		"outcome":            outcome,
		"outcome_updated_at": now,
	}
	if sentiment != nil {
		updates["reply_sentiment"] = *sentiment
	}
	if daysToReply != nil {
		updates["days_to_reply"] = *daysToReply
	}

	return r.db.WithContext(ctx).
		Model(&outreach.OutreachFeedback{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// GetScenarioStats gets statistics for scenarios
func (r *FeedbackRepository) GetScenarioStats(ctx context.Context, userID uuid.UUID) ([]*outreach.ScenarioStats, error) {
	var stats []*outreach.ScenarioStats

	query := `
		SELECT
			suggested_scenario as scenario,
			context_level,
			COUNT(*) as total_suggested,
			COUNT(CASE WHEN actual_action = 'used_draft' THEN 1 END) as drafts_used,
			COUNT(CASE WHEN actual_action = 'modified_draft' THEN 1 END) as drafts_modified,
			COUNT(CASE WHEN actual_action = 'wrote_own' THEN 1 END) as wrote_own,
			COUNT(CASE WHEN actual_action = 'skipped' THEN 1 END) as skipped,
			COUNT(CASE WHEN outcome = 'sent' OR outcome = 'replied' OR outcome = 'meeting_booked' OR outcome = 'meeting_done' THEN 1 END) as total_sent,
			COUNT(CASE WHEN outcome = 'replied' OR outcome = 'meeting_booked' OR outcome = 'meeting_done' THEN 1 END) as total_replied,
			COUNT(CASE WHEN outcome = 'meeting_booked' OR outcome = 'meeting_done' THEN 1 END) as total_meetings,
			COALESCE(
				ROUND(
					COUNT(CASE WHEN outcome IN ('replied', 'meeting_booked', 'meeting_done') THEN 1 END)::DECIMAL /
					NULLIF(COUNT(CASE WHEN outcome IN ('sent', 'replied', 'meeting_booked', 'meeting_done') THEN 1 END), 0) * 100, 2
				), 0
			) as reply_rate,
			COALESCE(
				ROUND(
					COUNT(CASE WHEN outcome IN ('meeting_booked', 'meeting_done') THEN 1 END)::DECIMAL /
					NULLIF(COUNT(CASE WHEN outcome IN ('replied', 'meeting_booked', 'meeting_done') THEN 1 END), 0) * 100, 2
				), 0
			) as meeting_rate,
			COALESCE(
				ROUND(
					(COUNT(CASE WHEN actual_action = 'used_draft' THEN 1 END) + COUNT(CASE WHEN actual_action = 'modified_draft' THEN 1 END))::DECIMAL /
					NULLIF(COUNT(*), 0) * 100, 2
				), 0
			) as draft_usage_rate,
			COALESCE(AVG(days_to_reply), 0) as avg_days_to_reply
		FROM outreach_feedback
		WHERE user_id = ?
		GROUP BY suggested_scenario, context_level
		ORDER BY total_suggested DESC
	`

	err := r.db.WithContext(ctx).Raw(query, userID).Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return stats, nil
}

// GetOverallStats gets overall statistics for a user
func (r *FeedbackRepository) GetOverallStats(ctx context.Context, userID uuid.UUID) (*outreach.ScenarioStats, error) {
	var stats outreach.ScenarioStats

	query := `
		SELECT
			'overall' as scenario,
			'ALL' as context_level,
			COUNT(*) as total_suggested,
			COUNT(CASE WHEN actual_action = 'used_draft' THEN 1 END) as drafts_used,
			COUNT(CASE WHEN actual_action = 'modified_draft' THEN 1 END) as drafts_modified,
			COUNT(CASE WHEN actual_action = 'wrote_own' THEN 1 END) as wrote_own,
			COUNT(CASE WHEN actual_action = 'skipped' THEN 1 END) as skipped,
			COUNT(CASE WHEN outcome IN ('sent', 'replied', 'meeting_booked', 'meeting_done') THEN 1 END) as total_sent,
			COUNT(CASE WHEN outcome IN ('replied', 'meeting_booked', 'meeting_done') THEN 1 END) as total_replied,
			COUNT(CASE WHEN outcome IN ('meeting_booked', 'meeting_done') THEN 1 END) as total_meetings,
			COALESCE(
				ROUND(
					COUNT(CASE WHEN outcome IN ('replied', 'meeting_booked', 'meeting_done') THEN 1 END)::DECIMAL /
					NULLIF(COUNT(CASE WHEN outcome IN ('sent', 'replied', 'meeting_booked', 'meeting_done') THEN 1 END), 0) * 100, 2
				), 0
			) as reply_rate,
			COALESCE(
				ROUND(
					COUNT(CASE WHEN outcome IN ('meeting_booked', 'meeting_done') THEN 1 END)::DECIMAL /
					NULLIF(COUNT(CASE WHEN outcome IN ('replied', 'meeting_booked', 'meeting_done') THEN 1 END), 0) * 100, 2
				), 0
			) as meeting_rate,
			COALESCE(
				ROUND(
					(COUNT(CASE WHEN actual_action = 'used_draft' THEN 1 END) + COUNT(CASE WHEN actual_action = 'modified_draft' THEN 1 END))::DECIMAL /
					NULLIF(COUNT(*), 0) * 100, 2
				), 0
			) as draft_usage_rate,
			COALESCE(AVG(days_to_reply), 0) as avg_days_to_reply
		FROM outreach_feedback
		WHERE user_id = ?
	`

	err := r.db.WithContext(ctx).Raw(query, userID).Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
