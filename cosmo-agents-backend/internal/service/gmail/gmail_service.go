package gmail

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/google_token_store"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	gmailRepo "github.com/rockship/cosmo-agents-go/internal/repository/gmail"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
)

// GmailNotificationPayload represents the payload pushed by Gmail notifications
type GmailNotificationPayload struct {
	EmailAddress string `json:"email_address"`
	HistoryID    string `json:"history_id"`
}

// UnmarshalJSON supports both string and numeric history IDs when decoding payloads
func (p *GmailNotificationPayload) UnmarshalJSON(data []byte) error {
	type alias struct {
		EmailAddress string          `json:"email_address"`
		HistoryID    json.RawMessage `json:"history_id"`
	}

	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	p.EmailAddress = raw.EmailAddress
	if len(raw.HistoryID) == 0 || string(raw.HistoryID) == "null" {
		p.HistoryID = ""
		return nil
	}

	// Handle string history IDs
	if raw.HistoryID[0] == '"' {
		if err := json.Unmarshal(raw.HistoryID, &p.HistoryID); err != nil {
			return err
		}
		return nil
	}

	// Handle numeric history IDs without converting to float64 to avoid precision loss
	var number json.Number
	if err := json.Unmarshal(raw.HistoryID, &number); err != nil {
		return fmt.Errorf("unsupported history_id type: %w", err)
	}

	p.HistoryID = number.String()
	return nil
}

// GmailService handles Gmail operations
type GmailService struct {
	oauthClient      *googleoauth.Client
	agentRepo        *agentRepo.AgentRepository
	gmailAccountRepo *gmailRepo.GmailAccountRepository
	redisClient      interface{} // Add proper Redis client type if needed
	stateSigner      *oauthStateSigner
	jwtSecret        string
	workerClient     interface{} // Add proper worker client type if needed
	pubsubTopic      string
}

// GmailConfig holds Gmail service configuration
type GmailConfig struct {
	OAuthClient      *googleoauth.Client
	AgentRepo        *agentRepo.AgentRepository
	GmailAccountRepo *gmailRepo.GmailAccountRepository
	RedisClient      interface{}
	JWTSecret        string
	WorkerClient     interface{}
	PubsubTopic      string
}

// NewGmailService creates a new Gmail service
func NewGmailService(config GmailConfig) *GmailService {
	return &GmailService{
		oauthClient:      config.OAuthClient,
		agentRepo:        config.AgentRepo,
		gmailAccountRepo: config.GmailAccountRepo,
		redisClient:      config.RedisClient,
		stateSigner:      newOAuthStateSigner(config.JWTSecret),
		jwtSecret:        config.JWTSecret,
		workerClient:     config.WorkerClient,
		pubsubTopic:      config.PubsubTopic,
	}
}

// oauthStateSigner handles OAuth state token signing and verification
type oauthStateSigner struct {
	secretKey []byte
}

// newOAuthStateSigner creates a new OAuth state signer
func newOAuthStateSigner(secretKey string) *oauthStateSigner {
	if secretKey == "" {
		panic("OAuthStateSigner: secret key cannot be empty")
	}
	return &oauthStateSigner{secretKey: []byte(secretKey)}
}

// statePayload represents the data stored in the state token
type statePayload struct {
	UserID    uuid.UUID `json:"user_id"`
	Timestamp int64     `json:"timestamp"`
	Nonce     string    `json:"nonce"`
}

// SignState creates a signed state token containing user ID
func (s *oauthStateSigner) SignState(userID uuid.UUID) (string, error) {
	payload := statePayload{
		UserID:    userID,
		Timestamp: time.Now().Unix(),
		Nonce:     uuid.New().String()[:8], // Short nonce for uniqueness
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal state payload: %w", err)
	}

	// Create signature
	hash := hmac.New(sha256.New, s.secretKey)
	hash.Write(payloadBytes)
	signature := hash.Sum(nil)

	// Base64 encode payload and signature
	payloadB64 := base64.URLEncoding.EncodeToString(payloadBytes)
	signatureB64 := base64.URLEncoding.EncodeToString(signature)

	return fmt.Sprintf("%s.%s", payloadB64, signatureB64), nil
}

// VerifyState verifies a signed state token and returns the user ID
func (s *oauthStateSigner) VerifyState(state string) (uuid.UUID, error) {
	parts := strings.Split(state, ".")
	if len(parts) != 2 {
		return uuid.Nil, fmt.Errorf("invalid state format")
	}

	payloadB64, signatureB64 := parts[0], parts[1]

	payloadBytes, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to decode state payload: %w", err)
	}

	signature, err := base64.URLEncoding.DecodeString(signatureB64)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to decode state signature: %w", err)
	}

	// Verify signature
	hash := hmac.New(sha256.New, s.secretKey)
	hash.Write(payloadBytes)
	expectedSignature := hash.Sum(nil)

	if !hmac.Equal(signature, expectedSignature) {
		return uuid.Nil, fmt.Errorf("invalid state signature")
	}

	var payload statePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return uuid.Nil, fmt.Errorf("failed to unmarshal state payload: %w", err)
	}

	// Check timestamp (state expires after 10 minutes)
	if time.Now().Unix()-payload.Timestamp > 600 {
		return uuid.Nil, fmt.Errorf("state token expired")
	}

	return payload.UserID, nil
}

// GetAuthorizationURL generates Gmail OAuth2 authorization URL
func (s *GmailService) GetAuthorizationURL(ctx context.Context, redirectURI string) (string, error) {
	logger.FromContext(ctx).Info().
		Str("redirect_uri", redirectURI).
		Msg("Generating Gmail OAuth authorization URL")

	if s.oauthClient == nil {
		return "", fmt.Errorf("oauth client not configured")
	}

	state, err := s.stateSigner.SignState(uuid.Nil) // State for general auth (no specific user)
	if err != nil {
		return "", fmt.Errorf("failed to sign state: %w", err)
	}

	authURL := s.oauthClient.GetAuthURL(state)

	logger.FromContext(ctx).Info().
		Str("auth_url", authURL).
		Msg("Generated Gmail OAuth authorization URL")

	return authURL, nil
}

// HandleOAuth2Callback processes Gmail OAuth2 callback and returns GmailAccount
func (s *GmailService) HandleOAuth2Callback(ctx context.Context, code, state, redirectURI string) (*domain.GmailAccount, error) {
	logger.FromContext(ctx).Info().
		Str("state", state).
		Str("redirect_uri", redirectURI).
		Msg("Processing Gmail OAuth2 callback")

	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}

	userID, err := s.stateSigner.VerifyState(state)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired state: %w", err)
	}

	gmailAccount, err := s.CreateGmailAccountFromAuth(ctx, userID, code, state, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail account from auth: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("gmail_account_id", gmailAccount.ID.String()).
		Msg("Successfully processed Gmail OAuth2 callback")

	return gmailAccount, nil
}

// CreateGmailAccountFromAuth creates or updates GmailAccount from Gmail OAuth authentication
func (s *GmailService) CreateGmailAccountFromAuth(ctx context.Context, userID uuid.UUID, code, state, redirectURI string) (*domain.GmailAccount, error) {
	logger.FromContext(ctx).Info().
		Str("user_id", userID.String()).
		Msg("Creating Gmail account from Gmail authentication")

	// Exchange authorization code for tokens
	token, err := s.oauthClient.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	// Get user info from Google
	info, err := s.oauthClient.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Find or create agent for this user
	agent, err := s.findOrCreateAgent(ctx, userID, info)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create agent: %w", err)
	}

	// Check if Gmail account already exists for this agent and email
	existingGmailAccount, err := s.gmailAccountRepo.FindByAgentID(ctx, agent.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing gmail account: %w", err)
	}

	var gmailAccount *domain.GmailAccount
	if existingGmailAccount == nil {
		// Create new Gmail account
		gmailAccount = &domain.GmailAccount{
			AgentID: agent.ID,
			Email:   info.Email,
			Name:    info.Name,
			Picture: info.Picture,
			Status:  domain.GmailAccountStatusActive,
		}

		gmailAccount, err = s.gmailAccountRepo.Create(ctx, gmailAccount)
		if err != nil {

			return nil, fmt.Errorf("failed to create gmail account: %w", err)
		}
	} else {
		// Update existing Gmail account
		gmailAccount = existingGmailAccount
		gmailAccount.Name = info.Name
		gmailAccount.Picture = info.Picture
		gmailAccount.Status = domain.GmailAccountStatusActive

		err = s.gmailAccountRepo.Update(ctx, gmailAccount.ID, gmailAccount)
		if err != nil {

			return nil, fmt.Errorf("failed to update gmail account: %w", err)
		}
	}

	// Store OAuth credentials in Gmail account
	if err := s.storeGmailAccountCredentials(ctx, gmailAccount.ID, token); err != nil {
		return nil, fmt.Errorf("failed to store credentials: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("gmail_account_id", gmailAccount.ID.String()).
		Str("email", gmailAccount.Email).
		Msg("Successfully created/updated Gmail account from auth")

	return gmailAccount, nil
}

// GetOrSyncGmailAccount retrieves agent by email or creates a new one if needed
func (s *GmailService) GetOrSyncGmailAccount(ctx context.Context, userID uuid.UUID, email string) (*domain.GmailAccount, error) {
	logger.FromContext(ctx).Info().
		Str("user_id", userID.String()).
		Str("email", email).
		Msg("Getting or syncing Gmail account")

	gmailAccount, err := s.gmailAccountRepo.FindByUserAndEmail(ctx, userID, email)
	if err != nil {
		return nil, fmt.Errorf("failed to find gmail account: %w", err)
	}

	if gmailAccount == nil {
		return nil, fmt.Errorf("gmail account not found for email: %s", email)
	}

	// Check if gmail account needs sync (e.g., token refresh)
	if gmailAccount.Status == domain.GmailAccountStatusInvalidGrant {
		return nil, fmt.Errorf("gmail account needs reauthorization: invalid grant")
	}
	return gmailAccount, nil
}

// StartWatch starts Gmail push notifications for a Gmail account
func (s *GmailService) StartWatch(ctx context.Context, gmailAccountID uuid.UUID) (*v2schema.GmailWatchResponse, error) {
	logger.FromContext(ctx).Info().
		Str("gmail_account_id", gmailAccountID.String()).
		Msg("Starting Gmail watch")

	if s.pubsubTopic == "" {
		return nil, fmt.Errorf("pubsub topic not configured")
	}

	// Get Gmail account and refresh token if needed
	gmailAccount, err := s.gmailAccountRepo.GetByID(ctx, gmailAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get gmail account: %w", err)
	}

	if gmailAccount == nil {
		return nil, fmt.Errorf("gmail account not found")
	}

	// Ensure valid token
	token, err := s.RefreshGmailAccountToken(ctx, gmailAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh gmail account token: %w", err)
	}

	// Create Gmail service
	service, err := s.createGmailService(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	// Start watching
	watchReq := &gmail.WatchRequest{
		TopicName:           s.pubsubTopic,
		LabelIds:            []string{"INBOX", "SENT"},
		LabelFilterBehavior: "include",
	}

	resp, err := service.Users.Watch("me", watchReq).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to start Gmail watch: %w", err)
	}

	// Update Gmail account with watch history ID and expiry
	historyID := fmt.Sprintf("%d", resp.HistoryId)
	gmailAccount.SetWatching(historyID, resp.Expiration)
	err = s.gmailAccountRepo.Update(ctx, gmailAccount.ID, gmailAccount)
	if err != nil {
		logger.FromContext(ctx).Warn().
			Err(err).
			Str("gmail_account_id", gmailAccount.ID.String()).
			Msg("Failed to update gmail account with watch data")
	}

	response := &v2schema.GmailWatchResponse{
		HistoryID:  historyID,
		Expiration: resp.Expiration,
	}

	logger.FromContext(ctx).Info().
		Str("gmail_account_id", gmailAccountID.String()).
		Str("history_id", historyID).
		Int64("expiration", resp.Expiration).
		Msg("Successfully started Gmail watch")

	return response, nil
}

// StopWatch stops Gmail push notifications for a Gmail account
func (s *GmailService) StopWatch(ctx context.Context, gmailAccountID uuid.UUID) error {
	logger.FromContext(ctx).Info().
		Str("gmail_account_id", gmailAccountID.String()).
		Msg("Stopping Gmail watch")

	// Get Gmail account and refresh token if needed
	gmailAccount, err := s.gmailAccountRepo.GetByID(ctx, gmailAccountID)
	if err != nil {
		return fmt.Errorf("failed to get gmail account: %w", err)
	}

	if gmailAccount == nil {
		return fmt.Errorf("gmail account not found")
	}

	// Ensure valid token
	token, err := s.RefreshGmailAccountToken(ctx, gmailAccountID)
	if err != nil {
		return fmt.Errorf("failed to refresh gmail account token: %w", err)
	}

	// Create Gmail service
	service, err := s.createGmailService(ctx, token)
	if err != nil {
		return fmt.Errorf("failed to create Gmail service: %w", err)
	}

	// Stop watching
	err = service.Users.Stop("me").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to stop Gmail watch: %w", err)
	}

	// Update gmail account to stop watching
	gmailAccount.StopWatching()
	err = s.gmailAccountRepo.Update(ctx, gmailAccount.ID, gmailAccount)
	if err != nil {
		logger.FromContext(ctx).Warn().
			Err(err).
			Str("gmail_account_id", gmailAccount.ID.String()).
			Msg("Failed to update gmail account after stopping watch")
	}

	logger.FromContext(ctx).Info().
		Str("gmail_account_id", gmailAccountID.String()).
		Msg("Successfully stopped Gmail watch")

	return nil
}

// HandleNotification processes Gmail push notifications
func (s *GmailService) HandleNotification(ctx context.Context, notification *GmailNotificationPayload) error {
	logger.FromContext(ctx).Info().
		Str("email_address", notification.EmailAddress).
		Str("history_id", notification.HistoryID).
		Msg("Handling Gmail notification")

	// Find Gmail accounts by email
	gmailAccounts, err := s.gmailAccountRepo.FindByEmailForWatching(ctx, notification.EmailAddress)
	if err != nil {
		return fmt.Errorf("failed to find gmail accounts by email: %w", err)
	}

	if len(gmailAccounts) == 0 {
		return fmt.Errorf("no gmail accounts found for email: %s", notification.EmailAddress)
	}

	// Enqueue background task for email processing
	for _, gmailAccount := range gmailAccounts {
		payload := map[string]interface{}{
			"gmail_account_id": gmailAccount.ID.String(),
			"email_address":    notification.EmailAddress,
			"history_id":       notification.HistoryID,
		}

		// TODO: Implement proper worker client enqueue
		// if s.workerClient != nil {
		// 	if err := s.workerClient.EnqueueTask(ctx, "gmail_notification", payload); err != nil {
		// 		logger.FromContext(ctx).Warn().
		// 			Err(err).
		// 			Str("gmail_account_id", gmailAccount.ID.String()).
		// 			Msg("Failed to enqueue Gmail notification task")
		// 	}
		// }
		_ = payload // TODO: Remove this when worker client is implemented
	}

	logger.FromContext(ctx).Info().
		Str("email_address", notification.EmailAddress).
		Int("gmail_account_count", len(gmailAccounts)).
		Msg("Successfully handled Gmail notification")

	return nil
}

// RefreshGmailAccountToken refreshes Gmail account's OAuth token
func (s *GmailService) RefreshGmailAccountToken(ctx context.Context, gmailAccountID uuid.UUID) (*oauth2.Token, error) {
	logger.FromContext(ctx).Info().
		Str("gmail_account_id", gmailAccountID.String()).
		Msg("Refreshing Gmail account token")

	// Use atomic refresh from repository
	store, err := s.gmailAccountRepo.AtomicRefreshToken(ctx, gmailAccountID, s.oauthClient)
	if err != nil {
		if strings.Contains(err.Error(), "invalid_grant") {
			// Update gmail account status to invalid grant
			updateErr := s.gmailAccountRepo.AtomicStatusUpdate(ctx, gmailAccountID, domain.GmailAccountStatusInvalidGrant)
			if updateErr != nil {
				logger.FromContext(ctx).Error().
					Err(updateErr).
					Str("gmail_account_id", gmailAccountID.String()).
					Msg("Failed to update gmail account status to invalid_grant")
			}

			return nil, fmt.Errorf("gmail account token invalid_grant: please reauthorize Gmail access")
		}
		return nil, fmt.Errorf("failed to refresh gmail account token: %w", err)
	}

	token := &oauth2.Token{
		AccessToken:  store.AccessToken,
		RefreshToken: store.RefreshToken,
		Expiry:       store.Expiry,
		TokenType:    "Bearer",
	}

	logger.FromContext(ctx).Info().
		Str("gmail_account_id", gmailAccountID.String()).
		Msg("Successfully refreshed Gmail account token")

	return token, nil
}

// CheckGmailAccountTokenStatus checks the status of Gmail account's OAuth token
func (s *GmailService) CheckGmailAccountTokenStatus(ctx context.Context, gmailAccountID uuid.UUID) (map[string]interface{}, error) {
	logger.FromContext(ctx).Info().
		Str("gmail_account_id", gmailAccountID.String()).
		Msg("Checking Gmail account token status")

	gmailAccount, err := s.gmailAccountRepo.GetByID(ctx, gmailAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get gmail account: %w", err)
	}

	if gmailAccount == nil {
		return nil, fmt.Errorf("gmail account not found")
	}

	// Get token store
	store, err := gmailAccount.GetGoogleTokenStore()
	if err != nil {
		return nil, fmt.Errorf("failed to get token store: %w", err)
	}

	status := map[string]interface{}{
		"gmail_account_id":  gmailAccount.ID.String(),
		"email":             gmailAccount.Email,
		"status":            string(gmailAccount.Status),
		"token_expiry":      store.Expiry,
		"has_refresh_token": store.RefreshToken != "",
		"is_valid":          gmailAccount.Status == domain.GmailAccountStatusActive,
		"is_watching":       gmailAccount.IsWatching,
		"watch_expiry":      gmailAccount.WatchExpiry,
	}

	logger.FromContext(ctx).Info().
		Str("gmail_account_id", gmailAccountID.String()).
		Str("status", string(gmailAccount.Status)).
		Bool("is_valid", gmailAccount.Status == domain.GmailAccountStatusActive).
		Msg("Successfully checked Gmail account token status")

	return status, nil
}

// Helper methods

func (s *GmailService) findOrCreateAgent(ctx context.Context, userID uuid.UUID, info *googleoauth.UserInfo) (*domain.Agent, error) {
	// Try to find existing agent by user ID and email
	agent, err := s.agentRepo.FindByUserAndEmail(ctx, userID, info.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to find existing agent: %w", err)
	}

	if agent != nil {
		// Update existing agent with Gmail info
		agent.Name = info.Name
		agent.Picture = info.Picture
		agent.EmailProvider = domain.AgentEmailProviderGmail
		agent.Status = domain.AgentStatusActive

		// Update agent
		err = s.agentRepo.Update(ctx, agent.ID, agent)
		if err != nil {
			return nil, fmt.Errorf("failed to update existing agent: %w", err)
		}

		return agent, nil
	}

	// Create new agent
	newAgent := &domain.Agent{
		UserID:        userID,
		Email:         info.Email,
		Name:          info.Name,
		Picture:       info.Picture,
		EmailProvider: domain.AgentEmailProviderGmail,
		Status:        domain.AgentStatusActive,
	}

	// Create agent
	createdAgent, err := s.agentRepo.Create(ctx, newAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to create new agent: %w", err)
	}

	return createdAgent, nil
}

func (s *GmailService) storeGmailAccountCredentials(ctx context.Context, gmailAccountID uuid.UUID, token *googleoauth.Token) error {
	// Get the gmail account
	gmailAccount, err := s.gmailAccountRepo.GetByID(ctx, gmailAccountID)
	if err != nil {
		return fmt.Errorf("failed to get gmail account: %w", err)
	}

	// Convert token to GoogleTokenStore
	tokenStore := &google_token_store.GoogleTokenStore{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		Scopes:       []string{}, // OAuth scopes will be managed separately
	}

	// Update credentials
	if err := gmailAccount.UpdateGoogleTokenStore(tokenStore); err != nil {
		return fmt.Errorf("failed to update google token store: %w", err)
	}

	// Save the updated gmail account
	return s.gmailAccountRepo.Update(ctx, gmailAccountID, gmailAccount)
}

func (s *GmailService) createGmailService(ctx context.Context, token *oauth2.Token) (*gmail.Service, error) {
	config := s.oauthClient.GetConfig()
	if config == nil {
		return nil, fmt.Errorf("oauth config not available")
	}

	client := config.Client(ctx, token)
	service, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	return service, nil
}
