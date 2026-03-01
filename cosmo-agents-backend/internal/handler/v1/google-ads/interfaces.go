package googleads

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	// ErrCampaignNotFound indicates the campaign is missing
	ErrCampaignNotFound = errors.New("campaign not found")
	// ErrWebhookSlugExists indicates the slug is already in use
	ErrWebhookSlugExists = errors.New("webhook slug already exists")
	// ErrCampaignInactive indicates campaign is not active/scheduled
	ErrCampaignInactive = errors.New("campaign is not running")
	// ErrMissingAgent indicates campaign lacks an assigned agent
	ErrMissingAgent = errors.New("campaign missing agent")
	// ErrContactListNotFound indicates contact list was not found
	ErrContactListNotFound = errors.New("contact list not found")
	// ErrInboundFormNotFound indicates inbound form was not found
	ErrInboundFormNotFound = errors.New("inbound form not found")
)

// GoogleAdsUseCase defines the business logic interface for Google Ads operations
type GoogleAdsUseCase interface {
	// GenerateWebhook creates a webhook endpoint for Google Ads lead forms
	GenerateWebhook(ctx context.Context, req GenerateWebhookRequest) (*GenerateWebhookResponse, error)

	// ProcessWebhook processes incoming webhook data from Google Ads
	ProcessWebhook(ctx context.Context, req ProcessWebhookRequest) error
}

// GenerateWebhookRequest represents the input for creating a webhook
type GenerateWebhookRequest struct {
	CampaignID    uuid.UUID
	ContactListID uuid.UUID
	Name          string
	Slug          string
	UserID        uuid.UUID
}

// GenerateWebhookResponse represents the output after webhook creation
type GenerateWebhookResponse struct {
	WebhookURL string
}

// ProcessWebhookRequest represents the input for processing a webhook
type ProcessWebhookRequest struct {
	Slug           string
	LeadID         string
	UserColumnData []ColumnEntry
}

// ColumnEntry represents a single field in the Google Ads lead data
type ColumnEntry struct {
	ColumnID    string
	StringValue string
}
