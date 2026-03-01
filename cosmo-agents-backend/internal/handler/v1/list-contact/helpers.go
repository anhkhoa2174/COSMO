package listcontact

import (
	"strconv"

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

// parsePagination extracts pagination parameters from query string
func parsePagination(c fiber.Ctx, defaultOffset, defaultLimit int) (int, int) {
	offset := defaultOffset
	limit := defaultLimit

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil {
			offset = val
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil {
			limit = val
		}
	}

	if limit <= 0 {
		limit = defaultLimit
	}
	if offset < 0 {
		offset = defaultOffset
	}
	return offset, limit
}

// Error response helpers
func badRequest(c fiber.Ctx, message string, err error) error {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(fiber.StatusBadRequest, message, errMsg))
}

func internalError(c fiber.Ctx, message string, err error) error {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(fiber.StatusInternalServerError, message, errMsg))
}

func unauthorizedResponse(c fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, "User not authenticated", ""))
}

func unauthorized(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, message, ""))
}

func notFound(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(fiber.StatusNotFound, message, ""))
}

func forbidden(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(fiber.StatusForbidden, message, ""))
}
