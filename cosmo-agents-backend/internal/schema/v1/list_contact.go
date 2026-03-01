package v1

import (
	"time"

	"github.com/google/uuid"
)

// InboundLeadFormEntity represents an inbound lead form in responses
type InboundLeadFormEntity struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	UIMetadata any       `json:"ui_metadata,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CreateListContactRequest represents the request to create a list contact
type CreateListContactRequest struct {
	Name               string      `json:"name" validate:"required"`
	Source             *string     `json:"source,omitempty"`
	SourceID           *string     `json:"source_id,omitempty"`
	HubspotID          *string     `json:"hubspot_id,omitempty"`
	InboundLeadFormIDs []uuid.UUID `json:"inbound_lead_form_ids,omitempty"`
	ContactIDs         []uuid.UUID `json:"contact_ids,omitempty"`
}

// UpdateListContactRequest represents the request to update a list contact
type UpdateListContactRequest struct {
	Name      *string `json:"name,omitempty"`
	Source    *string `json:"source,omitempty"`
	SourceID  *string `json:"source_id,omitempty"`
	HubspotID *string `json:"hubspot_id,omitempty"`
	// Allow adding inbound lead form associations to an existing list
	InboundLeadFormIDs []uuid.UUID `json:"inbound_lead_form_ids,omitempty"`
	ContactIDs         []uuid.UUID `json:"contact_ids,omitempty"`
}

// ListContactResponse represents a list contact entity
type ListContactResponse struct {
	ID               uuid.UUID               `json:"id"`
	Name             string                  `json:"name"`
	Source           *string                 `json:"source,omitempty"`
	SourceID         *string                 `json:"source_id,omitempty"`
	HubspotID        *string                 `json:"hubspot_id,omitempty"`
	InboundLeadForms []InboundLeadFormEntity `json:"inbound_lead_forms"`
	Contacts         []ContactResponse       `json:"contacts"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
}

// ListContactSearchResponseEntity represents a simplified list contact entity for search (without relationships)
type ListContactSearchResponseEntity struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Source    *string   `json:"source,omitempty"`
	SourceID  *string   `json:"source_id,omitempty"`
	HubspotID *string   `json:"hubspot_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListContactSearchRequest represents the search request
type ListContactSearchRequest struct {
	Filter map[string]interface{} `json:"filter"`
}

// ListContactListItem wraps a list contact entity with metadata
type ListContactListItem struct {
	Entity               ListContactSearchResponseEntity `json:"entity"`
	Creator              string                          `json:"creator"`
	NumberOfInboundForms int                             `json:"number_of_inbound_forms"`
	Size                 int                             `json:"size"`
}

// ListContactSearchResponse represents paginated search results
type ListContactSearchResponse struct {
	List   []ListContactListItem `json:"list"`
	Offset int                   `json:"offset"`
	Limit  int                   `json:"limit"`
	Total  int64                 `json:"total"`
}

// DeleteListContactRequest represents delete request
type DeleteListContactRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1"`
}
