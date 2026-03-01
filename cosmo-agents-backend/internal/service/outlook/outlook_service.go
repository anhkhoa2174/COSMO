package outlook

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rockship/cosmo-agents-go/pkg/config"
)

// OutlookService handles Microsoft OAuth2 and Graph API interactions.
type OutlookService struct {
	httpClient   *http.Client
	clientID     string
	clientSecret string
	redirectURI  string
	authority    string
	scopes       []string
}

// NewOutlookService creates a new OutlookService instance.
func NewOutlookService(cfg config.OutlookConfig) *OutlookService {
	if strings.TrimSpace(cfg.ClientID) == "" ||
		strings.TrimSpace(cfg.ClientSecret) == "" ||
		strings.TrimSpace(cfg.RedirectURI) == "" ||
		strings.TrimSpace(cfg.Authority) == "" {
		return nil
	}

	return &OutlookService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		redirectURI:  cfg.RedirectURI,
		authority:    strings.TrimRight(cfg.Authority, "/"),
		scopes:       parseOutlookScopes(cfg.Scopes),
	}
}

// AuthorizationURL builds the OAuth authorization URL.
func (s *OutlookService) AuthorizationURL(overrideRedirect string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("outlook integration not configured")
	}

	redirect := strings.TrimSpace(overrideRedirect)
	if redirect == "" {
		redirect = s.redirectURI
	}

	if redirect == "" {
		return "", fmt.Errorf("outlook redirect uri not configured")
	}

	q := url.Values{}
	q.Set("client_id", s.clientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirect)
	q.Set("response_mode", "query")
	q.Set("scope", strings.Join(s.scopes, " "))

	authURL := fmt.Sprintf("%s/oauth2/v2.0/authorize?%s", s.authority, q.Encode())
	return authURL, nil
}

// TokenResponse represents token exchange result.
type TokenResponse struct {
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope,omitempty"`
	ExpiresIn    int    `json:"expires_in"`
	ExtExpiresIn int    `json:"ext_expires_in,omitempty"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
}

// ExchangeCode exchanges authorization code for tokens.
func (s *OutlookService) ExchangeCode(ctx context.Context, code, redirect string) (*TokenResponse, error) {
	if s == nil {
		return nil, fmt.Errorf("outlook integration not configured")
	}

	redirectURI := strings.TrimSpace(redirect)
	if redirectURI == "" {
		redirectURI = s.redirectURI
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", s.clientID)
	form.Set("client_secret", s.clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("scope", strings.Join(s.scopes, " "))

	var resp TokenResponse
	if err := s.postForm(ctx, s.tokenEndpoint(), form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// RefreshToken refreshes the access token using a refresh token.
func (s *OutlookService) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	if s == nil {
		return nil, fmt.Errorf("outlook integration not configured")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", s.clientID)
	form.Set("client_secret", s.clientSecret)
	form.Set("refresh_token", refreshToken)
	form.Set("scope", strings.Join(s.scopes, " "))

	var resp TokenResponse
	if err := s.postForm(ctx, s.tokenEndpoint(), form, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetContacts retrieves Outlook contacts.
func (s *OutlookService) GetContacts(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	return s.getGraph(ctx, accessToken, "/me/contacts")
}

// GetMessages retrieves Outlook messages.
func (s *OutlookService) GetMessages(ctx context.Context, accessToken string) (map[string]interface{}, error) {
	return s.getGraph(ctx, accessToken, "/me/messages")
}

// SendMail sends an email via Microsoft Graph.
func (s *OutlookService) SendMail(ctx context.Context, accessToken string, payload map[string]interface{}) (map[string]interface{}, error) {
	return s.postGraph(ctx, accessToken, "/me/sendMail", payload)
}

// ReplyMail replies to an existing message.
func (s *OutlookService) ReplyMail(ctx context.Context, accessToken, messageID string, payload map[string]interface{}) (map[string]interface{}, error) {
	path := fmt.Sprintf("/me/messages/%s/reply", url.PathEscape(messageID))
	return s.postGraph(ctx, accessToken, path, payload)
}

// ForwardMail forwards an existing message.
func (s *OutlookService) ForwardMail(ctx context.Context, accessToken, messageID string, payload map[string]interface{}) (map[string]interface{}, error) {
	path := fmt.Sprintf("/me/messages/%s/forward", url.PathEscape(messageID))
	return s.postGraph(ctx, accessToken, path, payload)
}

func (s *OutlookService) tokenEndpoint() string {
	return fmt.Sprintf("%s/oauth2/v2.0/token", s.authority)
}

func (s *OutlookService) getGraph(ctx context.Context, accessToken, path string) (map[string]interface{}, error) {
	if s == nil {
		return nil, fmt.Errorf("outlook integration not configured")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://graph.microsoft.com/v1.0"+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("graph request failed: %s: %s", res.Status, string(body))
	}

	var data map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func (s *OutlookService) postGraph(ctx context.Context, accessToken, path string, payload map[string]interface{}) (map[string]interface{}, error) {
	if s == nil {
		return nil, fmt.Errorf("outlook integration not configured")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://graph.microsoft.com/v1.0"+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		responseBody, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("graph request failed: %s: %s", res.Status, string(responseBody))
	}

	if res.StatusCode == http.StatusAccepted || res.StatusCode == http.StatusNoContent {
		return map[string]interface{}{"status": "accepted"}, nil
	}

	var data map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func (s *OutlookService) postForm(ctx context.Context, endpoint string, form url.Values, dest interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("token request failed: %s: %s", res.Status, string(body))
	}

	return json.NewDecoder(res.Body).Decode(dest)
}

func parseOutlookScopes(scopes string) []string {
	if strings.TrimSpace(scopes) == "" {
		return []string{"User.Read"}
	}
	parts := strings.Fields(scopes)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return []string{"User.Read"}
	}
	return result
}

// BuildAttachment creates a Graph attachment map from raw data.
func BuildAttachment(filename, contentType string, data []byte) map[string]interface{} {
	return map[string]interface{}{
		"@odata.type":  "#microsoft.graph.fileAttachment",
		"name":         filename,
		"contentType":  contentType,
		"contentBytes": base64.StdEncoding.EncodeToString(data),
	}
}
