package v2

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// EmailSearchRequest represents the request for POST /v2/emails/search
type EmailSearchRequest struct {
	Filter map[string]interface{} `json:"filter"`
}

// EmailEntity represents an email entity
type EmailEntity struct {
	ID             uuid.UUID  `json:"id"`
	Subject        *string    `json:"subject"`
	Content        *string    `json:"content"`
	FromEmail      *string    `json:"from_email"`
	ToEmail        *string    `json:"to_email"`
	Attachments    []string   `json:"attachments"`
	Intents        []string   `json:"intents"`
	GmailMessageID *string    `json:"gmail_message_id"`
	ConversationID *uuid.UUID `json:"conversation_id"`
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

// EmailListItem represents a single item in the email list
type EmailListItem struct {
	Entity EmailEntity `json:"entity"`
}

// EmailSearchResponse represents the response for POST /v2/emails/search
type EmailSearchResponse = schema.PaginatedResponse[EmailListItem]

// EmailReplyRequest represents the request for POST /v2/emails/{email_id}/reply
type EmailReplyRequest struct {
	Subject *string  `json:"subject"`
	Content string   `json:"content" validate:"required"`
	CC      []string `json:"cc"`
}
