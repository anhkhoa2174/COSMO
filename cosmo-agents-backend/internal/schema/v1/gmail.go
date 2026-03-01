package v1

import "github.com/google/uuid"

// Gmail API schemas

// GmailSendRequest represents request to send email via Gmail API
type GmailSendRequest struct {
	AgentID   uuid.UUID `json:"agent_id" validate:"required"`
	To        string    `json:"to" validate:"required,email"`
	Cc        string    `json:"cc,omitempty" validate:"omitempty,email"`
	Bcc       string    `json:"bcc,omitempty" validate:"omitempty,email"`
	Subject   string    `json:"subject" validate:"required,min=1"`
	Body      string    `json:"body" validate:"required,min=1"`
	IsHTML    bool      `json:"is_html"`
	InReplyTo string    `json:"in_reply_to,omitempty"` // Message-ID for threading
}

// GmailAuthURLRequest represents request to get OAuth2 URL
type GmailAuthURLRequest struct {
	AgentID uuid.UUID `json:"agent_id" validate:"required"`
}

// GmailRefreshTokenRequest represents request to refresh OAuth2 token
type GmailRefreshTokenRequest struct {
	AgentID uuid.UUID `json:"agent_id" validate:"required"`
}
