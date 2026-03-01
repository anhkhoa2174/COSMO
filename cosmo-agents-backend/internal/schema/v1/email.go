package v1

import (
	"time"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// Email request/response schemas

// SendEmailRequest represents the request body for sending an email
type SendEmailRequest struct {
	ConversationID *uuid.UUID  `json:"conversation_id,omitempty"`
	CampaignID     *uuid.UUID  `json:"campaign_id,omitempty"`
	FromEmail      string      `json:"from_email" validate:"required,email"`
	ToEmail        string      `json:"to_email" validate:"required,email"`
	Subject        string      `json:"subject" validate:"required,min=1"`
	Content        string      `json:"content" validate:"required,min=1"`
	Attachments    []uuid.UUID `json:"attachments,omitempty"`
	Labels         []string    `json:"labels,omitempty"`
	Intents        []string    `json:"intents,omitempty"`
}

// EmailResponse matches machine.api.v1.email.schema.EmailResponse
type EmailResponse struct {
	ID         uuid.UUID         `json:"id"`
	FromEmail  *string           `json:"from_email,omitempty"`
	ToEmail    *string           `json:"to_email,omitempty"`
	Subject    *string           `json:"subject,omitempty"`
	ScheduleAt *time.Time        `json:"schedule_at"`
	DoneAt     *time.Time        `json:"done_at,omitempty"`
	Status     domain.TaskStatus `json:"status"`
	Error      *string           `json:"error,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

// EmailDetailResponse matches machine.api.v1.email.schema.EmailDetailResponse
type EmailDetailResponse struct {
	ID         uuid.UUID         `json:"id"`
	FromEmail  *string           `json:"from_email,omitempty"`
	ToEmail    *string           `json:"to_email,omitempty"`
	Subject    *string           `json:"subject,omitempty"`
	Content    *string           `json:"content,omitempty"`
	ScheduleAt *time.Time        `json:"schedule_at"`
	DoneAt     *time.Time        `json:"done_at,omitempty"`
	Status     domain.TaskStatus `json:"status"`
	Error      *string           `json:"error,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}
