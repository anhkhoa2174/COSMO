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
		// RESP3 (go-redis v9's default) replies with one map per module;
		// RESP2 with a flat [name, value, ver, value, ...] array.
		if m, ok := mod.(map[interface{}]interface{}); ok {
			mod = []interface{}{"name", m["name"]}
		}
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

		// RESP3 (go-redis v9's default) replies with a map, RESP2 with a flat
		// [key, value, ...] array; current servers send the counts as integers
		// under both, older ones as strings, so values are read with Sprint.
		fields := map[string]interface{}{}
		switch v := info.(type) {
		case map[interface{}]interface{}:
			for k, val := range v {
				fields[fmt.Sprint(k)] = val
			}
		case []interface{}:
			for j := 0; j+1 < len(v); j += 2 {
				fields[fmt.Sprint(v[j])] = v[j+1]
			}
		default:
			continue
		}

		stat := IndexStats{Name: idx}
		if v, ok := fields["num_docs"]; ok {
			stat.DocumentCount = fmt.Sprint(v)
		}
		if v, ok := fields["num_records"]; ok {
			stat.RecordCount = fmt.Sprint(v)
		}
		if v, ok := fields["indexing"]; ok {
			stat.IsIndexing = fmt.Sprint(v) == "1"
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
