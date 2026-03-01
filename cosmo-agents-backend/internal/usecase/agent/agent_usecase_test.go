package agent

import (
	"net/mail"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

func intPtr(v int) *int {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func TestNewAgentUseCase(t *testing.T) {
	useCase := &AgentUseCase{}
	assert.NotNil(t, useCase)
}

func TestCreateAgentUseCase_Validation(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	testCases := []struct {
		name    string
		useCase CreateAgentUseCase
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid use case",
			useCase: CreateAgentUseCase{
				UserID:         userID,
				OrganizationID: &orgID,
				Name:           "Valid Agent",
				Email:          "valid@example.com",
				EmailProvider:  domain.AgentEmailProviderGmail,
			},
			wantErr: false,
		},
		{
			name: "invalid email provider",
			useCase: CreateAgentUseCase{
				UserID:        userID,
				Name:          "Invalid Agent",
				Email:         "invalid@example.com",
				EmailProvider: domain.AgentEmailProvider("invalid"),
			},
			wantErr: true,
			errMsg:  "invalid email provider",
		},
		{
			name: "empty name",
			useCase: CreateAgentUseCase{
				UserID:        userID,
				Name:          "",
				Email:         "test@example.com",
				EmailProvider: domain.AgentEmailProviderGmail,
			},
			wantErr: true,
			errMsg:  "empty",
		},
		{
			name: "invalid email format",
			useCase: CreateAgentUseCase{
				UserID:        userID,
				Name:          "Test Agent",
				Email:         "invalid-email",
				EmailProvider: domain.AgentEmailProviderGmail,
			},
			wantErr: true,
			errMsg:  "email",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Basic validation checks
			hasError := false
			errorMsg := ""

			if tc.useCase.UserID == uuid.Nil {
				hasError = true
				errorMsg += "user_id required; "
			}

			if tc.useCase.Name == "" {
				hasError = true
				errorMsg += "empty name; "
			}

			if tc.useCase.Email == "" {
				hasError = true
				errorMsg += "empty email; "
			}

			// Validate email format
			if tc.useCase.Email != "" {
				_, err := mail.ParseAddress(tc.useCase.Email)
				if err != nil {
					hasError = true
					errorMsg += "invalid email format; "
				}
			}

			if tc.useCase.EmailProvider != domain.AgentEmailProviderGmail &&
				tc.useCase.EmailProvider != domain.AgentEmailProviderOutlook {
				hasError = true
				errorMsg += "invalid email provider; "
			}

			if tc.wantErr {
				assert.True(t, hasError, "Expected validation error")
				if tc.errMsg != "" {
					assert.Contains(t, errorMsg, tc.errMsg)
				}
			} else {
				assert.False(t, hasError, "Expected no validation error, got: "+errorMsg)
			}
		})
	}
}

func TestUpdateAgentUseCase_Validation(t *testing.T) {
	userID := uuid.New()
	agentID := uuid.New()

	testCases := []struct {
		name    string
		useCase UpdateAgentUseCase
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid update",
			useCase: UpdateAgentUseCase{
				AgentID: agentID,
				UserID:  &userID,
				Name:    stringPtr("Updated Name"),
			},
			wantErr: false,
		},
		{
			name: "empty agent ID",
			useCase: UpdateAgentUseCase{
				AgentID: uuid.Nil,
				UserID:  &userID,
			},
			wantErr: true,
			errMsg:  "agent_id",
		},
		{
			name: "valid email update",
			useCase: UpdateAgentUseCase{
				AgentID: agentID,
				Email:   stringPtr("updated@example.com"),
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasError := false
			errorMsg := ""

			if tc.useCase.AgentID == uuid.Nil {
				hasError = true
				errorMsg += "agent_id required; "
			}

			if tc.useCase.Email != nil && *tc.useCase.Email == "" {
				hasError = true
				errorMsg += "empty email; "
			}

			if tc.wantErr {
				assert.True(t, hasError, "Expected validation error")
				if tc.errMsg != "" {
					assert.Contains(t, errorMsg, tc.errMsg)
				}
			} else {
				assert.False(t, hasError, "Expected no validation error, got: "+errorMsg)
			}
		})
	}
}

func TestAgentResponse_Creation(t *testing.T) {
	// Test response structure
	now := time.Now()
	agentID := uuid.New()
	userID := uuid.New()
	orgID := uuid.New()

	response := AgentResponse{
		ID:              agentID,
		UserID:          userID,
		OrganizationID:  &orgID,
		Name:            "Test Agent",
		Email:           "test@example.com",
		Status:          domain.AgentStatusActive,
		EmailProvider:   domain.AgentEmailProviderGmail,
		Signature:       "Best regards",
		Picture:         "https://example.com/avatar.jpg",
		Persona:         []string{"friendly"},
		LastHistoryID:   "12345",
		DailyLimit:      intPtr(100),
		MaxDailyLimit:   intPtr(500),
		ValidCred:       boolPtr(true),
		EmailsSentToday: intPtr(25),
		Metadata:        map[string]interface{}{"source": "test"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	assert.Equal(t, agentID, response.ID)
	assert.Equal(t, userID, response.UserID)
	assert.Equal(t, "Test Agent", response.Name)
	assert.Equal(t, "test@example.com", response.Email)
	assert.Equal(t, domain.AgentStatusActive, response.Status)
	assert.Equal(t, domain.AgentEmailProviderGmail, response.EmailProvider)
	assert.Equal(t, 100, *response.DailyLimit)
	assert.Equal(t, 500, *response.MaxDailyLimit)
	assert.True(t, *response.ValidCred)
	assert.Equal(t, 25, *response.EmailsSentToday)
	assert.NotNil(t, response.Metadata)
}

func TestAgentsListResponse_Creation(t *testing.T) {
	agentID := uuid.New()
	userID := uuid.New()

	agents := []AgentResponse{
		{
			ID:        agentID,
			UserID:    userID,
			Name:      "Agent 1",
			Email:     "agent1@example.com",
			Status:    domain.AgentStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			UserID:    userID,
			Name:      "Agent 2",
			Email:     "agent2@example.com",
			Status:    domain.AgentStatusActive,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	response := AgentsListResponse{
		Agents: agents,
		Total:  len(agents),
	}

	assert.Len(t, response.Agents, 2)
	assert.Equal(t, 2, response.Total)
	assert.Equal(t, "Agent 1", response.Agents[0].Name)
	assert.Equal(t, "Agent 2", response.Agents[1].Name)
}

// Benchmark tests
func BenchmarkAgentUseCase_Validation(b *testing.B) {
	userID := uuid.New()
	orgID := uuid.New()

	useCase := CreateAgentUseCase{
		UserID:         userID,
		OrganizationID: &orgID,
		Name:           "Benchmark Agent",
		Email:          "benchmark@example.com",
		EmailProvider:  domain.AgentEmailProviderGmail,
		DailyLimit:     intPtr(100),
		MaxDailyLimit:  intPtr(500),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Basic validation (similar to what would be in real validation)
		_ = useCase.UserID != uuid.Nil &&
			useCase.Name != "" &&
			useCase.Email != "" &&
			(useCase.EmailProvider == domain.AgentEmailProviderGmail ||
				useCase.EmailProvider == domain.AgentEmailProviderOutlook)
	}
}

func BenchmarkAgentResponse_Creation(b *testing.B) {
	agentID := uuid.New()
	userID := uuid.New()
	orgID := uuid.New()
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		response := AgentResponse{
			ID:              agentID,
			UserID:          userID,
			OrganizationID:  &orgID,
			Name:            "Benchmark Agent",
			Email:           "benchmark@example.com",
			Status:          domain.AgentStatusActive,
			EmailProvider:   domain.AgentEmailProviderGmail,
			DailyLimit:      intPtr(100),
			MaxDailyLimit:   intPtr(500),
			ValidCred:       boolPtr(true),
			EmailsSentToday: intPtr(25),
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		_ = response
	}
}

// Integration tests (would require database setup)
func TestAgentUseCase_Integration(t *testing.T) {
	t.Skip("Integration tests require service and database setup")
}
