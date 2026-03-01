package agent

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/google_token_store"
)

func TestNewGoogleTokenStoreFromMap(t *testing.T) {
	input := map[string]interface{}{
		"token":         "access",
		"refresh_token": "refresh",
		"scopes":        []string{"scope1", "scope2"},
		"expiry":        "2024-10-01T12:30:45Z",
	}

	store, err := google_token_store.NewGoogleTokenStoreFromMap(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if store.AccessToken != "access" || store.RefreshToken != "refresh" {
		t.Fatalf("unexpected tokens: %+v", store)
	}

	expectExpiry := time.Date(2024, 10, 1, 12, 30, 45, 0, time.UTC)
	if !store.Expiry.Equal(expectExpiry) {
		t.Fatalf("expected expiry %s, got %s", expectExpiry, store.Expiry)
	}
}

func TestGoogleTokenStoreToMap(t *testing.T) {
	store := google_token_store.GoogleTokenStore{
		AccessToken:  "token",
		RefreshToken: "refresh",
		Scopes:       []string{"scope"},
		Expiry:       time.Date(2024, 10, 1, 12, 30, 45, 987000000, time.UTC),
	}

	output := store.ToMap()
	if output["token"] != "token" || output["refresh_token"] != "refresh" {
		t.Fatalf("unexpected map output: %+v", output)
	}

	expiry, ok := output["expiry"].(string)
	if !ok {
		t.Fatalf("expected expiry string: %+v", output["expiry"])
	}
	if expiry != "2024-10-01T12:30:45Z" {
		t.Fatalf("expected expiry 2024-10-01T12:30:45Z, got %s", expiry)
	}
}

func TestAgentGetGoogleTokenStore(t *testing.T) {
	creds := map[string]interface{}{
		"token":         "access",
		"refresh_token": "refresh",
		"scopes":        []string{},
		"expiry":        "2024-10-01T12:30:45Z",
	}
	payload, _ := json.Marshal(creds)

	agent := Agent{
		Credentials: base.JSON(payload),
	}

	store, err := agent.GetGoogleTokenStore()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if store.AccessToken != "access" || store.RefreshToken != "refresh" {
		t.Fatalf("unexpected store values: %+v", store)
	}
}

func TestAgentGetGoogleTokenStoreEmpty(t *testing.T) {
	agent := Agent{}
	_, err := agent.GetGoogleTokenStore()
	if err == nil {
		t.Fatal("expected error when credentials are empty")
	}
}

func TestAgentUpdateGoogleTokenStore(t *testing.T) {
	existing := map[string]interface{}{
		"foo": "bar",
	}
	payload, _ := json.Marshal(existing)
	agent := Agent{
		Credentials: base.JSON(payload),
	}

	store := &google_token_store.GoogleTokenStore{
		AccessToken:  "access",
		RefreshToken: "refresh",
		Scopes:       []string{"scope"},
		Expiry:       time.Date(2024, 10, 1, 12, 30, 45, 0, time.UTC),
	}

	if err := agent.UpdateGoogleTokenStore(store); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if agent.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}

	var merged map[string]interface{}
	if err := json.Unmarshal(agent.Credentials, &merged); err != nil {
		t.Fatalf("failed to unmarshal credentials: %v", err)
	}

	if merged["foo"] != "bar" {
		t.Fatalf("expected existing key to remain, got %v", merged["foo"])
	}
	if merged["token"] != "access" || merged["refresh_token"] != "refresh" {
		t.Fatalf("expected token data to be merged: %+v", merged)
	}
}

func TestAgentTableName(t *testing.T) {
	agent := Agent{}
	if agent.TableName() != "agents" {
		t.Fatalf("expected table name 'agents', got %q", agent.TableName())
	}
}

func TestAgentSetMetadata(t *testing.T) {
	agent := Agent{}
	metadata := map[string]any{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	if err := agent.SetMetadata(metadata); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the metadata was set
	var result map[string]any
	if err := json.Unmarshal(agent.CMetadata, &result); err != nil {
		t.Fatalf("failed to unmarshal metadata: %v", err)
	}

	if result["key1"] != "value1" || result["key2"] != float64(123) || result["key3"] != true {
		t.Fatalf("unexpected metadata values: %+v", result)
	}
}

func TestAgentGetMetadata(t *testing.T) {
	metadata := map[string]any{
		"test": "value",
		"num":  42,
	}
	data, _ := json.Marshal(metadata)
	agent := Agent{
		CMetadata: base.JSONB(data),
	}

	result, err := agent.GetMetadata()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result["test"] != "value" || result["num"] != float64(42) {
		t.Fatalf("unexpected metadata values: %+v", result)
	}
}

func TestAgentGetMetadataEmpty(t *testing.T) {
	agent := Agent{
		CMetadata: base.JSONB([]byte("{}")),
	}

	metadata, err := agent.GetMetadata()
	if err != nil {
		t.Fatalf("expected no error for empty metadata, got %v", err)
	}

	if metadata == nil {
		t.Fatal("expected empty map, not nil")
	}
}

func TestAgentSetCredentials(t *testing.T) {
	agent := Agent{}
	credentials := map[string]interface{}{
		"token":    "access123",
		"username": "test@example.com",
	}

	if err := agent.SetCredentials(credentials); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(agent.Credentials, &result); err != nil {
		t.Fatalf("failed to unmarshal credentials: %v", err)
	}

	if result["token"] != "access123" || result["username"] != "test@example.com" {
		t.Fatalf("unexpected credentials: %+v", result)
	}
}

func TestAgentGetCredentials(t *testing.T) {
	credentials := map[string]interface{}{
		"secret": "value123",
	}
	data, _ := json.Marshal(credentials)
	agent := Agent{
		Credentials: base.JSON(data),
	}

	result, err := agent.GetCredentials()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result["secret"] != "value123" {
		t.Fatalf("unexpected credentials value: %+v", result)
	}
}

func TestAgentGetCredentialsEmpty(t *testing.T) {
	agent := Agent{
		Credentials: base.JSON([]byte("{}")),
	}

	creds, err := agent.GetCredentials()
	if err != nil {
		t.Fatalf("expected no error for empty credentials, got %v", err)
	}

	if creds == nil {
		t.Fatal("expected empty map, not nil")
	}
}

func TestAgentCheckTokenStatusNonGmail(t *testing.T) {
	agent := Agent{
		EmailProvider: AgentEmailProviderOutlook,
		Status:        AgentStatusActive,
	}

	status, err := agent.CheckTokenStatus()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != AgentStatusActive {
		t.Fatalf("expected status %q, got %q", AgentStatusActive, status)
	}
}

func TestAgentCheckTokenStatusInactive(t *testing.T) {
	agent := Agent{
		EmailProvider: AgentEmailProviderGmail,
		Status:        AgentStatusInactive,
	}

	status, err := agent.CheckTokenStatus()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != AgentStatusInactive {
		t.Fatalf("expected status %q, got %q", AgentStatusInactive, status)
	}
}

func TestAgentCheckTokenStatusInvalidGrant(t *testing.T) {
	agent := Agent{
		EmailProvider: AgentEmailProviderGmail,
		Status:        AgentStatusInvalidGrant,
	}

	status, err := agent.CheckTokenStatus()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != AgentStatusInvalidGrant {
		t.Fatalf("expected status %q, got %q", AgentStatusInvalidGrant, status)
	}
}

func TestAgentCheckTokenStatusNoCredentials(t *testing.T) {
	agent := Agent{
		EmailProvider: AgentEmailProviderGmail,
		Status:        AgentStatusActive,
		Credentials:   base.JSON([]byte{}),
	}

	status, err := agent.CheckTokenStatus()
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}

	if status != AgentStatusInactive {
		t.Fatalf("expected status %q, got %q", AgentStatusInactive, status)
	}
}

func TestAgentCheckTokenStatusExpired(t *testing.T) {
	creds := map[string]interface{}{
		"token":         "access",
		"refresh_token": "refresh",
		"scopes":        []string{"email"},
		"expiry":        time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	}
	payload, _ := json.Marshal(creds)

	agent := Agent{
		EmailProvider: AgentEmailProviderGmail,
		Status:        AgentStatusActive,
		Credentials:   base.JSON(payload),
	}

	status, err := agent.CheckTokenStatus()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != AgentStatusInvalidGrant {
		t.Fatalf("expected status %q, got %q", AgentStatusInvalidGrant, status)
	}
}

func TestAgentCheckTokenStatusValid(t *testing.T) {
	creds := map[string]interface{}{
		"token":         "access",
		"refresh_token": "refresh",
		"scopes":        []string{"email"},
		"expiry":        time.Now().Add(2 * time.Hour).Format(time.RFC3339),
	}
	payload, _ := json.Marshal(creds)

	agent := Agent{
		EmailProvider: AgentEmailProviderGmail,
		Status:        AgentStatusActive,
		Credentials:   base.JSON(payload),
	}

	status, err := agent.CheckTokenStatus()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if status != AgentStatusActive {
		t.Fatalf("expected status %q, got %q", AgentStatusActive, status)
	}
}

func TestAgentUpdateGoogleTokenStoreNil(t *testing.T) {
	agent := Agent{}

	if err := agent.UpdateGoogleTokenStore(nil); err == nil {
		t.Fatal("expected error for nil token store")
	}
}

func TestAgentBeforeCreateDefaults(t *testing.T) {
	agent := Agent{}

	if err := agent.BeforeCreate(nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if agent.ID == uuid.Nil {
		t.Fatal("expected ID to be set")
	}
	if agent.Signature != defaultAgentSignature {
		t.Fatalf("expected default signature, got %q", agent.Signature)
	}
	if agent.Status != AgentStatusActive {
		t.Fatalf("expected status %q, got %q", AgentStatusActive, agent.Status)
	}
	if agent.DailyLimit == nil || *agent.DailyLimit != defaultDailyLimit {
		t.Fatalf("unexpected daily limit: %+v", agent.DailyLimit)
	}
	if agent.MaxDailyLimit == nil || *agent.MaxDailyLimit != defaultMaxDailyLimit {
		t.Fatalf("unexpected max daily limit: %+v", agent.MaxDailyLimit)
	}
	if agent.ValidCred == nil || !*agent.ValidCred {
		t.Fatalf("expected valid_cred default true, got %+v", agent.ValidCred)
	}
	if agent.EmailsSentToday == nil || *agent.EmailsSentToday != defaultEmailsSent {
		t.Fatalf("unexpected emails_sent_today: %+v", agent.EmailsSentToday)
	}
	if string(agent.CMetadata) != "{}" {
		t.Fatalf("expected empty cmetadata, got %s", string(agent.CMetadata))
	}
	if agent.Persona == nil {
		t.Fatal("expected persona to be initialised")
	}
}

func TestAgentBeforeCreateBaseError(t *testing.T) {
	agent := Agent{}
	// Create a mock DB that returns an error
	// Note: This test would require a more complex setup with GORM mock
	// For now, we'll just test that the method signature works
	_ = agent.BeforeCreate(nil)
}
