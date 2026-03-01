package vectorstore

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// Initializer handles vector store initialization on application startup
type Initializer struct {
	redisClient *redis.Client
	vectorStore *RedisVectorStore
}

// NewInitializer creates a new vector store initializer
func NewInitializer(redisClient *redis.Client) *Initializer {
	return &Initializer{
		redisClient: redisClient,
		vectorStore: NewRedisVectorStore(redisClient),
	}
}

// Initialize sets up all vector search indexes and verifies Redis Stack is available
func (i *Initializer) Initialize(ctx context.Context) error {
	log.Info().Msg("Initializing vector search infrastructure...")

	// Check if Redis Stack is available
	if err := i.verifyRedisStack(ctx); err != nil {
		return fmt.Errorf("Redis Stack verification failed: %w", err)
	}

	// Create vector search indexes
	if err := i.vectorStore.InitializeIndexes(ctx); err != nil {
		return fmt.Errorf("failed to initialize vector indexes: %w", err)
	}

	log.Info().Msg("✓ Vector search infrastructure initialized successfully")
	return nil
}

// verifyRedisStack checks if Redis Stack modules are loaded
func (i *Initializer) verifyRedisStack(ctx context.Context) error {
	// Check for RediSearch module
	modules, err := i.redisClient.Do(ctx, "MODULE", "LIST").Result()
	if err != nil {
		return fmt.Errorf("failed to check Redis modules: %w", err)
	}

	// Parse module list
	moduleList, ok := modules.([]interface{})
	if !ok {
		return fmt.Errorf("unexpected module list format")
	}

	hasSearch := false
	for _, mod := range moduleList {
		modInfo, ok := mod.([]interface{})
		if !ok || len(modInfo) < 2 {
			continue
		}

		// Module info format: [name, value, version, value, ...]
		for j := 0; j < len(modInfo); j += 2 {
			if fmt.Sprintf("%v", modInfo[j]) == "name" {
				modName := fmt.Sprintf("%v", modInfo[j+1])
				if modName == "search" || modName == "ft" {
					hasSearch = true
					log.Info().Str("module", modName).Msg("Found RediSearch module")
					break
				}
			}
		}
	}

	if !hasSearch {
		return fmt.Errorf("RediSearch module not found. Please use redis/redis-stack-server image")
	}

	log.Info().Msg("✓ Redis Stack verified")
	return nil
}

// GetVectorStore returns the initialized vector store instance
func (i *Initializer) GetVectorStore() *RedisVectorStore {
	return i.vectorStore
}

// HealthCheck verifies vector search is working correctly
func (i *Initializer) HealthCheck(ctx context.Context) error {
	// Check if indexes exist
	indexes := []string{
		ContactVectorIndex,
		InteractionVectorIndex,
		KnowledgeVectorIndex,
		SegmentVectorIndex,
	}

	for _, idx := range indexes {
		result := i.redisClient.Do(ctx, "FT.INFO", idx)
		if result.Err() != nil {
			return fmt.Errorf("index %s not found or unhealthy: %w", idx, result.Err())
		}
	}

	log.Debug().Msg("Vector search health check passed")
	return nil
}

// ResetIndexes drops and recreates all vector indexes (use with caution!)
func (i *Initializer) ResetIndexes(ctx context.Context) error {
	log.Warn().Msg("Resetting all vector indexes - this will delete all vectors!")

	indexes := []string{
		ContactVectorIndex,
		InteractionVectorIndex,
		KnowledgeVectorIndex,
		SegmentVectorIndex,
	}

	// Drop existing indexes
	for _, idx := range indexes {
		err := i.redisClient.Do(ctx, "FT.DROPINDEX", idx, "DD").Err()
		if err != nil && err.Error() != "Unknown Index name" {
			log.Warn().Str("index", idx).Err(err).Msg("Failed to drop index (might not exist)")
		}
	}

	// Recreate indexes
	return i.vectorStore.InitializeIndexes(ctx)
}

// GetIndexStats returns statistics about vector indexes
func (i *Initializer) GetIndexStats(ctx context.Context) (map[string]IndexStats, error) {
	indexes := []string{
		ContactVectorIndex,
		InteractionVectorIndex,
		KnowledgeVectorIndex,
		SegmentVectorIndex,
	}

	stats := make(map[string]IndexStats)

	for _, idx := range indexes {
		info, err := i.redisClient.Do(ctx, "FT.INFO", idx).Result()
		if err != nil {
			continue
		}

		infoList, ok := info.([]interface{})
		if !ok {
			continue
		}

		stat := IndexStats{
			Name: idx,
		}

		// Parse index info
		for j := 0; j < len(infoList); j += 2 {
			if j+1 >= len(infoList) {
				break
			}

			key := fmt.Sprintf("%v", infoList[j])
			value := infoList[j+1]

			switch key {
			case "num_docs":
				if v, ok := value.(string); ok {
					stat.DocumentCount = v
				}
			case "num_records":
				if v, ok := value.(string); ok {
					stat.RecordCount = v
				}
			case "indexing":
				if v, ok := value.(string); ok {
					stat.IsIndexing = v == "1"
				}
			}
		}

		stats[idx] = stat
	}

	return stats, nil
}

// IndexStats contains statistics about a vector index
type IndexStats struct {
	Name          string `json:"name"`
	DocumentCount string `json:"document_count"`
	RecordCount   string `json:"record_count"`
	IsIndexing    bool   `json:"is_indexing"`
}
