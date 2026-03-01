package workflow

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"gorm.io/gorm"
)

// Workflow represents the campaign workflow (CWorkflow in Python code).
type Workflow struct {
	base.Base
	base.SoftDeleteMixin
	base.TimestampMixin

	UserID    uuid.UUID      `gorm:"type:uuid;not null;index:idx_workflows_user_id" json:"user_id"`
	Nodes     pq.StringArray `gorm:"type:text[];default:'{}'" json:"nodes,omitempty"`
	Edges     base.JSONB     `gorm:"type:jsonb;default:'[]'" json:"edges,omitempty"`
	State     base.JSONB     `gorm:"type:jsonb;default:'{}'" json:"state,omitempty"`
	CMetadata base.JSONB     `gorm:"type:jsonb;default:'{}'" json:"cmetadata,omitempty"`
}

// TableName specifies the table name.
func (Workflow) TableName() string {
	return "workflows"
}

// BeforeCreate ensures defaults align with the Python model.
func (w *Workflow) BeforeCreate(tx *gorm.DB) error {
	if err := w.Base.BeforeCreate(tx); err != nil {
		return err
	}

	if len(w.Nodes) == 0 {
		w.Nodes = pq.StringArray{}
	}
	if len(w.Edges) == 0 {
		w.Edges = base.JSONB([]byte("[]"))
	}
	if len(w.State) == 0 {
		w.State = base.JSONB([]byte("{}"))
	}
	if len(w.CMetadata) == 0 {
		w.CMetadata = base.JSONB([]byte("{}"))
	}

	return nil
}
