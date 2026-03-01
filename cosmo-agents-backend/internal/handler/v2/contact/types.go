package contact

import (
	"time"

	"github.com/google/uuid"
)

// SearchRequest represents the contact search request body
type SearchRequest struct {
	Filter map[string]interface{} `json:"filter"`
}

// InboundLeadFormEntity represents inbound lead form info in contact response
type InboundLeadFormEntity struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ContactEntity represents a contact in the response
type ContactEntity struct {
	ID              uuid.UUID              `json:"id"`
	UserID          uuid.UUID              `json:"user_id"`
	Name            *string                `json:"name"`
	Email           *string                `json:"email"`
	Phone           *string                `json:"phone"`
	Company         *string                `json:"company"`
	JobTitle        *string                `json:"job_title"`
	Address         *string                `json:"address"`
	City            *string                `json:"city"`
	Country         *string                `json:"country"`
	State           *string                `json:"state"`
	Zip             *string                `json:"zip"`
	SourceID        *string                `json:"source_id"`
	Source          *string                `json:"source"`
	HubspotID       *string                `json:"hubspot_id"`
	OrganizationID  *uuid.UUID             `json:"organization_id"`
	Tags            map[string]interface{} `json:"tags"`
	InboundLeadForm *InboundLeadFormEntity `json:"inbound_lead_form"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`

	// Added by info - tracks which BD added this contact
	AddedByName  *string `json:"added_by_name,omitempty"`
	AddedByEmail *string `json:"added_by_email,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

type ContactEntityResponseItem struct {
	Entity ContactEntity `json:"entity"`
}

// BatchUpdateRequest represents the batch update request
type BatchUpdateRequest struct {
	Emails []string               `json:"emails" validate:"required,min=1"`
	Fields map[string]interface{} `json:"fields" validate:"required"`
}
