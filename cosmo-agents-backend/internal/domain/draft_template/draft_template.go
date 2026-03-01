package draft_template

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/template"
)

// DraftTemplate represents a draft intent template linked to a campaign.
type DraftTemplate struct {
	base.Base
	base.TimestampMixin

	Intent     string    `gorm:"type:text;not null" json:"intent"`
	CampaignID uuid.UUID `gorm:"type:uuid;not null;index:idx_draft_templates_campaign_id" json:"campaign_id"`
	TemplateID uuid.UUID `gorm:"type:uuid;not null" json:"template_id"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.DraftTemplateWithCampaign
	// - relations.DraftTemplateWithTemplate
	// - relations.DraftTemplateWithFullRelations
	Template *template.Template `gorm:"foreignKey:TemplateID;constraint:OnDelete:CASCADE" json:"template,omitempty"`
}

// TableName specifies the database table.
func (DraftTemplate) TableName() string {
	return "draft_templates"
}
