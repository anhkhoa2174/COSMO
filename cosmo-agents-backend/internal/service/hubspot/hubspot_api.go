package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rockship/cosmo-agents-go/pkg/config"
)

// HubspotAPI provides low-level access to HubSpot REST endpoints.
type HubspotAPI struct {
	client       *http.Client
	clientID     string
	clientSecret string
	redirectURI  string
	scopes       []string
}

// NewHubspotAPI creates a new HubspotAPI instance.
func NewHubspotAPI(cfg config.HubspotConfig) *HubspotAPI {
	return &HubspotAPI{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		redirectURI:  cfg.RedirectURI,
		scopes:       parseScopes(cfg.AuthScopes),
	}
}

func parseScopes(scopes string) []string {
	if strings.TrimSpace(scopes) == "" {
		return []string{"contacts"}
	}
	parts := strings.Split(scopes, ",")
	if len(parts) == 1 && strings.Contains(scopes, " ") {
		parts = strings.Split(scopes, " ")
	}
	var result []string
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return []string{"contacts"}
	}
	return result
}

// HubspotTokenResponse represents OAuth token response.
type HubspotTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// HubspotUserInfo represents HubSpot user information response.
type HubspotUserInfo struct {
	Token     string   `json:"token"`
	User      string   `json:"user"`
	HubDomain string   `json:"hub_domain"`
	Scopes    []string `json:"scopes"`
	HubID     int      `json:"hub_id"`
	ClientID  string   `json:"client_id"`
	UserID    int      `json:"user_id"`
	TokenType string   `json:"token_type"`
}

// AuthorizationURL generates OAuth authorization URL.
func (h *HubspotAPI) AuthorizationURL(overrideRedirect string) (string, error) {
	redirect := h.redirectURI
	if strings.TrimSpace(overrideRedirect) != "" {
		redirect = overrideRedirect
	}
	if redirect == "" {
		return "", fmt.Errorf("hubspot redirect URI is not configured")
	}

	u := url.URL{
		Scheme: "https",
		Host:   "app.hubspot.com",
		Path:   "/oauth/authorize",
	}
	q := url.Values{}
	q.Set("client_id", h.clientID)
	q.Set("redirect_uri", redirect)
	q.Set("scope", strings.Join(h.scopes, " "))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// ExchangeCode exchanges authorization code for tokens.
func (h *HubspotAPI) ExchangeCode(ctx context.Context, code, redirectURI string) (*HubspotTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", h.clientID)
	data.Set("client_secret", h.clientSecret)
	if strings.TrimSpace(redirectURI) == "" {
		redirectURI = h.redirectURI
	}
	data.Set("redirect_uri", redirectURI)
	data.Set("code", code)

	var resp HubspotTokenResponse
	if err := h.postForm(ctx, "https://api.hubapi.com/oauth/v1/token", data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RefreshToken refreshes access token using refresh token.
func (h *HubspotAPI) RefreshToken(ctx context.Context, refreshToken string) (*HubspotTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", h.clientID)
	data.Set("client_secret", h.clientSecret)
	data.Set("refresh_token", refreshToken)

	var resp HubspotTokenResponse
	if err := h.postForm(ctx, "https://api.hubapi.com/oauth/v1/token", data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetUserInfoFromRefreshToken retrieves user info using refresh token.
func (h *HubspotAPI) GetUserInfoFromRefreshToken(ctx context.Context, refreshToken string) (*HubspotUserInfo, error) {
	url := fmt.Sprintf("https://api.hubapi.com/oauth/v1/refresh-tokens/%s", refreshToken)
	var resp HubspotUserInfo
	if err := h.getJSON(ctx, url, "", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// HubspotContact holds contact data subset.
type HubspotContact struct {
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties"`
}

// HubspotContactsPage represents a page of contacts.
type HubspotContactsPage struct {
	Contacts []HubspotContact
	After    int
}

// GetContactsAll fetches a page of HubSpot contacts.
func (h *HubspotAPI) GetContactsAll(ctx context.Context, accessToken string, after int) (*HubspotContactsPage, error) {
	params := url.Values{}
	params.Set("limit", "100")
	properties := []string{"firstname", "lastname", "email", "phone", "company", "jobtitle", "address", "city", "country", "state", "zip"}
	for _, p := range properties {
		params.Add("properties", p)
	}
	if after > 0 {
		params.Set("after", strconv.Itoa(after))
	}

	reqURL := "https://api.hubapi.com/crm/v3/objects/contacts?" + params.Encode()
	var resp struct {
		Results []struct {
			ID         string                 `json:"id"`
			Properties map[string]interface{} `json:"properties"`
		} `json:"results"`
		Paging struct {
			Next struct {
				After string `json:"after"`
			} `json:"next"`
		} `json:"paging"`
	}

	if err := h.getJSON(ctx, reqURL, accessToken, &resp); err != nil {
		return nil, err
	}

	next := -1
	if resp.Paging.Next.After != "" {
		if parsed, err := strconv.Atoi(resp.Paging.Next.After); err == nil {
			next = parsed
		}
	}

	contacts := make([]HubspotContact, len(resp.Results))
	for i, item := range resp.Results {
		contacts[i] = HubspotContact{
			ID:         item.ID,
			Properties: item.Properties,
		}
	}

	return &HubspotContactsPage{
		Contacts: contacts,
		After:    next,
	}, nil
}

// HubspotList represents a contact list.
type HubspotList struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// HubspotListsPage represents paginated list response.
type HubspotListsPage struct {
	Lists  []HubspotList
	Offset int
}

// GetLists fetches HubSpot contact lists.
func (h *HubspotAPI) GetLists(ctx context.Context, accessToken string, offset int) (*HubspotListsPage, error) {
	params := url.Values{}
	params.Set("limit", "100")
	if offset > 0 {
		params.Set("after", strconv.Itoa(offset))
	}
	reqURL := "https://api.hubapi.com/crm/v3/lists?" + params.Encode()

	var resp struct {
		Results []struct {
			ListID string `json:"listId"`
			Name   string `json:"name"`
		} `json:"results"`
		Paging struct {
			Next struct {
				After string `json:"after"`
			} `json:"next"`
		} `json:"paging"`
	}

	if err := h.getJSON(ctx, reqURL, accessToken, &resp); err != nil {
		return nil, err
	}

	lists := make([]HubspotList, len(resp.Results))
	for i, item := range resp.Results {
		lists[i] = HubspotList{
			ID:   item.ListID,
			Name: item.Name,
		}
	}

	next := -1
	if resp.Paging.Next.After != "" {
		if parsed, err := strconv.Atoi(resp.Paging.Next.After); err == nil {
			next = parsed
		}
	}

	return &HubspotListsPage{
		Lists:  lists,
		Offset: next,
	}, nil
}

// GetContactsFromList returns contact IDs for given list.
func (h *HubspotAPI) GetContactsFromList(ctx context.Context, accessToken, listID string, after *string) ([]string, *string, error) {
	reqURL := fmt.Sprintf("https://api.hubapi.com/crm/v3/lists/%s/memberships?limit=100", listID)
	if after != nil && *after != "" {
		reqURL += "&after=" + *after
	}
	var resp struct {
		Results []struct {
			RecordID string `json:"recordId"`
		} `json:"results"`
		Paging struct {
			Next struct {
				After string `json:"after"`
			} `json:"next"`
		} `json:"paging"`
	}
	if err := h.getJSON(ctx, reqURL, accessToken, &resp); err != nil {
		return nil, nil, err
	}
	ids := make([]string, 0, len(resp.Results))

	var newAfter *string
	if resp.Paging.Next.After != "" {
		newAfter = &resp.Paging.Next.After
	}

	for _, item := range resp.Results {
		if item.RecordID != "" {
			ids = append(ids, item.RecordID)
		}
	}
	return ids, newAfter, nil
}

// BatchGetContacts retrieves contact details by IDs.
func (h *HubspotAPI) BatchGetContacts(ctx context.Context, accessToken string, ids []string, properties []string) ([]map[string]interface{}, error) {
	body := map[string]interface{}{
		"idProperty": "hs_object_id",
		"inputs":     make([]map[string]string, 0, len(ids)),
		"properties": properties,
	}
	for _, id := range ids {
		body["inputs"] = append(body["inputs"].([]map[string]string), map[string]string{"id": id})
	}

	var resp struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := h.postJSON(ctx, "https://api.hubapi.com/crm/v3/objects/contacts/batch/read?archived=false", accessToken, body, &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

// GetListByID returns details for a specific list.
func (h *HubspotAPI) GetListByID(ctx context.Context, accessToken, listID string) (map[string]interface{}, error) {
	reqURL := fmt.Sprintf("https://api.hubapi.com/crm/v3/lists/%s?includeFilters=false", listID)
	var resp map[string]interface{}
	if err := h.getJSON(ctx, reqURL, accessToken, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (h *HubspotAPI) getJSON(ctx context.Context, url string, accessToken string, destination interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return fmt.Errorf("hubspot request failed: %s", res.Status)
	}

	return json.NewDecoder(res.Body).Decode(destination)
}

func (h *HubspotAPI) postForm(ctx context.Context, endpoint string, form url.Values, destination interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return fmt.Errorf("hubspot request failed: %s", res.Status)
	}

	return json.NewDecoder(res.Body).Decode(destination)
}

func (h *HubspotAPI) postJSON(ctx context.Context, endpoint, accessToken string, payload interface{}, destination interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	res, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		return fmt.Errorf("hubspot request failed: %s", res.Status)
	}

	return json.NewDecoder(res.Body).Decode(destination)
}
