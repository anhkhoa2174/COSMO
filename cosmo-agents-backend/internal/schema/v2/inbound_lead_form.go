package v2

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// InboundLeadFormFieldRequest represents a field in the inbound lead form request
type InboundLeadFormFieldRequest struct {
	Name        string         `json:"name" validate:"required"`
	DisplayName string         `json:"display_name" validate:"required"`
	IsRequired  *bool          `json:"is_required"`
	UIMetadata  map[string]any `json:"ui_metadata"`
}

// InboundLeadFormCreateRequest represents the request for POST /v2/inbound-lead-forms
type InboundLeadFormCreateRequest struct {
	Name           string                        `json:"name" validate:"required"`
	Slug           string                        `json:"slug" validate:"required"`
	Fields         []InboundLeadFormFieldRequest `json:"fields" validate:"required,dive"`
	UIMetadata     map[string]any                `json:"ui_metadata"`
	ListContactIDs []uuid.UUID                   `json:"list_contact_ids,omitempty"`
}

// InboundLeadFormUpdateRequest represents the request for PUT /v2/inbound-lead-forms/{id}
type InboundLeadFormUpdateRequest struct {
	Name       string                        `json:"name" validate:"required"`
	Fields     []InboundLeadFormFieldRequest `json:"fields" validate:"required,dive"`
	UIMetadata map[string]any                `json:"ui_metadata"`
}

// InboundLeadFormFieldResponse represents a field in the inbound lead form response
type InboundLeadFormFieldResponse struct {
	Name          string         `json:"name"`
	DisplayName   string         `json:"display_name"`
	FieldType     string         `json:"field_type"`
	IsRequired    bool           `json:"is_required"`
	SelectOptions []string       `json:"select_options,omitempty"`
	FallbackValue any            `json:"fallback_value,omitempty"`
	UIMetadata    map[string]any `json:"ui_metadata,omitempty"`
}

// InboundLeadFormResponse represents the response for GET /v2/inbound-lead-forms/{id}
type InboundLeadFormResponse struct {
	ID         uuid.UUID                      `json:"id"`
	Name       string                         `json:"name"`
	Slug       string                         `json:"slug"`
	Fields     []InboundLeadFormFieldResponse `json:"fields"`
	UIMetadata map[string]any                 `json:"ui_metadata,omitempty"`
}

// InboundLeadFormListItem represents a single item in the inbound lead form list
type InboundLeadFormListItem struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

// LeadFormSubmitResponse represents the response for POST /v2/inbound-lead-forms/{id}/submit
type LeadFormSubmitResponse struct {
	ContactID uuid.UUID      `json:"contact_id"`
	Data      map[string]any `json:"data"`
}

// InboundLeadFormListResponse represents the paginated response for inbound lead forms
type InboundLeadFormListResponse = schema.PaginatedResponse[InboundLeadFormListItem]
