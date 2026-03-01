package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v2oauth "github.com/rockship/cosmo-agents-go/internal/handler/v2/oauth"
	agent "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// GmailHandler handles Gmail-related V2 endpoints to match Python API
type GmailHandler struct {
	oauthClient     *googleoauth.Client
	agentRepo       *agent.AgentRepository
	userRepo        *user.UserRepository
	pubsubTopicName string // Google Pub/Sub topic for Gmail notifications
	stateSigner     *v2oauth.OAuthStateSigner
	workerClient    WorkerClient  // For background task processing
	redisClient     *redis.Client // For rate limiting
}

// WorkerClient interface for enqueuing background tasks
type WorkerClient interface {
	EnqueueTask(ctx context.Context, taskType string, payload interface{}) error
}

// NewGmailHandler creates a new GmailHandler
func NewGmailHandler(
	oauthClient *googleoauth.Client,
	agentRepo *agent.AgentRepository,
	userRepo *user.UserRepository,
	pubsubTopicName string,
	jwtSecret string,
	workerClient WorkerClient,
	redisClient *redis.Client,
) *GmailHandler {
	return &GmailHandler{
		oauthClient:     oauthClient,
		agentRepo:       agentRepo,
		userRepo:        userRepo,
		pubsubTopicName: normalizePubSubTopicName(pubsubTopicName),
		stateSigner:     v2oauth.NewOAuthStateSigner(jwtSecret),
		workerClient:    workerClient,
		redisClient:     redisClient,
	}
}

// normalizePubSubTopicName ensures topic name follows "projects/<project>/topics/<topic>".
// Accepts bare topic IDs, "topics/<topic>" and full Pub/Sub URLs.
func normalizePubSubTopicName(topic string) string {
	raw := strings.TrimSpace(topic)
	if raw == "" {
		return ""
	}

	// Strip full Pub/Sub URL prefix if present.
	normalized := strings.TrimPrefix(raw, "//pubsub.googleapis.com/")

	// Already fully qualified.
	if strings.HasPrefix(normalized, "projects/") {
		return normalized
	}

	normalized = strings.TrimPrefix(normalized, "topics/")

	projectID := firstNonEmpty(
		os.Getenv("GOOGLE_PUBSUB_PROJECT_ID"),
		os.Getenv("GOOGLE_CLOUD_PROJECT"),
		os.Getenv("GCLOUD_PROJECT"),
		os.Getenv("GCP_PROJECT"),
		os.Getenv("PROJECT_ID"),
	)
	if projectID == "" {
		logger.Logger.Warn().
			Str("topic", raw).
			Msg("Gmail Pub/Sub topic is missing project prefix and no project ID env found; using raw topic")
		return normalized
	}

	resolved := fmt.Sprintf("projects/%s/topics/%s", projectID, normalized)
	logger.Logger.Info().
		Str("topic", raw).
		Str("resolved_topic", resolved).
		Msg("Normalized Gmail Pub/Sub topic name")
	return resolved
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// checkRateLimit checks if user is rate limited using Redis sliding window with Lua script
// Returns true if rate limit exceeded, false if allowed
func (h *GmailHandler) checkRateLimit(ctx context.Context, userID uuid.UUID, limit int, window time.Duration) (bool, error) {
	if h.redisClient == nil {
		// CRITICAL FIX: Fail-closed when Redis is not available to prevent DoS
		// Instead of silently bypassing rate limiting, block all requests
		logger.Logger.Error().
			Str("user_id", userID.String()).
			Msg("Rate limiting unavailable - Redis client is nil, blocking request")
		return true, fmt.Errorf("rate limiting service unavailable")
	}

	key := fmt.Sprintf("rate_limit:gmail_refresh:%s", userID.String())
	now := time.Now().Unix()
	windowSeconds := int64(window.Seconds())

	// CRITICAL FIX: Use Lua script for atomic check-and-increment to eliminate race condition
	// This ensures atomicity - no race window between count check and increment
	luaScript := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local current_time = now

		-- Remove expired entries
		redis.call('ZREMRANGEBYSCORE', key, 0, current_time - window)

		-- Count current requests
		local current_count = redis.call('ZCARD', key)

		-- Add current request
		redis.call('ZADD', key, current_time, current_time)
		redis.call('EXPIRE', key, window)

		-- Return if count exceeds limit (after adding current request)
		return current_count + 1 > limit and 1 or 0
	`

	result, err := h.redisClient.Eval(ctx, luaScript, []string{key}, now, windowSeconds, limit).Result()
	if err != nil {
		// Redis error, allow request but log the error
		logger.Logger.Error().Err(err).Str("user_id", userID.String()).Msg("Rate limit check failed")
		return false, nil
	}

	// CRITICAL FIX: Safe type assertion to prevent panic if Redis returns unexpected type
	limitExceeded := false
	if resultInt, ok := result.(int64); ok {
		limitExceeded = resultInt == 1
	} else {
		// Log unexpected result type but allow request (fail-safe)
		logger.Logger.Warn().
			Str("user_id", userID.String()).
			Interface("result", result).
			Msg("Unexpected Redis result type in rate limit check")
	}
	return limitExceeded, nil
}

// CheckAgentTokenStatus godoc
// @Summary Check agent token status
// @Description Check if Gmail agent token is valid, expired, or needs re-authorization
// @Tags Gmail
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param agent_id path string true "Agent ID"
// @Success 200 {object} map[string]interface{} "Token status retrieved"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Agent not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /v2/gmail/agents/{agent_id}/token-status [get]
func (h *GmailHandler) CheckAgentTokenStatus(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	agentIDStr := c.Params("agent_id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid agent ID format",
		})
	}

	// Get agent
	agent, err := h.agentRepo.GetByID(c.Context(), agentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Agent not found",
		})
	}

	// Verify agent belongs to user
	if agent.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Access denied",
		})
	}

	// Check token status
	store, err := agent.GetGoogleTokenStore()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to get token store",
		})
	}

	// CRITICAL FIX: Handle zero-time expiry to prevent panics in handler
	var isExpired, willExpireSoon bool
	if store.Expiry.IsZero() {
		// Malformed credentials - treat as expired
		isExpired = true
		willExpireSoon = true
	} else {
		isExpired = time.Now().After(store.Expiry)
		willExpireSoon = time.Now().Add(1 * time.Hour).After(store.Expiry)
	}

	tokenStatus := map[string]interface{}{
		"agent_id":          agent.ID.String(),
		"current_status":    agent.Status,
		"has_access_token":  store.AccessToken != "",
		"has_refresh_token": store.RefreshToken != "",
		"token_expiry":      store.Expiry,
		"is_expired":        isExpired,
		"needs_reauth":      agent.Status == domain.AgentStatusInvalidGrant,
		"will_expire_soon":  willExpireSoon,
	}

	// Log token status for debugging
	logger.Logger.Info().
		Str("agent_id", agent.ID.String()).
		Str("status", string(agent.Status)).
		Str("token_expiry", store.Expiry.String()).
		Bool("is_expired", tokenStatus["is_expired"].(bool)).
		Bool("needs_reauth", tokenStatus["needs_reauth"].(bool)).
		Msg("Agent token status checked")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   tokenStatus,
	})
}

// GetGmailAuthorizationURL godoc
// @Summary Get Gmail authorization URL
// @Description Generates Gmail OAuth2 authorization URL for accessing Gmail API
// @Tags Gmail
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param redirect_uri query string false "The URI to redirect after authentication"
// @Success 200 {object} schema.APIResponse[v2schema.GmailAuthURLResponse] "Authorization URL"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /v2/google/gmail [get]
func (h *GmailHandler) GetGmailAuthorizationURL(c fiber.Ctx) error {
	userVal := c.Locals("user_id")
	userUUID, ok := userVal.(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Unauthorized",
			},
		})
	}

	// Ensure user exists (state references this ID later)
	if _, err := h.userRepo.FindByID(c.Context(), userUUID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Failed to load user",
			},
		})
	}

	redirectURI := c.Query("redirect_uri")
	if redirectURI == "" {
		// Build default redirect URI
		scheme := "http"
		if c.Protocol() == "https" {
			scheme = "https"
		}
		redirectURI = fmt.Sprintf("%s://%s/v2/google/gmail/oauth2callback", scheme, c.Hostname())
	}

	oauthCfg := h.cloneOAuthConfig(redirectURI)
	if oauthCfg == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "OAuth configuration not available",
			},
		})
	}

	// Generate signed state token containing user ID
	// This is stateless and works across multiple server instances
	// Prevents CSRF attacks while supporting horizontal scaling
	stateToken, err := h.stateSigner.CreateState(userUUID, 10*time.Minute)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Failed to generate state token",
			},
		})
	}

	authURL := oauthCfg.AuthCodeURL(
		stateToken, // Use signed state token
		oauth2.AccessTypeOffline,
		oauth2.ApprovalForce,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	// Match Python response format
	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(v2schema.GmailAuthURLResponse{
		URL: authURL,
	}))
}

// HandleGmailOAuth2Callback godoc
// @Summary Gmail OAuth2 callback
// @Description Handles the Gmail OAuth2 callback and exchanges code for tokens
// @Tags Gmail
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param code query string true "Authorization code from Google"
// @Param state query string true "State token (server-generated CSRF token)"
// @Param redirect_uri query string false "Redirect URI"
// @Success 200 {object} schema.APIResponse[v1schema.AgentResponse] "Agent information"
// @Failure 400 {object} map[string]interface{} "Bad request - invalid/expired state or missing code"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /v2/google/gmail/oauth2callback [get]
func (h *GmailHandler) HandleGmailOAuth2Callback(c fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state") // state contains random token, not user ID
	redirectURI := c.Query("redirect_uri")

	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Missing authorization code",
			},
		})
	}

	if state == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Missing state parameter",
			},
		})
	}

	// Verify signed state token and extract user ID
	// This is stateless and works across multiple server instances
	// Prevents CSRF attacks by verifying the signature
	userID, err := h.stateSigner.VerifyState(state)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": fmt.Sprintf("Invalid or expired state: %v", err),
			},
		})
	}

	ctx := c.Context()
	oauthCfg := h.cloneOAuthConfig(redirectURI)
	if oauthCfg == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "OAuth configuration not available",
			},
		})
	}

	// Exchange code for tokens
	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Failed to exchange authorization code",
				"detail":  err.Error(),
			},
		})
	}

	// Get user info
	userInfo, err := h.oauthClient.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Failed to get user information",
			},
		})
	}

	user, err := h.userRepo.FindWithOrganizations(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Failed to load user",
			},
		})
	}
	if user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "User not found",
			},
		})
	}

	orgID, err := h.resolveOrganizationID(c.Context(), user)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": err.Error(),
			},
		})
	}

	agent, err := h.findOrCreateAgent(c.Context(), user, orgID, userInfo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Failed to persist agent",
			},
		})
	}

	// Build token store with new access token
	// IMPORTANT: Only update refresh token if Google provides a new one
	// Google may omit refresh token on subsequent authorizations
	tokenStore := &domain.GoogleTokenStore{
		AccessToken: token.AccessToken,
		Scopes:      oauthCfg.Scopes,
		Expiry:      token.Expiry.UTC(),
	}

	// Log what we got back from Google (masking refresh token presence only)
	logger.Logger.Info().
		Str("agent_id", agent.ID.String()).
		Str("user_id", user.ID.String()).
		Bool("has_refresh_token", token.RefreshToken != "").
		Int("scope_count", len(oauthCfg.Scopes)).
		Msg("Gmail OAuth callback received tokens")

	// Preserve existing refresh token if Google doesn't provide a new one
	if token.RefreshToken != "" {
		tokenStore.RefreshToken = token.RefreshToken
	} else {
		// Get existing refresh token from agent
		if existingStore, err := agent.GetGoogleTokenStore(); err == nil && existingStore.RefreshToken != "" {
			tokenStore.RefreshToken = existingStore.RefreshToken
			logger.Logger.Info().
				Str("agent_id", agent.ID.String()).
				Msg("Gmail OAuth callback reused existing refresh token for agent")
		}
	}

	if err := agent.UpdateGoogleTokenStore(tokenStore); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Failed to store Gmail credentials",
			},
		})
	}

	// Restore healthy status after successful re-authorization.
	agent.Status = domain.AgentStatusActive
	valid := true
	agent.ValidCred = &valid

	if err := h.agentRepo.Update(c.Context(), agent.ID, agent); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Failed to update agent credentials",
			},
		})
	}

	// Log after persisting credentials to DB
	logger.Logger.Info().
		Str("agent_id", agent.ID.String()).
		Str("user_id", user.ID.String()).
		Bool("has_refresh_token", tokenStore.RefreshToken != "").
		Msg("Gmail OAuth callback persisted agent credentials")

	var historyID *string
	if h.pubsubTopicName != "" {
		if watchID, _, watchErr := h.startAgentWatch(ctx, agent); watchErr == nil && watchID != "" {
			historyID = &watchID
			if err := h.agentRepo.Update(c.Context(), agent.ID, agent); err != nil {
				logger.Logger.Warn().
					Err(err).
					Str("agent_id", agent.ID.String()).
					Msg("Failed to persist agent after starting Gmail watch")
			}
		}
	}

	resp := mapAgentResponse(agent)
	if historyID != nil {
		resp.LastHistoryID = *historyID
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(resp))
}

// GmailWatch godoc
// @Summary Start watching Gmail
// @Description Starts watching Gmail for push notifications via Google Pub/Sub
// @Tags Gmail
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[[]v2schema.GmailWatchResponse] "Watch responses"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /v2/google/gmail/watch [post]
func (h *GmailHandler) GmailWatch(c fiber.Ctx) error {
	userVal := c.Locals("user_id")
	userUUID, ok := userVal.(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Unauthorized",
			},
		})
	}

	agents, err := h.agentRepo.GetByUserID(c.Context(), userUUID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Failed to load agents",
			},
		})
	}

	if len(agents) == 0 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "success",
			"data":   []v2schema.GmailWatchResponse{},
		})
	}

	responses := make([]v2schema.GmailWatchResponse, 0, len(agents))
	// OPTIMIZATION: Use parent context with overall timeout instead of creating contexts per iteration
	// This prevents resource waste while maintaining safety
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second) // Overall 30s timeout for all agents
	defer cancel()                                                  // This defer is safe - called once at function end

	for i := range agents {
		agent := &agents[i]
		historyID, expiration, err := h.startAgentWatch(ctx, agent)
		if err != nil {
			// Check if error is due to invalid_grant (refresh token expired)
			// This matches Python behavior in ctrl.py:142-146
			if strings.Contains(err.Error(), "invalid_grant") {
				agent.Status = domain.AgentStatusInvalidGrant
				invalid := false
				agent.ValidCred = &invalid
				if updateErr := h.agentRepo.Update(ctx, agent.ID, agent); updateErr != nil {
					logger.Logger.Error().
						Err(updateErr).
						Str("agent_id", agent.ID.String()).
						Msg("Failed to update agent status to invalid_grant")
				}
				logger.Logger.Error().
					Err(err).
					Str("agent_id", agent.ID.String()).
					Msg("Failed to refresh token for agent - marked as invalid_grant")
			} else {
				logger.Logger.Warn().
					Err(err).
					Str("agent_id", agent.ID.String()).
					Msg("Failed to start Gmail watch for agent")
			}
			continue
		}
		if historyID != "" {
			responses = append(responses, v2schema.GmailWatchResponse{
				HistoryID:  historyID,
				Expiration: expiration,
			})
		}
		if err := h.agentRepo.Update(c.Context(), agent.ID, agent); err != nil {
			logger.Logger.Warn().
				Err(err).
				Str("agent_id", agent.ID.String()).
				Msg("Failed to persist agent watch data")
		}
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(responses))
}

// GmailStop godoc
// @Summary Stop watching Gmail
// @Description Stops watching Gmail for push notifications
// @Tags Gmail
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[v2schema.GmailStopResponse] "Stop response"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Router /v2/google/gmail/stop [post]
func (h *GmailHandler) GmailStop(c fiber.Ctx) error {
	userVal := c.Locals("user_id")
	userUUID, ok := userVal.(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Unauthorized",
			},
		})
	}

	// Get the first (primary) agent for the user to match Python behavior
	// In Python, it uses current_user.credentials which represents the primary agent
	agents, err := h.agentRepo.GetByUserID(c.Context(), userUUID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "error",
			"error": fiber.Map{
				"message": "Failed to load agents",
			},
		})
	}

	if len(agents) == 0 {
		logger.Logger.Info().Str("user_id", userUUID.String()).Msg("No agents found for user")
		return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(v2schema.GmailStopResponse{
			Message: "Stop watching for Gmail notifications for the current user successfully.",
		}))
	}

	// Stop only the first agent (primary agent) to match Python behavior
	ctx := c.Context()
	agent := &agents[0]
	if err := h.stopAgentWatch(ctx, agent); err != nil {
		logger.Logger.Warn().
			Err(err).
			Str("agent_id", agent.ID.String()).
			Msg("Failed to stop Gmail watch for agent")
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(v2schema.GmailStopResponse{
		Message: "Stop watching for Gmail notifications for the current user successfully.",
	}))
}

// GmailNotifications godoc
// @Summary Handle Gmail Pub/Sub notifications
// @Description Handles Gmail push notifications from Google Pub/Sub webhook
// @Tags Gmail
// @Accept json
// @Produce json
// @Param notification body v2schema.GmailNotificationRequest true "Gmail notification payload"
// @Success 200 {object} map[string]interface{} "Notification processed"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Router /v2/google/gmail/notifications [post]
func (h *GmailHandler) GmailNotifications(c fiber.Ctx) error {
	var req v2schema.GmailNotificationRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		logger.Logger.Warn().Err(err).Msg("Gmail notification: invalid JSON body")
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "success",
			"data":   "No event messages received from Gmail. Ignored.",
		})
	}

	// Decode base64 message data
	if req.Message.Data == "" {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "success",
			"data":   "No event messages received from Gmail. Ignored.",
		})
	}

	decodedData, err := base64.StdEncoding.DecodeString(req.Message.Data)
	if err != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "success",
			"data":   "No event messages received from Gmail. Ignored.",
		})
	}

	// Parse Gmail message
	var gmailMsg v2schema.GmailMessage
	if err := json.Unmarshal(decodedData, &gmailMsg); err != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "success",
			"data":   "No event messages received from Gmail. Ignored.",
		})
	}

	// Enqueue background task to process the notification
	// This matches Python's background_tasks.add_task(...)
	if h.workerClient != nil {
		payload := map[string]interface{}{
			"email_address": gmailMsg.EmailAddress,
			"history_id":    gmailMsg.HistoryID,
		}

		if err := h.workerClient.EnqueueTask(c.Context(), worker.TypeGmailNotification, payload); err != nil {
			logger.Logger.Error().
				Err(err).
				Str("email", gmailMsg.EmailAddress).
				Int64("history_id", gmailMsg.HistoryID).
				Msg("Failed to enqueue Gmail notification task")
			// Don't fail the request - Google expects 200 OK
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   "healthy",
	})
}

// Helper function to create Gmail service
func (h *GmailHandler) createGmailService(ctx context.Context, token *oauth2.Token) (*gmail.Service, error) {
	client := h.oauthClient.GetConfig().Client(ctx, token)
	service, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}
	return service, nil
}

func (h *GmailHandler) cloneOAuthConfig(redirectURI string) *oauth2.Config {
	if h.oauthClient == nil {
		return nil
	}
	base := h.oauthClient.GetConfig()
	if base == nil {
		return nil
	}
	clone := *base
	if redirectURI != "" {
		clone.RedirectURL = redirectURI
	}
	return &clone
}

func (h *GmailHandler) resolveOrganizationID(ctx context.Context, user *domain.User) (*uuid.UUID, error) {
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Note: User.Roles relationship removed to avoid circular imports
	// Role data should be loaded separately using relations package if needed

	org, err := h.userRepo.FindFirstOrganizationOfUser(ctx, user.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if org != nil {
		return &org.ID, nil
	}

	return nil, errors.New("user must belong to an organization before connecting Gmail")
}

func (h *GmailHandler) findOrCreateAgent(ctx context.Context, user *domain.User, orgID *uuid.UUID, info *googleoauth.UserInfo) (*domain.Agent, error) {
	email := info.Email
	if email == "" {
		email = user.Email
	}
	agent, err := h.agentRepo.FindByUserAndEmail(ctx, user.ID, email)
	if err != nil {
		return nil, err
	}

	name := info.Name
	if name == "" {
		name = user.Name
	}

	var orgPtr *uuid.UUID
	if orgID != nil {
		idCopy := *orgID
		orgPtr = &idCopy
	}

	if agent == nil {
		newAgent := &domain.Agent{
			UserID:         user.ID,
			Email:          email,
			Name:           name,
			Picture:        info.Picture,
			OrganizationID: orgPtr,
			EmailProvider:  domain.AgentEmailProviderGmail,
			Status:         domain.AgentStatusActive,
		}
		created, err := h.agentRepo.Create(ctx, newAgent)
		if err != nil {
			return nil, err
		}
		return created, nil
	}

	agent.Name = name
	agent.Picture = info.Picture
	agent.EmailProvider = domain.AgentEmailProviderGmail
	agent.Status = domain.AgentStatusActive
	agent.OrganizationID = orgPtr

	return agent, nil
}

// ensureAgentTokenAtomic safely refreshes agent token using database locking to prevent race conditions
// CRITICAL FIX: Removed redundant pre-check to eliminate TOCTOU vulnerability
// AtomicRefreshToken handles its own locking and token expiry checking
func (h *GmailHandler) ensureAgentTokenAtomic(ctx context.Context, agent *domain.Agent) (*oauth2.Token, error) {
	store, err := h.agentRepo.AtomicRefreshToken(ctx, agent.ID, h.oauthClient)
	if err != nil {
		return nil, fmt.Errorf("failed to atomically refresh token: %w", err)
	}

	return &oauth2.Token{
		AccessToken:  store.AccessToken,
		RefreshToken: store.RefreshToken,
		Expiry:       store.Expiry,
		TokenType:    "Bearer",
	}, nil
}

func (h *GmailHandler) startAgentWatch(ctx context.Context, agent *domain.Agent) (string, int64, error) {
	if h.pubsubTopicName == "" {
		return "", 0, nil
	}

	token, err := h.ensureAgentTokenAtomic(ctx, agent)
	if err != nil {
		return "", 0, err
	}

	service, err := h.createGmailService(ctx, token)
	if err != nil {
		return "", 0, err
	}

	resp, err := service.Users.Watch("me", &gmail.WatchRequest{
		TopicName:           h.pubsubTopicName,
		LabelIds:            []string{"INBOX", "SENT"},
		LabelFilterBehavior: "include",
	}).Do()
	if err != nil {
		return "", 0, err
	}

	historyID := fmt.Sprintf("%d", resp.HistoryId)
	agent.LastHistoryID = historyID
	return historyID, resp.Expiration, nil
}

func (h *GmailHandler) stopAgentWatch(ctx context.Context, agent *domain.Agent) error {
	token, err := h.ensureAgentTokenAtomic(ctx, agent)
	if err != nil {
		return err
	}
	service, err := h.createGmailService(ctx, token)
	if err != nil {
		return err
	}
	err = service.Users.Stop("me").Context(ctx).Do()
	return err
}

// RefreshToken manually refreshes Gmail tokens for debugging
// @Summary Refresh Gmail token manually
// @Description Manually refresh Gmail access token for current user
// @Tags Gmail
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Refresh successful"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /gmail/refresh-token [post]
func (h *GmailHandler) RefreshToken(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	// Rate limiting: 1 request per minute per user to prevent DoS
	rateLimitExceeded, err := h.checkRateLimit(c.Context(), userID, 1, 1*time.Minute)
	if err != nil {
		// CRITICAL FIX: Enforce rate limiting even when Redis fails - fail-closed
		logger.Logger.Error().Err(err).Str("user_id", userID.String()).Msg("Rate limiting service failed, blocking request")
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":      "error",
			"message":     "Rate limiting service temporarily unavailable",
			"retry_after": "60s",
		})
	} else if rateLimitExceeded {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"status":      "error",
			"message":     "Rate limit exceeded. Please try again later.",
			"retry_after": "60s",
		})
	}

	agents, err := h.agentRepo.GetByUserID(c.Context(), userID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to get agents",
		})
	}

	if len(agents) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "No Gmail agents found for user",
		})
	}

	// Rate limiting: process maximum 10 agents per request to prevent DoS
	const maxAgentsPerRequest = 10
	totalAgentCount := len(agents) // Store original count before slicing
	wasRateLimited := len(agents) > maxAgentsPerRequest

	if wasRateLimited {
		logger.Logger.Warn().
			Int("total_agents", totalAgentCount).
			Int("processing_limit", maxAgentsPerRequest).
			Msg("Limiting agent refresh to prevent DoS")

		agents = agents[:maxAgentsPerRequest]
	}

	results := make([]map[string]interface{}, 0)
	successCount := 0
	errorCount := 0

	for i := range agents {
		agent := &agents[i] // Convert to pointer
		logger.Logger.Info().
			Str("agent_id", agent.ID.String()).
			Msg("Manually refreshing Gmail token")

		token, err := h.ensureAgentTokenAtomic(c.Context(), agent)
		if err != nil {
			errorCount++
			logger.Logger.Error().
				Err(err).
				Str("agent_id", agent.ID.String()).
				Msg("Manual token refresh failed")

			results = append(results, map[string]interface{}{
				"agent_id": agent.ID.String(),
				"status":   "error",
				"error":    err.Error(),
			})
			continue
		}

		successCount++
		results = append(results, map[string]interface{}{
			"agent_id": agent.ID.String(),
			"status":   "success",
			"expiry":   token.Expiry,
		})
	}

	// Log summary for better observability
	logger.Logger.Info().
		Int("total_agents", totalAgentCount).
		Int("processed_agents", len(agents)).
		Int("success_count", successCount).
		Int("error_count", errorCount).
		Bool("rate_limited", wasRateLimited).
		Msg("Gmail token refresh batch completed")

	response := fiber.Map{
		"status": "success",
		"data":   results,
	}

	// Include rate limiting info if we hit the limit
	if wasRateLimited {
		response["rate_limited"] = fiber.Map{
			"total_agents":     totalAgentCount,
			"processed_agents": len(agents),
			"limit":            maxAgentsPerRequest,
			"message":          "Use pagination or background job for large agent counts",
		}
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func mapAgentResponse(agent *domain.Agent) v1schema.AgentResponse {
	metadata := make(map[string]any)
	if len(agent.CMetadata) > 0 {
		if err := json.Unmarshal(agent.CMetadata, &metadata); err != nil {
			logger.Logger.Warn().
				Err(err).
				Str("agent_id", agent.ID.String()).
				Msg("Failed to unmarshal agent metadata")
		}
	}

	return v1schema.AgentResponse{
		ID:              agent.ID,
		CreatedAt:       agent.CreatedAt,
		UpdatedAt:       agent.UpdatedAt,
		UserID:          agent.UserID,
		OrganizationID:  agent.OrganizationID,
		Name:            agent.Name,
		Email:           agent.Email,
		Persona:         agent.Persona,
		EmailProvider:   string(agent.EmailProvider),
		Signature:       agent.Signature,
		Picture:         agent.Picture,
		Status:          string(agent.Status),
		DailyLimit:      agent.DailyLimit,
		MaxDailyLimit:   agent.MaxDailyLimit,
		ValidCred:       agent.ValidCred,
		EmailsSentToday: agent.EmailsSentToday,
		LastHistoryID:   agent.LastHistoryID,
		CMetadata:       metadata,
		IsDeleted:       agent.IsDeleted,
	}
}
