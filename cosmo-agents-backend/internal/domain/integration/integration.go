package integration

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// DuplicationOption indicates how to handle duplicates.
type DuplicationOption string

const (
	DuplicationOptionSkip      DuplicationOption = "skip"
	DuplicationOptionOverwrite DuplicationOption = "overwrite"
	DuplicationOptionMerge     DuplicationOption = "merge"
	DuplicationOptionKeepBoth  DuplicationOption = "keep_both"
)

// SourceIntegration denotes the integration provider.
type SourceIntegration string

const (
	SourceIntegrationHubspot  SourceIntegration = "hubspot"
	SourceIntegrationFacebook SourceIntegration = "facebook"
)

// Integration mirrors machine/models/integration.py.
type Integration struct {
	base.Base
	base.TimestampMixin

	ID         string            `gorm:"primaryKey" json:"id"`
	UserID     uuid.UUID         `gorm:"type:uuid;not null" json:"user_id"`
	Source     SourceIntegration `gorm:"type:text;not null" json:"source"`
	SourceID   string            `gorm:"type:text;not null" json:"source_id"`
	Credential base.JSONB        `gorm:"type:jsonb" json:"credential,omitempty"`
	Config     base.JSONB        `gorm:"type:jsonb" json:"config,omitempty"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.IntegrationWithUser
}

func (Integration) TableName() string {
	return "integrations"
}
