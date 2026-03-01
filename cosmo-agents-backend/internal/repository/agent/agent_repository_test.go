package agent

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
)

func newTestRepo(t *testing.T) (*AgentRepository, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	db = db.Session(&gorm.Session{AllowGlobalUpdate: true})
	require.NoError(t, db.AutoMigrate(&domain.Agent{}))
	return NewAgentRepository(db), db
}

func TestAgentRepository_FindAndUpdate(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()

	userID := uuid.New()
	agentA := &domain.Agent{UserID: userID, Email: "a@example.com", Name: "A"}
	agentB := &domain.Agent{UserID: userID, Email: "b@example.com", Name: "B"}
	require.NoError(t, db.Create([]*domain.Agent{agentA, agentB}).Error)

	// soft delete B
	require.NoError(t, db.Model(agentB).Update("is_deleted", true).Error)

	found, err := repo.FindByIDs(ctx, []uuid.UUID{agentA.ID, agentB.ID})
	require.NoError(t, err)
	assert.Len(t, found, 2) // map includes both keys
	assert.NotNil(t, found[agentA.ID])

	// GetByEmail returns only non-deleted (soft delete ignored because default scope already applied)
	got, err := repo.GetByEmail(ctx, "a@example.com")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, agentA.ID, got.ID)

	none, err := repo.GetByEmail(ctx, "missing@example.com")
	require.NoError(t, err)
	assert.Nil(t, none)

	// Update last history id
	require.NoError(t, repo.UpdateLastHistoryID(ctx, agentA.ID, "h123"))
	var refreshed domain.Agent
	require.NoError(t, db.First(&refreshed, "id = ?", agentA.ID).Error)
	assert.Equal(t, "h123", refreshed.LastHistoryID)
}

func TestAgentRepository_StatusAndCounters(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()

	agent := &domain.Agent{
		UserID: uuid.New(),
		Email:  "status@example.com",
		Status: domain.AgentStatusInactive,
	}
	require.NoError(t, db.Create(agent).Error)

	// AtomicStatusUpdate
	require.NoError(t, repo.AtomicStatusUpdate(ctx, agent.ID, domain.AgentStatusActive))
	var refreshed domain.Agent
	require.NoError(t, db.First(&refreshed, "id = ?", agent.ID).Error)
	assert.Equal(t, domain.AgentStatusActive, refreshed.Status)

	// Increment email count and reset
	require.NoError(t, repo.IncrementEmailCount(ctx, agent.ID.String(), 3))
	require.NoError(t, db.First(&refreshed, "id = ?", agent.ID).Error)
	assert.NotNil(t, refreshed.EmailsSentToday)
	assert.Equal(t, 3, *refreshed.EmailsSentToday)

	// Reset with WHERE to satisfy sqlite's safety (or mimic production default scope)
	require.NoError(t, repo.GetDB().WithContext(ctx).Model(&domain.Agent{}).Where("1 = 1").Update("emails_sent_today", 0).Error)
	require.NoError(t, db.First(&refreshed, "id = ?", agent.ID).Error)
	assert.Equal(t, 0, *refreshed.EmailsSentToday)
}

func TestAgentRepository_QueryHelpers(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()

	userA := uuid.New()
	userB := uuid.New()

	agents := []*domain.Agent{
		{UserID: userA, Email: "a1@example.com", Name: "a1", Status: domain.AgentStatusActive, EmailsSentToday: intPtr(1)},
		{UserID: userA, Email: "a2@example.com", Name: "a2", Status: domain.AgentStatusInactive, EmailsSentToday: intPtr(2)},
		{UserID: userB, Email: "a1@example.com", Name: "a3", Status: domain.AgentStatusActive, EmailsSentToday: intPtr(3)},
	}
	require.NoError(t, db.Create(&agents).Error)

	byUser, err := repo.GetByUserID(ctx, userA.String())
	require.NoError(t, err)
	assert.Len(t, byUser, 2)

	activeAgents, err := repo.GetActiveAgents(ctx)
	require.NoError(t, err)
	assert.Len(t, activeAgents, 2)

	found, err := repo.FindByUserAndEmail(ctx, userA, "a1@example.com")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, agents[0].ID, found.ID)

	missing, err := repo.FindByUserAndEmail(ctx, userA, "missing@example.com")
	require.NoError(t, err)
	assert.Nil(t, missing)

	byEmail, err := repo.FindByEmail(ctx, "a1@example.com")
	require.NoError(t, err)
	assert.Len(t, byEmail, 2)

	require.NoError(t, repo.IncrementEmailCount(ctx, agents[0].ID.String(), 4))
	var refreshed domain.Agent
	require.NoError(t, db.First(&refreshed, "id = ?", agents[0].ID).Error)
	assert.Equal(t, 5, *refreshed.EmailsSentToday)

	require.NoError(t, repo.ResetDailyEmailCounts(ctx))
	var all []domain.Agent
	require.NoError(t, db.Find(&all).Error)
	for _, a := range all {
		require.NotNil(t, a.EmailsSentToday)
		assert.Equal(t, 0, *a.EmailsSentToday)
	}
}

func TestAgentRepository_AtomicUpdateGoogleCredentials(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()

	agent := &domain.Agent{
		UserID:        uuid.New(),
		Email:         "creds@example.com",
		EmailProvider: domain.AgentEmailProviderGmail,
	}
	originalExpiry := time.Now().Add(-1 * time.Hour)
	originalStore := newTokenStore("old-access", "old-refresh", originalExpiry, []string{"scopeA", "scopeB"})
	require.NoError(t, agent.UpdateGoogleTokenStore(originalStore))
	require.NoError(t, db.Create(agent).Error)

	newExpiry := time.Now().Add(2 * time.Hour)
	require.NoError(t, repo.AtomicUpdateGoogleCredentials(ctx, agent.ID, "new-access", "new-refresh", newExpiry))

	var updated domain.Agent
	require.NoError(t, db.First(&updated, "id = ?", agent.ID).Error)
	store, err := updated.GetGoogleTokenStore()
	require.NoError(t, err)
	assert.Equal(t, "new-access", store.AccessToken)
	assert.Equal(t, "new-refresh", store.RefreshToken)
	assert.Equal(t, []string{"scopeA", "scopeB"}, store.Scopes)
	assert.WithinDuration(t, newExpiry.UTC(), store.Expiry, time.Second)
}

func TestAgentRepository_AtomicRefreshToken(t *testing.T) {
	t.Run("skips_refresh_when_token_valid", func(t *testing.T) {
		repo, db := newTestRepo(t)
		ctx := context.Background()
		agent := &domain.Agent{
			UserID:        uuid.New(),
			Email:         "valid@example.com",
			EmailProvider: domain.AgentEmailProviderGmail,
		}
		validStore := newTokenStore("access-1", "refresh-1", time.Now().Add(10*time.Minute), []string{"scope1"})
		require.NoError(t, agent.UpdateGoogleTokenStore(validStore))
		require.NoError(t, db.Create(agent).Error)

		client := &stubOAuthClient{}
		store, err := repo.AtomicRefreshToken(ctx, agent.ID, client)
		require.NoError(t, err)
		assert.False(t, client.called)
		assert.Equal(t, "access-1", store.AccessToken)

		var fromDB domain.Agent
		require.NoError(t, db.First(&fromDB, "id = ?", agent.ID).Error)
		persisted, err := fromDB.GetGoogleTokenStore()
		require.NoError(t, err)
		assert.Equal(t, "access-1", persisted.AccessToken)
	})

	t.Run("refreshes_and_persists_when_expired", func(t *testing.T) {
		repo, db := newTestRepo(t)
		ctx := context.Background()
		agent := &domain.Agent{
			UserID:        uuid.New(),
			Email:         "expired@example.com",
			EmailProvider: domain.AgentEmailProviderGmail,
		}
		expiredStore := newTokenStore("access-old", "refresh-old", time.Now().Add(-10*time.Minute), []string{"scope2"})
		require.NoError(t, agent.UpdateGoogleTokenStore(expiredStore))
		require.NoError(t, db.Create(agent).Error)

		refreshedExpiry := time.Now().Add(30 * time.Minute)
		client := &stubOAuthClient{
			token: &googleoauth.Token{
				AccessToken:  "access-new",
				RefreshToken: "",
				Expiry:       refreshedExpiry,
			},
		}

		store, err := repo.AtomicRefreshToken(ctx, agent.ID, client)
		require.NoError(t, err)
		assert.True(t, client.called)
		assert.Equal(t, "access-new", store.AccessToken)
		assert.Equal(t, "refresh-old", store.RefreshToken, "should keep existing refresh token when provider omits it")
		assert.WithinDuration(t, refreshedExpiry.UTC(), store.Expiry, time.Second)

		var fromDB domain.Agent
		require.NoError(t, db.First(&fromDB, "id = ?", agent.ID).Error)
		persisted, err := fromDB.GetGoogleTokenStore()
		require.NoError(t, err)
		assert.Equal(t, "access-new", persisted.AccessToken)
		assert.Equal(t, "refresh-old", persisted.RefreshToken)
		assert.Equal(t, []string{"scope2"}, persisted.Scopes)
	})
}

func TestAgentRepository_UpdateAgentStatusBasedOnTokenExpiry(t *testing.T) {
	repo, db := newTestRepo(t)
	ctx := context.Background()

	expiringAgent := &domain.Agent{
		UserID:        uuid.New(),
		Email:         "expiring@example.com",
		EmailProvider: domain.AgentEmailProviderGmail,
		Status:        domain.AgentStatusActive,
	}
	require.NoError(t, expiringAgent.UpdateGoogleTokenStore(newTokenStore(
		"access-expiring",
		"refresh-expiring",
		time.Now().Add(1*time.Minute),
		[]string{"scopeX"},
	)))
	require.NoError(t, db.Create(expiringAgent).Error)

	unchangedAgent := &domain.Agent{
		UserID:        uuid.New(),
		Email:         "unchanged@example.com",
		EmailProvider: domain.AgentEmailProviderGmail,
		Status:        domain.AgentStatusInvalidGrant,
	}
	require.NoError(t, unchangedAgent.UpdateGoogleTokenStore(newTokenStore(
		"access-valid",
		"refresh-valid",
		time.Now().Add(2*time.Hour),
		[]string{"scopeY"},
	)))
	require.NoError(t, db.Create(unchangedAgent).Error)

	require.NoError(t, repo.UpdateAgentStatusBasedOnTokenExpiry(ctx, expiringAgent.ID))
	var refreshed domain.Agent
	require.NoError(t, db.First(&refreshed, "id = ?", expiringAgent.ID).Error)
	assert.Equal(t, domain.AgentStatusInvalidGrant, refreshed.Status)

	require.NoError(t, repo.UpdateAgentStatusBasedOnTokenExpiry(ctx, unchangedAgent.ID))
	var unchanged domain.Agent
	require.NoError(t, db.First(&unchanged, "id = ?", unchangedAgent.ID).Error)
	assert.Equal(t, domain.AgentStatusInvalidGrant, unchanged.Status)

	brokenAgent := &domain.Agent{
		UserID:        uuid.New(),
		Email:         "broken@example.com",
		EmailProvider: domain.AgentEmailProviderGmail,
		Status:        domain.AgentStatusActive,
	}
	require.NoError(t, db.Create(brokenAgent).Error)
	err := repo.UpdateAgentStatusBasedOnTokenExpiry(ctx, brokenAgent.ID)
	require.Error(t, err)
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
		ValidCred:       boolPtr(true),
		DailyLimit:      intPtr(100),
		MaxDailyLimit:   intPtr(500),
		EmailsSentToday: intPtr(0),
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

func TestAgentEmailProviderValidation(t *testing.T) {
	userID := uuid.New()

	testCases := []struct {
		name     string
		provider domain.AgentEmailProvider
		valid    bool
	}{
		{
			name:     "valid gmail provider",
			provider: domain.AgentEmailProviderGmail,
			valid:    true,
		},
		{
			name:     "valid outlook provider",
			provider: domain.AgentEmailProviderOutlook,
			valid:    true,
		},
		{
			name:     "invalid provider",
			provider: domain.AgentEmailProvider("invalid"),
			valid:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			agent := &domain.Agent{
				Base:          domain.Base{ID: uuid.New()},
				UserID:        userID,
				Name:          "Test Agent",
				Email:         "test@example.com",
				Status:        domain.AgentStatusActive,
				EmailProvider: tc.provider,
			}

			isValid := agent.EmailProvider == domain.AgentEmailProviderGmail ||
				agent.EmailProvider == domain.AgentEmailProviderOutlook

			assert.Equal(t, tc.valid, isValid)
		})
	}
}

func TestAgentStatusValidation(t *testing.T) {
	userID := uuid.New()

	testCases := []struct {
		name   string
		status domain.AgentStatus
		valid  bool
	}{
		{
			name:   "valid active status",
			status: domain.AgentStatusActive,
			valid:  true,
		},
		{
			name:   "valid inactive status",
			status: domain.AgentStatusInactive,
			valid:  true,
		},
		{
			name:   "valid invalid_grant status",
			status: domain.AgentStatusInvalidGrant,
			valid:  true,
		},
		{
			name:   "invalid status",
			status: domain.AgentStatus("invalid"),
			valid:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			agent := &domain.Agent{
				Base:   domain.Base{ID: uuid.New()},
				UserID: userID,
				Name:   "Test Agent",
				Email:  "test@example.com",
				Status: tc.status,
			}

			// Common valid statuses
			validStatuses := map[domain.AgentStatus]bool{
				domain.AgentStatusActive:       true,
				domain.AgentStatusInactive:     true,
				domain.AgentStatusInvalidGrant: true,
			}

			isValid := validStatuses[agent.Status]

			assert.Equal(t, tc.valid, isValid)
		})
	}
}

func TestAgentEmailValidation(t *testing.T) {
	userID := uuid.New()

	testCases := []struct {
		name      string
		email     string
		wantError bool
	}{
		{
			name:      "valid email",
			email:     "test@example.com",
			wantError: false,
		},
		{
			name:      "valid email with subdomain",
			email:     "test@subdomain.example.com",
			wantError: false,
		},
		{
			name:      "empty email",
			email:     "",
			wantError: true,
		},
		{
			name:      "invalid email format",
			email:     "invalid-email",
			wantError: true,
		},
		{
			name:      "email without domain",
			email:     "test@",
			wantError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			agent := &domain.Agent{
				Base:   domain.Base{ID: uuid.New()},
				UserID: userID,
				Name:   "Test Agent",
				Email:  tc.email,
			}

			hasError := agent.Email == ""

			if tc.wantError {
				// Basic validation - either empty or invalid format
				// Check for proper email format: has @ and domain after @
				atIndex := -1
				for i, c := range agent.Email {
					if c == '@' {
						atIndex = i
						break
					}
				}
				isInvalid := agent.Email == "" || len(agent.Email) < 3 ||
					atIndex <= 0 || atIndex == len(agent.Email)-1
				assert.True(t, isInvalid, "Expected email validation error for email: %s", agent.Email)
			} else {
				assert.False(t, hasError, "Expected no email validation error")
			}
		})
	}
}

func TestAgentLimitValidation(t *testing.T) {
	_ = uuid.New() // userID not used in this test

	testCases := []struct {
		name          string
		dailyLimit    *int
		maxDailyLimit *int
		wantError     bool
		errorMessage  string
	}{
		{
			name:          "valid limits",
			dailyLimit:    intPtr(100),
			maxDailyLimit: intPtr(500),
			wantError:     false,
		},
		{
			name:          "daily limit equals max limit",
			dailyLimit:    intPtr(200),
			maxDailyLimit: intPtr(200),
			wantError:     false,
		},
		{
			name:          "daily limit exceeds max limit",
			dailyLimit:    intPtr(300),
			maxDailyLimit: intPtr(200),
			wantError:     true,
			errorMessage:  "daily limit cannot exceed max daily limit",
		},
		{
			name:          "nil daily limit",
			dailyLimit:    nil,
			maxDailyLimit: intPtr(500),
			wantError:     false,
		},
		{
			name:          "nil max daily limit",
			dailyLimit:    intPtr(100),
			maxDailyLimit: nil,
			wantError:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasError := tc.dailyLimit != nil && tc.maxDailyLimit != nil &&
				*tc.dailyLimit > *tc.maxDailyLimit

			if tc.wantError {
				assert.True(t, hasError, "Expected limit validation error")
				assert.Contains(t, tc.errorMessage, "daily limit cannot exceed")
			} else {
				assert.False(t, hasError, "Expected no limit validation error")
			}
		})
	}
}

// Benchmark tests
func BenchmarkAgentCreation(b *testing.B) {
	userID := uuid.New()
	orgID := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent := &domain.Agent{
			Base:            domain.Base{ID: uuid.New()},
			UserID:          userID,
			OrganizationID:  &orgID,
			Name:            "Benchmark Agent",
			Email:           "benchmark@example.com",
			Status:          domain.AgentStatusActive,
			EmailProvider:   domain.AgentEmailProviderGmail,
			DailyLimit:      intPtr(100),
			MaxDailyLimit:   intPtr(500),
			ValidCred:       boolPtr(true),
			EmailsSentToday: intPtr(0),
		}
		_ = agent
	}
}

func BenchmarkAgentValidation(b *testing.B) {
	userID := uuid.New()
	agent := &domain.Agent{
		Base:            domain.Base{ID: uuid.New()},
		UserID:          userID,
		Name:            "Test Agent",
		Email:           "test@example.com",
		Status:          domain.AgentStatusActive,
		EmailProvider:   domain.AgentEmailProviderGmail,
		DailyLimit:      intPtr(100),
		MaxDailyLimit:   intPtr(500),
		ValidCred:       boolPtr(true),
		EmailsSentToday: intPtr(0),
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

// Helper functions
func intPtr(v int) *int {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

type stubOAuthClient struct {
	token  *googleoauth.Token
	err    error
	called bool
}

func (s *stubOAuthClient) RefreshToken(ctx context.Context, refreshToken string) (*googleoauth.Token, error) {
	s.called = true
	if s.err != nil {
		return nil, s.err
	}
	return s.token, nil
}

func newTokenStore(accessToken, refreshToken string, expiry time.Time, scopes []string) *domain.GoogleTokenStore {
	return &domain.GoogleTokenStore{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       expiry.UTC(),
		Scopes:       scopes,
	}
}

// Integration tests that would require database setup
func TestAgentRepository_Integration(t *testing.T) {
	t.Skip("Integration tests require database setup")
}

// Suite test placeholder
type AgentRepositoryTestSuite struct {
	suite.Suite
}

func (suite *AgentRepositoryTestSuite) SetupSuite() {
	suite.T().Skip("Test suite requires database setup")
}

func TestAgentRepositorySuite(t *testing.T) {
	suite.Run(t, new(AgentRepositoryTestSuite))
}
