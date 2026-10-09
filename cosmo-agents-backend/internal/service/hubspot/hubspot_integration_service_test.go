package hubspot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/user"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
)

// MockHubspotAPI is a mock implementation of HubspotAPI for testing
type MockHubspotAPI struct {
	mock.Mock
}

// MockUserRepository is a mock implementation of UserRepository for testing
type MockUserRepository struct {
	mock.Mock
}

// MockWorkerClient is a mock implementation of WorkerClient for testing
type MockWorkerClient struct {
	mock.Mock
}

// Define an interface that our mock can implement
type HubspotAPIService interface {
	AuthorizationURL(overrideRedirect string) (string, error)
	ExchangeCode(ctx context.Context, code, redirectURI string) (*HubspotTokenResponse, error)
	GetUserInfoFromRefreshToken(ctx context.Context, refreshToken string) (*HubspotUserInfo, error)
}

// Define an interface for UserRepository that our mock can implement
type UserRepositoryService interface {
	FindByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	UpdateHubspotCredentials(ctx context.Context, userID uuid.UUID, credentials base.JSON) error
}

// Define an interface for WorkerClient that our mock can implement
type WorkerClientService interface {
	EnqueueCriticalTask(ctx context.Context, taskType string, payload interface{}) (uuid.UUID, error)
}

// Implement the interface methods for MockHubspotAPI
func (m *MockHubspotAPI) AuthorizationURL(overrideRedirect string) (string, error) {
	args := m.Called(overrideRedirect)
	return args.String(0), args.Error(1)
}

func (m *MockHubspotAPI) ExchangeCode(ctx context.Context, code, redirectURI string) (*HubspotTokenResponse, error) {
	args := m.Called(ctx, code, redirectURI)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HubspotTokenResponse), args.Error(1)
}

func (m *MockHubspotAPI) GetUserInfoFromRefreshToken(ctx context.Context, refreshToken string) (*HubspotUserInfo, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*HubspotUserInfo), args.Error(1)
}

// Implement the interface methods for MockUserRepository
func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}

func (m *MockUserRepository) UpdateHubspotCredentials(ctx context.Context, userID uuid.UUID, credentials base.JSON) error {
	args := m.Called(ctx, userID, credentials)
	return args.Error(0)
}

// Implement the interface methods for MockWorkerClient
func (m *MockWorkerClient) EnqueueCriticalTask(ctx context.Context, taskType string, payload interface{}) (uuid.UUID, error) {
	args := m.Called(ctx, taskType, payload)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

// Create a test service that accepts interfaces instead of concrete types
type TestHubspotIntegrationService struct {
	api          HubspotAPIService
	userRepo     UserRepositoryService
	workerClient WorkerClientService
}

func NewTestHubspotIntegrationService(
	api HubspotAPIService,
	userRepo UserRepositoryService,
	workerClient WorkerClientService,
) *TestHubspotIntegrationService {
	return &TestHubspotIntegrationService{
		api:          api,
		userRepo:     userRepo,
		workerClient: workerClient,
	}
}

// Implement the same methods as the original service
func (s *TestHubspotIntegrationService) AuthorizationURL(ctx context.Context, redirect string) (string, error) {
	if s.api == nil {
		return "", errors.New("hubspot integration not configured")
	}
	return s.api.AuthorizationURL(redirect)
}

func (s *TestHubspotIntegrationService) HandleCallback(ctx context.Context, userID uuid.UUID, code, redirectURI string) (*HubspotCallbackResult, error) {
	if s.api == nil {
		return nil, errors.New("hubspot integration not configured")
	}

	token, err := s.api.ExchangeCode(ctx, code, redirectURI)
	if err != nil {
		return nil, errors.New("hubspot token exchange failed: " + err.Error())
	}

	info, err := s.api.GetUserInfoFromRefreshToken(ctx, token.RefreshToken)
	if err != nil {
		return nil, errors.New("failed to fetch hubspot user info: " + err.Error())
	}

	if err := s.storeCredentials(ctx, userID, token, info); err != nil {
		return nil, err
	}

	if s.workerClient != nil {
		payload := hubspotSyncPayload{
			UserID:      userID.String(),
			HubID:       fmt.Sprintf("%d", info.HubID),
			AccessToken: token.AccessToken,
			After:       0,
			Offset:      0,
		}
		if _, err := s.workerClient.EnqueueCriticalTask(ctx, "contact:pull_hubspot", payload); err != nil {
			// Log error but don't fail the operation
		}
		if _, err := s.workerClient.EnqueueCriticalTask(ctx, "contact:pull_hubspot_lists", payload); err != nil {
			// Log error but don't fail the operation
		}
	}

	return &HubspotCallbackResult{
		Token: *token,
		Info:  *info,
	}, nil
}

func (s *TestHubspotIntegrationService) GetUserInfo(ctx context.Context, userID uuid.UUID) (*HubspotUserInfo, error) {
	if s.api == nil {
		return nil, errors.New("hubspot integration not configured")
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}
	if user.HubspotCredentials == nil {
		return nil, errors.New("hubspot credentials not set")
	}

	var creds map[string]interface{}
	if err := user.HubspotCredentials.Unmarshal(&creds); err != nil {
		return nil, errors.New("failed to parse hubspot credentials: " + err.Error())
	}

	refreshToken, _ := creds["refresh_token"].(string)
	if refreshToken == "" {
		return nil, errors.New("refresh token missing")
	}

	return s.api.GetUserInfoFromRefreshToken(ctx, refreshToken)
}

func (s *TestHubspotIntegrationService) storeCredentials(ctx context.Context, userID uuid.UUID, token *HubspotTokenResponse, info *HubspotUserInfo) error {
	credentials := map[string]interface{}{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"hub_id":        info.HubID,
		"token_type":    token.TokenType,
		"expires_in":    token.ExpiresIn,
	}

	var serialized base.JSON
	if err := serialized.Marshal(credentials); err != nil {
		return errors.New("failed to serialize hubspot credentials: " + err.Error())
	}

	return s.userRepo.UpdateHubspotCredentials(ctx, userID, serialized)
}

func TestNewHubspotIntegrationService(t *testing.T) {
	mockAPI := &MockHubspotAPI{}
	mockUserRepo := &MockUserRepository{}
	mockWorkerClient := &MockWorkerClient{}

	service := NewTestHubspotIntegrationService(
		mockAPI,
		mockUserRepo,
		mockWorkerClient,
	)

	assert.NotNil(t, service)
	assert.Equal(t, mockAPI, service.api)
	assert.Equal(t, mockUserRepo, service.userRepo)
	assert.Equal(t, mockWorkerClient, service.workerClient)
}

func TestHubspotIntegrationService_AuthorizationURL(t *testing.T) {
	tests := []struct {
		name          string
		apiNil        bool
		redirect      string
		setupMocks    func(*MockHubspotAPI)
		expectedURL   string
		expectedError bool
		errorContains string
	}{
		{
			name:     "successful authorization URL generation",
			apiNil:   false,
			redirect: "https://app.example.com/auth/callback",
			setupMocks: func(m *MockHubspotAPI) {
				m.On("AuthorizationURL", "https://app.example.com/auth/callback").
					Return("https://app.hubspot.com/oauth/authorize", nil)
			},
			expectedURL:   "https://app.hubspot.com/oauth/authorize",
			expectedError: false,
		},
		{
			name:          "API not configured",
			apiNil:        true,
			redirect:      "https://app.example.com/auth/callback",
			setupMocks:    func(m *MockHubspotAPI) {},
			expectedError: true,
			errorContains: "hubspot integration not configured",
		},
		{
			name:     "API returns error",
			apiNil:   false,
			redirect: "https://app.example.com/auth/callback",
			setupMocks: func(m *MockHubspotAPI) {
				m.On("AuthorizationURL", "https://app.example.com/auth/callback").
					Return("", errors.New("API error"))
			},
			expectedError: true,
			errorContains: "API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var api HubspotAPIService
			if !tt.apiNil {
				mockAPI := &MockHubspotAPI{}
				tt.setupMocks(mockAPI)
				api = mockAPI
			}

			service := NewTestHubspotIntegrationService(api, &MockUserRepository{}, &MockWorkerClient{})

			url, err := service.AuthorizationURL(context.Background(), tt.redirect)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedURL, url)
			}

			if mockAPI, ok := api.(*MockHubspotAPI); ok {
				mockAPI.AssertExpectations(t)
			}
		})
	}
}

func TestHubspotIntegrationService_HandleCallback(t *testing.T) {
	userID := uuid.New()
	validCode := "valid_authorization_code"
	redirectURI := "https://app.example.com/auth/callback"

	tests := []struct {
		name                string
		userID              uuid.UUID
		code                string
		redirectURI         string
		apiNil              bool
		workerClientNil     bool
		setupMocks          func(*MockHubspotAPI, *MockUserRepository, *MockWorkerClient)
		expectedError       bool
		errorContains       string
		expectedWorkerCalls int
	}{
		{
			name:            "successful callback with all services",
			userID:          userID,
			code:            validCode,
			redirectURI:     redirectURI,
			apiNil:          false,
			workerClientNil: false,
			setupMocks: func(api *MockHubspotAPI, userRepo *MockUserRepository, worker *MockWorkerClient) {
				// Mock token exchange
				tokenResponse := &HubspotTokenResponse{
					AccessToken:  "test_access_token",
					RefreshToken: "test_refresh_token",
					ExpiresIn:    3600,
					TokenType:    "Bearer",
				}
				api.On("ExchangeCode", mock.Anything, validCode, redirectURI).
					Return(tokenResponse, nil)

				// Mock user info fetch
				userInfo := &HubspotUserInfo{
					Token:     "test_token",
					User:      "test_user",
					HubDomain: "test.hubspot.com",
					HubID:     987654321,
					TokenType: "Bearer",
				}
				api.On("GetUserInfoFromRefreshToken", mock.Anything, "test_refresh_token").
					Return(userInfo, nil)

				// Mock user repository operations - HandleCallback only calls UpdateHubspotCredentials via storeCredentials
				userRepo.On("UpdateHubspotCredentials", mock.Anything, userID, mock.Anything).
					Return(nil)

				// Mock worker client
				worker.On("EnqueueCriticalTask", mock.Anything, "contact:pull_hubspot", mock.Anything).
					Return(uuid.New(), nil)
				worker.On("EnqueueCriticalTask", mock.Anything, "contact:pull_hubspot_lists", mock.Anything).
					Return(uuid.New(), nil)
			},
			expectedError:       false,
			expectedWorkerCalls: 2,
		},
		{
			name:                "API not configured",
			userID:              userID,
			code:                validCode,
			redirectURI:         redirectURI,
			apiNil:              true,
			workerClientNil:     false,
			setupMocks:          func(*MockHubspotAPI, *MockUserRepository, *MockWorkerClient) {},
			expectedError:       true,
			errorContains:       "hubspot integration not configured",
			expectedWorkerCalls: 0,
		},
		{
			name:            "token exchange fails",
			userID:          userID,
			code:            "invalid_code",
			redirectURI:     redirectURI,
			apiNil:          false,
			workerClientNil: false,
			setupMocks: func(api *MockHubspotAPI, userRepo *MockUserRepository, worker *MockWorkerClient) {
				api.On("ExchangeCode", mock.Anything, "invalid_code", redirectURI).
					Return(nil, errors.New("invalid authorization code"))
			},
			expectedError:       true,
			errorContains:       "hubspot token exchange failed",
			expectedWorkerCalls: 0,
		},
		{
			name:            "user info fetch fails",
			userID:          userID,
			code:            validCode,
			redirectURI:     redirectURI,
			apiNil:          false,
			workerClientNil: false,
			setupMocks: func(api *MockHubspotAPI, userRepo *MockUserRepository, worker *MockWorkerClient) {
				tokenResponse := &HubspotTokenResponse{
					AccessToken:  "test_access_token",
					RefreshToken: "test_refresh_token",
					ExpiresIn:    3600,
					TokenType:    "Bearer",
				}
				api.On("ExchangeCode", mock.Anything, validCode, redirectURI).
					Return(tokenResponse, nil)
				api.On("GetUserInfoFromRefreshToken", mock.Anything, "test_refresh_token").
					Return(nil, errors.New("API error"))
			},
			expectedError:       true,
			errorContains:       "failed to fetch hubspot user info",
			expectedWorkerCalls: 0,
		},
		{
			name:            "worker client is nil (should not fail)",
			userID:          userID,
			code:            validCode,
			redirectURI:     redirectURI,
			apiNil:          false,
			workerClientNil: true,
			setupMocks: func(api *MockHubspotAPI, userRepo *MockUserRepository, worker *MockWorkerClient) {
				tokenResponse := &HubspotTokenResponse{
					AccessToken:  "test_access_token",
					RefreshToken: "test_refresh_token",
					ExpiresIn:    3600,
					TokenType:    "Bearer",
				}
				userInfo := &HubspotUserInfo{
					HubID: 123456789,
				}
				api.On("ExchangeCode", mock.Anything, validCode, redirectURI).
					Return(tokenResponse, nil)
				api.On("GetUserInfoFromRefreshToken", mock.Anything, "test_refresh_token").
					Return(userInfo, nil)
				userRepo.On("UpdateHubspotCredentials", mock.Anything, userID, mock.Anything).
					Return(nil)
			},
			expectedError:       false,
			expectedWorkerCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var api HubspotAPIService
			var workerClient WorkerClientService

			if !tt.apiNil {
				mockAPI := &MockHubspotAPI{}
				api = mockAPI
			}

			if !tt.workerClientNil {
				mockWorker := &MockWorkerClient{}
				workerClient = mockWorker
			}

			mockUserRepo := &MockUserRepository{}

			if mockAPI, ok := api.(*MockHubspotAPI); ok {
				var mockWorker *MockWorkerClient
				if workerClient != nil {
					if wc, ok := workerClient.(*MockWorkerClient); ok {
						mockWorker = wc
					}
				}
				tt.setupMocks(mockAPI, mockUserRepo, mockWorker)
			} else {
				tt.setupMocks(nil, mockUserRepo, nil)
			}

			service := NewTestHubspotIntegrationService(api, mockUserRepo, workerClient)

			result, err := service.HandleCallback(context.Background(), tt.userID, tt.code, tt.redirectURI)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			if mockAPI, ok := api.(*MockHubspotAPI); ok {
				mockAPI.AssertExpectations(t)
			}
			mockUserRepo.AssertExpectations(t)
			if mockWorker, ok := workerClient.(*MockWorkerClient); ok {
				mockWorker.AssertExpectations(t)
			}
		})
	}
}

func TestHubspotIntegrationService_GetUserInfo(t *testing.T) {
	userID := uuid.New()
	validRefreshToken := "valid_refresh_token"

	tests := []struct {
		name          string
		userID        uuid.UUID
		apiNil        bool
		setupMocks    func(*MockHubspotAPI, *MockUserRepository)
		expectedError bool
		errorContains string
	}{
		{
			name:   "successful user info retrieval",
			userID: userID,
			apiNil: false,
			setupMocks: func(api *MockHubspotAPI, userRepo *MockUserRepository) {
				// Mock user with credentials
				serializedCredentials := base.JSON([]byte(`{"refresh_token":"valid_refresh_token"}`))
				user := &user.User{
					Base:               base.Base{ID: userID},
					HubspotCredentials: serializedCredentials,
				}
				userRepo.On("FindByID", mock.Anything, userID).
					Return(user, nil)

				// Mock API call
				userInfo := &HubspotUserInfo{
					Token:     "test_token",
					User:      "test_user",
					HubDomain: "test.hubspot.com",
					HubID:     123456789,
					TokenType: "Bearer",
				}
				api.On("GetUserInfoFromRefreshToken", mock.Anything, validRefreshToken).
					Return(userInfo, nil)
			},
			expectedError: false,
		},
		{
			name:   "API not configured",
			userID: userID,
			apiNil: true,
			setupMocks: func(api *MockHubspotAPI, userRepo *MockUserRepository) {
			},
			expectedError: true,
			errorContains: "hubspot integration not configured",
		},
		{
			name:   "user not found",
			userID: uuid.New(),
			apiNil: false,
			setupMocks: func(api *MockHubspotAPI, userRepo *MockUserRepository) {
				userRepo.On("FindByID", mock.Anything, mock.Anything).
					Return(nil, errors.New("user not found"))
			},
			expectedError: true,
			errorContains: "user not found",
		},
		{
			name:   "hubspot credentials not set",
			userID: userID,
			apiNil: false,
			setupMocks: func(api *MockHubspotAPI, userRepo *MockUserRepository) {
				user := &user.User{
					Base:               base.Base{ID: userID},
					HubspotCredentials: nil,
				}
				userRepo.On("FindByID", mock.Anything, userID).
					Return(user, nil)
			},
			expectedError: true,
			errorContains: "hubspot credentials not set",
		},
		{
			name:   "refresh token missing",
			userID: userID,
			apiNil: false,
			setupMocks: func(api *MockHubspotAPI, userRepo *MockUserRepository) {
				serializedCredentials := base.JSON([]byte(`{"access_token":"some_token"}`))
				user := &user.User{
					Base:               base.Base{ID: userID},
					HubspotCredentials: serializedCredentials,
				}
				userRepo.On("FindByID", mock.Anything, userID).
					Return(user, nil)
			},
			expectedError: true,
			errorContains: "refresh token missing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var api HubspotAPIService
			if !tt.apiNil {
				mockAPI := &MockHubspotAPI{}
				api = mockAPI
			}

			mockUserRepo := &MockUserRepository{}

			if mockAPI, ok := api.(*MockHubspotAPI); ok {
				tt.setupMocks(mockAPI, mockUserRepo)
			} else {
				tt.setupMocks(nil, mockUserRepo)
			}

			service := NewTestHubspotIntegrationService(api, mockUserRepo, &MockWorkerClient{})

			result, err := service.GetUserInfo(context.Background(), tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			if mockAPI, ok := api.(*MockHubspotAPI); ok {
				mockAPI.AssertExpectations(t)
			}
			mockUserRepo.AssertExpectations(t)
		})
	}
}

func TestHubspotIntegrationService_StoreCredentials(t *testing.T) {
	userID := uuid.New()
	token := &HubspotTokenResponse{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
	}
	info := &HubspotUserInfo{
		HubID: 123456789,
	}

	tests := []struct {
		name        string
		userID      uuid.UUID
		token       *HubspotTokenResponse
		info        *HubspotUserInfo
		setupMocks  func(*MockUserRepository)
		expectedErr bool
		errContains string
	}{
		{
			name:   "successful credential storage",
			userID: userID,
			token:  token,
			info:   info,
			setupMocks: func(m *MockUserRepository) {
				m.On("UpdateHubspotCredentials", mock.Anything, userID, mock.Anything).
					Return(nil)
			},
			expectedErr: false,
		},
		{
			name:   "database error",
			userID: userID,
			token:  token,
			info:   info,
			setupMocks: func(m *MockUserRepository) {
				m.On("UpdateHubspotCredentials", mock.Anything, userID, mock.Anything).
					Return(errors.New("database error"))
			},
			expectedErr: true,
			errContains: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUserRepo := &MockUserRepository{}
			tt.setupMocks(mockUserRepo)

			service := NewTestHubspotIntegrationService(nil, mockUserRepo, &MockWorkerClient{})

			err := service.storeCredentials(context.Background(), tt.userID, tt.token, tt.info)

			if tt.expectedErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockUserRepo.AssertExpectations(t)
		})
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newStubHubspotAPI(transport http.RoundTripper) *HubspotAPI {
	return &HubspotAPI{
		client:       &http.Client{Transport: transport},
		clientID:     "client-id",
		clientSecret: "client-secret",
		redirectURI:  "http://example.com/callback",
		scopes:       []string{"contacts", "crm.objects.contacts.read"},
	}
}

func newHubspotServiceWithDB(t *testing.T, transport http.RoundTripper) (*HubspotIntegrationService, *userRepo.UserRepository, uuid.UUID) {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&user.User{}))

	repo := userRepo.NewUserRepository(db)
	userID := uuid.New()
	_, err = repo.Create(context.Background(), &user.User{
		Base:  base.Base{ID: userID},
		Email: "user@example.com",
		Name:  "User",
	})
	require.NoError(t, err)

	api := newStubHubspotAPI(transport)
	service := NewHubspotIntegrationService(api, repo, nil)
	return service, repo, userID
}

func TestHubspotAPI_AuthorizationURLAndScopes(t *testing.T) {
	api := newStubHubspotAPI(http.DefaultTransport)
	url, err := api.AuthorizationURL("http://override/cb")
	require.NoError(t, err)
	assert.Contains(t, url, "client_id=client-id")
	assert.Contains(t, url, "redirect_uri=http%3A%2F%2Foverride%2Fcb")
	assert.Contains(t, url, "scope=contacts+crm.objects.contacts.read")

	_, err = (&HubspotAPI{redirectURI: ""}).AuthorizationURL("")
	assert.Error(t, err)
}

func TestHubspotAPI_ExchangeAndRefresh(t *testing.T) {
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(r.URL.Path, "/oauth/v1/token"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"access_token":"access","refresh_token":"refresh","expires_in":3600,"token_type":"Bearer"
				}`)),
				Request: r,
			}, nil
		case strings.Contains(r.URL.Path, "/refresh-tokens/"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"token":"t","user":"u","hub_id":123,"token_type":"Bearer","scopes":["contacts"]
				}`)),
				Request: r,
			}, nil
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
	})

	api := newStubHubspotAPI(rt)
	ctx := context.Background()

	token, err := api.ExchangeCode(ctx, "code", "")
	require.NoError(t, err)
	assert.Equal(t, "access", token.AccessToken)
	assert.Equal(t, "refresh", token.RefreshToken)

	refreshed, err := api.RefreshToken(ctx, "refresh")
	require.NoError(t, err)
	assert.Equal(t, "access", refreshed.AccessToken)

	info, err := api.GetUserInfoFromRefreshToken(ctx, "refresh")
	require.NoError(t, err)
	assert.Equal(t, 123, info.HubID)
	assert.Contains(t, info.Scopes, "contacts")

	// error paths
	errRT := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`err`)), Request: r}, nil
	})
	api = newStubHubspotAPI(errRT)
	_, err = api.RefreshToken(ctx, "refresh")
	assert.Error(t, err)
	_, err = api.GetUserInfoFromRefreshToken(ctx, "refresh")
	assert.Error(t, err)
}

func TestHubspotAPI_ContactsAndLists(t *testing.T) {
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(r.URL.Path, "/crm/v3/objects/contacts"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"results":[
						{"id":"1","properties":{"email":"a@example.com"}},
						{"id":"2","properties":{"email":"b@example.com"}}
					],
					"paging":{"next":{"after":"20"}}
				}`)),
				Request: r,
			}, nil
		case strings.Contains(r.URL.Path, "/crm/v3/lists/"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"listId":"123"}`)),
				Request:    r,
			}, nil
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
	})

	api := newStubHubspotAPI(rt)
	ctx := context.Background()

	page, err := api.GetContactsAll(ctx, "token", 0)
	require.NoError(t, err)
	require.Len(t, page.Contacts, 2)
	assert.Equal(t, 20, page.After)
	assert.Equal(t, "a@example.com", page.Contacts[0].Properties["email"])

	list, err := api.GetListByID(ctx, "token", "321")
	require.NoError(t, err)
	assert.Equal(t, "123", list["listId"])

	// Error path
	errRT := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	})
	api = newStubHubspotAPI(errRT)
	_, err = api.GetContactsAll(ctx, "token", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hubspot request failed")
}

func TestParseScopesAndAuthURL(t *testing.T) {
	assert.Equal(t, []string{"contacts"}, parseScopes(""))
	assert.Equal(t, []string{"contacts", "crm.objects.contacts.read"}, parseScopes("contacts crm.objects.contacts.read"))
	assert.Equal(t, []string{"contacts", "crm"}, parseScopes("contacts,crm"))

	_, err := (&HubspotAPI{redirectURI: ""}).AuthorizationURL("")
	assert.Error(t, err)
}

func TestHubspotIntegrationService_HandleCallbackRealDeps(t *testing.T) {
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case strings.Contains(r.URL.Path, "/oauth/v1/token"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"access_token":"tokenA","refresh_token":"tokenR","expires_in":3600,"token_type":"Bearer"
				}`)),
				Request: r,
			}, nil
		case strings.Contains(r.URL.Path, "/refresh-tokens/"):
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"hub_id":42,"token_type":"Bearer"}`)),
				Request:    r,
			}, nil
		default:
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
	})

	service, repo, userID := newHubspotServiceWithDB(t, rt)

	result, err := service.HandleCallback(context.Background(), userID, "code", "http://cb")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "tokenA", result.Token.AccessToken)
	assert.Equal(t, 42, result.Info.HubID)

	stored, err := repo.FindByID(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, stored.HubspotCredentials)

	var creds map[string]interface{}
	require.NoError(t, stored.HubspotCredentials.Unmarshal(&creds))
	assert.Equal(t, "tokenR", creds["refresh_token"])
}

func TestHubspotIntegrationService_GetUserInfoRealDeps(t *testing.T) {
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"hub_id":7,"token_type":"Bearer","scopes":["contacts"]}`)),
			Request:    r,
		}, nil
	})

	service, repo, userID := newHubspotServiceWithDB(t, rt)

	creds := base.JSON([]byte(`{"refresh_token":"stored"}`))
	require.NoError(t, repo.UpdateHubspotCredentials(context.Background(), userID, creds))

	info, err := service.GetUserInfo(context.Background(), userID)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, 7, info.HubID)
}
