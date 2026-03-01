package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// Config holds Google OAuth2 configuration.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// Client wraps Google OAuth2 operations.
type Client struct {
	config *oauth2.Config
}

// NewClient creates a new Google OAuth2 client.
func NewClient(cfg Config) *Client {
	return &Client{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURI,
			Scopes: []string{
				gmail.GmailSendScope,                               // Send emails
				gmail.GmailReadonlyScope,                           // Read emails
				gmail.GmailModifyScope,                             // Modify emails (labels, etc)
				gmail.GmailLabelsScope,                             // Manage labels
				"https://www.googleapis.com/auth/userinfo.email",   // User email
				"https://www.googleapis.com/auth/userinfo.profile", // User profile
			},
			Endpoint: google.Endpoint,
		},
	}
}

// GetAuthURL generates the OAuth2 authorization URL.
// State should be a random string to prevent CSRF attacks.
func (c *Client) GetAuthURL(state string) string {
	return c.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeCode exchanges authorization code for tokens.
func (c *Client) ExchangeCode(ctx context.Context, code string) (*Token, error) {
	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return &Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}, nil
}

// RefreshToken refreshes an expired access token using the refresh token.
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*Token, error) {
	tokenSource := c.config.TokenSource(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
	})

	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return &Token{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		TokenType:    newToken.TokenType,
		Expiry:       newToken.Expiry,
	}, nil
}

// GetConfig returns the OAuth2 config.
func (c *Client) GetConfig() *oauth2.Config {
	return c.config
}

// GetUserInfo fetches user information from Google.
func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	client := c.config.Client(ctx, &oauth2.Token{
		AccessToken: accessToken,
	})

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// RevokeToken revokes an access (or refresh) token via Google's revoke endpoint.
func (c *Client) RevokeToken(ctx context.Context, token string) error {
	if token == "" {
		return fmt.Errorf("empty token")
	}
	values := url.Values{}
	values.Set("token", token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/revoke", strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Google's revoke returns 200 on success; 400 on failure.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to revoke token: status %d", resp.StatusCode)
	}
	return nil
}

// Token represents OAuth2 token information.
type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Expiry       time.Time
}

// IsExpired checks if the token is expired.
func (t *Token) IsExpired() bool {
	return t.Expiry.Before(time.Now().Add(5 * time.Minute))
}

// UserInfo represents Google user information.
type UserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}
