package v1

import (
	"github.com/google/uuid"
)

// MCPContactCreateRequest represents a contact in the MCP campaign creation request
type MCPContactCreateRequest struct {
	Name     string                 `json:"name" validate:"required"`
	Email    string                 `json:"email" validate:"required,email"`
	Phone    *string                `json:"phone,omitempty"`
	Company  *string                `json:"company,omitempty"`
	JobTitle *string                `json:"job_title,omitempty"`
	Address  *string                `json:"address,omitempty"`
	City     *string                `json:"city,omitempty"`
	Country  *string                `json:"country,omitempty"`
	State    *string                `json:"state,omitempty"`
	Zip      *string                `json:"zip,omitempty"`
	Tags     map[string]interface{} `json:"tags,omitempty"`
}

// MCPCampaignCreateRequest represents POST /v1/mcp/campaigns request body
type MCPCampaignCreateRequest struct {
	Contacts []MCPContactCreateRequest `json:"contacts" validate:"required,min=1,dive"`
	Name     *string                   `json:"name,omitempty"`
	Playbook string                    `json:"playbook" validate:"required"`
}

// MCPCampaignCreateResponse represents POST /v1/mcp/campaigns response
type MCPCampaignCreateResponse struct {
	ID              uuid.UUID  `json:"id"`
	CampaignURL     string     `json:"campaign_url"`
	Name            *string    `json:"name,omitempty"`
	ContactListID   *uuid.UUID `json:"contact_list_id,omitempty"`
	Playbook        *string    `json:"playbook,omitempty"`
	ContactsCreated int        `json:"contacts_created"`
}
