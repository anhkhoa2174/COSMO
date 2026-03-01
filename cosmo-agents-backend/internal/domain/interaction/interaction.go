package interaction

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// Interaction represents a logged interaction (email/linkedin/call/etc.)
type Interaction struct {
	base.Base
	base.TimestampMixin

	ContactID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"contact_id"`
	CampaignID     *uuid.UUID `gorm:"type:uuid" json:"campaign_id,omitempty"`
	SegmentationID *uuid.UUID `gorm:"type:uuid" json:"segmentation_id,omitempty"`

	InteractionType string     `json:"interaction_type"` // sent/open/reply/bounce/linkedin_message/call/meeting
	Channel         string     `json:"channel"`          // email/linkedin/phone/in_person
	Direction       string     `json:"direction"`        // inbound/outbound
	Content         base.JSONB `gorm:"type:jsonb;default:'{}'" json:"content"`
	AIAnalysis      base.JSONB `gorm:"type:jsonb;default:'{}'" json:"ai_analysis"`
	OccurredAt      time.Time  `gorm:"not null" json:"occurred_at"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
}

func (Interaction) TableName() string {
	return "interactions"
}
