package inbound_lead_form

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	customFieldDomain "github.com/rockship/cosmo-agents-go/internal/domain/custom_field"
)

// FormField represents a single form field configuration.
type FormField struct {
	base.Base
	base.TimestampMixin

	FormID          uuid.UUID  `gorm:"type:uuid;not null;index:idx_form_fields_form_id" json:"form_id"`
	CustomFieldID   *uuid.UUID `gorm:"type:uuid" json:"custom_field_id,omitempty"`
	DisplayName     string     `gorm:"type:text;not null" json:"display_name"`
	IsSystemField   bool       `gorm:"not null;default:false" json:"is_system_field"`
	SystemFieldName *string    `gorm:"type:text" json:"system_field_name,omitempty"`
	UIMetadata      base.JSONB `gorm:"type:jsonb" json:"ui_metadata,omitempty"`
	IsRequired      bool       `gorm:"not null;default:false" json:"is_required"`

	Form        *InboundLeadForm               `gorm:"foreignKey:FormID;constraint:OnDelete:CASCADE" json:"form,omitempty"`
	CustomField *customFieldDomain.CustomField `gorm:"foreignKey:CustomFieldID;constraint:OnDelete:CASCADE" json:"custom_field,omitempty"`
}

func (FormField) TableName() string {
	return "form_fields"
}

// InboundLeadForm holds lead capture configuration.
type InboundLeadForm struct {
	base.Base
	base.TimestampMixin

	UserID     *uuid.UUID `gorm:"type:uuid;index:idx_inbound_lead_forms_user_id" json:"user_id,omitempty"`
	Name       string     `gorm:"type:text;not null" json:"name"`
	Slug       string     `gorm:"type:text;not null;unique" json:"slug"`
	UIMetadata base.JSONB `gorm:"type:jsonb" json:"ui_metadata,omitempty"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.InboundLeadFormWithFields
	// - relations.InboundLeadFormWithListContacts
	// - relations.InboundLeadFormWithFullRelations
	Fields []FormField `gorm:"foreignKey:FormID;constraint:OnDelete:CASCADE" json:"fields,omitempty"`
}

func (InboundLeadForm) TableName() string {
	return "inbound_lead_forms"
}

// InboundLeadFormListContactAssociation maps lead forms to contact lists.
type InboundLeadFormListContactAssociation struct {
	base.Base

	InboundLeadFormID uuid.UUID `gorm:"type:uuid;not null" json:"inbound_lead_form_id"`
	ListContactID     uuid.UUID `gorm:"type:uuid;not null" json:"list_contact_id"`
}

func (InboundLeadFormListContactAssociation) TableName() string {
	return "inbound_lead_form_list_contact_association"
}
