package template_knowledge

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// TemplateKnowledge represents the join table between templates and knowledges.
type TemplateKnowledge struct {
	base.Base

	TemplateID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_template_knowledges_pair" json:"template_id"`
	KnowledgeID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_template_knowledges_pair" json:"knowledge_id"`
}

// TableName specifies the table name.
func (TemplateKnowledge) TableName() string {
	return "template_knowledges"
}
