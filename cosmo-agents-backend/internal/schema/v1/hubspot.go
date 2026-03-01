package v1

// AuthHubspotResponse represents authorization URL response.
type AuthHubspotResponse struct {
	URL string `json:"url"`
}

// AuthHubspotCallbackResponse represents OAuth callback response.
type AuthHubspotCallbackResponse struct {
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// HubspotUserInfoRead represents HubSpot user information.
type HubspotUserInfoRead struct {
	Token     string   `json:"token"`
	User      string   `json:"user"`
	HubDomain string   `json:"hub_domain"`
	Scopes    []string `json:"scopes"`
	HubID     int      `json:"hub_id"`
	ClientID  string   `json:"client_id"`
	UserID    int      `json:"user_id"`
	TokenType string   `json:"token_type"`
}
