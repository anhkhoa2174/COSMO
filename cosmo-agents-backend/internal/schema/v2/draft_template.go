package v2

import "github.com/google/uuid"

// DraftTemplateResponse represents a draft template response
type DraftTemplateResponse struct {
	ID         uuid.UUID              `json:"id"`
	CampaignID uuid.UUID              `json:"campaign_id"`
	Intent     string                 `json:"intent"`
	Subject    string                 `json:"subject"`
	Content    string                 `json:"content"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}
