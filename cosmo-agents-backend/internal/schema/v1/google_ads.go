package v1

import "github.com/google/uuid"

// CreateGoogleAdsWebhookRequest represents the payload to create a webhook.
type CreateGoogleAdsWebhookRequest struct {
	CampaignID    uuid.UUID `json:"campaign_id" validate:"required"`
	ContactListID uuid.UUID `json:"contact_list_id" validate:"required"`
	Name          string    `json:"name" validate:"required"`
	Slug          string    `json:"slug" validate:"required"`
}

// GoogleAdsWebhookPayload captures the incoming webhook JSON.
type GoogleAdsWebhookPayload struct {
	LeadID         string                 `json:"lead_id"`
	UserColumnData []GoogleAdsColumnEntry `json:"user_column_data"`
}

// GoogleAdsColumnEntry holds a single user column entry.
type GoogleAdsColumnEntry struct {
	ColumnID    string `json:"column_id"`
	StringValue string `json:"string_value"`
}
