package interaction

import (
	"context"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	"gorm.io/gorm"
)

// Repository for interactions.
type Repository struct {
	*gormpkg.GormRepository[domain.Interaction]
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{GormRepository: gormpkg.NewGormRepository[domain.Interaction](db), db: db}
}

func (r *Repository) ListByContact(ctx context.Context, contactID uuid.UUID, limit int) ([]*domain.Interaction, error) {
	if limit <= 0 {
		limit = 50
	}
	var interactions []*domain.Interaction
	if err := r.db.WithContext(ctx).
		Where("contact_id = ?", contactID).
		Order("occurred_at DESC").
		Limit(limit).
		Find(&interactions).Error; err != nil {
		return nil, err
	}
	return interactions, nil
}
