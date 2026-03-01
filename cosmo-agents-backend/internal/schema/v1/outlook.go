package v1

// OutlookAuthResponse represents the authorization URL response.
type OutlookAuthResponse struct {
	URL string `json:"url"`
}

// OutlookTokenResponse represents token payload returned to clients.
type OutlookTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}
