package v1

import (
	"time"

	"github.com/google/uuid"
)

// Template request/response schemas

// CreateTemplateRequest represents the request body for creating a template
type CreateTemplateRequest struct {
	CampaignID *uuid.UUID     `json:"campaign_id,omitempty"`
	Type       string         `json:"type,omitempty"`
	Category   string         `json:"category" validate:"required,oneof=Draft Outreach 'Out of office outreach'"`
	Subject    string         `json:"subject" validate:"required,min=1"`
	Content    string         `json:"content" validate:"required,min=1"`
	Position   *float64       `json:"position,omitempty"`
	SendAfter  *int           `json:"send_after,omitempty" validate:"omitempty,min=0"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// UpdateTemplateRequest represents the request body for updating a template
type UpdateTemplateRequest struct {
	Type      *string        `json:"type,omitempty"`
	Category  *string        `json:"category,omitempty" validate:"omitempty,oneof=Draft Outreach 'Out of office outreach'"`
	Subject   *string        `json:"subject,omitempty" validate:"omitempty,min=1"`
	Content   *string        `json:"content,omitempty" validate:"omitempty,min=1"`
	Position  *float64       `json:"position,omitempty"`
	SendAfter *int           `json:"send_after,omitempty" validate:"omitempty,min=0"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// TemplateResponse represents the response for a template
type TemplateResponse struct {
	ID         uuid.UUID      `json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	UserID     uuid.UUID      `json:"user_id"`
	CampaignID *uuid.UUID     `json:"campaign_id,omitempty"`
	Type       string         `json:"type,omitempty"`
	Category   string         `json:"category"`
	Subject    string         `json:"subject"`
	Content    string         `json:"content"`
	Position   float64        `json:"position"`
	SendAfter  int            `json:"send_after"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// TemplateListResponse represents a paginated list of templates
type TemplateListResponse struct {
	Items      []TemplateResponse `json:"items"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	TotalPages int                `json:"total_pages"`
}

// ReorderTemplatesRequest represents request to reorder templates
type ReorderTemplatesRequest struct {
	TemplatePositions []TemplatePosition `json:"template_positions" validate:"required,min=1,dive"`
}

// TemplatePosition represents a template's new position
type TemplatePosition struct {
	TemplateID uuid.UUID `json:"template_id" validate:"required"`
	Position   float64   `json:"position" validate:"required"`
}
