package middleware

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/repository/personal_api_key"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	"github.com/rockship/cosmo-agents-go/pkg/auth"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

var errUserNotFoundForAPIKey = errors.New("user not found for personal API key")

// AuthMiddleware creates an authentication middleware
func AuthMiddleware(jwtManager *auth.JWTManager, apiKeyRepo *personalapikey.PersonalApiKeyRepository, userRepo *userRepo.UserRepository) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Skip auth for specific public endpoints only
		path := c.Path()

		// List of truly public endpoints (login, oauth, etc.)
		publicPaths := []string{
			"/v1/auth/login",
			"/v1/auth/refresh",
			"/v1/auth/oauth2callback",
			"/v1/pubsub/",                    // All pubsub endpoints are public
			"/v2/google/gmail/notifications", // Pub/Sub push endpoint (validated upstream)
			"/v1/sse-test",                   // SSE test (temporary)
			"/v1/public/",                    // Unauthenticated pages, e.g. lead forms; each route limits itself
		}

		// Exact match for specific paths, or prefix match for wildcards
		for _, publicPath := range publicPaths {
			if strings.HasSuffix(publicPath, "/") {
				// Prefix match for paths ending with /
				if strings.HasPrefix(path, publicPath) {
					return c.Next()
				}
			} else {
				// Exact match for specific endpoints
				if path == publicPath {
					return c.Next()
				}
			}
		}

		// Special case: /v1/auth without any suffix (the authorize endpoint)
		if path == "/v1/auth" {
			return c.Next()
		}

		// Get Authorization header
		authHeader := c.Get("Authorization")
		// SSE EventSource API cannot set custom headers — accept token from query param
		if authHeader == "" {
			if queryToken := c.Query("token"); queryToken != "" {
				authHeader = "Bearer " + queryToken
			}
		}
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
				fiber.StatusUnauthorized,
				"Missing authorization header",
				"Please provide a valid Bearer token",
			))
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
				fiber.StatusUnauthorized,
				"Invalid authorization format",
				"Expected format: 'Bearer <token>'",
			))
		}

		tokenString := parts[1]

		// Prefer personal API key when prefix matches, otherwise fall back to JWT
		if strings.HasPrefix(tokenString, personalapikey.KEY_PREFIX) {
			if apiKeyRepo == nil {
				return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
					fiber.StatusUnauthorized,
					"Personal API key authentication not configured",
					"Missing personal API key repository",
				))
			}

			key, err := apiKeyRepo.ValidateRawKey(c.Context(), tokenString)
			if err != nil {
				status := fiber.StatusUnauthorized
				message := "Invalid or expired token"

				switch {
				case errors.Is(err, personalapikey.ErrExpiredPersonalAPIKey):
					message = "Personal API key expired"
				case errors.Is(err, personalapikey.ErrMissingPersonalAPIKeySecret):
					message = "Personal API key authentication not configured"
				case errors.Is(err, errUserNotFoundForAPIKey):
					message = "Invalid or expired token"
				default:
					status = fiber.StatusInternalServerError
					message = "Failed to validate personal API key"
				}

				logger.Logger.Warn().
					Err(err).
					Str("request_id", GetRequestID(c)).
					Msg("Invalid personal API key")

				return c.Status(status).JSON(schema.ErrorResponse(
					status,
					message,
					err.Error(),
				))
			}

			if key == nil {
				return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
					fiber.StatusUnauthorized,
					"Invalid or expired token",
					"Personal API key not found",
				))
			}

			var email string
			if userRepo != nil {
				user, err := userRepo.GetDetailByID(c.Context(), key.UserID)
				if err != nil {
					logger.Logger.Error().
						Err(err).
						Str("request_id", GetRequestID(c)).
						Msg("Failed to load user for personal API key")

					return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
						fiber.StatusInternalServerError,
						"Failed to validate personal API key",
						err.Error(),
					))
				}

				if user == nil {
					logger.Logger.Warn().
						Err(errUserNotFoundForAPIKey).
						Str("api_key_id", key.ID.String()).
						Str("request_id", GetRequestID(c)).
						Msg("User not found for personal API key")

					return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
						fiber.StatusUnauthorized,
						"Invalid or expired token",
						errUserNotFoundForAPIKey.Error(),
					))
				}

				email = strings.TrimSpace(user.Email)
			}

			if err := apiKeyRepo.UpdateLastUsed(c.Context(), key.ID); err != nil {
				logger.Logger.Warn().
					Err(err).
					Str("api_key_id", key.ID.String()).
					Str("request_id", GetRequestID(c)).
					Msg("Failed to update personal API key last_used_at")
			}

			// Store user info in context for use in handlers
			c.Locals("userID", key.UserID)
			c.Locals("user_id", key.UserID)
			if email != "" {
				c.Locals("email", email)
				c.Locals("user_email", email)
			}

			logger.Logger.Debug().
				Str("user_id", key.UserID.String()).
				Str("auth_type", "personal_api_key").
				Str("request_id", GetRequestID(c)).
				Msg("Authenticated request")

			return c.Next()
		}

		// Validate JWT token (fallback)
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			logger.Logger.Warn().
				Err(err).
				Str("request_id", GetRequestID(c)).
				Msg("Invalid token")

			return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
				fiber.StatusUnauthorized,
				"Invalid or expired token",
				err.Error(),
			))
		}

		// Store user info in context for use in handlers
		// Note: Many handlers expect "user_id" (snake_case) while others use "userID" (camelCase).
		// To maintain backward compatibility, set both keys.
		c.Locals("userID", claims.UserID)
		c.Locals("user_id", claims.UserID)
		c.Locals("email", claims.Email)
		c.Locals("organizationID", claims.OrganizationID)
		c.Locals("claims", claims)

		// Log authenticated request
		logger.Logger.Debug().
			Str("user_id", claims.UserID.String()).
			Str("email", claims.Email).
			Str("request_id", GetRequestID(c)).
			Msg("Authenticated request")

		return c.Next()
	}
}

// OptionalAuth middleware that allows both authenticated and unauthenticated requests
func OptionalAuth(jwtManager *auth.JWTManager, apiKeyRepo *personalapikey.PersonalApiKeyRepository, userRepo *userRepo.UserRepository) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			// No auth header, continue without authentication
			return c.Next()
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString := parts[1]

			if strings.HasPrefix(tokenString, personalapikey.KEY_PREFIX) && apiKeyRepo != nil {
				if key, err := apiKeyRepo.ValidateRawKey(c.Context(), tokenString); err == nil && key != nil {
					c.Locals("userID", key.UserID)
					c.Locals("user_id", key.UserID)

					if userRepo != nil {
						if user, err := userRepo.GetDetailByID(c.Context(), key.UserID); err == nil && user != nil {
							email := strings.TrimSpace(user.Email)
							if email != "" {
								c.Locals("email", email)
								c.Locals("user_email", email)
							}
						}
					}

					if err := apiKeyRepo.UpdateLastUsed(c.Context(), key.ID); err != nil {
						logger.Logger.Warn().
							Err(err).
							Str("api_key_id", key.ID.String()).
							Str("request_id", GetRequestID(c)).
							Msg("Failed to update personal API key last_used_at")
					}
					return c.Next()
				}
			}

			claims, err := jwtManager.ValidateToken(tokenString)
			if err == nil {
				// Valid token, store claims
				c.Locals("userID", claims.UserID)
				c.Locals("user_id", claims.UserID)
				c.Locals("email", claims.Email)
				c.Locals("organizationID", claims.OrganizationID)
				c.Locals("claims", claims)
			}
		}

		return c.Next()
	}
}

// GetUserID retrieves the authenticated user ID from context
func GetUserID(c fiber.Ctx) (uuid.UUID, bool) {
	userID, ok := c.Locals("userID").(uuid.UUID)
	return userID, ok
}

// GetEmail retrieves the authenticated user email from context
func GetEmail(c fiber.Ctx) (string, bool) {
	email, ok := c.Locals("email").(string)
	return email, ok
}

// GetOrganizationID retrieves the organization ID from context
func GetOrganizationID(c fiber.Ctx) (uuid.UUID, bool) {
	orgID, ok := c.Locals("organizationID").(uuid.UUID)
	return orgID, ok
}

// GetClaims retrieves all JWT claims from context
func GetClaims(c fiber.Ctx) (*auth.Claims, bool) {
	claims, ok := c.Locals("claims").(*auth.Claims)
	return claims, ok
}

// RequireOrganization middleware ensures the user belongs to a specific organization
func RequireOrganization() fiber.Handler {
	return func(c fiber.Ctx) error {
		orgID, ok := GetOrganizationID(c)
		if !ok || orgID == uuid.Nil {
			return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
				fiber.StatusForbidden,
				"Organization required",
				"This endpoint requires an organization context",
			))
		}

		return c.Next()
	}
}
