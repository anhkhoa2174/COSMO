package lead_form_integration

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// IntegrationType enumerates supported lead form integrations.
type IntegrationType string

const (
	IntegrationTypeFacebook IntegrationType = "facebook"
)

// LeadFormIntegration represents an external lead form tied to a campaign.
type LeadFormIntegration struct {
	base.Base
	base.TimestampMixin

	CampaignID      uuid.UUID       `gorm:"type:uuid;not null" json:"campaign_id"`
	IntegrationType IntegrationType `gorm:"type:text;not null" json:"integration_type"`
	Config          base.JSONB      `gorm:"type:jsonb;default:'{}'" json:"config,omitempty"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.LeadFormIntegrationWithCampaign
	// - relations.LeadFormIntegrationWithFieldMappings
	// - relations.LeadFormIntegrationWithFullRelations
	FieldMappings []LeadFieldMapping `gorm:"foreignKey:FormIntegrationID;constraint:OnDelete:CASCADE" json:"field_mappings,omitempty"`
}

func (LeadFormIntegration) TableName() string {
	return "lead_form_integrations"
}

// MappingType indicates how external fields are mapped.
type MappingType string

const (
	MappingTypeContactField MappingType = "contact_field"
	MappingTypeCustomField  MappingType = "custom_field"
)

// LeadFieldMapping connects external fields to contact/custom fields.
type LeadFieldMapping struct {
	base.Base
	base.TimestampMixin

	CampaignID        uuid.UUID   `gorm:"type:uuid;not null" json:"campaign_id"`
	FormIntegrationID uuid.UUID   `gorm:"type:uuid;not null;index:idx_lead_field_mappings_integration_id" json:"form_integration_id"`
	CustomFieldID     *uuid.UUID  `gorm:"type:uuid" json:"custom_field_id,omitempty"`
	ExternalFieldName string      `gorm:"type:text;not null" json:"external_field_name"`
	MappingType       MappingType `gorm:"type:text;not null" json:"mapping_type"`
	ContactFieldName  *string     `gorm:"type:text" json:"contact_field_name,omitempty"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.LeadFieldMappingWithFormIntegration
	// - relations.LeadFieldMappingWithCustomField
	// - relations.LeadFieldMappingWithFullRelations
	FormIntegration *LeadFormIntegration `gorm:"foreignKey:FormIntegrationID;constraint:OnDelete:CASCADE" json:"lead_form_integration,omitempty"`
}

func (LeadFieldMapping) TableName() string {
	return "lead_field_mappings"
}
