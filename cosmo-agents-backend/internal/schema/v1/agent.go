package v1

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Agent request/response schemas

// CreateAgentRequest represents the request body for creating an agent
type CreateAgentRequest struct {
	Name           string         `json:"name,omitempty" validate:"required,min=1,max=255"`
	Email          string         `json:"email,omitempty" validate:"required,email"`
	OrganizationID *uuid.UUID     `json:"organization_id,omitempty"`
	Persona        []string       `json:"persona,omitempty"`
	EmailProvider  string         `json:"email_provider,omitempty" validate:"omitempty,oneof=gmail outlook"`
	Signature      string         `json:"signature,omitempty"`
	Picture        string         `json:"picture,omitempty"`
	Credentials    map[string]any `json:"credentials,omitempty"`
	DailyLimit     *int           `json:"daily_limit,omitempty" validate:"omitempty,min=1,max=1000"`
	MaxDailyLimit  *int           `json:"max_daily_limit,omitempty" validate:"omitempty,min=1,max=1000"`
	CMetadata      map[string]any `json:"cmetadata,omitempty"`
}

// UpdateAgentRequest represents the request body for updating an agent
type UpdateAgentRequest struct {
	Name          *string         `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	DailyLimit    *int            `json:"daily_limit,omitempty" validate:"omitempty,min=1,max=1000"`
	WorkingHours  *map[string]any `json:"working_hours,omitempty"`
	Persona       *[]string       `json:"persona,omitempty"`
	Email         *string         `json:"email,omitempty" validate:"omitempty,email"`
	EmailProvider *string         `json:"email_provider,omitempty" validate:"omitempty,oneof=gmail outlook"`
	Status        *string         `json:"status,omitempty" validate:"omitempty,oneof=active inactive suspended"`
	Signature     *string         `json:"signature,omitempty"`
	Picture       *string         `json:"picture,omitempty"`
}

// AgentResponse represents the response for an agent
type AgentResponse struct {
	ID              uuid.UUID      `json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	UserID          uuid.UUID      `json:"user_id"`
	OrganizationID  *uuid.UUID     `json:"organization_id,omitempty"`
	Name            string         `json:"name"`
	Email           string         `json:"email"`
	Persona         pq.StringArray `json:"persona,omitempty"`
	EmailProvider   string         `json:"email_provider,omitempty"`
	Signature       string         `json:"signature,omitempty"`
	Picture         string         `json:"picture,omitempty"`
	Status          string         `json:"status"`
	DailyLimit      *int           `json:"daily_limit,omitempty"`
	MaxDailyLimit   *int           `json:"max_daily_limit,omitempty"`
	ValidCred       *bool          `json:"valid_cred,omitempty"`
	EmailsSentToday *int           `json:"emails_sent_today,omitempty"`
	LastHistoryID   string         `json:"last_history_id,omitempty"`
	CMetadata       map[string]any `json:"cmetadata,omitempty"`
	IsDeleted       bool           `json:"is_deleted"`
}

// AgentInboxDetail contains summary information about the agent's inbox
type AgentInboxDetail struct {
	// ConnectedDate is when the inbox was connected
	ConnectedDate time.Time `json:"connected_date"`
	// ConnectedInbox is the connected email address
	ConnectedInbox string `json:"connected_inbox"`
	// EmailProvider is the provider name (e.g., gmail, outlook)
	EmailProvider string `json:"email_provider"`
	// AddedBy displays who added/connected this inbox (e.g., user email or name)
	AddedBy string `json:"added_by"`
}

// AgentEmailStatistics aggregates basic email metrics for the agent
type AgentEmailStatistics struct {
	// Sent total emails sent by this agent
	Sent int `json:"sent"`
	// ReplyRate fraction 0..1 of emails that got a reply
	ReplyRate float64 `json:"reply_rate"`
	// OpenRate fraction 0..1 of emails that were opened
	OpenRate float64 `json:"open_rate"`
	// BounceRate fraction 0..1 of emails that bounced
	BounceRate float64 `json:"bounce_rate"`
}

// AgentGetResponse wraps the agent entity and related computed info
type AgentGetResponse struct {
	Entity          AgentResponse        `json:"entity"`
	InboxDetail     AgentInboxDetail     `json:"inbox_detail"`
	EmailStatistics AgentEmailStatistics `json:"email_statistics"`
}

// AgentListResponse represents a paginated list of agents
type AgentListResponse struct {
	Items      []AgentResponse `json:"items"`
	Total      int64           `json:"total"`
	Page       int             `json:"page"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages"`
}

// AgentSearchRequest represents the search request payload
type AgentSearchRequest struct {
	Filter map[string]interface{} `json:"filter,omitempty"`
}

// AgentListItem wraps an agent entity for search listings
type AgentListItem struct {
	Entity AgentResponse `json:"entity"`
}

// AgentSearchResponse represents paginated search results for agents
type AgentSearchResponse struct {
	List   []AgentListItem `json:"list"`
	Offset int             `json:"offset"`
	Limit  int             `json:"limit"`
	Total  int64           `json:"total"`
}

// SyncAgentRequest represents the request to sync an agent's emails
type SyncAgentRequest struct {
	FullSync bool `json:"full_sync,omitempty"` // If true, sync all emails; otherwise, incremental sync
}

// SyncAgentResponse represents the response after syncing an agent
type SyncAgentResponse struct {
	Message       string    `json:"message"`
	EmailsSynced  int       `json:"emails_synced"`
	TasksEnqueued int       `json:"tasks_enqueued"`
	SyncedAt      time.Time `json:"synced_at"`
}

// AgentGetConversationRequest represents the request to search agent conversations
type AgentGetConversationRequest struct {
	Filter map[string]any `json:"filter,omitempty"`
}

// ConversationEntity represents a conversation entity matching Python structure
type ConversationEntity struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	Labels     []string   `json:"labels"`
	Replied    bool       `json:"replied"`
	CampaignID *uuid.UUID `json:"campaign_id"`
	AssigneeID *uuid.UUID `json:"assignee_id"`
	Intents    []string   `json:"intents"`
	Status     string     `json:"status"`
	IsDeleted  bool       `json:"is_deleted"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// EmailEntity represents an email entity matching Python structure
type EmailEntity struct {
	ID          *uuid.UUID `json:"id"`
	Subject     *string    `json:"subject"`
	Content     *string    `json:"content"`
	FromEmail   *string    `json:"from_email"`
	ToEmail     *string    `json:"to_email"`
	Attachments []string   `json:"attachments"`
	Intents     []string   `json:"intents"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

// AgentGetConversationsResponse represents the response structure matching Python
type AgentGetConversationsResponse struct {
	Entity      ConversationEntity `json:"entity"`
	LatestEmail EmailEntity        `json:"latest_email"`
}
