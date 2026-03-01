package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// ResponseHelper provides helper functions for consistent API responses
type ResponseHelper struct{}

// NewResponseHelper creates a new ResponseHelper
func NewResponseHelper() *ResponseHelper {
	return &ResponseHelper{}
}

// HandleAuthError returns appropriate error response for authentication errors
func (h *ResponseHelper) HandleAuthError(c fiber.Ctx, err error) error {
	switch err {
	case middleware.ErrUnauthorized, middleware.ErrNoRolesFound:
		return c.Status(fiber.StatusUnauthorized).JSON(
			schema.ErrorResponse(fiber.StatusUnauthorized, "Unauthorized", err.Error()),
		)
	case middleware.ErrUserNotFound:
		return c.Status(fiber.StatusNotFound).JSON(
			schema.ErrorResponse(fiber.StatusNotFound, "User not found", err.Error()),
		)
	case middleware.ErrInvalidUserID:
		return c.Status(fiber.StatusBadRequest).JSON(
			schema.ErrorResponse(fiber.StatusBadRequest, "Invalid user ID", err.Error()),
		)
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(
			schema.ErrorResponse(fiber.StatusInternalServerError, "Internal server error", err.Error()),
		)
	}
}

// BadRequest returns a 400 Bad Request response
func (h *ResponseHelper) BadRequest(c fiber.Ctx, message string, err error) error {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	return c.Status(fiber.StatusBadRequest).JSON(
		schema.ErrorResponse(fiber.StatusBadRequest, message, errMsg),
	)
}

// Unauthorized returns a 401 Unauthorized response
func (h *ResponseHelper) Unauthorized(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(
		schema.ErrorResponse(fiber.StatusUnauthorized, message, ""),
	)
}

// NotFound returns a 404 Not Found response
func (h *ResponseHelper) NotFound(c fiber.Ctx, message string, err error) error {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	return c.Status(fiber.StatusNotFound).JSON(
		schema.ErrorResponse(fiber.StatusNotFound, message, errMsg),
	)
}

// InternalServerError returns a 500 Internal Server Error response
func (h *ResponseHelper) InternalServerError(c fiber.Ctx, message string, err error) error {
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	return c.Status(fiber.StatusInternalServerError).JSON(
		schema.ErrorResponse(fiber.StatusInternalServerError, message, errMsg),
	)
}

// Success returns a 200 OK response with data
func (h *ResponseHelper) Success(c fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(
		schema.SuccessResponse(data),
	)
}

// Created returns a 201 Created response with data
func (h *ResponseHelper) Created(c fiber.Ctx, data interface{}) error {
	return c.Status(fiber.StatusCreated).JSON(
		schema.SuccessResponse(data),
	)
}

// ValidationError returns a 400 Bad Request response for validation errors
func (h *ResponseHelper) ValidationError(c fiber.Ctx, fieldErrors map[string]string) error {
	return c.Status(fiber.StatusBadRequest).JSON(
		schema.ErrorResponse(fiber.StatusBadRequest, "Validation failed", fieldErrors),
	)
}
