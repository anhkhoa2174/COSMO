package daily_action

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/core"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
)

// ActionRepository handles DailyAction entity operations.
type ActionRepository struct {
	*gormpkg.GormRepository[domain.DailyAction]
	db *gorm.DB
}

// NewActionRepository creates a new action repository.
func NewActionRepository(db *gorm.DB) *ActionRepository {
	return &ActionRepository{
		GormRepository: gormpkg.NewGormRepository[domain.DailyAction](db),
		db:             db,
	}
}

// FindByIDAndUserID finds an action by ID scoped to a user.
func (r *ActionRepository) FindByIDAndUserID(ctx context.Context, userID uuid.UUID, actionID uuid.UUID) (*domain.DailyAction, error) {
	var action domain.DailyAction
	err := core.DB(ctx, r.db).WithContext(ctx).
		Where("id = ? AND user_id = ? AND is_deleted = ?", actionID, userID, false).
		First(&action).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &action, nil
}

// FindByGenerationID finds all actions for a generation, ordered by priority.
func (r *ActionRepository) FindByGenerationID(ctx context.Context, generationID uuid.UUID) ([]domain.DailyAction, error) {
	var actions []domain.DailyAction
	err := core.DB(ctx, r.db).WithContext(ctx).
		Where("generation_id = ? AND is_deleted = ?", generationID, false).
		Order("priority ASC").
		Find(&actions).Error
	return actions, err
}

// FindByCategoryWithPagination finds actions for a generation and category with offset pagination.
// Excludes snoozed actions whose snooze_until is still in the future.
func (r *ActionRepository) FindByCategoryWithPagination(ctx context.Context, generationID uuid.UUID, categoryID string, offset, limit int, includeCompleted bool) ([]domain.DailyAction, int64, error) {
	query := core.DB(ctx, r.db).WithContext(ctx).
		Where("generation_id = ? AND category_id = ? AND is_deleted = ?", generationID, categoryID, false)

	if !includeCompleted {
		query = query.Where("status NOT IN ?", []string{
			string(domain.ActionStatusCompleted),
			string(domain.ActionStatusSkipped),
		})
	}

	// Exclude currently snoozed actions
	query = query.Where("(snooze_until IS NULL OR snooze_until <= ?)", time.Now())

	var total int64
	if err := query.Model(&domain.DailyAction{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var actions []domain.DailyAction
	err := query.
		Order("priority ASC").
		Offset(offset).
		Limit(limit).
		Find(&actions).Error
	return actions, total, err
}

// UpdateStatus updates an action's status and status_changed_at timestamp.
func (r *ActionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ActionStatus, extras map[string]interface{}) error {
	now := time.Now()
	updates := map[string]interface{}{
		"status":            status,
		"status_changed_at": now,
	}
	for k, v := range extras {
		updates[k] = v
	}
	return core.DB(ctx, r.db).WithContext(ctx).
		Model(&domain.DailyAction{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(updates).Error
}

// CountByGenerationAndStatus counts actions for a generation grouped by status.
func (r *ActionRepository) CountByGenerationAndStatus(ctx context.Context, generationID uuid.UUID) (map[domain.ActionStatus]int, error) {
	type result struct {
		Status string
		Count  int
	}
	var results []result
	err := core.DB(ctx, r.db).WithContext(ctx).
		Model(&domain.DailyAction{}).
		Select("status, count(*) as count").
		Where("generation_id = ? AND is_deleted = ?", generationID, false).
		Group("status").
		Find(&results).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[domain.ActionStatus]int)
	for _, r := range results {
		counts[domain.ActionStatus(r.Status)] = r.Count
	}
	return counts, nil
}

// BatchCreate creates multiple actions in a single insert.
func (r *ActionRepository) BatchCreate(ctx context.Context, actions []domain.DailyAction) error {
	if len(actions) == 0 {
		return nil
	}
	return core.DB(ctx, r.db).WithContext(ctx).Create(&actions).Error
}

// DeleteByGenerationID soft-deletes all actions for a generation (used on force_refresh).
func (r *ActionRepository) DeleteByGenerationID(ctx context.Context, generationID uuid.UUID) error {
	return core.DB(ctx, r.db).WithContext(ctx).
		Model(&domain.DailyAction{}).
		Where("generation_id = ? AND is_deleted = ?", generationID, false).
		Update("is_deleted", true).Error
}
