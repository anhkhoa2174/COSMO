package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

// RequestID middleware adds a unique request ID to each request
func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Check if request ID already exists in header
		requestID := c.Get(RequestIDHeader)

		// Generate new UUID if not provided
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set request ID in response header
		c.Set(RequestIDHeader, requestID)

		// Store in context for use in handlers/logging
		c.Locals("requestID", requestID)

		return c.Next()
	}
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(c fiber.Ctx) string {
	if requestID, ok := c.Locals("requestID").(string); ok {
		return requestID
	}
	return ""
}
