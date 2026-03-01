package outlook

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/rockship/cosmo-agents-go/pkg/config"
)

type mockTransport struct {
	handler  func(req *http.Request) *http.Response
	requests []*http.Request
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.requests = append(m.requests, req)
	if m.handler == nil {
		return nil, nil
	}
	resp := m.handler(req)
	if resp.Body == nil {
		resp.Body = io.NopCloser(strings.NewReader(""))
	}
	return resp, nil
}

func TestAuthorizationURL(t *testing.T) {
	svc := NewOutlookService(config.OutlookConfig{
		ClientID:     "client",
		ClientSecret: "secret",
		RedirectURI:  "https://app.example.com/callback",
		Authority:    "https://login.example.com/",
		Scopes:       "User.Read Mail.Read",
	})

	authURL, err := svc.AuthorizationURL("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse auth url: %v", err)
	}

	q := parsed.Query()
	if q.Get("client_id") != "client" {
		t.Fatalf("client_id not set correctly: %v", q.Get("client_id"))
	}
	if !strings.Contains(q.Get("scope"), "Mail.Read") {
		t.Fatalf("expected scopes to contain Mail.Read, got %s", q.Get("scope"))
	}
}

func TestAuthorizationURLNilService(t *testing.T) {
	var svc *OutlookService
	if _, err := svc.AuthorizationURL(""); err == nil {
		t.Fatal("expected error when service is nil")
	}
}

func TestExchangeAndRefreshToken(t *testing.T) {
	mt := &mockTransport{
		handler: func(req *http.Request) *http.Response {
			body := `{"token_type":"Bearer","access_token":"at","refresh_token":"rt","expires_in":3600}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     make(http.Header),
			}
		},
	}

	svc := &OutlookService{
		httpClient:   &http.Client{Transport: mt},
		clientID:     "client",
		clientSecret: "secret",
		redirectURI:  "https://redirect",
		authority:    "https://login.example.com",
		scopes:       []string{"User.Read"},
	}

	token, err := svc.ExchangeCode(context.Background(), "code123", "")
	if err != nil {
		t.Fatalf("exchange code error: %v", err)
	}
	if token.AccessToken != "at" || token.RefreshToken != "rt" {
		t.Fatalf("unexpected token response: %+v", token)
	}

	refreshed, err := svc.RefreshToken(context.Background(), "old-refresh")
	if err != nil {
		t.Fatalf("refresh token error: %v", err)
	}
	if refreshed.AccessToken != "at" {
		t.Fatalf("unexpected refreshed token: %+v", refreshed)
	}

	if len(mt.requests) != 2 {
		t.Fatalf("expected two token requests, got %d", len(mt.requests))
	}

	firstBody, _ := io.ReadAll(mt.requests[0].Body)
	if !strings.Contains(string(firstBody), "authorization_code") {
		t.Fatalf("expected authorization_code grant, got %s", string(firstBody))
	}
}

func TestGetGraphError(t *testing.T) {
	mt := &mockTransport{
		handler: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("boom")),
				Header:     make(http.Header),
			}
		},
	}

	svc := &OutlookService{
		httpClient: &http.Client{Transport: mt},
		scopes:     []string{"User.Read"},
	}

	if _, err := svc.GetContacts(context.Background(), "token"); err == nil {
		t.Fatal("expected error for graph failure")
	}
}

func TestPostGraphBranches(t *testing.T) {
	mt := &mockTransport{
		handler: func(req *http.Request) *http.Response {
			switch {
			case req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/sendMail"):
				return &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}
			case req.Method == http.MethodPost && strings.Contains(req.URL.Path, "/reply"):
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true}`)), Header: make(http.Header)}
			default:
				return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader("bad")), Header: make(http.Header)}
			}
		},
	}

	svc := &OutlookService{
		httpClient: &http.Client{Transport: mt},
		scopes:     []string{"User.Read"},
	}

	accepted, err := svc.SendMail(context.Background(), "token", map[string]interface{}{"a": 1})
	if err != nil {
		t.Fatalf("send mail error: %v", err)
	}
	if accepted["status"] != "accepted" {
		t.Fatalf("expected accepted status, got %v", accepted)
	}

	reply, err := svc.ReplyMail(context.Background(), "token", "123", map[string]interface{}{"b": 2})
	if err != nil {
		t.Fatalf("reply mail error: %v", err)
	}
	if ok, found := reply["ok"]; !found || !ok.(bool) {
		t.Fatalf("expected ok reply, got %v", reply)
	}

	if _, err := svc.ForwardMail(context.Background(), "token", "123", map[string]interface{}{}); err == nil {
		t.Fatal("expected error for bad status")
	}
}

func TestParseScopesAndAttachment(t *testing.T) {
	if scopes := parseOutlookScopes(""); len(scopes) != 1 || scopes[0] != "User.Read" {
		t.Fatalf("unexpected default scopes: %v", scopes)
	}

	scopes := parseOutlookScopes("User.Read  Calendars.Read")
	if len(scopes) != 2 || scopes[1] != "Calendars.Read" {
		t.Fatalf("unexpected parsed scopes: %v", scopes)
	}

	att := BuildAttachment("file.txt", "text/plain", []byte("hello"))
	if att["name"] != "file.txt" || att["contentType"] != "text/plain" {
		t.Fatalf("attachment metadata mismatch: %v", att)
	}
	contentBytes, _ := att["contentBytes"].(string)
	if contentBytes == "" || !strings.Contains(contentBytes, "aGVsbG8") {
		t.Fatalf("attachment content not base64 encoded: %v", contentBytes)
	}
}
