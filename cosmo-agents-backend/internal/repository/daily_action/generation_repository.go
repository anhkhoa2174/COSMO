package daily_action

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/core"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
)

// GenerationRepository handles DailyActionGeneration entity operations.
type GenerationRepository struct {
	*gormpkg.GormRepository[domain.DailyActionGeneration]
	db *gorm.DB
}

// NewGenerationRepository creates a new generation repository.
func NewGenerationRepository(db *gorm.DB) *GenerationRepository {
	return &GenerationRepository{
		GormRepository: gormpkg.NewGormRepository[domain.DailyActionGeneration](db),
		db:             db,
	}
}

// FindLatestByUserAndDate finds the latest non-replaced generation for a user and date.
func (r *GenerationRepository) FindLatestByUserAndDate(ctx context.Context, userID uuid.UUID, date string) (*domain.DailyActionGeneration, error) {
	var gen domain.DailyActionGeneration
	err := core.DB(ctx, r.db).WithContext(ctx).
		Where("user_id = ? AND date = ? AND is_deleted = ? AND replaced_by_id IS NULL", userID, date, false).
		Order("created_at DESC").
		First(&gen).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &gen, nil
}

// FindActiveByUserAndDate finds an in-progress generation (started or generating) for a user and date.
func (r *GenerationRepository) FindActiveByUserAndDate(ctx context.Context, userID uuid.UUID, date string) (*domain.DailyActionGeneration, error) {
	var gen domain.DailyActionGeneration
	err := core.DB(ctx, r.db).WithContext(ctx).
		Where("user_id = ? AND date = ? AND is_deleted = ? AND replaced_by_id IS NULL AND status IN ?",
			userID, date, false, []string{string(domain.GenerationStatusStarted), string(domain.GenerationStatusGenerating)}).
		First(&gen).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &gen, nil
}

// UpdateStatus updates the generation status and optionally sets generated_at and action_count.
func (r *GenerationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.GenerationStatus, attrs map[string]interface{}) error {
	updates := map[string]interface{}{"status": status}
	for k, v := range attrs {
		updates[k] = v
	}
	return core.DB(ctx, r.db).WithContext(ctx).
		Model(&domain.DailyActionGeneration{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(updates).Error
}

// MarkReplaced marks a generation as replaced by a newer one.
func (r *GenerationRepository) MarkReplaced(ctx context.Context, oldID uuid.UUID, newID uuid.UUID) error {
	return core.DB(ctx, r.db).WithContext(ctx).
		Model(&domain.DailyActionGeneration{}).
		Where("id = ? AND is_deleted = ?", oldID, false).
		Update("replaced_by_id", newID).Error
}

// SoftDelete sets is_deleted = true on a generation record.
func (r *GenerationRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return core.DB(ctx, r.db).WithContext(ctx).
		Model(&domain.DailyActionGeneration{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error
}
