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

// SSEEventRepository handles SSEEvent entity operations.
type SSEEventRepository struct {
	*gormpkg.GormRepository[domain.SSEEvent]
	db *gorm.DB
}

// NewSSEEventRepository creates a new SSE event repository.
func NewSSEEventRepository(db *gorm.DB) *SSEEventRepository {
	return &SSEEventRepository{
		GormRepository: gormpkg.NewGormRepository[domain.SSEEvent](db),
		db:             db,
	}
}

// FindByUserSince finds SSE events for a user after a given event ID (for Last-Event-ID replay).
func (r *SSEEventRepository) FindByUserSince(ctx context.Context, userID uuid.UUID, sinceEventID uuid.UUID) ([]domain.SSEEvent, error) {
	// First find the timestamp of the sinceEventID
	var refEvent domain.SSEEvent
	err := core.DB(ctx, r.db).WithContext(ctx).
		Where("id = ? AND user_id = ?", sinceEventID, userID).
		First(&refEvent).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	var events []domain.SSEEvent
	err = core.DB(ctx, r.db).WithContext(ctx).
		Where("user_id = ? AND created_at > ?", userID, refEvent.CreatedAt).
		Order("created_at ASC").
		Find(&events).Error
	return events, err
}

// FindRecentByUser finds recent SSE events for a user (for initial load).
func (r *SSEEventRepository) FindRecentByUser(ctx context.Context, userID uuid.UUID, limit int) ([]domain.SSEEvent, error) {
	var events []domain.SSEEvent
	err := core.DB(ctx, r.db).WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

// CleanupOlderThan deletes SSE events older than the given duration.
func (r *SSEEventRepository) CleanupOlderThan(ctx context.Context, before time.Time) error {
	return core.DB(ctx, r.db).WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&domain.SSEEvent{}).Error
}
