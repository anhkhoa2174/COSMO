package customfield

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// Helper functions for common response patterns

func unauthorized(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
		fiber.StatusUnauthorized, message, "",
	))
}

func badRequest(c fiber.Ctx, message string, err error) error {
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
		fiber.StatusBadRequest, message, detail,
	))
}

func notFound(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
		fiber.StatusNotFound, message, "",
	))
}

func forbidden(c fiber.Ctx, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
		fiber.StatusForbidden, message, "",
	))
}

func internalError(c fiber.Ctx, message string, err error) error {
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
		fiber.StatusInternalServerError, message, detail,
	))
}

func parsePagination(c fiber.Ctx, defaultOffset, defaultLimit int) (int, int) {
	offset := defaultOffset
	limit := defaultLimit

	if offsetParam := c.Query("offset"); offsetParam != "" {
		if o, err := parseIntParam(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}

	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := parseIntParam(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	return offset, limit
}

func parseIntParam(param string) (int, error) {
	var result int
	_, err := fmt.Sscanf(param, "%d", &result)
	return result, err
}

func parseUUIDParam(c fiber.Ctx, paramName string) (uuid.UUID, error) {
	idStr := c.Params(paramName)
	return uuid.Parse(idStr)
}
