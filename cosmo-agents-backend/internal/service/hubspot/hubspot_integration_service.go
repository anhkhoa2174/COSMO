package hubspot

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// HubspotIntegrationService orchestrates HubSpot OAuth and synchronization.
type HubspotIntegrationService struct {
	api          *HubspotAPI
	userRepo     *user.UserRepository
	workerClient *worker.Client
}

// NewHubspotIntegrationService creates a new integration service.
func NewHubspotIntegrationService(
	api *HubspotAPI,
	userRepo *user.UserRepository,
	workerClient *worker.Client,
) *HubspotIntegrationService {
	return &HubspotIntegrationService{
		api:          api,
		userRepo:     userRepo,
		workerClient: workerClient,
	}
}

// AuthorizationURL generates the authorization URL.
func (s *HubspotIntegrationService) AuthorizationURL(ctx context.Context, redirect string) (string, error) {
	if s.api == nil {
		return "", fmt.Errorf("hubspot integration not configured")
	}
	return s.api.AuthorizationURL(redirect)
}

// HubspotCallbackResult captures callback response details.
type HubspotCallbackResult struct {
	Token HubspotTokenResponse
	Info  HubspotUserInfo
}

// HandleCallback exchanges code, stores credentials, and triggers sync.
func (s *HubspotIntegrationService) HandleCallback(ctx context.Context, userID uuid.UUID, code, redirectURI string) (*HubspotCallbackResult, error) {
	if s.api == nil {
		return nil, fmt.Errorf("hubspot integration not configured")
	}

	token, err := s.api.ExchangeCode(ctx, code, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("hubspot token exchange failed: %w", err)
	}

	info, err := s.api.GetUserInfoFromRefreshToken(ctx, token.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch hubspot user info: %w", err)
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
		if _, err := s.workerClient.EnqueueCriticalTask(ctx, worker.TypePullHubspotContacts, payload); err != nil {
			logger.FromContext(ctx).Error().Err(err).Msg("failed to enqueue hubspot contacts pull")
		}
		if _, err := s.workerClient.EnqueueCriticalTask(ctx, worker.TypePullHubspotListContacts, payload); err != nil {
			logger.FromContext(ctx).Error().Err(err).Msg("failed to enqueue hubspot list pull")
		}
	}

	return &HubspotCallbackResult{
		Token: *token,
		Info:  *info,
	}, nil
}

// GetUserInfo retrieves HubSpot user info based on stored credentials.
func (s *HubspotIntegrationService) GetUserInfo(ctx context.Context, userID uuid.UUID) (*HubspotUserInfo, error) {
	if s.api == nil {
		return nil, fmt.Errorf("hubspot integration not configured")
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if user.HubspotCredentials == nil {
		return nil, fmt.Errorf("hubspot credentials not set")
	}

	var creds map[string]interface{}
	if err := user.HubspotCredentials.Unmarshal(&creds); err != nil {
		return nil, fmt.Errorf("failed to parse hubspot credentials: %w", err)
	}

	refreshToken, _ := creds["refresh_token"].(string)
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token missing")
	}

	return s.api.GetUserInfoFromRefreshToken(ctx, refreshToken)
}

func (s *HubspotIntegrationService) storeCredentials(ctx context.Context, userID uuid.UUID, token *HubspotTokenResponse, info *HubspotUserInfo) error {
	credentials := map[string]interface{}{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"hub_id":        info.HubID,
		"token_type":    token.TokenType,
		"expires_in":    token.ExpiresIn,
	}

	var serialized domain.JSON
	if err := serialized.Marshal(credentials); err != nil {
		return fmt.Errorf("failed to serialize hubspot credentials: %w", err)
	}

	return s.userRepo.UpdateHubspotCredentials(ctx, userID, serialized)
}

type hubspotSyncPayload struct {
	UserID      string `json:"user_id"`
	HubID       string `json:"hub_id"`
	AccessToken string `json:"access_token"`
	After       int    `json:"after"`
	Offset      int    `json:"offset"`
}
