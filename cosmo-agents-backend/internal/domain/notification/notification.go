package notification

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// Notification stores user campaign notification records.
type Notification struct {
	base.Base
	base.SoftDeleteMixin
	base.TimestampMixin

	UserID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_notifications_user_campaign" json:"user_id"`
	CampaignID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_notifications_user_campaign" json:"campaign_id"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.UserWithNotifications
	// - relations.CampaignWithFullRelations
}

func (Notification) TableName() string {
	return "notifications"
}
