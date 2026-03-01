package knowledge

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"gorm.io/gorm"
)

// KnowledgeSourceType defines knowledge ingestion source.
type KnowledgeSourceType string

const (
	KnowledgeSourceWebsite KnowledgeSourceType = "website"
	KnowledgeSourceUpload  KnowledgeSourceType = "upload"
)

// Knowledge mirrors machine/models/knowledge.py.
type Knowledge struct {
	base.Base
	base.SoftDeleteMixin
	base.TimestampMixin

	SourceType    KnowledgeSourceType `gorm:"type:text;default:'upload'" json:"source_type"`
	SummaryPair   pq.StringArray      `gorm:"type:text[];default:'{}'" json:"summary_pair,omitempty"`
	EmbeddingGID  *string             `gorm:"column:embedding_gid;type:text;uniqueIndex" json:"embedding_gid,omitempty"`
	CozeDatasetID *string             `gorm:"column:coze_dataset_id;type:text;uniqueIndex" json:"coze_dataset_id,omitempty"`
	UserID        uuid.UUID           `gorm:"type:uuid;not null;index:idx_knowledges_user_id" json:"user_id"`
	CMetadata     base.JSONB          `gorm:"column:cmetadata;type:jsonb;default:'{}'" json:"cmetadata,omitempty"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.KnowledgeWithTemplates
	// - relations.KnowledgeWithFullRelations
}

// TableName specifies the table name.
func (Knowledge) TableName() string {
	return "knowledges"
}

// BeforeCreate sets defaults similarly to Python model.
func (k *Knowledge) BeforeCreate(tx *gorm.DB) error {
	if err := k.Base.BeforeCreate(tx); err != nil {
		return err
	}

	if len(k.SummaryPair) == 0 {
		k.SummaryPair = pq.StringArray{}
	}
	if len(k.CMetadata) == 0 {
		k.CMetadata = base.JSONB([]byte("{}"))
	}
	if k.SourceType == "" {
		k.SourceType = KnowledgeSourceUpload
	}

	return nil
}
