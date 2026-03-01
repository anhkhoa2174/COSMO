package response

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rs/zerolog"
)

// Response represents a standardized API response
type Response[T any] struct {
	Status  string     `json:"status"`
	Message string     `json:"message,omitempty"`
	Data    T          `json:"data,omitempty"`
	Error   *ErrorInfo `json:"error,omitempty"`
}

// ErrorInfo contains detailed error information
type ErrorInfo struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Helper provides standardized response methods
type Helper struct {
	logger zerolog.Logger
}

// NewHelper creates a new response helper
func NewHelper() *Helper {
	return &Helper{
		logger: logger.Logger,
	}
}

// Success returns a successful response
func (h *Helper) Success(c fiber.Ctx, data interface{}) error {
	return c.Status(http.StatusOK).JSON(Response[interface{}]{
		Status: "success",
		Data:   data,
	})
}

// SuccessWithMessage returns a successful response with message
func (h *Helper) SuccessWithMessage(c fiber.Ctx, data interface{}, message string) error {
	return c.Status(http.StatusOK).JSON(Response[interface{}]{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// Error returns an error response
func (h *Helper) Error(c fiber.Ctx, statusCode int, code, message string, details interface{}) error {
	h.logger.Error().Err(fmt.Errorf("API error: %s", message)).
		Str("code", code).
		Str("path", c.Path()).
		Str("method", c.Method()).
		Interface("details", details).
		Msg("API error occurred")

	return c.Status(statusCode).JSON(Response[interface{}]{
		Status:  "error",
		Message: message,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// BadRequest returns a 400 error
func (h *Helper) BadRequest(c fiber.Ctx, message string, details ...interface{}) error {
	detail := interface{}(0)
	if len(details) > 0 {
		detail = details[0]
	}
	return h.Error(c, http.StatusBadRequest, "BAD_REQUEST", message, detail)
}

// Unauthorized returns a 401 error
func (h *Helper) Unauthorized(c fiber.Ctx, message string, details ...interface{}) error {
	detail := interface{}(0)
	if len(details) > 0 {
		detail = details[0]
	}
	return h.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", message, detail)
}

// Forbidden returns a 403 error
func (h *Helper) Forbidden(c fiber.Ctx, message string, details ...interface{}) error {
	detail := interface{}(0)
	if len(details) > 0 {
		detail = details[0]
	}
	return h.Error(c, http.StatusForbidden, "FORBIDDEN", message, detail)
}

// NotFound returns a 404 error
func (h *Helper) NotFound(c fiber.Ctx, message string, details ...interface{}) error {
	detail := interface{}(0)
	if len(details) > 0 {
		detail = details[0]
	}
	return h.Error(c, http.StatusNotFound, "NOT_FOUND", message, detail)
}

// InternalServerError returns a 500 error
func (h *Helper) InternalServerError(c fiber.Ctx, err error, details ...interface{}) error {
	message := "Internal server error"
	if err != nil {
		message = err.Error()
	}

	detail := interface{}(0)
	if len(details) > 0 {
		detail = details[0]
	}

	h.logger.Error().Err(err).
		Str("path", c.Path()).
		Str("method", c.Method()).
		Interface("details", detail).
		Msg("Internal server error")

	return h.Error(c, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message, detail)
}

// ServiceUnavailable returns a 503 error
func (h *Helper) ServiceUnavailable(c fiber.Ctx, message string, details ...interface{}) error {
	detail := interface{}(0)
	if len(details) > 0 {
		detail = details[0]
	}
	return h.Error(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", message, detail)
}

// ValidationFailed returns a 422 error for validation issues
func (h *Helper) ValidationFailed(c fiber.Ctx, message string, details ...interface{}) error {
	detail := interface{}(0)
	if len(details) > 0 {
		detail = details[0]
	}
	return h.Error(c, http.StatusUnprocessableEntity, "VALIDATION_FAILED", message, detail)
}

// PagedResponse represents a paginated response
type PagedResponse[T any] struct {
	Status     string `json:"status"`
	Data       []T    `json:"data"`
	Page       int    `json:"page"`
	PerPage    int    `json:"per_page"`
	Total      int64  `json:"total"`
	TotalPages int    `json:"total_pages"`
}

// Paged returns a paginated response
func (h *Helper) Paged(c fiber.Ctx, data interface{}, page, perPage int, total int64) error {
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))

	// Convert data to []interface{} for JSON response
	var dataSlice []interface{}
	switch v := data.(type) {
	case []interface{}:
		dataSlice = v
	default:
		// For single items or other types, wrap in slice
		if v != nil {
			dataSlice = []interface{}{v}
		}
	}

	return c.Status(http.StatusOK).JSON(PagedResponse[interface{}]{
		Status:     "success",
		Data:       dataSlice,
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	})
}
