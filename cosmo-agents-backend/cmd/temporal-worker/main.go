package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rockship/cosmo-agents-go/internal/repository/contact"
	segRepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	intelService "github.com/rockship/cosmo-agents-go/internal/service/intelligence"
	"github.com/rockship/cosmo-agents-go/internal/skills"
	"github.com/rockship/cosmo-agents-go/internal/temporal/worker"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/config"
	"github.com/rockship/cosmo-agents-go/pkg/database"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rockship/cosmo-agents-go/pkg/vectorstore"
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

	log.Printf("Starting Temporal worker with real dependencies...")

	// Get Temporal config from environment
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	temporalNamespace := os.Getenv("TEMPORAL_NAMESPACE")
	if temporalNamespace == "" {
		temporalNamespace = "default"
	}

	log.Printf("Temporal Host: %s", temporalHost)
	log.Printf("Temporal Namespace: %s", temporalNamespace)

	// Connect to PostgreSQL
	db, err := database.NewPostgresDB(database.PostgresConfig{
		DSN:         cfg.Database.PostgresURI,
		MaxOpenConn: cfg.Database.MaxOpenConn,
		MaxIdleConn: cfg.Database.MaxIdleConn,
		MaxLifetime: 1 * time.Hour,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(db)
	log.Println("PostgreSQL connected successfully")

	// Connect to Redis
	redisClient, err := database.NewRedisClient(database.RedisConfig{
		URL:      cfg.Redis.URL,
		Password: cfg.Redis.Password,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer database.CloseRedis(redisClient)
	log.Println("Redis connected successfully")

	// Initialize OpenAI client
	var openAIClient *ai.OpenAIClient
	if cfg.AI.OpenAIAPIKey != "" {
		openAIClient = ai.NewOpenAIClient(ai.Config{
			APIKey: cfg.AI.OpenAIAPIKey,
			Model:  cfg.AI.OpenAIModel,
		})
		log.Println("OpenAI client initialized")
	} else {
		log.Println("Warning: OPENAI_API_KEY not configured - AI activities will use placeholders")
	}

	// Initialize repositories
	contactRepo := contact.NewContactRepository(db)
	segmentationRepo := segRepo.NewSegmentationRepository(db)
	scoreRepo := segRepo.NewScoreRepository(db)

	log.Println("Repositories initialized")

	// Initialize Vector Search infrastructure
	redisVectorStore := vectorstore.NewRedisVectorStore(redisClient)
	ctx := context.Background()
	if err := redisVectorStore.InitializeIndexes(ctx); err != nil {
		log.Printf("Warning: Failed to initialize vector search indexes: %v", err)
	}
	vectorSearchSkill := skills.NewVectorSearchSkill(redisVectorStore, openAIClient)

	// Initialize Intelligence service
	var intelligenceSvc *intelService.Service
	if openAIClient != nil {
		// Note: We're passing contactRepo as interactionRepo placeholder
		// In production, use proper interaction repository
		intelligenceSvc = intelService.NewService(
			contactRepo,
			nil, // interaction repo - can be nil for basic operations
			segmentationRepo,
			scoreRepo,
			openAIClient,
			vectorSearchSkill,
			redisVectorStore,
		)
		log.Println("Intelligence service initialized")
	}

	// Create worker dependencies
	deps := &worker.Dependencies{
		ContactRepo:      contactRepo,
		IntelligenceSvc:  intelligenceSvc,
		SegmentationRepo: segmentationRepo,
		ScoreRepo:        scoreRepo,
		OpenAI:           openAIClient,
	}

	// Create worker with real dependencies
	w, err := worker.NewTemporalWorker(worker.Config{
		HostPort:  temporalHost,
		Namespace: temporalNamespace,
	}, deps)
	if err != nil {
		log.Fatalf("Failed to create Temporal worker: %v", err)
	}

	log.Println("Temporal worker created with real dependencies")

	// Start worker (blocking)
	if err := w.Start(); err != nil {
		log.Fatalf("Temporal worker failed: %v", err)
	}
}
