package listcontact

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// Helper functions for extracting user ID from context
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

// Error response helpers
func badRequest(c fiber.Ctx, message string, err error) error {
	if err == nil {
		err = errors.New(message)
	}
	return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(fiber.StatusBadRequest, message, err.Error()))
}

func internalError(c fiber.Ctx, message string, err error) error {
	return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(fiber.StatusInternalServerError, message, err.Error()))
}

func unauthorizedResponse(c fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, "unauthorized", ""))
}

func notFound(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(fiber.StatusNotFound, message, ""))
}

func forbidden(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(fiber.StatusForbidden, message, ""))
}

func serviceUnavailable(c fiber.Ctx, message string, err error) error {
	if err == nil {
		err = errors.New(message)
	}
	return c.Status(fiber.StatusServiceUnavailable).JSON(schema.ErrorResponse(fiber.StatusServiceUnavailable, message, err.Error()))
}
