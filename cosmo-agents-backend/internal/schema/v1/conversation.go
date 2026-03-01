package v1

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Conversation request/response schemas

// CreateConversationRequest represents the request body for creating a conversation
type CreateConversationRequest struct {
	GmailThreadID string         `json:"gmail_thread_id" validate:"required"`
	CampaignID    *uuid.UUID     `json:"campaign_id,omitempty"`
	AgentID       *uuid.UUID     `json:"agent_id,omitempty"`
	AssigneeID    *uuid.UUID     `json:"assignee_id,omitempty"`
	Labels        []string       `json:"labels,omitempty"`
	Intents       []string       `json:"intents,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// UpdateConversationRequest represents the request body for updating a conversation
type UpdateConversationRequest struct {
	Labels     *[]string      `json:"labels,omitempty"`
	Intents    *[]string      `json:"intents,omitempty"`
	Status     *string        `json:"status,omitempty" validate:"omitempty,oneof=read unread"`
	Replied    *bool          `json:"replied,omitempty"`
	AssigneeID *uuid.UUID     `json:"assignee_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// ConversationResponse represents the response for a conversation
type ConversationResponse struct {
	ID            uuid.UUID      `json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	UserID        uuid.UUID      `json:"user_id"`
	GmailThreadID string         `json:"gmail_thread_id"`
	CampaignID    *uuid.UUID     `json:"campaign_id"`
	AgentID       *uuid.UUID     `json:"agent_id"`
	AssigneeID    *uuid.UUID     `json:"assignee_id"`
	Labels        pq.StringArray `json:"labels"`
	Intents       pq.StringArray `json:"intents"`
	Replied       bool           `json:"replied"`
	Status        string         `json:"status"`
	Metadata      map[string]any `json:"metadata"`
	IsDeleted     bool           `json:"is_deleted"`
}

// ConversationListResponse represents a paginated list of conversations
type ConversationListResponse struct {
	Items      []ConversationResponse `json:"items"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"page_size"`
	TotalPages int                    `json:"total_pages"`
}

// MarkAsReadRequest represents marking a conversation as read
type MarkAsReadRequest struct {
	Read bool `json:"read"` // true = mark as read, false = mark as unread
}

// AssignConversationRequest represents assigning a conversation to a sales rep
type AssignConversationRequest struct {
	AssigneeID uuid.UUID `json:"assignee_id" validate:"required"`
}

// ConversationSearchRequest represents the request body for searching conversations
type ConversationSearchRequest struct {
	Filter map[string]interface{} `json:"filter,omitempty"`
}

// ConversationListItem represents a conversation with its latest email
type ConversationListItem struct {
	Entity      ConversationEntity `json:"entity"`
	LatestEmail EmailEntity        `json:"latest_email"`
}

// ConversationSearchResponse represents the response for conversation search
type ConversationSearchResponse struct {
	List   []ConversationListItem `json:"list"`
	Offset int                    `json:"offset"`
	Limit  int                    `json:"limit"`
	Total  int64                  `json:"total"`
}

// ConversationReadResponse represents detailed conversation with emails
type ConversationReadResponse struct {
	ID         uuid.UUID      `json:"id"`
	UserID     uuid.UUID      `json:"user_id"`
	Labels     pq.StringArray `json:"labels"`
	Replied    bool           `json:"replied"`
	CampaignID *uuid.UUID     `json:"campaign_id"`
	AssigneeID *uuid.UUID     `json:"assignee_id"`
	Status     string         `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	Emails     []EmailEntity  `json:"emails"`
	IsDeleted  bool           `json:"is_deleted"`
}

// AssigneeListResponse represents a paginated list of assignees
type AssigneeListResponse struct {
	List   []AssigneeEntity `json:"list"`
	Offset int              `json:"offset"`
	Limit  int              `json:"limit"`
	Total  int64            `json:"total"`
}

// AssigneeEntity represents an assignee (simplified user entity)
type AssigneeEntity struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
}
