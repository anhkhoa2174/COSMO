package segmentation

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// Segmentation represents a dynamic segment definition.
type Segmentation struct {
	base.Base
	base.TimestampMixin

	UserID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	Priority      int        `gorm:"default:5" json:"priority"`
	Criteria      base.JSONB `gorm:"type:jsonb;default:'{}'" json:"criteria"`
	ICPDefinition base.JSONB `gorm:"type:jsonb;default:'{}'" json:"icp_definition"`
	IsActive      bool       `gorm:"default:true" json:"is_active"`
}

func (Segmentation) TableName() string {
	return "segmentations"
}

// ContactSegmentScore represents fit score of a contact in a segment.
type ContactSegmentScore struct {
	base.Base
	base.TimestampMixin

	ContactID          uuid.UUID  `gorm:"type:uuid;not null;index" json:"contact_id"`
	SegmentationID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"segmentation_id"`
	FitScore           int        `gorm:"default:0" json:"fit_score"`
	ScoreBreakdown     base.JSONB `gorm:"type:jsonb;default:'{}'" json:"score_breakdown"`
	ScoreType          string     `gorm:"default:'auto'" json:"score_type"`
	Status             string     `gorm:"default:'qualified'" json:"status"`
	PassesFilters      bool       `gorm:"default:true" json:"passes_filters"`
	EnrolledInCampaign bool       `gorm:"default:false" json:"enrolled_in_campaign"`
	CurrentCampaignID  *uuid.UUID `gorm:"type:uuid" json:"current_campaign_id,omitempty"`
}

func (ContactSegmentScore) TableName() string {
	return "contact_segment_scores"
}
