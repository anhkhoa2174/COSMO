package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	appMiddleware "github.com/rockship/cosmo-agents-go/internal/middleware"
	"github.com/rockship/cosmo-agents-go/pkg/auth"
	"github.com/rockship/cosmo-agents-go/pkg/config"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rockship/cosmo-agents-go/pkg/metrics"
)

// App holds the Fiber application and configuration
type App struct {
	Fiber      *fiber.App
	Config     *config.Config
	JWTManager *auth.JWTManager
}

type metricsWriter struct {
	c fiber.Ctx
}

func (w *metricsWriter) Header() http.Header {
	return make(http.Header)
}

func (w *metricsWriter) Write(data []byte) (int, error) {
	return w.c.Write(data)
}

func (w *metricsWriter) WriteHeader(statusCode int) {
	w.c.Status(statusCode)
}

// NewApp creates and configures a new Fiber application
func NewApp(cfg *config.Config) *App {
	// Initialize JWT manager with Python-compatible configuration
	accessSecret := cfg.Auth.AccessTokenSecret
	if accessSecret == "" {
		accessSecret = cfg.Auth.JWTSecret
	}
	refreshSecret := cfg.Auth.RefreshTokenSecret
	if refreshSecret == "" {
		refreshSecret = cfg.Auth.JWTSecret
	}

	accessDuration := time.Duration(cfg.Auth.AccessTokenExpireDays) * 24 * time.Hour
	if accessDuration <= 0 && cfg.Auth.JWTExpiration > 0 {
		accessDuration = time.Duration(cfg.Auth.JWTExpiration) * time.Hour
	}
	if accessDuration <= 0 {
		accessDuration = time.Hour * 24
	}

	refreshDuration := time.Duration(cfg.Auth.RefreshTokenExpireDays) * 24 * time.Hour
	if refreshDuration <= 0 && cfg.Auth.JWTRefreshExpiration > 0 {
		refreshDuration = time.Duration(cfg.Auth.JWTRefreshExpiration) * time.Hour
	}
	if refreshDuration <= 0 {
		refreshDuration = 7 * 24 * time.Hour
	}

	jwtManager := auth.NewJWTManager(accessSecret, accessDuration, refreshSecret, refreshDuration, cfg.Auth.JWTAlgorithm)
	if cfg.Auth.JWTIssuer != "" {
		jwtManager.SetIssuer(cfg.Auth.JWTIssuer)
	}
	if cfg.Auth.JWTAudience != "" {
		jwtManager.SetAudience(cfg.Auth.JWTAudience)
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ServerHeader: "Cosmo-Agents-Go",
	})

	// Setup global middleware
	setupMiddleware(app, cfg, jwtManager)

	// Setup basic routes
	setupBasicRoutes(app)

	return &App{
		Fiber:      app,
		Config:     cfg,
		JWTManager: jwtManager,
	}
}

// setupMiddleware configures all global middleware
func setupMiddleware(app *fiber.App, cfg *config.Config, jwtManager *auth.JWTManager) {
	// Global middleware
	app.Use(appMiddleware.Recovery())  // Panic recovery
	app.Use(appMiddleware.RequestID()) // Request ID tracking
	app.Use(appMiddleware.Logger())    // Request logging
	// Enable Prometheus middleware for automatic metrics collection
	app.Use(appMiddleware.Prometheus())                            // HTTP metrics
	app.Use(appMiddleware.CORS(appMiddleware.DefaultCORSConfig())) // CORS
	app.Use(appMiddleware.OptionalAuth(jwtManager, nil, nil))      // Auth

	// Rate limiting (if enabled)
	if cfg.Auth.RateLimitEnabled {
		rateLimitWindow := time.Duration(cfg.Auth.RateLimitWindow) * time.Minute
		app.Use(appMiddleware.PerIPRateLimit(cfg.Auth.RateLimitMax, rateLimitWindow))
		logger.Logger.Info().
			Int("max", cfg.Auth.RateLimitMax).
			Dur("window", rateLimitWindow).
			Msg("Rate limiting enabled")
	}
}

// setupBasicRoutes configures health check and Swagger routes
func setupBasicRoutes(app *fiber.App) {
	healthHandler := func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	}
	// Health check endpoints
	app.Get("/ping", healthHandler)
	app.Get("/health", healthHandler)
	app.Get("/healthz", healthHandler)

	// Prometheus metrics endpoint at root
	app.Get("/metrics", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/plain")

		handler := metrics.Handler()

		w := &metricsWriter{
			c: c,
		}

		req := &http.Request{
			Method: "GET",
			URL:    nil,
		}

		handler.ServeHTTP(w, req)

		return nil
	})

	// Swagger JSON endpoint (disable caching)
	app.Get("/swagger/doc.json", func(c fiber.Ctx) error {
		c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
		return c.SendFile("./docs/swagger.json")
	})

	// Swagger UI endpoint
	app.Get("/swagger", func(c fiber.Ctx) error {
		html := `<!DOCTYPE html>
<html>
<head>
    <title>API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        window.onload = () => {
			// Add a cache-busting query to always fetch latest spec
			const specUrl = '/swagger/doc.json?v=' + Date.now();
			window.ui = SwaggerUIBundle({
				url: specUrl,
                dom_id: '#swagger-ui',
            });
        };
    </script>
</body>
</html>`
		c.Set("Content-Type", "text/html")
		return c.SendString(html)
	})
}

// Start starts the Fiber server
func (a *App) Start() error {
	logger.Logger.Info().Msgf("Starting server on port %d", a.Config.App.Port)
	return a.Fiber.Listen(fmt.Sprintf(":%d", a.Config.App.Port))
}
