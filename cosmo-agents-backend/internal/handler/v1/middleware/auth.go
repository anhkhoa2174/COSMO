package middleware

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// Context keys for storing user information
type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	RequestIDKey contextKey = "request_id"
	RoleKey      contextKey = "role"
)

// GetUserID retrieves the user ID from the Fiber context.
func GetUserID(c fiber.Ctx) (uuid.UUID, bool) {
	userID, exists := c.Locals("user_id").(uuid.UUID)
	return userID, exists
}

// GetRequestID retrieves the request ID from the Fiber context.
func GetRequestID(c fiber.Ctx) string {
	requestID, exists := c.Locals("request_id").(string)
	if !exists {
		return ""
	}
	return requestID
}

// GetUserRole retrieves the user role from the Fiber context.
func GetUserRole(c fiber.Ctx) (string, bool) {
	role, exists := c.Locals("user_role").(string)
	return role, exists
}

// IsAdmin checks if the user has admin role from Fiber context.
func IsAdmin(c fiber.Ctx) bool {
	role, exists := GetUserRole(c)
	return exists && role == "admin"
}

// IsAdminFromContext checks if the user has admin role from context (for backward compatibility).
func IsAdminFromContext(ctx context.Context) bool {
	role, exists := ctx.Value(RoleKey).(string)
	return exists && role == "admin"
}

// RequireAdmin returns a middleware that requires admin role.
func RequireAdmin() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Get user info from locals or context
		userRole, exists := c.Locals("user_role").(string)
		if !exists || userRole != "admin" {
			logger.FromContext(context.Background()).Warn().
				Msg("Access denied: user is not admin")
			return fiber.ErrForbidden
		}
		return c.Next()
	}
}

// SetUserContext sets user information in the Fiber context.
func SetUserContext(c fiber.Ctx, userID uuid.UUID, role string, requestID string) {
	c.Locals("user_id", userID)
	c.Locals("user_role", role)
	c.Locals("request_id", requestID)
}
