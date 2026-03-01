package feedback

import (
	"context"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	"gorm.io/gorm"
)

type Repository struct {
	*gormpkg.GormRepository[domain.UserFeedback]
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{GormRepository: gormpkg.NewGormRepository[domain.UserFeedback](db), db: db}
}

func (r *Repository) ListByUser(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.UserFeedback, error) {
	if limit <= 0 {
		limit = 50
	}
	var items []*domain.UserFeedback
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
