package gmail

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/google_token_store"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	gmailRepo "github.com/rockship/cosmo-agents-go/internal/repository/gmail"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
)

// Test OAuth state signer functionality
func TestOAuthStateSigner(t *testing.T) {
	secretKey := "test-secret-key"
	signer := newOAuthStateSigner(secretKey)

	t.Run("sign and verify valid state", func(t *testing.T) {
		userID := uuid.New()

		state, err := signer.SignState(userID)
		assert.NoError(t, err)
		assert.NotEmpty(t, state)

		verifiedUserID, err := signer.VerifyState(state)
		assert.NoError(t, err)
		assert.Equal(t, userID, verifiedUserID)
	})

	t.Run("verify invalid state format", func(t *testing.T) {
		invalidState := "invalid.state.format"

		_, err := signer.VerifyState(invalidState)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state format")
	})

	t.Run("verify expired state", func(t *testing.T) {
		userID := uuid.New()

		// Create a signer with a very short timeout for testing
		signer := &oauthStateSigner{secretKey: []byte(secretKey)}

		// Manually create an expired state
		payload := statePayload{
			UserID:    userID,
			Timestamp: time.Now().Unix() - 700, // More than 10 minutes ago
			Nonce:     uuid.New().String()[:8],
		}

		payloadBytes, _ := json.Marshal(payload)
		hash := hmac.New(sha256.New, signer.secretKey)
		hash.Write(payloadBytes)
		signature := hash.Sum(nil)

		payloadB64 := base64.URLEncoding.EncodeToString(payloadBytes)
		signatureB64 := base64.URLEncoding.EncodeToString(signature)

		expiredState := payloadB64 + "." + signatureB64

		_, err := signer.VerifyState(expiredState)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "state token expired")
	})

	t.Run("empty secret key panic", func(t *testing.T) {
		assert.Panics(t, func() {
			newOAuthStateSigner("")
		})
	})

	t.Run("verify tampered signature", func(t *testing.T) {
		userID := uuid.New()
		state, err := signer.SignState(userID)
		require.NoError(t, err)
		parts := strings.Split(state, ".")
		tampered := parts[0] + ".invalidsignature"

		_, verifyErr := signer.VerifyState(tampered)
		assert.Error(t, verifyErr)
		assert.Contains(t, verifyErr.Error(), "invalid state signature")
	})
}

// Test Gmail notification payload unmarshaling
func TestGmailNotificationPayload_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		expected    *GmailNotificationPayload
		expectError bool
	}{
		{
			name: "string history ID",
			data: `{
				"email_address": "test@example.com",
				"history_id": "12345"
			}`,
			expected: &GmailNotificationPayload{
				EmailAddress: "test@example.com",
				HistoryID:    "12345",
			},
			expectError: false,
		},
		{
			name: "numeric history ID",
			data: `{
				"email_address": "test@example.com",
				"history_id": 12345
			}`,
			expected: &GmailNotificationPayload{
				EmailAddress: "test@example.com",
				HistoryID:    "12345",
			},
			expectError: false,
		},
		{
			name: "null history ID",
			data: `{
				"email_address": "test@example.com",
				"history_id": null
			}`,
			expected: &GmailNotificationPayload{
				EmailAddress: "test@example.com",
				HistoryID:    "",
			},
			expectError: false,
		},
		{
			name: "missing history ID",
			data: `{
				"email_address": "test@example.com"
			}`,
			expected: &GmailNotificationPayload{
				EmailAddress: "test@example.com",
				HistoryID:    "",
			},
			expectError: false,
		},
		{
			name:        "invalid JSON",
			data:        `{"invalid": json}`,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var payload GmailNotificationPayload
			err := json.Unmarshal([]byte(tt.data), &payload)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.EmailAddress, payload.EmailAddress)
				assert.Equal(t, tt.expected.HistoryID, payload.HistoryID)
			}
		})
	}
}

// Test GmailService constructor
func TestNewGmailService(t *testing.T) {
	config := GmailConfig{
		OAuthClient:      nil,
		AgentRepo:        nil,
		GmailAccountRepo: nil,
		RedisClient:      nil,
		JWTSecret:        "test-secret",
		WorkerClient:     nil,
		PubsubTopic:      "test-topic",
	}

	service := NewGmailService(config)

	assert.NotNil(t, service)
	assert.Equal(t, config.OAuthClient, service.oauthClient)
	assert.Equal(t, config.AgentRepo, service.agentRepo)
	assert.Equal(t, config.GmailAccountRepo, service.gmailAccountRepo)
	assert.Equal(t, config.RedisClient, service.redisClient)
	assert.Equal(t, config.JWTSecret, service.jwtSecret)
	assert.Equal(t, config.WorkerClient, service.workerClient)
	assert.Equal(t, config.PubsubTopic, service.pubsubTopic)
	assert.NotNil(t, service.stateSigner)
}

// Mock GmailAccount for testing
type MockGmailAccount struct {
	mock.Mock
}

func (m *MockGmailAccount) GetGoogleTokenStore() (*domain.GoogleTokenStore, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.GoogleTokenStore), args.Error(1)
}

func (m *MockGmailAccount) UpdateGoogleTokenStore(store *domain.GoogleTokenStore) error {
	args := m.Called(store)
	return args.Error(0)
}

// Test GmailService error scenarios
func TestGmailService_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name         string
		setupService func() *GmailService
		testFunc     func(*GmailService)
	}{
		{
			name: "GetAuthorizationURL with nil OAuth client",
			setupService: func() *GmailService {
				config := GmailConfig{
					OAuthClient:      nil,
					AgentRepo:        nil,
					GmailAccountRepo: nil,
					JWTSecret:        "test-secret",
				}
				return NewGmailService(config)
			},
			testFunc: func(s *GmailService) {
				result, err := s.GetAuthorizationURL(context.Background(), "http://localhost:8080/callback")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "oauth client not configured")
				assert.Empty(t, result)
			},
		},
		{
			name: "HandleOAuth2Callback with missing code",
			setupService: func() *GmailService {
				config := GmailConfig{
					OAuthClient:      nil,
					AgentRepo:        nil,
					GmailAccountRepo: nil,
					JWTSecret:        "test-secret",
				}
				return NewGmailService(config)
			},
			testFunc: func(s *GmailService) {
				result, err := s.HandleOAuth2Callback(context.Background(), "", "valid.state", "http://localhost:8080/callback")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "missing authorization code")
				assert.Nil(t, result)
			},
		},
		{
			name: "HandleOAuth2Callback with invalid state",
			setupService: func() *GmailService {
				config := GmailConfig{
					OAuthClient:      nil,
					AgentRepo:        nil,
					GmailAccountRepo: nil,
					JWTSecret:        "test-secret",
				}
				return NewGmailService(config)
			},
			testFunc: func(s *GmailService) {
				result, err := s.HandleOAuth2Callback(context.Background(), "valid-code", "invalid.state", "http://localhost:8080/callback")
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid or expired state")
				assert.Nil(t, result)
			},
		},
		{
			name: "StartWatch with no pubsub topic",
			setupService: func() *GmailService {
				config := GmailConfig{
					OAuthClient:      nil,
					AgentRepo:        nil,
					GmailAccountRepo: nil,
					JWTSecret:        "test-secret",
					PubsubTopic:      "",
				}
				return NewGmailService(config)
			},
			testFunc: func(s *GmailService) {
				gmailAccountID := uuid.New()
				result, err := s.StartWatch(context.Background(), gmailAccountID)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "pubsub topic not configured")
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := tt.setupService()
			tt.testFunc(service)
		})
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newGmailServiceWithDB(t *testing.T, transport http.RoundTripper) (*GmailService, context.Context, *domain.GmailAccount) {
	t.Helper()

	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Agent{}, &domain.GmailAccount{}))

	agentRepository := agentRepo.NewAgentRepository(db)
	gmailRepository := gmailRepo.NewGmailAccountRepository(db)
	oauthClient := googleoauth.NewClient(googleoauth.Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURI:  "http://localhost/callback",
	})

	service := NewGmailService(GmailConfig{
		OAuthClient:      oauthClient,
		AgentRepo:        agentRepository,
		GmailAccountRepo: gmailRepository,
		JWTSecret:        "jwt-secret",
		PubsubTopic:      "projects/demo/topics/gmail",
	})

	ctx := context.Background()
	if transport != nil {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Transport: transport})
	}

	agent := &domain.Agent{
		Base:          domain.Base{ID: uuid.New()},
		UserID:        uuid.New(),
		Email:         "user@example.com",
		Name:          "Agent",
		EmailProvider: domain.AgentEmailProviderGmail,
		Status:        domain.AgentStatusActive,
	}
	require.NoError(t, db.Create(agent).Error)

	gmailAccount := &domain.GmailAccount{
		Base:    domain.Base{ID: uuid.New()},
		AgentID: agent.ID,
		Email:   "user@example.com",
		Name:    "User",
		Status:  domain.GmailAccountStatusActive,
	}

	tokenStore := &google_token_store.GoogleTokenStore{
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}
	require.NoError(t, gmailAccount.UpdateGoogleTokenStore(tokenStore))
	require.NoError(t, db.Create(gmailAccount).Error)

	return service, ctx, gmailAccount
}

func TestGmailService_GetAuthorizationURL(t *testing.T) {
	service, ctx, _ := newGmailServiceWithDB(t, nil)

	url, err := service.GetAuthorizationURL(ctx, "http://localhost/callback")
	assert.NoError(t, err)
	assert.NotEmpty(t, url)
	assert.Contains(t, url, "state=")
}

func TestGmailService_RefreshGmailAccountTokenInvalidGrant(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`)),
			Request:    req,
		}, nil
	})
	service, ctx, gmailAccount := newGmailServiceWithDB(t, transport)

	_, err := service.RefreshGmailAccountToken(ctx, gmailAccount.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_grant")

	updated, getErr := service.gmailAccountRepo.GetByID(ctx, gmailAccount.ID)
	require.NoError(t, getErr)
	assert.Equal(t, domain.GmailAccountStatusInvalidGrant, updated.Status)
}

func TestGmailService_HandleNotification(t *testing.T) {
	service, ctx, gmailAccount := newGmailServiceWithDB(t, nil)

	err := service.HandleNotification(ctx, &GmailNotificationPayload{
		EmailAddress: gmailAccount.Email,
		HistoryID:    "123",
	})
	assert.NoError(t, err)

	// Remove all accounts to trigger not found branch
	db := service.gmailAccountRepo.GetDB()
	require.NoError(t, db.Where("1 = 1").Delete(&domain.GmailAccount{}).Error)

	err = service.HandleNotification(ctx, &GmailNotificationPayload{
		EmailAddress: "missing@example.com",
		HistoryID:    "456",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no gmail accounts found")
}

func TestGmailService_CheckGmailAccountTokenStatus(t *testing.T) {
	service, ctx, gmailAccount := newGmailServiceWithDB(t, nil)

	status, err := service.CheckGmailAccountTokenStatus(ctx, gmailAccount.ID)
	assert.NoError(t, err)
	assert.Equal(t, gmailAccount.ID.String(), status["gmail_account_id"])
	assert.Equal(t, gmailAccount.Email, status["email"])
	assert.Equal(t, string(domain.GmailAccountStatusActive), status["status"])
	assert.Equal(t, true, status["has_refresh_token"])
	assert.Equal(t, true, status["is_valid"])
}

func TestGmailService_CreateGmailService(t *testing.T) {
	service, ctx, _ := newGmailServiceWithDB(t, nil)
	token := &oauth2.Token{AccessToken: "token"}

	gmailService, err := service.createGmailService(ctx, token)
	assert.NoError(t, err)
	assert.NotNil(t, gmailService)

	service.oauthClient = &googleoauth.Client{}
	gmailService, err = service.createGmailService(ctx, token)
	assert.Nil(t, gmailService)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "oauth config not available")
}

// Test GmailAccount domain structure
func TestGmailAccount_DomainStructure(t *testing.T) {
	gmailAccountID := uuid.New()
	agentID := uuid.New()

	gmailAccount := &domain.GmailAccount{
		Base:          domain.Base{ID: gmailAccountID},
		AgentID:       agentID,
		Email:         "test@gmail.com",
		Name:          "Test User",
		Picture:       "https://example.com/picture.jpg",
		Status:        domain.GmailAccountStatusActive,
		IsWatching:    true,
		LastHistoryID: "12345",
		WatchExpiry:   &[]time.Time{time.Now().Add(time.Hour)}[0],
	}

	assert.Equal(t, gmailAccountID, gmailAccount.ID)
	assert.Equal(t, agentID, gmailAccount.AgentID)
	assert.Equal(t, "test@gmail.com", gmailAccount.Email)
	assert.Equal(t, "Test User", gmailAccount.Name)
	assert.Equal(t, "https://example.com/picture.jpg", gmailAccount.Picture)
	assert.Equal(t, domain.GmailAccountStatusActive, gmailAccount.Status)
	assert.True(t, gmailAccount.IsWatching)
	assert.Equal(t, "12345", gmailAccount.LastHistoryID)
	assert.True(t, gmailAccount.WatchExpiry.After(time.Now()))
}

// Test helper methods for GmailAccount
func TestGmailAccount_HelperMethods(t *testing.T) {
	gmailAccount := &domain.GmailAccount{
		Base:   domain.Base{ID: uuid.New()},
		Email:  "test@gmail.com",
		Status: domain.GmailAccountStatusActive,
	}

	t.Run("SetWatching", func(t *testing.T) {
		historyID := "67890"
		expiry := time.Now().Unix()
		gmailAccount.SetWatching(historyID, expiry)

		assert.True(t, gmailAccount.IsWatching)
		assert.Equal(t, historyID, gmailAccount.LastHistoryID)
		assert.NotNil(t, gmailAccount.WatchExpiry)
	})

	t.Run("StopWatching", func(t *testing.T) {
		gmailAccount.IsWatching = true
		gmailAccount.LastHistoryID = "12345"
		expiryTime := time.Now().Add(time.Hour)
		gmailAccount.WatchExpiry = &expiryTime

		gmailAccount.StopWatching()

		assert.False(t, gmailAccount.IsWatching)
		assert.Equal(t, "12345", gmailAccount.LastHistoryID) // StopWatching doesn't clear LastHistoryID
		assert.Nil(t, gmailAccount.WatchExpiry)
	})

	t.Run("GetGoogleTokenStore and UpdateGoogleTokenStore", func(t *testing.T) {
		// This would test the actual domain methods, but since we can't mock them easily here,
		// we'll just verify they exist and can be called
		assert.NotNil(t, gmailAccount)

		// Test that UpdateGoogleTokenStore returns an error with nil input (as expected)
		err := gmailAccount.UpdateGoogleTokenStore(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token store cannot be nil")

		// Test GetGoogleTokenStore with empty credentials
		_, err = gmailAccount.GetGoogleTokenStore()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gmail account credentials is empty")
	})
}

// Test GmailAccountStatus constants
func TestGmailAccountStatus_Constants(t *testing.T) {
	// Test that status constants are defined correctly
	assert.Equal(t, domain.GmailAccountStatus("active"), domain.GmailAccountStatusActive)
	assert.Equal(t, domain.GmailAccountStatus("inactive"), domain.GmailAccountStatusInactive)
	assert.Equal(t, domain.GmailAccountStatus("invalid Google grant"), domain.GmailAccountStatusInvalidGrant)
}

// Test GmailAccountStatus constants - validation is done in the domain layer
func TestGmailAccountStatus_ConstantsValidation(t *testing.T) {
	validStatuses := []domain.GmailAccountStatus{
		domain.GmailAccountStatusActive,
		domain.GmailAccountStatusInactive,
		domain.GmailAccountStatusMissingScopes,
		domain.GmailAccountStatusSyncRequired,
		domain.GmailAccountStatusInvalidGrant,
	}

	for _, status := range validStatuses {
		// Just verify the constants are properly defined
		assert.NotEmpty(t, string(status), "Status should not be empty")
	}

	invalidStatus := domain.GmailAccountStatus("invalid")
	assert.NotEmpty(t, string(invalidStatus), "Invalid status should still have a string representation")
}

// Test Gmail service configuration validation
func TestGmailConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  GmailConfig
		isValid bool
	}{
		{
			name: "valid config",
			config: GmailConfig{
				OAuthClient:      nil,
				AgentRepo:        nil,
				GmailAccountRepo: nil,
				RedisClient:      nil,
				JWTSecret:        "valid-secret",
				WorkerClient:     nil,
				PubsubTopic:      "test-topic",
			},
			isValid: true,
		},
		{
			name: "empty JWT secret",
			config: GmailConfig{
				OAuthClient:      nil,
				AgentRepo:        nil,
				GmailAccountRepo: nil,
				RedisClient:      nil,
				JWTSecret:        "",
				WorkerClient:     nil,
				PubsubTopic:      "test-topic",
			},
			isValid: false, // Empty JWT secret would cause issues with state signing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.JWTSecret == "" {
				// This should cause issues with state signer creation
				assert.Panics(t, func() {
					newOAuthStateSigner(tt.config.JWTSecret)
				})
			} else {
				// Should not panic
				assert.NotPanics(t, func() {
					newOAuthStateSigner(tt.config.JWTSecret)
				})
			}
		})
	}
}

func newGmailServiceWithAgents(t *testing.T) (*GmailService, context.Context, *gmailRepo.GmailAccountRepository, *agentRepo.AgentRepository, uuid.UUID) {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Agent{}, &domain.GmailAccount{}))

	agentRepository := agentRepo.NewAgentRepository(db)
	gmailRepository := gmailRepo.NewGmailAccountRepository(db)
	service := NewGmailService(GmailConfig{
		GmailAccountRepo: gmailRepository,
		AgentRepo:        agentRepository,
		JWTSecret:        "secret",
	})
	return service, context.Background(), gmailRepository, agentRepository, uuid.New()
}

func TestGmailService_GetOrSyncGmailAccount(t *testing.T) {
	service, ctx, gmailRepository, agentRepository, userID := newGmailServiceWithAgents(t)

	agent := &domain.Agent{Base: domain.Base{ID: uuid.New()}, UserID: userID, Email: "a@example.com"}
	_, err := agentRepository.Create(ctx, agent)
	require.NoError(t, err)

	account := &domain.GmailAccount{
		Base:    domain.Base{ID: uuid.New()},
		AgentID: agent.ID,
		Email:   "a@example.com",
		Status:  domain.GmailAccountStatusActive,
	}
	_, err = gmailRepository.Create(ctx, account)
	require.NoError(t, err)

	found, err := service.GetOrSyncGmailAccount(ctx, userID, "a@example.com")
	require.NoError(t, err)
	assert.Equal(t, account.ID, found.ID)

	_, err = service.GetOrSyncGmailAccount(ctx, userID, "missing@example.com")
	assert.Error(t, err)

	require.NoError(t, gmailRepository.GetDB().Model(&domain.GmailAccount{}).Where("id = ?", account.ID).
		Update("status", domain.GmailAccountStatusInvalidGrant).Error)
	_, err = service.GetOrSyncGmailAccount(ctx, userID, "a@example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "needs reauthorization")
}

func TestGmailService_StoreCredentials(t *testing.T) {
	service, ctx, gmailRepository, agentRepository, userID := newGmailServiceWithAgents(t)
	agent := &domain.Agent{Base: domain.Base{ID: uuid.New()}, UserID: userID, Email: "a@example.com"}
	_, err := agentRepository.Create(ctx, agent)
	require.NoError(t, err)

	account := &domain.GmailAccount{
		Base:    domain.Base{ID: uuid.New()},
		AgentID: agent.ID,
		Email:   "a@example.com",
		Status:  domain.GmailAccountStatusActive,
	}
	_, err = gmailRepository.Create(ctx, account)
	require.NoError(t, err)

	token := &googleoauth.Token{
		AccessToken:  "acc",
		RefreshToken: "ref",
		Expiry:       time.Now().Add(time.Hour),
	}
	err = service.storeGmailAccountCredentials(ctx, account.ID, token)
	require.NoError(t, err)

	stored, err := gmailRepository.GetByID(ctx, account.ID)
	require.NoError(t, err)
	store, err := stored.GetGoogleTokenStore()
	require.NoError(t, err)
	assert.Equal(t, "acc", store.AccessToken)
	assert.Equal(t, "ref", store.RefreshToken)
}

func TestGmailService_CreateAccountAndWatchFlow(t *testing.T) {
	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(req.URL.Path, "/token"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"access_token":"acc","refresh_token":"ref","token_type":"Bearer","expires_in":3600}`)),
				Request:    req,
			}, nil
		case strings.Contains(req.URL.Path, "/userinfo"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"email":"watch@example.com","name":"Watcher"}`)),
				Request:    req,
			}, nil
		case strings.Contains(req.URL.Path, "/watch"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"historyId":"5","expiration":"99"}`)),
				Request:    req,
			}, nil
		case strings.Contains(req.URL.Path, "/stop"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Request:    req,
			}, nil
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
		}
	})

	// Build service with DB and stub HTTP client
	service, ctx, _ := newGmailServiceWithDB(t, transport)
	var agent domain.Agent
	require.NoError(t, service.agentRepo.GetDB().First(&agent).Error)

	// CreateGmailAccountFromAuth success path
	account, err := service.CreateGmailAccountFromAuth(ctx, agent.UserID, "auth-code", "state", "http://cb")
	require.NoError(t, err)
	assert.Equal(t, "watch@example.com", account.Email)

	// StartWatch / StopWatch flow
	service.pubsubTopic = "projects/demo/topics/gmail"
	watchResp, err := service.StartWatch(ctx, account.ID)
	require.NoError(t, err)
	assert.Equal(t, "5", watchResp.HistoryID)
	assert.Equal(t, int64(99), watchResp.Expiration)

	require.NoError(t, service.StopWatch(ctx, account.ID))
}

// Test notification handling structure
func TestGmailNotificationPayload_Structure(t *testing.T) {
	payload := &GmailNotificationPayload{
		EmailAddress: "test@gmail.com",
		HistoryID:    "12345",
	}

	assert.Equal(t, "test@gmail.com", payload.EmailAddress)
	assert.Equal(t, "12345", payload.HistoryID)

	// Test JSON marshaling
	jsonBytes, err := json.Marshal(payload)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonBytes), "test@gmail.com")
	assert.Contains(t, string(jsonBytes), "12345")

	// Test JSON unmarshaling
	var unmarshaled GmailNotificationPayload
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, payload.EmailAddress, unmarshaled.EmailAddress)
	assert.Equal(t, payload.HistoryID, unmarshaled.HistoryID)
}
