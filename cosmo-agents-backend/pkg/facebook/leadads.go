package facebook

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

const (
	graphAPIVersion = "v19.0"
	baseURL         = "https://graph.facebook.com/" + graphAPIVersion
	authURL         = "https://www.facebook.com/" + graphAPIVersion + "/dialog/oauth"
)

// Client handles Facebook Lead Ads API operations
type Client struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
}

// Page represents a Facebook Page
type Page struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AccessToken string `json:"access_token"`
}

// PagesResponse represents the pages list response
type PagesResponse struct {
	Data []Page `json:"data"`
}

// LeadFormQuestion represents a question in a lead form
type LeadFormQuestion struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"type"`
}

// LeadForm represents a Facebook lead generation form
type LeadForm struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Questions []LeadFormQuestion `json:"questions,omitempty"`
}

// LeadFormsResponse represents the lead forms list response
type LeadFormsResponse struct {
	Data []LeadForm `json:"data"`
}

// LeadFieldData represents a field in lead data
type LeadFieldData struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

// LeadData represents a Facebook lead
type LeadData struct {
	ID          string          `json:"id"`
	FormID      string          `json:"form_id"`
	CreatedTime string          `json:"created_time"`
	FieldData   []LeadFieldData `json:"field_data"`
	AdID        string          `json:"ad_id,omitempty"`
	AdgroupID   string          `json:"adgroup_id,omitempty"`
	CampaignID  string          `json:"campaign_id,omitempty"`
	IsOrganic   bool            `json:"is_organic,omitempty"`
}

// AccessToken represents a Facebook access token
type AccessToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
}

// IsShortLived checks if the token is short-lived (< 2 hours)
func (t *AccessToken) IsShortLived() bool {
	return t.ExpiresIn > 0 && t.ExpiresIn < 7200
}

// SubscribeResponse represents webhook subscription response
type SubscribeResponse struct {
	Success bool `json:"success"`
}

// NewClient creates a new Facebook Lead Ads client
func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAuthorizationURL generates the Facebook OAuth authorization URL
func (c *Client) GetAuthorizationURL(redirectURI string, scopes []string, state string) string {
	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", strings.Join(scopes, ","))
	params.Set("state", state)
	params.Set("response_type", "code")

	return fmt.Sprintf("%s?%s", authURL, params.Encode())
}

// ExchangeCodeForToken exchanges authorization code for access token
func (c *Client) ExchangeCodeForToken(ctx context.Context, code, redirectURI string) (*AccessToken, error) {
	endpoint := fmt.Sprintf("%s/oauth/access_token", baseURL)

	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("client_secret", c.clientSecret)
	params.Set("redirect_uri", redirectURI)
	params.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var token AccessToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return &token, nil
}

// ExchangeForLongLivedToken exchanges short-lived token for long-lived token (60 days)
func (c *Client) ExchangeForLongLivedToken(ctx context.Context, shortLivedToken string) (*AccessToken, error) {
	endpoint := fmt.Sprintf("%s/oauth/access_token", baseURL)

	params := url.Values{}
	params.Set("grant_type", "fb_exchange_token")
	params.Set("client_id", c.clientID)
	params.Set("client_secret", c.clientSecret)
	params.Set("fb_exchange_token", shortLivedToken)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var token AccessToken
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return &token, nil
}

// GetUserPages retrieves pages managed by the user
func (c *Client) GetUserPages(ctx context.Context, userAccessToken string) ([]Page, error) {
	endpoint := fmt.Sprintf("%s/me/accounts", baseURL)

	params := url.Values{}
	params.Set("access_token", userAccessToken)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get pages (status %d): %s", resp.StatusCode, string(body))
	}

	var pagesResp PagesResponse
	if err := json.Unmarshal(body, &pagesResp); err != nil {
		return nil, fmt.Errorf("failed to parse pages: %w", err)
	}

	return pagesResp.Data, nil
}

// GetPageLeadForms retrieves lead generation forms for a page
func (c *Client) GetPageLeadForms(ctx context.Context, pageID, pageAccessToken string) ([]LeadForm, error) {
	endpoint := fmt.Sprintf("%s/%s/leadgen_forms", baseURL, pageID)

	params := url.Values{}
	params.Set("access_token", pageAccessToken)
	params.Set("fields", "id,name,questions")

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get lead forms (status %d): %s", resp.StatusCode, string(body))
	}

	var formsResp LeadFormsResponse
	if err := json.Unmarshal(body, &formsResp); err != nil {
		return nil, fmt.Errorf("failed to parse lead forms: %w", err)
	}

	return formsResp.Data, nil
}

// SubscribePage subscribes the app to page webhook events
func (c *Client) SubscribePage(ctx context.Context, pageID, pageAccessToken string, subscribedFields []string) (bool, error) {
	endpoint := fmt.Sprintf("%s/%s/subscribed_apps", baseURL, pageID)

	params := url.Values{}
	params.Set("access_token", pageAccessToken)
	params.Set("subscribed_fields", strings.Join(subscribedFields, ","))

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to subscribe (status %d): %s", resp.StatusCode, string(body))
	}

	var subResp SubscribeResponse
	if err := json.Unmarshal(body, &subResp); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	return subResp.Success, nil
}

// GetLeadDetails retrieves detailed information for a lead
func (c *Client) GetLeadDetails(ctx context.Context, leadgenID, pageAccessToken string) (*LeadData, error) {
	endpoint := fmt.Sprintf("%s/%s", baseURL, leadgenID)

	params := url.Values{}
	params.Set("access_token", pageAccessToken)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get lead details (status %d): %s", resp.StatusCode, string(body))
	}

	var leadData LeadData
	if err := json.Unmarshal(body, &leadData); err != nil {
		return nil, fmt.Errorf("failed to parse lead data: %w", err)
	}

	return &leadData, nil
}

// VerifyWebhook verifies Facebook webhook subscription
func VerifyWebhook(mode, token, verifyToken string) (string, bool) {
	if mode == "subscribe" && token == verifyToken {
		return "", true
	}
	return "Verification failed", false
}
