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

	// Captured during onboarding — which CRM the team runs on, and how they
	// handle inbound leads today.
	CRM               string         `json:"crm"`
	LeadHandling      pq.StringArray `gorm:"type:text[]" json:"lead_handling"`
	LeadHandlingOther string         `json:"lead_handling_other"`

	// Per-organisation outreach cadence, set by an admin. Nil, or any key
	// left out of it, falls back to the service defaults — so an organisation
	// that never opens the settings page behaves exactly as before.
	OutreachSettings base.JSONB `gorm:"type:jsonb" json:"outreach_settings,omitempty"`

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
