package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/pkg/metrics"
)

// Prometheus returns a middleware that collects HTTP metrics
func Prometheus() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Skip metrics for health check and metrics endpoints
		if c.Path() == "/health" || c.Path() == "/metrics" || c.Path() == "/ping" || c.Path() == "/healthz" {
			return c.Next()
		}

		start := time.Now()
		// Use Path to get the request path
		// Create a copy of the path to avoid potential string corruption
		pathStr := c.Path()
		path := string([]byte(pathStr))
		method := c.Method()

		// Increment active connections
		metrics.IncrementActiveConnections()
		defer metrics.DecrementActiveConnections()

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)
		statusCode := c.Response().StatusCode()

		// Record HTTP metrics
		metrics.ObserveHTTPRequest(
			method,
			path,
			strconv.Itoa(statusCode),
			duration,
			int64(len(c.Request().Body())),
			int64(len(c.Response().Body())),
		)

		return err
	}
}
