package daily_action

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/core"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
)

// CompletionLogRepository handles ActionCompletionLog entity operations (append-only).
type CompletionLogRepository struct {
	*gormpkg.GormRepository[domain.ActionCompletionLog]
	db *gorm.DB
}

// NewCompletionLogRepository creates a new completion log repository.
func NewCompletionLogRepository(db *gorm.DB) *CompletionLogRepository {
	return &CompletionLogRepository{
		GormRepository: gormpkg.NewGormRepository[domain.ActionCompletionLog](db),
		db:             db,
	}
}

// FindByUserAndDate finds all completion logs for a user on a given date.
func (r *CompletionLogRepository) FindByUserAndDate(ctx context.Context, userID uuid.UUID, date string) ([]domain.ActionCompletionLog, error) {
	var logs []domain.ActionCompletionLog
	err := core.DB(ctx, r.db).WithContext(ctx).
		Where("user_id = ? AND date = ?", userID, date).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

// CountByUserDateAndTransition counts completion logs grouped by transition for a user and date.
func (r *CompletionLogRepository) CountByUserDateAndTransition(ctx context.Context, userID uuid.UUID, date string) (map[string]int, error) {
	type result struct {
		Transition string
		Count      int
	}
	var results []result
	err := core.DB(ctx, r.db).WithContext(ctx).
		Model(&domain.ActionCompletionLog{}).
		Select("transition, count(*) as count").
		Where("user_id = ? AND date = ?", userID, date).
		Group("transition").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, r := range results {
		counts[r.Transition] = r.Count
	}
	return counts, nil
}

// CountByUserDateAndActionType counts logs grouped by action_type for summary breakdown.
func (r *CompletionLogRepository) CountByUserDateAndActionType(ctx context.Context, userID uuid.UUID, date string, transition string) (map[string]int, error) {
	type result struct {
		ActionType string
		Count      int
	}
	var results []result
	err := core.DB(ctx, r.db).WithContext(ctx).
		Model(&domain.ActionCompletionLog{}).
		Select("action_type, count(*) as count").
		Where("user_id = ? AND date = ? AND transition = ?", userID, date, transition).
		Group("action_type").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[string]int)
	for _, r := range results {
		counts[r.ActionType] = r.Count
	}
	return counts, nil
}
