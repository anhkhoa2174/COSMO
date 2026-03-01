package auth

import (
	"net/url"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	googleService "github.com/rockship/cosmo-agents-go/internal/service/google"
)

// AuthHandler handles V2 authentication requests
type AuthHandler struct {
	authService *googleService.GoogleAuthService
	userRepo    *userRepo.UserRepository
}

// NewAuthHandler creates a new V2 AuthHandler
func NewAuthHandler(
	userRepo *userRepo.UserRepository,
	authService *googleService.GoogleAuthService,
	// AdapterGoogle godoc
) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		userRepo:    userRepo,
	}
}

// @Summary Get Google OAuth authorization URL
// @Description Returns Google OAuth authorization URL for user login
// @Tags Authentication
// @Accept json
// @Produce json
// @Param redirect_uri query string false "The URI to redirect after authentication"
// @Success 200 {object} map[string]interface{} "Authorization URL"
// @Failure 503 {object} map[string]interface{} "Service unavailable"
// @Router /v2/auth/adapter/google [get]
func (h *AuthHandler) AdapterGoogle(c fiber.Ctx) error {
	if h.authService == nil {
		return h.serviceUnavailable(c)
	}
	redirectURI := c.Query("redirect_uri")
	url, err := h.authService.GetAuthorizationURL(c.Context(), redirectURI, "", "")
	if err != nil {
		return h.serviceUnavailable(c)
	}
	return c.JSON(fiber.Map{"status": "success", "data": fiber.Map{"url": url}})
}

// MembersAuthURL godoc
// @Summary Get member invite authorization URL
// @Description Returns Google OAuth authorization URL for member invitation flow
// @Tags Authentication
// @Accept json
// @Produce json
// @Param redirect_uri query string false "The URI to redirect after authentication"
// @Param state query string false "State parameter for invitation"
// @Success 200 {object} map[string]interface{} "Authorization URL"
// @Failure 503 {object} map[string]interface{} "Service unavailable"
// @Router /v2/auth/members [get]
func (h *AuthHandler) MembersAuthURL(c fiber.Ctx) error {
	if h.authService == nil {
		return h.serviceUnavailable(c)
	}
	redirectURI := c.Query("redirect_uri")
	state := c.Query("state")
	url, err := h.authService.GetAuthorizationURL(c.Context(), redirectURI, state, "")
	if err != nil {
		return h.serviceUnavailable(c)
	}
	return c.JSON(fiber.Map{"status": "success", "data": fiber.Map{"url": url}})
}

// MembersOAuth2Callback godoc
// @Summary Complete member OAuth2 flow
// @Description Completes the OAuth2 flow for invited members
// @Tags Authentication
// @Accept json
// @Produce json
// @Param authorization_response query string false "Full authorization response URL containing code and state"
// @Param code query string false "Authorization code from Google"
// @Param state query string false "State parameter from invitation"
// @Param redirect_uri query string false "Redirect URI"
// @Success 200 {object} map[string]interface{} "Authentication tokens"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 503 {object} map[string]interface{} "Service unavailable"
// @Router /v2/auth/members/oauth2callback [get]
func (h *AuthHandler) MembersOAuth2Callback(c fiber.Ctx) error {
	if h.authService == nil {
		return h.serviceUnavailable(c)
	}
	authorizationResponse := coalesceAuthorizationResponse(c)
	redirectURI := c.Query("redirect_uri")
	state := c.Query("state")
	if state == "" {
		state = extractQueryParamValue(authorizationResponse, "state")
	}

	_, tokens, err := h.authService.HandleMemberOAuth2Callback(c.Context(), authorizationResponse, state, redirectURI)
	if err != nil {
		return badRequest(c, err.Error())
	}

	return c.JSON(fiber.Map{"status": "success", "data": tokens})
}

// MembersInviteCallback godoc
// @Summary Process member invitation callback
// @Description Processes member invitation and either returns auth URL or tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param state query string true "State parameter from invitation"
// @Param redirect_uri query string false "Redirect URI"
// @Success 200 {object} map[string]interface{} "Authorization URL or tokens"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 503 {object} map[string]interface{} "Service unavailable"
// @Router /v2/auth/members/invite-callback [get]
func (h *AuthHandler) MembersInviteCallback(c fiber.Ctx) error {
	if h.authService == nil {
		return h.serviceUnavailable(c)
	}
	redirectURI := c.Query("redirect_uri")
	state := c.Query("state")

	result, err := h.authService.HandleInviteMemberCallback(c.Context(), state, redirectURI)
	if err != nil {
		return badRequest(c, err.Error())
	}

	if result.AuthorizationURL != "" {
		return c.JSON(fiber.Map{"status": "success", "data": result.AuthorizationURL})
	}
	if result.Tokens != nil {
		return c.JSON(fiber.Map{"status": "success", "data": result.Tokens})
	}

	return c.JSON(fiber.Map{"status": "success", "data": nil})
}

// OAuth2Callback godoc
// @Summary Exchange authorization code for tokens
// @Description Exchanges Google authorization code for access, refresh, and ID tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param authorization_response query string false "Full authorization response URL containing code and state"
// @Param code query string false "Authorization code from Google"
// @Param state query string false "State parameter from authorization request"
// @Param redirect_uri query string false "Redirect URI used in authorization request"
// @Success 200 {object} map[string]interface{} "Authentication tokens"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 503 {object} map[string]interface{} "Service unavailable"
// @Router /v2/auth/oauth2callback [get]
func (h *AuthHandler) OAuth2Callback(c fiber.Ctx) error {
	if h.authService == nil {
		return h.serviceUnavailable(c)
	}
	authorizationResponse := coalesceAuthorizationResponse(c)
	redirectURI := c.Query("redirect_uri")
	state := c.Query("state")
	if state == "" {
		state = extractQueryParamValue(authorizationResponse, "state")
	}

	_, tokens, err := h.authService.HandleOAuth2Callback(c.Context(), authorizationResponse, state, redirectURI)
	if err != nil {
		return badRequest(c, err.Error())
	}

	return c.JSON(fiber.Map{"status": "success", "data": tokens})
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Refresh access token using refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param refresh_token query string true "Refresh token"
// @Success 200 {object} map[string]interface{} "New tokens"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Invalid refresh token"
// @Failure 503 {object} map[string]interface{} "Service unavailable"
// @Router /v2/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c fiber.Ctx) error {
	if h.authService == nil {
		return h.serviceUnavailable(c)
	}
	refreshToken := c.Query("refresh_token")
	if refreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Missing refresh_token parameter",
		})
	} else if refreshToken == "undefined" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid refresh_token parameter",
		})
	}

	_, tokens, err := h.authService.RefreshAccessToken(c.Context(), refreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid refresh token",
		})
	}

	return c.JSON(fiber.Map{"status": "success", "data": tokens})
}

func coalesceAuthorizationResponse(c fiber.Ctx) string {
	if raw := c.Query("authorization_response"); raw != "" {
		return raw
	}
	if raw := c.Query("url"); raw != "" {
		return raw
	}
	return string(c.Request().URI().FullURI())
}

func extractQueryParamValue(rawURL, key string) string {
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Query().Get(key)
}

// SignOut godoc
// @Summary Sign out user
// @Description Sign out the currently authenticated user
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Successfully signed out"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 503 {object} map[string]interface{} "Service unavailable"
// @Router /v2/auth/signout [post]
func (h *AuthHandler) SignOut(c fiber.Ctx) error {
	if h.authService == nil {
		return h.serviceUnavailable(c)
	}
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	user, err := h.userRepo.FindByID(c.Context(), userID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	credentials := map[string]interface{}{}
	if err := user.Credentials.Unmarshal(&credentials); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid stored credentials",
		})
	}

	if err := h.authService.Signout(c.Context(), credentials); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"status": "success", "data": nil})
}

func (h *AuthHandler) serviceUnavailable(c fiber.Ctx) error {
	return c.Status(fiber.StatusServiceUnavailable).JSON(schema.ErrorResponse(
		fiber.StatusServiceUnavailable,
		"OAuth client not configured",
		"Google OAuth credentials are missing",
	))
}

func badRequest(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
		fiber.StatusBadRequest,
		"Failed to process request",
		message,
	))
}
