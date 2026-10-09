package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// Logger middleware logs HTTP requests with structured logging
func Logger() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Skip logging for health check endpoints to reduce spam
		if c.Path() == "/health" || c.Path() == "/ping" || c.Path() == "/healthz" {
			return c.Next()
		}

		// Start timer
		start := time.Now()

		// Get request ID
		requestID := GetRequestID(c)

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Log the request
		logEvent := logger.Logger.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Int("status", c.Response().StatusCode()).
			Dur("duration_ms", duration).
			Str("user_agent", c.Get("User-Agent"))

		if requestID != "" {
			logEvent = logEvent.Str("request_id", requestID)
		}

		if err != nil {
			logEvent = logEvent.Err(err)
		}

		logEvent.Msg("HTTP Request")

		return err
	}
}
