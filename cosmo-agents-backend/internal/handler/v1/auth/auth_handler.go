package auth

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/relations"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	helperRepo "github.com/rockship/cosmo-agents-go/internal/repository/helper"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	cozeService "github.com/rockship/cosmo-agents-go/internal/service/coze"
	googleService "github.com/rockship/cosmo-agents-go/internal/service/google"
	"github.com/rockship/cosmo-agents-go/pkg/auth"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"gorm.io/gorm"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	userRepo          *user.UserRepository
	userRelationsRepo *helperRepo.UserWithRelationsRepository
	jwtManager        *auth.JWTManager
	googleAuthService *googleService.GoogleAuthService
	cozeService       *cozeService.CozeService
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(
	userRepo *user.UserRepository,
	jwtManager *auth.JWTManager,
	googleAuthService *googleService.GoogleAuthService,
	db *gorm.DB,
) *AuthHandler {
	// Initialize CozeService once to avoid per-request I/O
	cozeSvc, err := cozeService.NewCozeService()
	if err != nil {
		logger.Logger.Warn().Err(err).Msg("Coze service not available - /v1/auth/coze/token will return 503")
	}

	return &AuthHandler{
		userRepo:          userRepo,
		userRelationsRepo: helperRepo.NewUserWithRelationsRepository(db),
		jwtManager:        jwtManager,
		googleAuthService: googleAuthService,
		cozeService:       cozeSvc,
	}
}

// LoginRequest represents login request
type LoginRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// LoginResponse represents login response with JWT token
type LoginResponse struct {
	Token     string                `json:"token"`
	User      v1schema.UserResponse `json:"user"`
	ExpiresIn int                   `json:"expires_in"` // seconds
}

// RefreshTokenRequest represents refresh token request
type RefreshTokenRequest struct {
	Token string `json:"token" validate:"required"`
}

// CozeTokenRequest represents Coze token request payload
type CozeTokenRequest struct {
	SessionName     *string `json:"session_name,omitempty" validate:"omitempty,max=255,alphanumunicode"`
	DurationSeconds *int    `json:"duration_seconds,omitempty" validate:"omitempty,min=1,max=86399"`
}

// Login handles user login and returns JWT token
// @Summary User login
// @Description Authenticate user with email and return JWT token
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} schema.APIResponse[LoginResponse] "Login successful"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Invalid credentials"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Router /v1/auth/login [post]
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req LoginRequest

	// Validate request
	if err := v1validation.ValidateRequest(c, &req); err != nil {
		return err
	}

	// Find user by email
	user, err := h.userRepo.FindByEmail(c.Context(), req.Email)
	if err != nil || user == nil {
		logger.Logger.Warn().
			Str("email", req.Email).
			Msg("Login attempt for non-existent user")

		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Invalid credentials",
			"User not found or credentials incorrect",
		))
	}

	// Get user's organization ID (if any)
	var orgID uuid.UUID
	org, err := h.userRepo.FindFirstOrganizationOfUser(c.Context(), user.ID)
	if err == nil && org != nil {
		orgID = org.ID
	}

	// Generate JWT token
	token, err := h.jwtManager.GenerateToken(user.ID, user.Email, orgID)
	if err != nil {
		logger.Error(err).Msg("Failed to generate token")
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to generate token",
			err.Error(),
		))
	}

	logger.Logger.Info().
		Str("user_id", user.ID.String()).
		Str("email", user.Email).
		Msg("User logged in successfully")

	// Load user with relations for response
	userResponse, err := h.getUserWithRelationsResponse(c.Context(), user.ID)
	if err != nil {
		logger.Logger.Warn().Err(err).Msg("Failed to load user relations, using basic user")
		userResponse = v1schema.ToUserResponse(user)
	}

	response := LoginResponse{
		Token:     token,
		User:      userResponse,
		ExpiresIn: h.jwtManager.GetExpirationSeconds(),
	}

	return c.JSON(schema.SuccessResponse(response))
}

// RefreshToken refreshes an existing JWT token
// @Summary Refresh JWT token
// @Description Refresh an existing JWT token to extend session
// @Tags Auth
// @Accept json
// @Produce json
// @Param refresh_token query string false "Refresh token"
// @Param request body RefreshTokenRequest false "Refresh token request (legacy)"
// @Success 200 {object} schema.APIResponse[any] "Token refreshed successfully"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Invalid or expired token"
// @Router /v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c fiber.Ctx) error {
	// Prefer Python-compatible query param first
	refreshToken := c.Query("refresh_token")

	if refreshToken == "" {
		// Fallback to legacy JSON body { token: string }
		var req RefreshTokenRequest
		if err := c.Bind().JSON(&req); err != nil || req.Token == "" {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest,
				"Missing refresh token",
				"Provide refresh_token as query or token in body",
			))
		}
		refreshToken = req.Token
	}

	if h.googleAuthService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			fiber.StatusServiceUnavailable,
			"OAuth client not configured",
			"Google OAuth credentials are missing",
		))
	}

	_, tokens, err := h.googleAuthService.RefreshAccessToken(c.Context(), refreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Invalid or expired token",
			err.Error(),
		))
	}

	response := fiber.Map{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// Authorize generates a Google OAuth authorization URL
// @Summary Generate Google OAuth authorization URL
// @Tags Auth
// @Produce json
// @Param redirect_url query string false "Optional redirect URI override"
// @Success 200 {object} schema.APIResponse[map[string]string]
// @Failure 503 {object} schema.APIResponse[any]
// @Router /v1/auth [get]
func (h *AuthHandler) Authorize(c fiber.Ctx) error {
	if h.googleAuthService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			fiber.StatusServiceUnavailable,
			"OAuth client not configured",
			"Google OAuth credentials are missing",
		))
	}

	redirectURI := c.Query("redirect_url")
	authURL, err := h.googleAuthService.GetAuthorizationURL(c.Context(), redirectURI, "", "")
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			fiber.StatusServiceUnavailable,
			"OAuth configuration unavailable",
			err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"url": authURL,
	}))
}

// OAuth2Callback handles Google OAuth callback
// @Summary Handle Google OAuth callback
// @Tags Auth
// @Produce json
// @Param code query string true "Authorization code"
// @Param state query string false "State parameter"
// @Param redirect_uri query string false "Redirect URI used during authorization"
// @Success 200 {object} schema.APIResponse[any]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 503 {object} schema.APIResponse[any]
// @Router /v1/auth/oauth2callback [get]
func (h *AuthHandler) OAuth2Callback(c fiber.Ctx) error {
	if h.googleAuthService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			fiber.StatusServiceUnavailable,
			"OAuth client not configured",
			"Google OAuth credentials are missing",
		))
	}

	fullURL := string(c.Request().URI().FullURI())
	redirectURI := c.Query("redirect_uri")
	state := c.Query("state")

	user, tokens, err := h.googleAuthService.HandleOAuth2Callback(c.Context(), fullURL, state, redirectURI)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Failed to process OAuth callback",
			err.Error(),
		))
	}

	// Load user with relations for response
	userResponse, err := h.getUserWithRelationsResponse(c.Context(), user.ID)
	if err != nil {
		logger.Logger.Warn().Err(err).Msg("Failed to load user relations in OAuth callback, using basic user")
		userResponse = v1schema.ToUserResponse(user)
	}

	response := fiber.Map{
		"user":          userResponse,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// Me returns the current authenticated user
// @Summary Get current user
// @Description Get information about the currently authenticated user
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[v1schema.UserResponse] "User information"
// @Failure 401 {object} schema.APIResponse[any] "Not authenticated"
// @Failure 404 {object} schema.APIResponse[any] "User not found"
// @Router /v1/auth/me [get]
func (h *AuthHandler) Me(c fiber.Ctx) error {
	userID, ok := c.Locals("userID").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Not authenticated",
			"Please login first",
		))
	}

	// Find user
	user, err := h.userRepo.FindByID(c.Context(), userID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound,
			"User not found",
			"",
		))
	}

	return c.JSON(schema.SuccessResponse(v1schema.ToUserResponse(user)))
}

// Logout logs out the current user
// @Summary User logout
// @Description Logout current authenticated user (client should remove token)
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[any] "Logged out successfully"
// @Router /v1/auth/logout [post]
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	// In a JWT-based system, logout is typically handled client-side
	// by removing the token from storage
	// Server-side logout would require a token blacklist (Redis)

	logger.Logger.Info().
		Str("user_id", c.Locals("userID").(uuid.UUID).String()).
		Msg("User logged out")

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"message": "Logged out successfully",
	}))
}

// CozeToken exchanges app-signed JWT for a Coze access token
// @Summary Get Coze API token
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body CozeTokenRequest false "Coze token request"
// @Success 200 {object} schema.APIResponse[any]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 503 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/auth/coze/token [post]
func (h *AuthHandler) CozeToken(c fiber.Ctx) error {
	var req CozeTokenRequest
	if err := v1validation.ValidateRequest(c, &req); err != nil {
		return err
	}

	if h.cozeService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			fiber.StatusServiceUnavailable,
			"Coze not configured",
			"Missing COZE_* configuration",
		))
	}

	// Validate duration (1..86399)
	duration := 86399
	if req.DurationSeconds != nil {
		if *req.DurationSeconds < 1 || *req.DurationSeconds > 86399 {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest,
				"Invalid duration",
				"duration_seconds must be between 1 and 86399",
			))
		}
		duration = *req.DurationSeconds
	}

	tokenResp, err := h.cozeService.GetAccessToken(c.Context(), req.SessionName, duration)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Failed to get Coze token",
			"Unable to generate Coze access token",
		))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"access_token": tokenResp.AccessToken,
		"expires_in":   tokenResp.ExpiresIn,
	}))
}

// getUserWithRelationsResponse loads user with relations and converts to response
func (h *AuthHandler) getUserWithRelationsResponse(ctx context.Context, userID uuid.UUID) (v1schema.UserResponse, error) {
	userWithRelations, err := h.userRelationsRepo.GetUserWithFullRelations(ctx, userID)
	if err != nil {
		return v1schema.UserResponse{}, err
	}

	if userWithRelations == nil {
		// Fallback to basic user if no relations found
		user, err := h.userRepo.GetDetailByID(ctx, userID)
		if err != nil || user == nil {
			return v1schema.UserResponse{}, err
		}
		return v1schema.ToUserResponse(user), nil
	}

	// Convert UserWithFullRelations to UserResponse with relations data
	response := h.convertUserWithRelationsToResponse(userWithRelations)
	return response, nil
}

// convertUserWithRelationsToResponse converts UserWithFullRelations to UserResponse with relations data
func (h *AuthHandler) convertUserWithRelationsToResponse(userWithFullRelations *relations.UserWithFullRelations) v1schema.UserResponse {
	// Convert the base user
	response := v1schema.ToUserResponse(&userWithFullRelations.User)

	// Convert organizations
	orgs := make([]v1schema.OrganizationResponse, len(userWithFullRelations.Organizations))
	for i, org := range userWithFullRelations.Organizations {
		orgResponse := v1schema.ToOrganizationResponse(&org)
		orgs[i] = *orgResponse
	}
	response.Organizations = orgs

	// Convert roles
	roles := make([]v1schema.RoleResponse, len(userWithFullRelations.Roles))
	for i, role := range userWithFullRelations.Roles {
		roles[i] = v1schema.RoleResponse{
			ID:             role.ID,
			UserID:         role.UserID,
			OrganizationID: role.OrganizationID,
			Name:           string(role.Name),
			JobTitle:       role.JobTitle,
			Status:         string(role.Status),
			CreatedAt:      role.CreatedAt,
			UpdatedAt:      role.UpdatedAt,
		}
	}
	response.Roles = roles

	// Convert notifications
	notifications := make([]v1schema.NotificationResponse, len(userWithFullRelations.Notifications))
	for i, notification := range userWithFullRelations.Notifications {
		notifications[i] = v1schema.NotificationResponse{
			ID:         notification.ID,
			UserID:     notification.UserID,
			CampaignID: notification.CampaignID,
			CreatedAt:  notification.CreatedAt,
			UpdatedAt:  notification.UpdatedAt,
		}
	}
	response.Notifications = notifications

	return response
}
