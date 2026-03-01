package v2

import "github.com/google/uuid"

// AuthHubspotCallbackResponse represents the response for GET /v2/hubspot/callback
type AuthHubspotCallbackResponse struct {
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// HubspotUserInfoRead represents the response for GET /v2/hubspot/users/me
type HubspotUserInfoRead struct {
	ID                uuid.UUID              `json:"id"`
	Email             string                 `json:"email"`
	SourceID          string                 `json:"source_id"`
	AccessToken       string                 `json:"access_token"`
	RefreshToken      string                 `json:"refresh_token"`
	FieldMapping      map[string]interface{} `json:"field_mapping"`
	DuplicationOption *string                `json:"duplication_option"`
}

// HubspotUserRequest represents the request for PATCH /v2/hubspot/users/me
type HubspotUserRequest struct {
	FieldMapping      map[string]interface{} `json:"field_mapping,omitempty"`
	DuplicationOption *string                `json:"duplication_option,omitempty"`
}
