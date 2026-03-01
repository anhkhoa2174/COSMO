package campaign

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// userIDFromContext extracts user ID from fiber context
func userIDFromContext(c fiber.Ctx) (uuid.UUID, bool) {
	userID := c.Locals("user_id")
	if userID == nil {
		return uuid.Nil, false
	}

	// Handle both string and UUID types
	switch v := userID.(type) {
	case uuid.UUID:
		return v, true
	case string:
		uid, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, false
		}
		return uid, true
	default:
		return uuid.Nil, false
	}
}

// badRequest returns a 400 Bad Request error response
func badRequest(c fiber.Ctx, message string, detail string) error {
	return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
		fiber.StatusBadRequest, message, detail,
	))
}

// notFound returns a 404 Not Found error response
func notFound(c fiber.Ctx, message string, detail string) error {
	return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
		fiber.StatusNotFound, message, detail,
	))
}

// unauthorized returns a 401 Unauthorized error response
func unauthorized(c fiber.Ctx, message string, detail string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
		fiber.StatusUnauthorized, message, detail,
	))
}

// internalError returns a 500 Internal Server Error response
func internalError(c fiber.Ctx, message string, detail string) error {
	return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
		fiber.StatusInternalServerError, message, detail,
	))
}
