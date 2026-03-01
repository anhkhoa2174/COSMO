package v1

import (
	"time"

	"github.com/google/uuid"
)

// Task enqueue request schemas

// EnqueueSendEmailRequest represents request to enqueue an email sending task
type EnqueueSendEmailRequest struct {
	AgentID     uuid.UUID  `json:"agent_id" validate:"required"`
	ContactID   uuid.UUID  `json:"contact_id" validate:"required"`
	CampaignID  *uuid.UUID `json:"campaign_id,omitempty"`
	TemplateID  *uuid.UUID `json:"template_id,omitempty"`
	TaskID      *uuid.UUID `json:"task_id,omitempty"`
	To          string     `json:"to" validate:"required,email"`
	Cc          string     `json:"cc,omitempty" validate:"omitempty,email"`
	Bcc         string     `json:"bcc,omitempty" validate:"omitempty,email"`
	Subject     string     `json:"subject" validate:"required,min=1"`
	Body        string     `json:"body" validate:"required,min=1"`
	IsHTML      bool       `json:"is_html"`
	InReplyTo   string     `json:"in_reply_to,omitempty"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
}

// EnqueueExecuteCampaignRequest represents request to execute a campaign
type EnqueueExecuteCampaignRequest struct {
	CampaignID uuid.UUID   `json:"campaign_id" validate:"required"`
	AgentID    uuid.UUID   `json:"agent_id" validate:"required"`
	ContactIDs []uuid.UUID `json:"contact_ids,omitempty"` // Optional: specific contacts
}

// EnqueueScheduleTasksRequest represents request to schedule campaign tasks
type EnqueueScheduleTasksRequest struct {
	CampaignID uuid.UUID   `json:"campaign_id" validate:"required"`
	ContactIDs []uuid.UUID `json:"contact_ids" validate:"required,min=1"`
	AgentID    uuid.UUID   `json:"agent_id" validate:"required"`
}

// EnqueueSyncAgentRequest represents request to sync agent credentials
type EnqueueSyncAgentRequest struct {
	AgentID uuid.UUID `json:"agent_id" validate:"required"`
}

// EnqueueGenerateEmailRequest represents request to generate AI email
type EnqueueGenerateEmailRequest struct {
	CampaignID uuid.UUID              `json:"campaign_id" validate:"required"`
	ContactID  uuid.UUID              `json:"contact_id" validate:"required"`
	TemplateID uuid.UUID              `json:"template_id" validate:"required"`
	TaskID     *uuid.UUID             `json:"task_id,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}
