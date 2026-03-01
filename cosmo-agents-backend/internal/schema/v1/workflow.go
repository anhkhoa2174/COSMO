package v1

import (
	"time"

	"github.com/google/uuid"
)

// CreateWorkflowRequest represents the request to create a workflow
type CreateWorkflowRequest struct {
	Name        string                 `json:"name" validate:"required"`
	Description *string                `json:"description,omitempty"`
	IsActive    bool                   `json:"is_active"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// UpdateWorkflowRequest represents the request to update a workflow
type UpdateWorkflowRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	IsActive    *bool                  `json:"is_active,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// WorkflowResponse represents a workflow entity
type WorkflowResponse struct {
	ID             uuid.UUID              `json:"id"`
	UserID         uuid.UUID              `json:"user_id"`
	OrganizationID uuid.UUID              `json:"organization_id"`
	Name           string                 `json:"name"`
	Description    *string                `json:"description,omitempty"`
	IsActive       bool                   `json:"is_active"`
	Config         map[string]interface{} `json:"config,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// WorkflowListResponse represents paginated workflow results
type WorkflowListResponse struct {
	List   []WorkflowResponse `json:"list"`
	Offset int                `json:"offset"`
	Limit  int                `json:"limit"`
	Total  int64              `json:"total"`
}
