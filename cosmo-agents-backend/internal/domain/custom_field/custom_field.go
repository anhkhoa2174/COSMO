package custom_field

import (
	"errors"
	"strings"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// CustomFieldEntity enumerates entities that can hold custom fields.
type CustomFieldEntity string

const (
	CustomFieldEntityContact CustomFieldEntity = "contact"
	CustomFieldEntityCompany CustomFieldEntity = "company"
)

// CustomFieldDataType represents the supported data types.
type CustomFieldDataType string

const (
	CustomFieldDataTypeText   CustomFieldDataType = "text"
	CustomFieldDataTypeNumber CustomFieldDataType = "number"
	CustomFieldDataTypeEmail  CustomFieldDataType = "email"
	CustomFieldDataTypeSelect CustomFieldDataType = "select"
	CustomFieldDataTypeDate   CustomFieldDataType = "date"
	CustomFieldDataTypeURL    CustomFieldDataType = "url"
)

// allowedDataTypes for quick validation lookup.
var allowedDataTypes = map[CustomFieldDataType]struct{}{
	CustomFieldDataTypeText:   {},
	CustomFieldDataTypeNumber: {},
	CustomFieldDataTypeEmail:  {},
	CustomFieldDataTypeSelect: {},
	CustomFieldDataTypeDate:   {},
	CustomFieldDataTypeURL:    {},
}

// CustomField mirrors machine/models/custom_field.py.
type CustomField struct {
	base.Base
	base.TimestampMixin

	UserID         uuid.UUID           `gorm:"type:uuid;not null;index:idx_custom_fields_user_id" json:"user_id"`
	OrganizationID *uuid.UUID          `gorm:"type:uuid" json:"organization_id,omitempty"`
	Name           string              `gorm:"type:text;not null" json:"name"`
	NormalizedName string              `gorm:"type:text;not null" json:"normalized_name"`
	DataType       CustomFieldDataType `gorm:"type:text;not null" json:"data_type"`
	EntityType     CustomFieldEntity   `gorm:"type:text;not null" json:"entity_type"`
	IsRequired     bool                `gorm:"not null;default:false" json:"is_required"`
	Options        pq.StringArray      `gorm:"type:text[]" json:"options,omitempty"`
	SampleData     *string             `gorm:"type:text" json:"sample_data,omitempty"`
	FallbackValue  *string             `gorm:"type:text" json:"fallback_value,omitempty"`
}

func (CustomField) TableName() string {
	return "custom_fields"
}

// BeforeCreate normalizes data and ensures defaults.
func (cf *CustomField) BeforeCreate(tx *gorm.DB) error {
	if err := cf.Base.BeforeCreate(tx); err != nil {
		return err
	}
	return cf.normalizeAndValidate()
}

// BeforeUpdate ensures updates keep normalized values in sync.
func (cf *CustomField) BeforeUpdate(tx *gorm.DB) error {
	return cf.normalizeAndValidate()
}

// normalizeAndValidate is shared between create/update hooks.
func (cf *CustomField) normalizeAndValidate() error {
	cf.Name = strings.TrimSpace(cf.Name)
	if cf.Name == "" {
		return errors.New("custom field name cannot be empty")
	}

	cf.NormalizedName = normalizeCustomFieldName(cf.Name)
	if _, ok := allowedDataTypes[cf.DataType]; !ok {
		return errors.New("invalid data_type for custom field")
	}

	if cf.Options == nil {
		cf.Options = pq.StringArray{}
	}

	if cf.EntityType != CustomFieldEntityContact && cf.EntityType != CustomFieldEntityCompany {
		return errors.New("invalid entity_type for custom field")
	}

	return nil
}

func normalizeCustomFieldName(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "_")
	return slug
}
