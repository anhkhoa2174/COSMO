package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

const (
	// AuthProviderHeader is the header used to signal which upstream auth provider
	// should be used when mirroring the original Python service contract.
	AuthProviderHeader = "X-Auth-Provider"
	// DefaultAuthProvider replicates the FastAPI behaviour which defaults to google.
	DefaultAuthProvider = "google"
)

// AuthProviderMiddleware stores the requested auth provider in the context locals so handlers
// can honour the FastAPI-style dependency signature. The Python service accepts the header but
// treats it as optional – mirroring that by defaulting to "google".
func AuthProviderMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		provider := strings.ToLower(c.Get(AuthProviderHeader))
		if provider == "" {
			provider = DefaultAuthProvider
		}
		c.Locals("auth_provider", provider)
		return c.Next()
	}
}
