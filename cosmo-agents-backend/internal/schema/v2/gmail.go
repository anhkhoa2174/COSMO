package v2

import (
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// GmailAuthURLResponse represents the response for GET /v2/google/gmail
type GmailAuthURLResponse struct {
	URL string `json:"url" example:"https://accounts.google.com/o/oauth2/auth?..."`
}

// GmailOAuth2CallbackResponse represents the response for GET /v2/google/gmail/oauth2callback
// Returns agent information after successful OAuth2 callback
type GmailOAuth2CallbackResponse = v1schema.AgentResponse

// GmailWatchResponse represents the response for POST /v2/google/gmail/watch
type GmailWatchResponse struct {
	HistoryID  string `json:"history_id" example:"1234567"`
	Expiration int64  `json:"expiration" example:"1609459200000"`
}

// GmailStopResponse represents the response for POST /v2/google/gmail/stop
type GmailStopResponse struct {
	Message string `json:"message" example:"Stop watching for Gmail notifications for the current user successfully."`
}

// PubsubMessage represents a message published by publishers and consumed by subscribers in Google Cloud Pub/Sub
// Reference: https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
type PubsubMessage struct {
	Data        string            `json:"data,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
	MessageID   string            `json:"messageId,omitempty"`
	PublishTime string            `json:"publishTime,omitempty"`
	OrderingKey string            `json:"orderingKey,omitempty"`
}

// GmailMessage represents decoded message data from PubsubMessage
// Reference: https://developers.google.com/gmail/api/guides/push#python
type GmailMessage struct {
	EmailAddress string `json:"emailAddress"`
	HistoryID    int64  `json:"historyId"`
}

// GmailNotificationRequest represents the request body for POST /v2/google/gmail/notifications
// Reference: https://developers.google.com/gmail/api/guides/push#python
type GmailNotificationRequest struct {
	Message      PubsubMessage `json:"message" binding:"required"`
	Subscription string        `json:"subscription" binding:"required"`
}
