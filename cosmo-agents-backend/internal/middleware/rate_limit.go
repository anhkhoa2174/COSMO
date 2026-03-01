package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Max        int           // Maximum number of requests
	Expiration time.Duration // Time window
	ByIP       bool          // Limit by IP address
	ByUser     bool          // Limit by authenticated user
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Max:        100,             // 100 requests
		Expiration: 1 * time.Minute, // per minute
		ByIP:       true,
		ByUser:     false,
	}
}

// StrictRateLimitConfig for sensitive endpoints (auth, etc)
func StrictRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Max:        10,              // 10 requests
		Expiration: 1 * time.Minute, // per minute
		ByIP:       true,
		ByUser:     false,
	}
}

// RateLimit creates a rate limiting middleware
func RateLimit(config RateLimitConfig) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        config.Max,
		Expiration: config.Expiration,
		KeyGenerator: func(c fiber.Ctx) string {
			if config.ByUser {
				// Try to get user ID from context (if authenticated)
				if userID, ok := GetUserID(c); ok {
					return userID.String()
				}
			}

			// Default to IP-based limiting
			if config.ByIP {
				return c.IP()
			}

			// Fallback to request ID
			return GetRequestID(c)
		},
		LimitReached: func(c fiber.Ctx) error {
			logger.Logger.Warn().
				Str("ip", c.IP()).
				Str("path", c.Path()).
				Str("request_id", GetRequestID(c)).
				Int("max", config.Max).
				Dur("window", config.Expiration).
				Msg("Rate limit exceeded")

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"status": "error",
				"error": fiber.Map{
					"code":    fiber.StatusTooManyRequests,
					"message": "Rate limit exceeded",
					"details": "Too many requests. Please try again later.",
				},
			})
		},
		SkipFailedRequests:     false,
		SkipSuccessfulRequests: false,
		LimiterMiddleware:      limiter.SlidingWindow{},
	})
}

// PerUserRateLimit creates a rate limiter based on authenticated user
func PerUserRateLimit(max int, window time.Duration) fiber.Handler {
	return RateLimit(RateLimitConfig{
		Max:        max,
		Expiration: window,
		ByIP:       false,
		ByUser:     true,
	})
}

// PerIPRateLimit creates a rate limiter based on IP address
func PerIPRateLimit(max int, window time.Duration) fiber.Handler {
	return RateLimit(RateLimitConfig{
		Max:        max,
		Expiration: window,
		ByIP:       true,
		ByUser:     false,
	})
}
