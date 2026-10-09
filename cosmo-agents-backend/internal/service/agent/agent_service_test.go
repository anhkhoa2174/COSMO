package agent

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

func intPtr(v int) *int {
	return &v
}

func TestAgentService_ValidateCreateRequest(t *testing.T) {
	service := &AgentService{}

	ctx := context.Background()
	userID := uuid.New()

	testCases := []struct {
		name    string
		req     CreateAgentRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: CreateAgentRequest{
				UserID:        userID,
				Name:          "Valid Agent",
				Email:         "valid@example.com",
				EmailProvider: domain.AgentEmailProviderGmail,
			},
			wantErr: false,
		},
		{
			name: "daily limit exceeds max",
			req: CreateAgentRequest{
				UserID:        userID,
				Name:          "Invalid Agent",
				Email:         "invalid@example.com",
				EmailProvider: domain.AgentEmailProviderGmail,
				DailyLimit:    intPtr(100),
				MaxDailyLimit: intPtr(50),
			},
			wantErr: true,
			errMsg:  "daily limit cannot exceed max daily limit",
		},
		{
			name: "invalid email provider",
			req: CreateAgentRequest{
				UserID:        userID,
				Name:          "Invalid Agent",
				Email:         "invalid@example.com",
				EmailProvider: domain.AgentEmailProvider("invalid"),
			},
			wantErr: true,
			errMsg:  "invalid email provider",
		},
		{
			name: "valid outlook provider",
			req: CreateAgentRequest{
				UserID:        userID,
				Name:          "Valid Outlook Agent",
				Email:         "valid@outlook.com",
				EmailProvider: domain.AgentEmailProviderOutlook,
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := service.validateCreateRequest(ctx, tc.req)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func newAgentService(t *testing.T) (*AgentService, *gorm.DB) {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Agent{}))
	return NewAgentService(db), db
}

func TestAgentService_CreateUpdateDelete(t *testing.T) {
	service, db := newAgentService(t)
	ctx := context.Background()

	req := CreateAgentRequest{
		UserID:        uuid.New(),
		Name:          "Agent One",
		Email:         "agent1@example.com",
		EmailProvider: domain.AgentEmailProviderGmail,
		Signature:     "Sig",
		DailyLimit:    intPtr(10),
		MaxDailyLimit: intPtr(20),
		Metadata:      map[string]interface{}{"team": "alpha"},
	}

	created, err := service.CreateAgent(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, req.Email, created.Email)
	assert.Equal(t, req.Name, created.Name)

	got, err := service.GetAgent(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)

	newName := "Agent Updated"
	newEmail := "new@example.com"
	reqUpdate := UpdateAgentRequest{Name: &newName, Email: &newEmail}
	updated, err := service.UpdateAgent(ctx, created.ID, reqUpdate)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	assert.Equal(t, newEmail, updated.Email)

	require.NoError(t, service.DeleteAgent(ctx, created.ID))
	none, err := service.agentRepository.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, none)

	// duplicate email check
	other, err := service.CreateAgent(ctx, CreateAgentRequest{
		UserID:        req.UserID,
		Name:          "Second",
		Email:         "dup@example.com",
		EmailProvider: domain.AgentEmailProviderGmail,
	})
	require.NoError(t, err)
	_, err = service.CreateAgent(ctx, CreateAgentRequest{
		UserID:        req.UserID,
		Name:          "Third",
		Email:         "dup@example.com",
		EmailProvider: domain.AgentEmailProviderGmail,
	})
	assert.Error(t, err)

	// conflict on update
	conflictEmail := "conflict@example.com"
	_, err = service.CreateAgent(ctx, CreateAgentRequest{
		UserID:        req.UserID,
		Name:          "Fourth",
		Email:         conflictEmail,
		EmailProvider: domain.AgentEmailProviderGmail,
	})
	require.NoError(t, err)
	reqUpdate.Email = &conflictEmail
	_, err = service.UpdateAgent(ctx, other.ID, reqUpdate)
	assert.Error(t, err)
	_ = db
}

func TestAgentService_GetAgentsAndActive(t *testing.T) {
	service, _ := newAgentService(t)
	ctx := context.Background()
	userID := uuid.New()

	_, err := service.CreateAgent(ctx, CreateAgentRequest{
		UserID:        userID,
		Name:          "Active",
		Email:         "active@example.com",
		EmailProvider: domain.AgentEmailProviderGmail,
	})
	require.NoError(t, err)

	agent2, err := service.CreateAgent(ctx, CreateAgentRequest{
		UserID:        userID,
		Name:          "Inactive",
		Email:         "inactive@example.com",
		EmailProvider: domain.AgentEmailProviderGmail,
	})
	require.NoError(t, err)
	inactive := domain.AgentStatusInactive
	_, err = service.UpdateAgent(ctx, agent2.ID, UpdateAgentRequest{Status: &inactive})
	require.NoError(t, err)

	agents, err := service.GetAgentsByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, agents, 2)

	activeAgents, err := service.GetActiveAgents(ctx)
	require.NoError(t, err)
	require.Len(t, activeAgents, 1)
	assert.Equal(t, "Active", activeAgents[0].Name)
}

// Tests that would require database integration
func TestAgentService_Integration(t *testing.T) {
	t.Skip("Integration tests require database setup - see repository tests for full examples")
}

// Simple test that doesn't require full service construction
func TestIntHelper(t *testing.T) {
	// Test intPtr
	v := 42
	ptr := intPtr(v)
	assert.NotNil(t, ptr)
	assert.Equal(t, v, *ptr)
}

// Benchmark tests for simple operations
func BenchmarkAgentService_Validation(b *testing.B) {
	service := &AgentService{}

	ctx := context.Background()
	userID := uuid.New()

	req := CreateAgentRequest{
		UserID:        userID,
		Name:          "Valid Agent",
		Email:         "valid@example.com",
		EmailProvider: domain.AgentEmailProviderGmail,
		DailyLimit:    intPtr(100),
		MaxDailyLimit: intPtr(500),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := service.validateCreateRequest(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
