package organization

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// Organization represents a company/organization
// Converted from machine/models/organization.py
type Organization struct {
	base.Base
	base.TimestampMixin
	base.SoftDeleteMixin

	UserID                  *uuid.UUID     `gorm:"type:uuid;index" json:"user_id"`
	Name                    string         `json:"name"`
	CompanyURL              string         `json:"company_url"`
	CompanyDescription      string         `json:"company_description"`
	CompanyTargetingPersona pq.StringArray `gorm:"type:text[]" json:"company_targeting_persona"`
	ValueOffering           string         `json:"value_offering"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.OrganizationWithUser
	// - relations.OrganizationWithRoles
	// - relations.OrganizationWithAgents
	// - relations.OrganizationWithFullRelations
}

// TableName specifies the table name
func (Organization) TableName() string {
	return "organizations"
}
