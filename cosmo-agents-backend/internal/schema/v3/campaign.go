package v3

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// CampaignGenerateTemplateRequest represents the request for POST /v3/campaigns/{campaign_id}/templates
type CampaignGenerateTemplateRequest struct {
	Prompt       *string                `json:"prompt"`
	DocumentGids []string               `json:"document_gids"`
	ContactData  map[string]interface{} `json:"contact_data"`
	Tone         *string                `json:"tone"`
}

// CampaignRegenerateTemplateRequest represents the request for POST /v3/campaigns/{campaign_id}/templates/{template_id}
type CampaignRegenerateTemplateRequest struct {
	Prompt       string                 `json:"prompt"`
	ContactData  map[string]interface{} `json:"contact_data"`
	DocumentGids []string               `json:"document_gids"`
	Tone         *string                `json:"tone"`
}

// CampaignGenerateSampleResponseRequest represents the request for POST /v3/campaign/{campaign_id}/generate-sample-response
type CampaignGenerateSampleResponseRequest struct {
	Intent           domain.IntentType      `json:"intent"`
	ContactData      map[string]interface{} `json:"contact_data"`
	OutreachTemplate string                 `json:"outreach_template"`
}

// CampaignSaveExternalTemplateRequest represents the request for POST /v3/campaigns/{campaign_id}/templates/external
type CampaignSaveExternalTemplateRequest struct {
	Subject   string  `json:"subject" validate:"required"`
	Content   string  `json:"content" validate:"required"`
	Type      *string `json:"type"`       // Optional: if provided, use this type instead of auto-calculating
	SendAfter *int    `json:"send_after"` // Optional: days to wait before sending
}

// CampaignSaveExternalTemplateResponse represents the response for POST /v3/campaigns/{campaign_id}/templates/external
type CampaignSaveExternalTemplateResponse struct {
	ID        uuid.UUID `json:"id"`
	Subject   string    `json:"subject"`
	Content   string    `json:"content"`
	SendAfter int       `json:"send_after"`
	Type      string    `json:"type"`
}
