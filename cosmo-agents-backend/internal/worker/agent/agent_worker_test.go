package agent

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

func TestTaskTypeConstants(t *testing.T) {
	assert.Equal(t, "agent:create", TypeAgentCreate)
	assert.Equal(t, "agent:update", TypeAgentUpdate)
	assert.Equal(t, "agent:delete", TypeAgentDelete)
	assert.Equal(t, "agent:sync_emails", TypeAgentSyncEmails)
	assert.Equal(t, "agent:refresh_token", TypeAgentRefreshToken)

	assert.NotEmpty(t, TypeAgentCreate)
	assert.NotEmpty(t, TypeAgentUpdate)
	assert.NotEmpty(t, TypeAgentDelete)
	assert.NotEmpty(t, TypeAgentSyncEmails)
	assert.NotEmpty(t, TypeAgentRefreshToken)
}
func TestAgentCreatePayload(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	payload := AgentCreatePayload{
		UserID:         userID,
		OrganizationID: &orgID,
		Name:           "Test Agent",
		Email:          "test@example.com",
		EmailProvider:  "gmail",
		Credentials:    map[string]interface{}{"access_token": "test"},
	}

	assert.Equal(t, userID, payload.UserID)
	assert.Equal(t, &orgID, payload.OrganizationID)
	assert.Equal(t, "Test Agent", payload.Name)
	assert.Equal(t, "test@example.com", payload.Email)
	assert.Equal(t, "gmail", payload.EmailProvider)
	assert.NotNil(t, payload.Credentials)
	assert.Equal(t, "test", payload.Credentials["access_token"])
}

func TestAgentUpdatePayload(t *testing.T) {
	agentID := uuid.New()
	userID := uuid.New()
	orgID := uuid.New()
	dailyLimit := 100
	maxDailyLimit := 500

	payload := AgentUpdatePayload{
		AgentID:        agentID,
		UserID:         &userID,
		OrganizationID: &orgID,
		Name:           testStringPtr("Updated Agent"),
		Email:          testStringPtr("updated@example.com"),
		Status:         testStringPtr("active"),
		EmailProvider:  testStringPtr("outlook"),
		DailyLimit:     &dailyLimit,
		MaxDailyLimit:  &maxDailyLimit,
		Credentials:    map[string]interface{}{"refresh_token": "test"},
	}

	assert.Equal(t, agentID, payload.AgentID)
	assert.Equal(t, &userID, payload.UserID)
	assert.Equal(t, &orgID, payload.OrganizationID)
	assert.Equal(t, "Updated Agent", *payload.Name)
	assert.Equal(t, "updated@example.com", *payload.Email)
	assert.Equal(t, "active", *payload.Status)
	assert.Equal(t, "outlook", *payload.EmailProvider)
	assert.Equal(t, 100, *payload.DailyLimit)
	assert.Equal(t, 500, *payload.MaxDailyLimit)
	assert.NotNil(t, payload.Credentials)
	assert.Equal(t, "test", payload.Credentials["refresh_token"])
}

func TestAgentDeletePayload(t *testing.T) {
	agentID := uuid.New()
	userID := uuid.New()

	payload := AgentDeletePayload{
		AgentID: agentID,
		UserID:  userID,
	}

	assert.Equal(t, agentID, payload.AgentID)
	assert.Equal(t, userID, payload.UserID)
}

func TestAgentSyncEmailsPayload(t *testing.T) {
	agentID := uuid.New()
	userID := uuid.New()
	since := time.Now()

	payload := AgentSyncEmailsPayload{
		AgentID:   agentID,
		UserID:    userID,
		SyncAll:   true,
		SyncSince: &since,
	}

	assert.Equal(t, agentID, payload.AgentID)
	assert.Equal(t, userID, payload.UserID)
	assert.True(t, payload.SyncAll)
	assert.Equal(t, &since, payload.SyncSince)
}

func TestAgentRefreshTokenPayload(t *testing.T) {
	agentID := uuid.New()
	userID := uuid.New()

	payload := AgentRefreshTokenPayload{
		AgentID: agentID,
		UserID:  userID,
	}

	assert.Equal(t, agentID, payload.AgentID)
	assert.Equal(t, userID, payload.UserID)
}

func TestAgentValidation(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	agent := &domain.Agent{
		Base: domain.Base{
			ID: uuid.New(),
		},
		UserID:          userID,
		OrganizationID:  &orgID,
		Name:            "Test Agent",
		Email:           "test@example.com",
		Status:          domain.AgentStatusActive,
		EmailProvider:   domain.AgentEmailProviderGmail,
		ValidCred:       testBoolPtr(true),
		DailyLimit:      testIntPtr(100),
		MaxDailyLimit:   testIntPtr(500),
		EmailsSentToday: testIntPtr(0),
	}

	assert.NotEqual(t, uuid.Nil, agent.Base.ID)
	assert.Equal(t, userID, agent.UserID)
	assert.Equal(t, "Test Agent", agent.Name)
	assert.Equal(t, "test@example.com", agent.Email)
	assert.Equal(t, domain.AgentStatusActive, agent.Status)
	assert.Equal(t, domain.AgentEmailProviderGmail, agent.EmailProvider)
	assert.True(t, *agent.ValidCred)
	assert.Equal(t, 100, *agent.DailyLimit)
	assert.Equal(t, 500, *agent.MaxDailyLimit)
	assert.Equal(t, 0, *agent.EmailsSentToday)
}

// Benchmark tests
func BenchmarkAgentPayloadCreation(b *testing.B) {
	userID := uuid.New()
	orgID := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		payload := AgentCreatePayload{
			UserID:         userID,
			OrganizationID: &orgID,
			Name:           "Benchmark Agent",
			Email:          "benchmark@example.com",
			EmailProvider:  "gmail",
			Credentials:    map[string]interface{}{"test": "value"},
		}
		_ = payload
	}
}

func BenchmarkAgentValidation(b *testing.B) {
	userID := uuid.New()
	orgID := uuid.New()
	agent := &domain.Agent{
		Base:            domain.Base{ID: uuid.New()},
		UserID:          userID,
		OrganizationID:  &orgID,
		Name:            "Test Agent",
		Email:           "test@example.com",
		Status:          domain.AgentStatusActive,
		EmailProvider:   domain.AgentEmailProviderGmail,
		DailyLimit:      testIntPtr(100),
		MaxDailyLimit:   testIntPtr(500),
		ValidCred:       testBoolPtr(true),
		EmailsSentToday: testIntPtr(0),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Basic validation checks
		_ = agent.UserID != uuid.Nil &&
			agent.Name != "" &&
			agent.Email != "" &&
			(agent.Status == domain.AgentStatusActive ||
				agent.Status == domain.AgentStatusInactive ||
				agent.Status == domain.AgentStatusInvalidGrant) &&
			(agent.EmailProvider == domain.AgentEmailProviderGmail ||
				agent.EmailProvider == domain.AgentEmailProviderOutlook)
	}
}

func TestAgentWorker_Integration(t *testing.T) {
	t.Skip("Integration tests require database setup")
}

func TestBoolPtr(t *testing.T) {
	ptr := boolPtr(true)
	if ptr == nil || !*ptr {
		t.Fatalf("expected pointer to true")
	}
}

// Enqueue helpers depend on real asynq client; skipped to avoid Redis dependency.

type AgentWorkerTestSuite struct {
	suite.Suite
}

func (suite *AgentWorkerTestSuite) SetupSuite() {
	suite.T().Skip("Test suite requires database setup")
}

func TestAgentWorkerSuite(t *testing.T) {
	suite.Run(t, new(AgentWorkerTestSuite))
}
func testIntPtr(v int) *int {
	return &v
}

func testBoolPtr(v bool) *bool {
	return &v
}

func testStringPtr(v string) *string {
	return &v
}
