package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client handles Microsoft/Outlook OAuth2 authentication
type Client struct {
	clientID     string
	clientSecret string
	authority    string
	redirectURI  string
	scopes       []string
	httpClient   *http.Client
}

// TokenResponse represents the OAuth2 token response from Microsoft
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token,omitempty"`
}

// NewClient creates a new Microsoft OAuth2 client
func NewClient(clientID, clientSecret, authority, redirectURI string, scopes []string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		authority:    authority,
		redirectURI:  redirectURI,
		scopes:       scopes,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAuthorizationURL generates the Microsoft OAuth2 authorization URL
func (c *Client) GetAuthorizationURL(state string) string {
	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", c.redirectURI)
	params.Set("scope", strings.Join(c.scopes, " "))
	params.Set("state", state)
	params.Set("response_mode", "query")

	authURL := fmt.Sprintf("%s/oauth2/v2.0/authorize?%s", c.authority, params.Encode())
	return authURL
}

// ExchangeCode exchanges an authorization code for tokens
func (c *Client) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/oauth2/v2.0/token", c.authority)

	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", c.redirectURI)
	data.Set("grant_type", "authorization_code")
	data.Set("scope", strings.Join(c.scopes, " "))

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// RefreshToken refreshes an access token using a refresh token
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/oauth2/v2.0/token", c.authority)

	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")
	data.Set("scope", strings.Join(c.scopes, " "))

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// RevokeToken revokes an access or refresh token
func (c *Client) RevokeToken(ctx context.Context, token string) error {
	revokeURL := fmt.Sprintf("%s/oauth2/v2.0/logout", c.authority)

	data := url.Values{}
	data.Set("token", token)

	req, err := http.NewRequestWithContext(ctx, "POST", revokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token revocation failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
