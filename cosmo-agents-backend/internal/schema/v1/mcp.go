package v1

import (
	"github.com/google/uuid"
)

// MCPContactCreateRequest represents a contact in the MCP campaign creation request
type MCPContactCreateRequest struct {
	Name        string                 `json:"name" validate:"required"`
	Email       string                 `json:"email" validate:"required,email"`
	Phone       *string                `json:"phone,omitempty"`
	Company     *string                `json:"company,omitempty"`
	JobTitle    *string                `json:"job_title,omitempty"`
	LinkedInURL *string                `json:"linkedin_url,omitempty"`
	Address     *string                `json:"address,omitempty"`
	City        *string                `json:"city,omitempty"`
	Country     *string                `json:"country,omitempty"`
	State       *string                `json:"state,omitempty"`
	Zip         *string                `json:"zip,omitempty"`
	Tags        map[string]interface{} `json:"tags,omitempty"`
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

// ============================================
// MCP AI-Native Tools (Agentic Daily Actions)
// ============================================

// MCPContactPipelineItem represents a contact with its outreach pipeline state.
type MCPContactPipelineItem struct {
	ContactID         uuid.UUID `json:"contact_id"`
	Name              string    `json:"name"`
	Email             string    `json:"email"`
	Company           string    `json:"company"`
	JobTitle          string    `json:"job_title"`
	Industry          string    `json:"industry"`
	Source            string    `json:"source"`
	OutreachStage     string    `json:"outreach_stage"`
	BusinessStage     string    `json:"business_stage"`
	ConversationState string    `json:"conversation_state"`
	NextStep          string    `json:"next_step"`
	DaysSinceContact  int       `json:"days_since_contact"`
	FollowupCount     int       `json:"followup_count"`
	ContextLevel      string    `json:"context_level"`
	MessageDraft      string    `json:"message_draft,omitempty"`
	Type              string    `json:"type"`
}

// MCPContactsPipelineResponse wraps the pipeline contacts list.
type MCPContactsPipelineResponse struct {
	Contacts []MCPContactPipelineItem `json:"contacts"`
	Total    int                      `json:"total"`
}

// MCPInteractionItem represents a single interaction in the history.
type MCPInteractionItem struct {
	ID        uuid.UUID `json:"id"`
	Channel   string    `json:"channel"`
	Direction string    `json:"direction"`
	Content   string    `json:"content"`
	Subject   *string   `json:"subject,omitempty"`
	Sentiment *string   `json:"sentiment,omitempty"`
	Timestamp string    `json:"timestamp"`
}

// MCPContactInteractionsResponse wraps interactions for a contact.
type MCPContactInteractionsResponse struct {
	ContactID    uuid.UUID            `json:"contact_id"`
	ContactName  string               `json:"contact_name"`
	Interactions []MCPInteractionItem `json:"interactions"`
	Total        int                  `json:"total"`
}

// MCPDailyActionsStatusItem represents a single daily action for the status response.
type MCPDailyActionsStatusItem struct {
	ActionID   uuid.UUID `json:"action_id"`
	ContactID  uuid.UUID `json:"contact_id"`
	Name       string    `json:"contact_name"`
	Company    string    `json:"company"`
	ActionType string    `json:"action_type"`
	CategoryID string    `json:"category_id"`
	Priority   int       `json:"priority"`
	Status     string    `json:"status"`
	Reasoning  string    `json:"reasoning"`
}

// MCPDailyActionsStatusResponse wraps the daily actions status.
type MCPDailyActionsStatusResponse struct {
	GenerationID *uuid.UUID                  `json:"generation_id,omitempty"`
	Date         string                      `json:"date"`
	Status       string                      `json:"status"`
	Actions      []MCPDailyActionsStatusItem `json:"actions"`
	TotalActions int                         `json:"total_actions"`
	Completed    int                         `json:"completed"`
	Pending      int                         `json:"pending"`
}
