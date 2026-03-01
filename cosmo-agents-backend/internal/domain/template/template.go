package template

import (
	"errors"
	"strconv"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"gorm.io/gorm"
)

// TemplateCategory represents the business classification of a template.
type TemplateCategory string

const (
	TemplateCategoryDraft       TemplateCategory = "Draft"
	TemplateCategoryOutreach    TemplateCategory = "Outreach"
	TemplateCategoryOutOfOffice TemplateCategory = "Out of office outreach"
	defaultTemplatePosition                      = 1000.0
	positionIncrement                            = 1000.0
)

// Template mirrors machine/models/template.py.
type Template struct {
	base.Base
	base.TimestampMixin

	UserID     uuid.UUID        `gorm:"type:uuid;not null;index:idx_templates_user_id;uniqueIndex:uq_templates_type" json:"user_id"`
	CampaignID *uuid.UUID       `gorm:"type:uuid;index:idx_templates_campaign_id;uniqueIndex:uq_templates_type" json:"campaign_id,omitempty"`
	Type       string           `gorm:"type:text;uniqueIndex:uq_templates_type" json:"type"`
	Category   TemplateCategory `gorm:"type:text" json:"category"`
	Subject    string           `gorm:"type:text" json:"subject"`
	Content    string           `gorm:"type:text" json:"content"`
	CMetadata  base.JSONB       `gorm:"type:jsonb;default:'{}';column:cmetadata" json:"cmetadata,omitempty"`
	Position   float64          `gorm:"type:double precision;not null" json:"position"`
	SendAfter  int              `gorm:"default:0" json:"send_after"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.TemplateWithCampaign
	// - relations.TemplateWithKnowledges
	// - relations.TemplateWithFullRelations
}

// TableName specifies the table name.
func (Template) TableName() string {
	return "templates"
}

// BeforeCreate sets defaults aligned with the Python implementation.
func (t *Template) BeforeCreate(tx *gorm.DB) error {
	if err := t.Base.BeforeCreate(tx); err != nil {
		return err
	}

	if len(t.CMetadata) == 0 {
		t.CMetadata = base.JSONB([]byte("{}"))
	}

	if t.Position == 0 {
		t.Position = defaultTemplatePosition
	}

	return nil
}

// EmailTypeForIndex mimics Template.get_email_type from Python.
func EmailTypeForIndex(index int) string {
	if index <= 0 {
		return "First Email"
	}
	if index == 1 {
		return "Follow-up Email 1"
	}
	return "Follow-up Email " + strconv.Itoa(index)
}

// CalculatePositionBetween returns the midpoint between two positions.
func CalculatePositionBetween(before, after float64) (float64, error) {
	if before == 0 && after == 0 {
		return 0, errors.New("invalid template positions")
	}
	return (before + after) / 2, nil
}

// CalculateNextPosition returns a monotonically increasing position value.
func CalculateNextPosition(current float64) float64 {
	if current == 0 {
		return defaultTemplatePosition
	}
	return current + positionIncrement
}

// NormalizePosition returns the normalized position for a given index.
func NormalizePosition(index int) float64 {
	return float64(index+1) * positionIncrement
}
