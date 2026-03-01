package feedback

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// UserFeedback stores human feedback for learning and audit.
type UserFeedback struct {
	base.Base
	base.TimestampMixin

	UserID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	EntityType   string     `json:"entity_type"` // contact/segmentation/etc.
	EntityID     uuid.UUID  `gorm:"type:uuid;not null" json:"entity_id"`
	FeedbackType string     `json:"feedback_type"` // score_adjustment/insight_validation/custom_fact
	FeedbackData base.JSONB `gorm:"type:jsonb;default:'{}'" json:"feedback_data"`
	Applied      bool       `gorm:"default:false" json:"applied"`
	AppliedAt    *time.Time `json:"applied_at,omitempty"`
}

func (UserFeedback) TableName() string {
	return "user_feedback"
}
