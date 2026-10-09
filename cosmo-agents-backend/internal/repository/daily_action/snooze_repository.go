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

// SnoozeRepository handles ActionSnooze entity operations.
type SnoozeRepository struct {
	*gormpkg.GormRepository[domain.ActionSnooze]
	db *gorm.DB
}

// NewSnoozeRepository creates a new snooze repository.
func NewSnoozeRepository(db *gorm.DB) *SnoozeRepository {
	return &SnoozeRepository{
		GormRepository: gormpkg.NewGormRepository[domain.ActionSnooze](db),
		db:             db,
	}
}

// FindActiveByActionID finds the active (non-cleared) snooze for an action.
func (r *SnoozeRepository) FindActiveByActionID(ctx context.Context, actionID uuid.UUID) (*domain.ActionSnooze, error) {
	var snooze domain.ActionSnooze
	err := core.DB(ctx, r.db).WithContext(ctx).
		Where("action_id = ? AND cleared_at IS NULL", actionID).
		Order("created_at DESC").
		First(&snooze).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &snooze, nil
}

// ClearByActionID clears active snoozes for an action (sets cleared_at).
func (r *SnoozeRepository) ClearByActionID(ctx context.Context, actionID uuid.UUID) error {
	now := time.Now()
	return core.DB(ctx, r.db).WithContext(ctx).
		Model(&domain.ActionSnooze{}).
		Where("action_id = ? AND cleared_at IS NULL", actionID).
		Update("cleared_at", now).Error
}

// FindExpiredSnoozes finds snoozes that have expired but not been cleared.
func (r *SnoozeRepository) FindExpiredSnoozes(ctx context.Context, userID uuid.UUID) ([]domain.ActionSnooze, error) {
	var snoozes []domain.ActionSnooze
	err := core.DB(ctx, r.db).WithContext(ctx).
		Where("user_id = ? AND cleared_at IS NULL AND snooze_until <= ?", userID, time.Now()).
		Find(&snoozes).Error
	return snoozes, err
}
