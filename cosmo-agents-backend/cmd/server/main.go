// @title Cosmo Agents API
// @version 1.0
// @description API for managing AI agents, campaigns, tasks, and email automation
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/rockship/cosmo-agents-go
// @contact.email support@cosmo-agents.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host
// @BasePath /
// @schemes

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rockship/cosmo-agents-go/pkg/config"
	"github.com/rockship/cosmo-agents-go/pkg/database"
	"github.com/rockship/cosmo-agents-go/pkg/logger"

	_ "github.com/rockship/cosmo-agents-go/docs" // Import docs for Swagger
)

func main() {
	// Load .env file (ignore error if not found)
	_ = godotenv.Load()

	// Load configuration
	cfg := config.MustLoad()

	// Initialize logger
	logger.Init(logger.Config{
		Level:       cfg.Logger.Level,
		Environment: cfg.App.Environment,
		Output:      os.Stdout,
	})

	logger.Info("Starting Cosmo Agents Go Server")
	logger.Logger.Info().
		Str("environment", cfg.App.Environment).
		Int("port", cfg.App.Port).
		Msg("Configuration loaded")

	// Connect to PostgreSQL
	db, err := database.NewPostgresDB(database.PostgresConfig{
		DSN:         cfg.Database.PostgresURI,
		MaxOpenConn: cfg.Database.MaxOpenConn,
		MaxIdleConn: cfg.Database.MaxIdleConn,
		MaxLifetime: 1 * time.Hour,
	})
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	defer database.Close(db)
	logger.Logger.Info().Msg("PostgreSQL connected successfully")

	// Connect to Redis
	redisClient, err := database.NewRedisClient(database.RedisConfig{
		URL:      cfg.Redis.URL,
		Password: cfg.Redis.Password,
	})
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}
	defer database.CloseRedis(redisClient)
	logger.Logger.Info().Msg("Redis connected successfully")

	// Create Fiber application
	app := NewApp(cfg)

	// Initialize all dependencies (repos, services, handlers)
	deps := InitDependencies(db, redisClient, cfg, app.JWTManager)
	defer deps.Close()

	// Register all routes
	RegisterV1Routes(app, deps)
	RegisterV2Routes(app, deps)
	RegisterV3Routes(app, deps)

	// Start server
	if err := app.Start(); err != nil {
		logger.Fatal(fmt.Sprintf("Server failed: %v", err))
	}
}
