package v1

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// CreateCustomFieldRequest represents the request to create a custom field
type CreateCustomFieldRequest struct {
	Name          string                     `json:"name" validate:"required"`
	DataType      domain.CustomFieldDataType `json:"data_type" validate:"required"`
	EntityType    domain.CustomFieldEntity   `json:"entity_type" validate:"required"`
	IsRequired    bool                       `json:"is_required"`
	Options       []string                   `json:"options,omitempty"`
	SampleData    *string                    `json:"sample_data,omitempty"`
	FallbackValue *string                    `json:"fallback_value,omitempty"`
}

// UpdateCustomFieldRequest represents the request to update a custom field
type UpdateCustomFieldRequest struct {
	Name          *string                     `json:"name,omitempty"`
	DataType      *domain.CustomFieldDataType `json:"data_type,omitempty"`
	EntityType    *domain.CustomFieldEntity   `json:"entity_type,omitempty"`
	IsRequired    *bool                       `json:"is_required,omitempty"`
	Options       []string                    `json:"options,omitempty"`
	SampleData    *string                     `json:"sample_data,omitempty"`
	FallbackValue *string                     `json:"fallback_value,omitempty"`
}

// CustomFieldResponse represents a custom field entity
type CustomFieldResponse struct {
	ID             uuid.UUID                  `json:"id"`
	UserID         uuid.UUID                  `json:"user_id"`
	OrganizationID *uuid.UUID                 `json:"organization_id"`
	Name           string                     `json:"name"`
	NormalizedName string                     `json:"normalized_name"`
	DataType       domain.CustomFieldDataType `json:"data_type"`
	EntityType     domain.CustomFieldEntity   `json:"entity_type"`
	IsRequired     bool                       `json:"is_required"`
	Options        []string                   `json:"options"`
	SampleData     *string                    `json:"sample_data"`
	FallbackValue  *string                    `json:"fallback_value"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

// CustomFieldListResponse represents paginated custom field results
type CustomFieldListResponse struct {
	List   []CustomFieldResponse `json:"list"`
	Offset int                   `json:"offset"`
	Limit  int                   `json:"limit"`
	Total  int64                 `json:"total"`
}
