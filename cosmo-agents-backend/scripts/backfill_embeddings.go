// Package main provides a CLI tool to backfill embeddings for existing contacts
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/vectorstore"
)

var (
	dryRun      = flag.Bool("dry-run", false, "Run without actually generating embeddings")
	batchSize   = flag.Int("batch-size", 50, "Number of contacts to process in each batch")
	rateLimit   = flag.Duration("rate-limit", 200*time.Millisecond, "Delay between API calls")
	userIDFlag  = flag.String("user-id", "", "Process only contacts for specific user (optional)")
	maxContacts = flag.Int("max", 0, "Maximum number of contacts to process (0 = unlimited)")
)

func main() {
	flag.Parse()

	// Setup logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	log.Info().Msg("🚀 Starting embedding backfill process")

	// Load configuration from environment
	cfg := loadConfig()

	// Connect to PostgreSQL
	db, err := connectDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}

	// Initialize vector store
	vectorStore := vectorstore.NewRedisVectorStore(redisClient)

	// Initialize OpenAI client
	var openAIClient *ai.OpenAIClient
	if !*dryRun {
		if cfg.OpenAIAPIKey == "" {
			log.Fatal().Msg("OPENAI_API_KEY not set")
		}
		openAIClient = ai.NewOpenAIClient(ai.Config{
			APIKey: cfg.OpenAIAPIKey,
			Model:  "text-embedding-3-small", // or whatever embedding model you want to use
		})
	}

	// Create backfiller
	backfiller := &EmbeddingBackfiller{
		db:          db,
		vectorStore: vectorStore,
		openAI:      openAIClient,
		dryRun:      *dryRun,
		batchSize:   *batchSize,
		rateLimit:   *rateLimit,
	}

	// Run backfill
	stats, err := backfiller.Run(ctx, *userIDFlag, *maxContacts)
	if err != nil {
		log.Fatal().Err(err).Msg("Backfill failed")
	}

	// Print summary
	printSummary(stats)
}

type Config struct {
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	OpenAIAPIKey  string
}

func loadConfig() Config {
	return Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/cosmo_agents?sslmode=disable"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6381"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
		OpenAIAPIKey:  getEnv("OPENAI_API_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func connectDB(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

type EmbeddingBackfiller struct {
	db          *gorm.DB
	vectorStore *vectorstore.RedisVectorStore
	openAI      *ai.OpenAIClient
	dryRun      bool
	batchSize   int
	rateLimit   time.Duration
}

type BackfillStats struct {
	TotalContacts     int
	ProcessedContacts int
	SkippedContacts   int
	FailedContacts    int
	TotalDuration     time.Duration
	AvgEmbeddingTime  time.Duration
}

func (b *EmbeddingBackfiller) Run(ctx context.Context, userID string, maxContacts int) (*BackfillStats, error) {
	startTime := time.Now()
	stats := &BackfillStats{}

	// Count total contacts
	query := b.db.Model(&domain.Contact{})
	if userID != "" {
		uid, err := uuid.Parse(userID)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID: %w", err)
		}
		query = query.Where("user_id = ?", uid)
	}

	if err := query.Count(&[]int64{int64(stats.TotalContacts)}[0]).Error; err != nil {
		return nil, fmt.Errorf("failed to count contacts: %w", err)
	}

	log.Info().Int("total", stats.TotalContacts).Msg("Found contacts to process")

	if maxContacts > 0 && stats.TotalContacts > maxContacts {
		stats.TotalContacts = maxContacts
	}

	// Process in batches
	offset := 0
	embeddingTimes := []time.Duration{}

	for {
		// Fetch batch
		var contacts []domain.Contact
		query := b.db.Limit(b.batchSize).Offset(offset)
		if userID != "" {
			uid, _ := uuid.Parse(userID)
			query = query.Where("user_id = ?", uid)
		}

		if err := query.Find(&contacts).Error; err != nil {
			return nil, fmt.Errorf("failed to fetch contacts: %w", err)
		}

		if len(contacts) == 0 {
			break
		}

		// Process batch
		for _, contact := range contacts {
			if maxContacts > 0 && stats.ProcessedContacts >= maxContacts {
				goto done
			}

			// Check if embedding already exists
			exists, err := b.checkEmbeddingExists(ctx, contact.ID)
			if err != nil {
				log.Warn().Err(err).Str("contact_id", contact.ID.String()).Msg("Failed to check embedding")
			}
			if exists {
				log.Debug().Str("contact_id", contact.ID.String()).Msg("Embedding already exists, skipping")
				stats.SkippedContacts++
				continue
			}

			// Build embedding text
			embeddingText := buildEmbeddingText(&contact)

			if b.dryRun {
				log.Info().
					Str("contact_id", contact.ID.String()).
					Str("name", contact.Name).
					Str("embedding_text", embeddingText).
					Msg("[DRY RUN] Would generate embedding")
				stats.ProcessedContacts++
				continue
			}

			// Generate embedding
			embStart := time.Now()
			vector, err := b.openAI.GenerateEmbedding(ctx, embeddingText)
			embDuration := time.Since(embStart)
			embeddingTimes = append(embeddingTimes, embDuration)

			if err != nil {
				log.Error().Err(err).Str("contact_id", contact.ID.String()).Msg("Failed to generate embedding")
				stats.FailedContacts++
				continue
			}

			// Store in Redis
			metadata := map[string]interface{}{
				"name":            contact.Name,
				"company":         contact.Company,
				"job_title":       contact.JobTitle,
				"industry":        contact.Industry,
				"contact_channel": contact.ContactChannel,
				"outreach_stage":  contact.OutreachStage,
				"city":            contact.City,
				"country":         contact.Country,
				"backfilled":      true,
			}

			err = b.vectorStore.StoreContactVector(ctx, contact.ID, contact.UserID, vector, metadata)
			if err != nil {
				log.Error().Err(err).Str("contact_id", contact.ID.String()).Msg("Failed to store vector")
				stats.FailedContacts++
				continue
			}

			log.Info().
				Str("contact_id", contact.ID.String()).
				Str("name", contact.Name).
				Dur("embedding_time", embDuration).
				Msg("✓ Generated and stored embedding")

			stats.ProcessedContacts++

			// Rate limiting
			time.Sleep(b.rateLimit)
		}

		offset += b.batchSize

		// Progress update
		log.Info().
			Int("processed", stats.ProcessedContacts).
			Int("skipped", stats.SkippedContacts).
			Int("failed", stats.FailedContacts).
			Int("total", stats.TotalContacts).
			Msg("Progress update")
	}

done:
	stats.TotalDuration = time.Since(startTime)

	// Calculate average embedding time
	if len(embeddingTimes) > 0 {
		var total time.Duration
		for _, d := range embeddingTimes {
			total += d
		}
		stats.AvgEmbeddingTime = total / time.Duration(len(embeddingTimes))
	}

	return stats, nil
}

func (b *EmbeddingBackfiller) checkEmbeddingExists(ctx context.Context, contactID uuid.UUID) (bool, error) {
	// Simplified: always generate for now (no lightweight exists check yet)
	return false, nil
}

// extractEmailFromProfile extracts email from contact's profile JSONB
func extractEmailFromProfile(profile domain.JSONB) string {
	if len(profile) == 0 {
		return ""
	}
	var profileData map[string]interface{}
	if err := json.Unmarshal(profile, &profileData); err != nil {
		return ""
	}
	if email, ok := profileData["email"].(string); ok {
		return email
	}
	return ""
}

func buildEmbeddingText(contact *domain.Contact) string {
	var parts []string

	// Basic info
	if contact.Name != "N/A" && contact.Name != "" {
		parts = append(parts, contact.Name)
	}

	if contact.JobTitle != "N/A" {
		parts = append(parts, fmt.Sprintf("Job Title: %s", contact.JobTitle))
		// Boost job title for semantic search
		parts = append(parts, fmt.Sprintf("Job Title: %s", contact.JobTitle))
	}

	if contact.Company != "N/A" {
		parts = append(parts, fmt.Sprintf("Company: %s", contact.Company))
		// Boost company for semantic search
		parts = append(parts, fmt.Sprintf("Company: %s", contact.Company))
	}

	// Industry - important for semantic search like "find contacts in fintech"
	if contact.Industry != "" {
		parts = append(parts, fmt.Sprintf("Industry: %s", contact.Industry))
		// Boost industry for semantic search
		parts = append(parts, fmt.Sprintf("Industry: %s", contact.Industry))
	}

	// Contact channel
	if contact.ContactChannel != "" {
		parts = append(parts, fmt.Sprintf("Contact Channel: %s", contact.ContactChannel))
	}

	// Outreach stage
	if contact.OutreachStage != "" && contact.OutreachStage != "COLD" {
		parts = append(parts, fmt.Sprintf("Outreach Stage: %s", contact.OutreachStage))
	}

	// Outreach context
	if contact.ContextLevel != "" && contact.ContextLevel != "LOW" {
		parts = append(parts, fmt.Sprintf("Context Level: %s", contact.ContextLevel))
	}

	if contact.OutreachDecision != "" && contact.OutreachDecision != "INTRO" {
		parts = append(parts, fmt.Sprintf("Outreach Decision: %s", contact.OutreachDecision))
	}

	if contact.Scenario != "" {
		parts = append(parts, fmt.Sprintf("Scenario: %s", contact.Scenario))
	}

	if contact.LastOutcome != "" {
		parts = append(parts, fmt.Sprintf("Last Outcome: %s", contact.LastOutcome))
	}

	if contact.NextStep != "" && contact.NextStep != "SEND" {
		parts = append(parts, fmt.Sprintf("Next Step: %s", contact.NextStep))
	}

	// Extract email from profile JSONB
	contactEmail := extractEmailFromProfile(contact.Profile)
	if contactEmail != "" && contactEmail != "N/A" {
		parts = append(parts, fmt.Sprintf("Email: %s", contactEmail))
	}

	// Location info
	if contact.City != "N/A" {
		parts = append(parts, fmt.Sprintf("City: %s", contact.City))
	}
	if contact.Country != "N/A" {
		parts = append(parts, fmt.Sprintf("Country: %s", contact.Country))
	}

	// Profile data (if available)
	// Note: You'll need to unmarshal JSONB fields properly
	// This is a simplified version

	if len(parts) == 0 {
		return "Contact with minimal information"
	}

	return strings.Join(parts, ". ")
}

func printSummary(stats *BackfillStats) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 EMBEDDING BACKFILL SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total Contacts:      %d\n", stats.TotalContacts)
	fmt.Printf("Processed:           %d ✓\n", stats.ProcessedContacts)
	fmt.Printf("Skipped:             %d ⊘\n", stats.SkippedContacts)
	fmt.Printf("Failed:              %d ✗\n", stats.FailedContacts)
	fmt.Printf("Total Duration:      %s\n", stats.TotalDuration.Round(time.Second))
	if stats.AvgEmbeddingTime > 0 {
		fmt.Printf("Avg Embedding Time:  %s\n", stats.AvgEmbeddingTime.Round(time.Millisecond))
	}
	fmt.Println(strings.Repeat("=", 60))

	if stats.ProcessedContacts > 0 {
		successRate := float64(stats.ProcessedContacts) / float64(stats.TotalContacts) * 100
		fmt.Printf("\n✨ Success Rate: %.1f%%\n", successRate)
	}
}
