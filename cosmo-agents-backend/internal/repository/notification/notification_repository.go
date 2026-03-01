package notification

import (
	"context"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"

	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NotificationRepository handles Notification entity operations.
type NotificationRepository struct {
	*gormpkg.GormRepository[domain.Notification]
	db *gorm.DB
}

// NewNotificationRepository creates a new notification gormpkg.
func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{
		GormRepository: gormpkg.NewGormRepository[domain.Notification](db),
		db:             db,
	}
}

// UpsertMany creates or updates notifications based on (user_id, campaign_id) uniqueness.
func (r *NotificationRepository) UpsertMany(ctx context.Context, notifications []domain.Notification) ([]domain.Notification, error) {
	if len(notifications) == 0 {
		return nil, nil
	}

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "campaign_id"},
			},
			DoNothing: true,
		}).
		Create(&notifications).Error

	if err != nil {
		return nil, err
	}

	return notifications, nil
}

// FindByCampaignID retrieves all notifications for a campaign
func (r *NotificationRepository) FindByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*domain.Notification, error) {
	var notifications []*domain.Notification
	err := r.db.WithContext(ctx).
		Where("campaign_id = ?", campaignID).
		Find(&notifications).Error
	return notifications, err
}

// DeleteByIDs removes notifications by IDs for a given campaign.
func (r *NotificationRepository) DeleteByIDs(ctx context.Context, campaignID uuid.UUID, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Where("campaign_id = ? AND id IN ?", campaignID, ids).
		Delete(&domain.Notification{}).Error
}
