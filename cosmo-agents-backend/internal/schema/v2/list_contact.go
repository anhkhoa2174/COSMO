package v2

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// ListContactSearchRequest represents the request for POST /v2/list-contacts/search
type ListContactSearchRequest struct {
	Filter map[string]interface{} `json:"filter"`
}

// ListContactEntity represents a list contact entity
type ListContactEntity struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Source        *string    `json:"source"`
	SourceID      *string    `json:"source_id"`
	HubspotID     *string    `json:"hubspot_id"`
	ListContactID *uuid.UUID `json:"list_contact_id"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ListContactListItem represents a single item in the list contact search
type ListContactListItem struct {
	Entity           ListContactEntity `json:"entity"`
	Creator          string            `json:"creator"`
	NumberOfCampaign int               `json:"number_of_campaign"`
	Size             int               `json:"size"`
}

// ListContactSearchResponse represents the response for POST /v2/list-contacts/search
type ListContactSearchResponse = schema.PaginatedResponse[ListContactListItem]
