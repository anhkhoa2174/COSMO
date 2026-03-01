package hubspot

import (
	"bytes"
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
	baseAPIURL = "https://api.hubapi.com"
	authURL    = "https://app.hubspot.com/oauth/authorize"
	tokenURL   = "https://api.hubapi.com/oauth/v1/token"
	refreshURL = "https://api.hubapi.com/oauth/v1/refresh-tokens"
)

// Client handles HubSpot API operations
type Client struct {
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
	accessToken  string
	httpClient   *http.Client
}

// TokenResponse represents OAuth2 token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// Contact represents a HubSpot contact
type Contact struct {
	ID         string                 `json:"id,omitempty"`
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  string                 `json:"createdAt,omitempty"`
	UpdatedAt  string                 `json:"updatedAt,omitempty"`
	Archived   bool                   `json:"archived,omitempty"`
}

// ContactsResponse represents paginated contacts response
type ContactsResponse struct {
	Results []Contact `json:"results"`
	Paging  *Paging   `json:"paging,omitempty"`
}

// Paging represents pagination information
type Paging struct {
	Next *PagingNext `json:"next,omitempty"`
}

// PagingNext represents next page information
type PagingNext struct {
	After string `json:"after"`
	Link  string `json:"link,omitempty"`
}

// List represents a HubSpot list
type List struct {
	ListID int    `json:"listId"`
	Name   string `json:"name"`
}

// ListsResponse represents lists response
type ListsResponse struct {
	Lists   []List `json:"lists"`
	Offset  int    `json:"offset"`
	HasMore bool   `json:"has-more"`
}

// ListMembership represents list membership
type ListMembership struct {
	RecordID string `json:"recordId"`
}

// ListMembershipsResponse represents list memberships response
type ListMembershipsResponse struct {
	Results []ListMembership `json:"results"`
	Paging  *Paging          `json:"paging,omitempty"`
}

// BatchReadRequest represents batch read request
type BatchReadRequest struct {
	IDProperty string              `json:"idProperty"`
	Inputs     []map[string]string `json:"inputs"`
	Properties []string            `json:"properties"`
}

// BatchReadResponse represents batch read response
type BatchReadResponse struct {
	Results []Contact `json:"results"`
	Status  string    `json:"status,omitempty"`
	Message string    `json:"message,omitempty"`
}

// UserInfo represents HubSpot user information
type UserInfo struct {
	User      string `json:"user"`
	HubID     int    `json:"hub_id"`
	HubDomain string `json:"hub_domain"`
}

// NewClient creates a new HubSpot client
func NewClient(clientID, clientSecret, redirectURI string, scopes []string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		scopes:       scopes,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithToken creates a client with an access token
func NewClientWithToken(accessToken string) *Client {
	return &Client{
		accessToken: accessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAuthorizationURL generates the HubSpot OAuth2 authorization URL
func (c *Client) GetAuthorizationURL(state string) string {
	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("redirect_uri", c.redirectURI)
	params.Set("scope", strings.Join(c.scopes, " "))
	if state != "" {
		params.Set("state", state)
	}

	return fmt.Sprintf("%s?%s", authURL, params.Encode())
}

// ExchangeCode exchanges authorization code for tokens
func (c *Client) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("redirect_uri", c.redirectURI)
	data.Set("code", code)

	return c.requestToken(ctx, data)
}

// RefreshToken refreshes an access token
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("refresh_token", refreshToken)

	return c.requestToken(ctx, data)
}

// requestToken makes a token request
func (c *Client) requestToken(ctx context.Context, data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
		return nil, fmt.Errorf("token request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &tokenResp, nil
}

// GetUserInfoFromRefreshToken gets user info from refresh token
func (c *Client) GetUserInfoFromRefreshToken(ctx context.Context, refreshToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", refreshURL+"/"+refreshToken, nil)
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
		return nil, fmt.Errorf("request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var userInfo UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return &userInfo, nil
}

// doRequest performs an authenticated API request
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	url := baseAPIURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetContactsAll retrieves all contacts with pagination
func (c *Client) GetContactsAll(ctx context.Context, after string, limit int) (*ContactsResponse, error) {
	if limit <= 0 {
		limit = 100
	}

	properties := []string{"firstname", "lastname", "email", "phone", "company", "jobtitle", "address", "city", "country", "state", "zip"}
	endpoint := fmt.Sprintf("/crm/v3/objects/contacts?limit=%d&properties=%s", limit, strings.Join(properties, "&properties="))

	if after != "" {
		endpoint += "&after=" + after
	}

	respBody, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var contacts ContactsResponse
	if err := json.Unmarshal(respBody, &contacts); err != nil {
		return nil, fmt.Errorf("failed to parse contacts: %w", err)
	}

	return &contacts, nil
}

// GetContactIDsFromList retrieves contact IDs from a list
func (c *Client) GetContactIDsFromList(ctx context.Context, listID string, after string, limit int) ([]string, string, error) {
	if limit <= 0 {
		limit = 100
	}

	endpoint := fmt.Sprintf("/crm/v3/lists/%s/memberships?limit=%d", listID, limit)
	if after != "" {
		endpoint += "&after=" + after
	}

	respBody, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, "", err
	}

	var memberships ListMembershipsResponse
	if err := json.Unmarshal(respBody, &memberships); err != nil {
		return nil, "", fmt.Errorf("failed to parse memberships: %w", err)
	}

	contactIDs := make([]string, len(memberships.Results))
	for i, membership := range memberships.Results {
		contactIDs[i] = membership.RecordID
	}

	nextAfter := ""
	if memberships.Paging != nil && memberships.Paging.Next != nil {
		nextAfter = memberships.Paging.Next.After
	}

	return contactIDs, nextAfter, nil
}

// BatchGetContactsWithProperties retrieves contacts in batch with specific properties
func (c *Client) BatchGetContactsWithProperties(ctx context.Context, contactIDs []string, properties []string) ([]Contact, error) {
	inputs := make([]map[string]string, len(contactIDs))
	for i, id := range contactIDs {
		inputs[i] = map[string]string{"id": id}
	}

	req := BatchReadRequest{
		IDProperty: "hs_object_id",
		Inputs:     inputs,
		Properties: properties,
	}

	respBody, err := c.doRequest(ctx, "POST", "/crm/v3/objects/contacts/batch/read?archived=false", req)
	if err != nil {
		return nil, err
	}

	var batchResp BatchReadResponse
	if err := json.Unmarshal(respBody, &batchResp); err != nil {
		return nil, fmt.Errorf("failed to parse batch response: %w", err)
	}

	if batchResp.Status == "error" {
		return nil, fmt.Errorf("batch read failed: %s", batchResp.Message)
	}

	return batchResp.Results, nil
}

// GetLists retrieves contact lists
func (c *Client) GetLists(ctx context.Context, offset int, count int) (*ListsResponse, error) {
	if count <= 0 {
		count = 250
	}

	endpoint := fmt.Sprintf("/contacts/v1/lists?count=%d&offset=%d", count, offset)

	respBody, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var lists ListsResponse
	if err := json.Unmarshal(respBody, &lists); err != nil {
		return nil, fmt.Errorf("failed to parse lists: %w", err)
	}

	return &lists, nil
}

// GetListByID retrieves a specific list by ID
func (c *Client) GetListByID(ctx context.Context, listID string) (*List, error) {
	endpoint := fmt.Sprintf("/crm/v3/lists/%s?includeFilters=false", listID)

	respBody, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var list List
	if err := json.Unmarshal(respBody, &list); err != nil {
		return nil, fmt.Errorf("failed to parse list: %w", err)
	}

	return &list, nil
}

// CreateContact creates a new contact
func (c *Client) CreateContact(ctx context.Context, properties map[string]interface{}) (*Contact, error) {
	req := map[string]interface{}{
		"properties": properties,
	}

	respBody, err := c.doRequest(ctx, "POST", "/crm/v3/objects/contacts", req)
	if err != nil {
		return nil, err
	}

	var contact Contact
	if err := json.Unmarshal(respBody, &contact); err != nil {
		return nil, fmt.Errorf("failed to parse contact: %w", err)
	}

	return &contact, nil
}

// UpdateContact updates a contact
func (c *Client) UpdateContact(ctx context.Context, contactID string, properties map[string]interface{}) (*Contact, error) {
	req := map[string]interface{}{
		"properties": properties,
	}

	endpoint := fmt.Sprintf("/crm/v3/objects/contacts/%s", contactID)

	respBody, err := c.doRequest(ctx, "PATCH", endpoint, req)
	if err != nil {
		return nil, err
	}

	var contact Contact
	if err := json.Unmarshal(respBody, &contact); err != nil {
		return nil, fmt.Errorf("failed to parse contact: %w", err)
	}

	return &contact, nil
}
