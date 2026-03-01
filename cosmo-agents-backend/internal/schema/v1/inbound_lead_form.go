package v1

import (
	"github.com/google/uuid"
)

type InboundLeadFormFieldRequest struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	IsRequired  *bool          `json:"is_required"`
	UIMetadata  map[string]any `json:"ui_metadata"`
}

type InboundLeadFormCreateRequest struct {
	Name       string                        `json:"name"`
	Slug       string                        `json:"slug"`
	Fields     []InboundLeadFormFieldRequest `json:"fields"`
	UIMetadata map[string]any                `json:"ui_metadata"`
	// Optional list contact IDs to associate this form with existing lists
	ListContactIDs []uuid.UUID `json:"list_contact_ids,omitempty"`
}

type InboundLeadFormUpdateRequest struct {
	Name       string                        `json:"name"`
	Fields     []InboundLeadFormFieldRequest `json:"fields"`
	UIMetadata map[string]any                `json:"ui_metadata"`
}

type InboundLeadFormFieldResponse struct {
	Name          string         `json:"name"`
	DisplayName   string         `json:"display_name"`
	FieldType     string         `json:"field_type"`
	IsRequired    bool           `json:"is_required"`
	SelectOptions []string       `json:"select_options,omitempty"`
	FallbackValue any            `json:"fallback_value,omitempty"`
	UIMetadata    map[string]any `json:"ui_metadata,omitempty"`
}

type InboundLeadFormResponse struct {
	ID         uuid.UUID                      `json:"id"`
	Name       string                         `json:"name"`
	Slug       string                         `json:"slug"`
	Fields     []InboundLeadFormFieldResponse `json:"fields"`
	UIMetadata map[string]any                 `json:"ui_metadata,omitempty"`
}

type InboundLeadFormListItem struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}

type LeadFormSubmitResponse struct {
	ContactID uuid.UUID      `json:"contact_id"`
	Data      map[string]any `json:"data"`
}
