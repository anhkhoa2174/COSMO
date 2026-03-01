package dto

import (
	"time"

	"github.com/google/uuid"
)

// SendEmailPayload represents the payload for sending an email.
type SendEmailPayload struct {
	AgentID     uuid.UUID  `json:"agent_id"`
	ContactID   uuid.UUID  `json:"contact_id"`
	CampaignID  *uuid.UUID `json:"campaign_id,omitempty"`
	TemplateID  *uuid.UUID `json:"template_id,omitempty"`
	TaskID      *uuid.UUID `json:"task_id,omitempty"`
	To          string     `json:"to"`
	Cc          string     `json:"cc,omitempty"`
	Bcc         string     `json:"bcc,omitempty"`
	Subject     string     `json:"subject"`
	Body        string     `json:"body"`
	IsHTML      bool       `json:"is_html"`
	InReplyTo   string     `json:"in_reply_to,omitempty"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
}

// ProcessIncomingEmailPayload represents payload for processing incoming email.
type ProcessIncomingEmailPayload struct {
	AgentID        uuid.UUID `json:"agent_id"`
	GmailMessageID string    `json:"gmail_message_id"`
	GmailThreadID  string    `json:"gmail_thread_id"`
}

// SyncGmailHistoryPayload represents payload for syncing Gmail history.
type SyncGmailHistoryPayload struct {
	AgentID        uuid.UUID `json:"agent_id"`
	StartHistoryID string    `json:"start_history_id"`
}

type SendInviteMemberEmailPayload struct {
	MemberName string `json:"member_name"`
	Email      string `json:"email"`
	Url        string `json:"url"`
}
