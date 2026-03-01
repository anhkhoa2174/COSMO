package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// Recovery middleware recovers from panics and logs them
func Recovery() fiber.Handler {
	return func(c fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				// Get stack trace
				stack := debug.Stack()

				// Log the panic
				logger.Logger.Error().
					Interface("panic", r).
					Str("stack", string(stack)).
					Str("path", c.Path()).
					Str("method", c.Method()).
					Str("request_id", GetRequestID(c)).
					Msg("Panic recovered")

				// Return 500 error
				err := c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"status": "error",
					"error": fiber.Map{
						"code":    fiber.StatusInternalServerError,
						"message": "Internal server error",
						"details": fmt.Sprintf("%v", r),
					},
				})

				if err != nil {
					logger.Error(err).Msg("Failed to send error response")
				}
			}
		}()

		return c.Next()
	}
}
