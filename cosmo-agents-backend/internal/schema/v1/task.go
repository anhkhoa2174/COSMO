package v1

import (
	"time"

	"github.com/google/uuid"
)

// Task request/response schemas

// CreateTaskRequest represents the request body for creating a task
type CreateTaskRequest struct {
	ContactID  uuid.UUID      `json:"contact_id" validate:"required"`
	CampaignID uuid.UUID      `json:"campaign_id" validate:"required"`
	TemplateID uuid.UUID      `json:"template_id" validate:"required"`
	ScheduleAt *time.Time     `json:"schedule_at,omitempty"`
	Priority   string         `json:"priority,omitempty" validate:"omitempty,oneof=low medium high critical"`
	Payload    map[string]any `json:"payload,omitempty"`
}

// UpdateTaskRequest represents the request body for updating a task
type UpdateTaskRequest struct {
	ScheduleAt *time.Time     `json:"schedule_at,omitempty"`
	Priority   *string        `json:"priority,omitempty" validate:"omitempty,oneof=low medium high critical"`
	Status     *string        `json:"status,omitempty" validate:"omitempty,oneof=pending running cancelled failed done"`
	Payload    map[string]any `json:"payload,omitempty"`
	Error      *string        `json:"error,omitempty"`
}

// TaskResponse represents the response for a task
type TaskResponse struct {
	ID          uuid.UUID      `json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	ContactID   uuid.UUID      `json:"contact_id"`
	CampaignID  uuid.UUID      `json:"campaign_id"`
	TemplateID  uuid.UUID      `json:"template_id"`
	ScheduleAt  *time.Time     `json:"schedule_at,omitempty"`
	RespondedAt *time.Time     `json:"responded_at,omitempty"`
	TriggeredAt *time.Time     `json:"triggered_at,omitempty"`
	DoneAt      *time.Time     `json:"done_at,omitempty"`
	Priority    string         `json:"priority"`
	Status      string         `json:"status"`
	Payload     map[string]any `json:"payload,omitempty"`
	Error       *string        `json:"error,omitempty"`
}

// TaskListResponse represents a paginated list of tasks
type TaskListResponse struct {
	Items      []TaskResponse `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// TaskRespondRequest represents marking a task as responded
type TaskRespondRequest struct {
	ResponseData map[string]any `json:"response_data,omitempty"`
}

// TaskResolveRequest represents marking a task as resolved/done
type TaskResolveRequest struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
