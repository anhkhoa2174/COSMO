package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	appMiddleware "github.com/rockship/cosmo-agents-go/internal/middleware"
	"github.com/rockship/cosmo-agents-go/pkg/auth"
	"github.com/rockship/cosmo-agents-go/pkg/config"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// App holds the Fiber application and configuration
type App struct {
	Fiber      *fiber.App
	Config     *config.Config
	JWTManager *auth.JWTManager
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
	// Behind the Cloudflare tunnel (or nginx) every request arrives from the
	// proxy, so c.IP() was the proxy's address and per-IP rate limits became
	// one bucket shared by all visitors. The client address is taken from the
	// proxy header, but only for requests that come from a trusted proxy:
	// loopback, private ranges, and the Tailscale range the tunnel host
	// connects from. A request reaching the server directly cannot spoof it.
	proxyHeader := os.Getenv("TRUSTED_PROXY_HEADER")
	if proxyHeader == "" {
		proxyHeader = "Cf-Connecting-Ip"
	}
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ServerHeader: "Cosmo-Agents-Go",
		BodyLimit:    50 * 1024 * 1024, // 50 MB
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // Disabled for SSE long-lived connections
		TrustProxy:   true,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Loopback: true,
			Private:  true,
			Proxies:  []string{"100.64.0.0/10"},
		},
		ProxyHeader: proxyHeader,
	})

	// NOTE: middleware is NOT set up here — call app.SetupMiddleware() after
	// registering SSE routes to avoid SSE response buffering.

	// Setup basic routes
	setupBasicRoutes(app)

	return &App{
		Fiber:      app,
		Config:     cfg,
		JWTManager: jwtManager,
	}
}

// SetupMiddleware configures global middleware. Must be called AFTER SSE routes.
func (a *App) SetupMiddleware() {
	setupMiddleware(a.Fiber, a.Config, a.JWTManager)
}

// setupMiddleware configures all global middleware
func setupMiddleware(app *fiber.App, cfg *config.Config, jwtManager *auth.JWTManager) {
	// Global middleware
	app.Use(appMiddleware.Recovery())                              // Panic recovery
	app.Use(appMiddleware.RequestID())                             // Request ID tracking
	app.Use(appMiddleware.Logger())                                // Request logging
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
