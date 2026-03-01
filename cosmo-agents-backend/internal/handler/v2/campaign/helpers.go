package campaign

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// parsePagination extracts pagination parameters from query string
func parsePagination(c fiber.Ctx, defaultOffset, defaultLimit int) (int, int) {
	offset := defaultOffset
	limit := defaultLimit
	offsetProvided := false

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil {
			offset = val
		}
		offsetProvided = true
	}

	// Allow alternate limit keys from FE (page_size, pageSize, per_page)
	if limit == defaultLimit {
		if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
			if val, err := strconv.Atoi(pageSizeStr); err == nil && val > 0 {
				limit = val
			}
		} else if pageSizeStr := c.Query("pageSize"); pageSizeStr != "" {
			if val, err := strconv.Atoi(pageSizeStr); err == nil && val > 0 {
				limit = val
			}
		} else if perPageStr := c.Query("per_page"); perPageStr != "" {
			if val, err := strconv.Atoi(perPageStr); err == nil && val > 0 {
				limit = val
			}
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil {
			limit = val
		}
	}

	// Allow page-based pagination for clients that pass ?page=N (1-indexed) or ?page_index=N (0-indexed).
	pageParamProvided := false
	if offset == defaultOffset {
		pageIndexStr := c.Query("page_index")
		if pageIndexStr == "" {
			pageIndexStr = c.Query("pageIndex")
		}
		if pageIndexStr != "" {
			if val, err := strconv.Atoi(pageIndexStr); err == nil && val >= 0 {
				offset = val * limit
				pageParamProvided = true
			}
		} else if pageStr := c.Query("page"); pageStr != "" {
			if val, err := strconv.Atoi(pageStr); err == nil && val > 0 {
				offset = (val - 1) * limit
				pageParamProvided = true
			}
		}
	}

	if limit <= 0 {
		limit = defaultLimit
	}
	if offset < 0 {
		offset = defaultOffset
	}

	// Heuristic: some clients send offset as pageIndex while also sending limit.
	// If offset was explicitly provided, no page params, and offset looks like a page index (< limit),
	// treat it as pageIndex to avoid repeating page 1.
	if offsetProvided && !pageParamProvided && offset > 0 && limit > 0 && offset < limit {
		offset = offset * limit
	}

	return offset, limit
}

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

// Error response helpers
func badRequest(c fiber.Ctx, message string, err error) error {
	if err == nil {
		err = errors.New(message)
	}
	if c == nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("%s: %v", message, err))
	}
	return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(fiber.StatusBadRequest, message, err.Error()))
}

func internalError(c fiber.Ctx, message string, err error) error {
	if c == nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("%s: %v", message, err))
	}
	return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(fiber.StatusInternalServerError, message, err.Error()))
}

func notFound(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(fiber.StatusNotFound, message, ""))
}

func unauthorizedResponse(c fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, "unauthorized", ""))
}

func unauthorizedAccessResponse(c fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(fiber.StatusUnauthorized, "You are not authorized to access this resource", ""))
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
