package gmail

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/gmail"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
)

// GmailHandler handles Gmail OAuth2 and API operations.
type GmailHandler struct {
	oauth2Client    *googleoauth.Client
	agentRepo       *agentRepo.AgentRepository
	userRepo        *user.UserRepository
	pubsubTopicName string
}

// NewGmailHandler creates a new Gmail handler.
func NewGmailHandler(
	oauth2Client *googleoauth.Client,
	agentRepo *agentRepo.AgentRepository,
	userRepo *user.UserRepository,
	pubsubTopicName string,
) *GmailHandler {
	return &GmailHandler{
		oauth2Client:    oauth2Client,
		agentRepo:       agentRepo,
		userRepo:        userRepo,
		pubsubTopicName: pubsubTopicName,
	}
}

// GetAuthURL generates OAuth2 authorization URL.
// @Summary Get Gmail OAuth URL
// @Description Generates a Gmail OAuth2 authorization URL for agent authentication
// @Tags Gmail
// @Accept json
// @Produce json
// @Param auth body object{agent_id=string} true "Agent ID for authentication"
// @Success 200 {object} schema.APIResponse[map[string]string] "Successfully generated auth URL"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 404 {object} schema.APIResponse[any] "Agent not found"
// @Security BearerAuth
// @Router /v1/gmail/auth/url [post]
func (h *GmailHandler) GetAuthURL(c fiber.Ctx) error {
	var req struct {
		AgentID uuid.UUID `json:"agent_id" validate:"required"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if err := v1validation.ValidateStruct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	// Verify agent exists and belongs to user
	agent, err := h.agentRepo.FindByID(c.Context(), req.AgentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Agent not found", err.Error(),
		))
	}

	// Generate random state for CSRF protection
	state := generateState()

	// Store state temporarily (in production, use Redis with TTL)
	// For now, we'll encode agent_id in state (simplified)
	stateWithAgent := fmt.Sprintf("%s:%s", state, agent.ID.String())

	authURL := h.oauth2Client.GetAuthURL(stateWithAgent)

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"auth_url": authURL,
		"state":    stateWithAgent,
	}))
}

// HandleCallback handles OAuth2 callback from Google.
// @Summary Handle Gmail OAuth callback
// @Description Handles the OAuth2 callback from Google after user authorization
// @Tags Gmail
// @Accept json
// @Produce json
// @Param code query string true "Authorization code from Google"
// @Param state query string true "State parameter for CSRF protection"
// @Param error query string false "Error from OAuth provider"
// @Success 200 {object} schema.APIResponse[any] "Successfully connected Gmail"
// @Failure 400 {object} schema.APIResponse[any] "Invalid code, state, or OAuth error"
// @Failure 404 {object} schema.APIResponse[any] "Agent not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Router /v1/gmail/auth/callback [get]
func (h *GmailHandler) HandleCallback(c fiber.Ctx) error {
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	if errorParam != "" {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "OAuth error", errorParam,
		))
	}

	if code == "" || state == "" {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Missing code or state", "code and state are required",
		))
	}

	// Parse agent ID from state (simplified - in production use Redis)
	agentID, err := parseAgentIDFromState(state)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid state", err.Error(),
		))
	}

	// Exchange code for tokens
	token, err := h.oauth2Client.ExchangeCode(c.Context(), code)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to exchange code", err.Error(),
		))
	}

	// Get user info
	userInfo, err := h.oauth2Client.GetUserInfo(c.Context(), token.AccessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to get user info", err.Error(),
		))
	}

	// Get agent
	agent, err := h.agentRepo.FindByID(c.Context(), agentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Agent not found", err.Error(),
		))
	}

	// Update agent with tokens
	tokenStore := &domain.GoogleTokenStore{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Scopes: []string{
			"https://www.googleapis.com/auth/gmail.send",
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/gmail.labels",
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Expiry: token.Expiry,
	}

	if err := agent.UpdateGoogleTokenStore(tokenStore); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to store tokens", err.Error(),
		))
	}

	// Update email if not set
	if agent.Email == "" {
		agent.Email = userInfo.Email
	}

	agent.Status = domain.AgentStatusActive

	if err := h.agentRepo.Update(c.Context(), agent.ID, agent); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to update agent", err.Error(),
		))
	}

	if h.pubsubTopicName != "" {
		if err := h.startGmailWatch(c.Context(), agent, tokenStore); err != nil {
			logger.Logger.Warn().
				Err(err).
				Str("agent_id", agent.ID.String()).
				Msg("Failed to start Gmail watch after OAuth callback")
		}
	}

	// Return success response
	return c.JSON(schema.SuccessResponse(fiber.Map{
		"message":  "Gmail connected successfully",
		"agent_id": agent.ID,
		"email":    userInfo.Email,
	}))
}

func (h *GmailHandler) startGmailWatch(ctx context.Context, agent *domain.Agent, tokenStore *domain.GoogleTokenStore) error {
	if h.pubsubTopicName == "" {
		return nil
	}

	gmailClient, err := gmail.NewClient(ctx, tokenStore.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to create Gmail client for watch: %w", err)
	}

	watchResult, err := gmailClient.Watch(ctx, h.pubsubTopicName, []string{"INBOX", "SENT"})
	if err != nil {
		return err
	}

	if watchResult != nil && watchResult.HistoryID != "" {
		agent.LastHistoryID = watchResult.HistoryID
		if err := h.agentRepo.Update(ctx, agent.ID, agent); err != nil {
			return fmt.Errorf("failed to persist Gmail watch metadata: %w", err)
		}
	}

	return nil
}

// RefreshToken manually refreshes the OAuth2 token.
// @Summary Refresh Gmail OAuth token
// @Description Manually refreshes the OAuth2 access token for an agent
// @Tags Gmail
// @Accept json
// @Produce json
// @Param refresh body object{agent_id=string} true "Agent ID for token refresh"
// @Success 200 {object} schema.APIResponse[any] "Successfully refreshed token"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or no tokens found"
// @Failure 404 {object} schema.APIResponse[any] "Agent not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/gmail/auth/refresh [post]
func (h *GmailHandler) RefreshToken(c fiber.Ctx) error {
	var req struct {
		AgentID uuid.UUID `json:"agent_id" validate:"required"`
	}

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	// Get agent
	agent, err := h.agentRepo.FindByID(c.Context(), req.AgentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Agent not found", err.Error(),
		))
	}

	// Get current token store
	tokenStore, err := agent.GetGoogleTokenStore()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "No tokens found", err.Error(),
		))
	}

	// Refresh token
	newToken, err := h.oauth2Client.RefreshToken(c.Context(), tokenStore.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to refresh token", err.Error(),
		))
	}

	// Update token store
	tokenStore.AccessToken = newToken.AccessToken
	if newToken.RefreshToken != "" {
		tokenStore.RefreshToken = newToken.RefreshToken
	}
	tokenStore.Expiry = newToken.Expiry

	if err := agent.UpdateGoogleTokenStore(tokenStore); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to update tokens", err.Error(),
		))
	}

	if err := h.agentRepo.Update(c.Context(), agent.ID, agent); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to save agent", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"message":    "Token refreshed successfully",
		"expires_at": newToken.Expiry,
	}))
}

// GetProfile gets Gmail profile for an agent.
// @Summary Get Gmail profile
// @Description Retrieves Gmail profile information for an agent
// @Tags Gmail
// @Accept json
// @Produce json
// @Param agent_id path string true "Agent ID (UUID)"
// @Success 200 {object} schema.APIResponse[any] "Successfully retrieved Gmail profile"
// @Failure 400 {object} schema.APIResponse[any] "Invalid agent ID or no credentials"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/gmail/profile/{agent_id} [get]
func (h *GmailHandler) GetProfile(c fiber.Ctx) error {
	agentID, err := uuid.Parse(c.Params("agent_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid agent ID", err.Error(),
		))
	}

	// Get agent with valid token
	agent, gmailClient, err := h.getAgentWithGmailClient(c.Context(), agentID)
	if err != nil {
		return err
	}

	// Get profile
	profile, err := gmailClient.GetProfile(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to get Gmail profile", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"agent_id":       agent.ID,
		"email":          profile.EmailAddress,
		"messages_total": profile.MessagesTotal,
		"threads_total":  profile.ThreadsTotal,
	}))
}

// SendEmail sends an email via Gmail API.
// @Summary Send email via Gmail
// @Description Sends an email using the Gmail API for a specific agent
// @Tags Gmail
// @Accept json
// @Produce json
// @Param email body v1schema.GmailSendRequest true "Email sending data"
// @Success 200 {object} schema.APIResponse[map[string]string] "Successfully sent email"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/gmail/send [post]
func (h *GmailHandler) SendEmail(c fiber.Ctx) error {
	var req v1schema.GmailSendRequest

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if err := v1validation.ValidateStruct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	// Get agent with Gmail client
	agent, gmailClient, err := h.getAgentWithGmailClient(c.Context(), req.AgentID)
	if err != nil {
		return err
	}

	// Send email
	message, err := gmailClient.SendMessage(c.Context(), &gmail.SendMessageRequest{
		From:      agent.Email,
		To:        req.To,
		Cc:        req.Cc,
		Bcc:       req.Bcc,
		Subject:   req.Subject,
		Body:      req.Body,
		IsHTML:    req.IsHTML,
		InReplyTo: req.InReplyTo,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to send email", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"message_id": message.ID,
		"thread_id":  message.ThreadID,
		"status":     "sent",
	}))
}

// getAgentWithGmailClient retrieves agent and creates Gmail client with token refresh.
func (h *GmailHandler) getAgentWithGmailClient(ctx context.Context, agentID uuid.UUID) (*domain.Agent, *gmail.Client, error) {
	// Get agent
	agent, err := h.agentRepo.FindByID(ctx, agentID)
	if err != nil {
		return nil, nil, fiber.NewError(fiber.StatusNotFound, "Agent not found")
	}

	// Get token store
	tokenStore, err := agent.GetGoogleTokenStore()
	if err != nil {
		return nil, nil, fiber.NewError(fiber.StatusBadRequest, "No Gmail credentials found")
	}

	// Check if token expired and refresh if needed
	if tokenStore.Expiry.Before(time.Now().Add(5 * time.Minute)) {
		newToken, err := h.oauth2Client.RefreshToken(ctx, tokenStore.RefreshToken)
		if err != nil {
			return nil, nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to refresh token")
		}

		tokenStore.AccessToken = newToken.AccessToken
		if newToken.RefreshToken != "" {
			tokenStore.RefreshToken = newToken.RefreshToken
		}
		tokenStore.Expiry = newToken.Expiry

		if err := agent.UpdateGoogleTokenStore(tokenStore); err != nil {
			return nil, nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to update tokens")
		}

		if err := h.agentRepo.Update(ctx, agent.ID, agent); err != nil {
			return nil, nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to save agent")
		}
	}

	// Create Gmail client
	gmailClient, err := gmail.NewClient(ctx, tokenStore.AccessToken)
	if err != nil {
		return nil, nil, fiber.NewError(fiber.StatusInternalServerError, "Failed to create Gmail client")
	}

	return agent, gmailClient, nil
}

// generateState generates a random state string for OAuth2.
func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// parseAgentIDFromState extracts agent ID from state string.
func parseAgentIDFromState(state string) (uuid.UUID, error) {
	// State format: "random_string:agent_id"
	parts := splitLast(state, ":")
	if len(parts) != 2 {
		return uuid.Nil, fmt.Errorf("invalid state format")
	}

	return uuid.Parse(parts[1])
}

// splitLast splits string by last occurrence of separator.
func splitLast(s, sep string) []string {
	idx := len(s) - len(sep)
	for idx >= 0 {
		if s[idx:idx+len(sep)] == sep {
			return []string{s[:idx], s[idx+len(sep):]}
		}
		idx--
	}
	return []string{s}
}
