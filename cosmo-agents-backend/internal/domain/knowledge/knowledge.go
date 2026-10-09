package knowledge

import (
	"strings"

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

// KnowledgeType says what a document is about, so retrieval can prefer the
// documents that answer a given kind of question: a pricing question is
// answered from the pricing sheet before anything else.
type KnowledgeType string

const (
	KnowledgeTypePricing   KnowledgeType = "pricing"
	KnowledgeTypeProduct   KnowledgeType = "product"
	KnowledgeTypeCaseStudy KnowledgeType = "case_study"
	KnowledgeTypeFAQ       KnowledgeType = "faq"
	KnowledgeTypeOther     KnowledgeType = "other"
)

// KnowledgeTypes lists every accepted type, in display order.
func KnowledgeTypes() []KnowledgeType {
	return []KnowledgeType{
		KnowledgeTypePricing, KnowledgeTypeProduct, KnowledgeTypeCaseStudy,
		KnowledgeTypeFAQ, KnowledgeTypeOther,
	}
}

// ParseKnowledgeType accepts a type name in any case and with surrounding
// space. An empty value is "other", which is what an upload that does not say
// is; anything else unrecognised is rejected rather than stored as a type no
// search will ever ask for.
func ParseKnowledgeType(raw string) (KnowledgeType, bool) {
	norm := strings.ToLower(strings.TrimSpace(raw))
	if norm == "" {
		return KnowledgeTypeOther, true
	}
	for _, t := range KnowledgeTypes() {
		if string(t) == norm {
			return t, true
		}
	}
	return "", false
}

// Knowledge mirrors machine/models/knowledge.py.
type Knowledge struct {
	base.Base
	base.SoftDeleteMixin
	base.TimestampMixin

	SourceType    KnowledgeSourceType `gorm:"type:text;default:'upload'" json:"source_type"`
	Type          KnowledgeType       `gorm:"type:text;not null;default:'other'" json:"type"`
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
	if k.Type == "" {
		k.Type = KnowledgeTypeOther
	}

	return nil
}
