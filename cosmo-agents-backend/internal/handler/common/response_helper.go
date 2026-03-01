package common

import (
	"github.com/gofiber/fiber/v3"
)

// APIResponse represents a standard API response structure
type APIResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo represents error details in API responses
type ErrorInfo struct {
	Code    string      `json:"code,omitempty"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Status     string         `json:"status"`
	Data       interface{}    `json:"data"`
	Pagination PaginationInfo `json:"pagination"`
}

// ResponseHelper provides common response formatting utilities
type ResponseHelper struct{}

// NewResponseHelper creates a new response helper
func NewResponseHelper() *ResponseHelper {
	return &ResponseHelper{}
}

// Success returns a successful response
func (h *ResponseHelper) Success(c fiber.Ctx, data interface{}) error {
	return c.JSON(APIResponse{
		Status: "success",
		Data:   data,
	})
}

// SuccessWithMessage returns a successful response with message
func (h *ResponseHelper) SuccessWithMessage(c fiber.Ctx, data interface{}, message string) error {
	return c.JSON(APIResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// Error returns an error response
func (h *ResponseHelper) Error(c fiber.Ctx, statusCode int, message string, details interface{}) error {
	return c.Status(statusCode).JSON(APIResponse{
		Status:  "error",
		Message: message,
		Error: &ErrorInfo{
			Message: message,
			Details: details,
		},
	})
}

// ErrorWithCode returns an error response with error code
func (h *ResponseHelper) ErrorWithCode(c fiber.Ctx, statusCode int, code, message string, details interface{}) error {
	return c.Status(statusCode).JSON(APIResponse{
		Status:  "error",
		Message: message,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// BadRequest returns a 400 error response
func (h *ResponseHelper) BadRequest(c fiber.Ctx, message string, details interface{}) error {
	return h.ErrorWithCode(c, fiber.StatusBadRequest, "BAD_REQUEST", message, details)
}

// Unauthorized returns a 401 error response
func (h *ResponseHelper) Unauthorized(c fiber.Ctx, message string, details interface{}) error {
	return h.ErrorWithCode(c, fiber.StatusUnauthorized, "UNAUTHORIZED", message, details)
}

// Forbidden returns a 403 error response
func (h *ResponseHelper) Forbidden(c fiber.Ctx, message string, details interface{}) error {
	return h.ErrorWithCode(c, fiber.StatusForbidden, "FORBIDDEN", message, details)
}

// NotFound returns a 404 error response
func (h *ResponseHelper) NotFound(c fiber.Ctx, message string, details interface{}) error {
	return h.ErrorWithCode(c, fiber.StatusNotFound, "NOT_FOUND", message, details)
}

// InternalServerError returns a 500 error response
func (h *ResponseHelper) InternalServerError(c fiber.Ctx, message string, details interface{}) error {
	return h.ErrorWithCode(c, fiber.StatusInternalServerError, "INTERNAL_SERVER_ERROR", message, details)
}

// ValidationFailed returns a 422 error response
func (h *ResponseHelper) ValidationFailed(c fiber.Ctx, message string, details interface{}) error {
	return h.ErrorWithCode(c, fiber.StatusUnprocessableEntity, "VALIDATION_FAILED", message, details)
}

// Paginated returns a paginated response
func (h *ResponseHelper) Paginated(c fiber.Ctx, data interface{}, page, pageSize int, total int64) error {
	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return c.JSON(PaginatedResponse{
		Status: "success",
		Data:   data,
		Pagination: PaginationInfo{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// LegacyMapResponse converts APIResponse to legacy fiber.Map format for backward compatibility
func (h *ResponseHelper) LegacyMapResponse(status string, data interface{}, message string) fiber.Map {
	response := fiber.Map{
		"status": status,
	}

	if data != nil {
		response["data"] = data
	}

	if message != "" {
		response["message"] = message
	}

	return response
}
