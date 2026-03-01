package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	ExposeHeaders    []string
	MaxAge           int
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			fiber.MethodGet,
			fiber.MethodPost,
			fiber.MethodPut,
			fiber.MethodPatch,
			fiber.MethodDelete,
			fiber.MethodOptions,
			fiber.MethodHead,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			RequestIDHeader,
		},
		AllowCredentials: false,
		ExposeHeaders: []string{
			RequestIDHeader,
		},
		MaxAge: 3600, // 1 hour
	}
}

// CORS returns a configured CORS middleware
func CORS(config CORSConfig) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     config.AllowOrigins,
		AllowMethods:     config.AllowMethods,
		AllowHeaders:     config.AllowHeaders,
		AllowCredentials: config.AllowCredentials,
		ExposeHeaders:    config.ExposeHeaders,
		MaxAge:           config.MaxAge,
	})
}
