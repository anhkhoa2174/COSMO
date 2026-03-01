package vectorstore

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// Vector dimensions for OpenAI text-embedding-ada-002
	VectorDimensions = 1536

	// Index names
	ContactVectorIndex     = "idx:contacts:vector"
	InteractionVectorIndex = "idx:interactions:vector"
	KnowledgeVectorIndex   = "idx:knowledge:vector"
	SegmentVectorIndex     = "idx:segments:vector"

	// Key prefixes
	ContactVectorPrefix     = "vector:contact:"
	InteractionVectorPrefix = "vector:interaction:"
	KnowledgeVectorPrefix   = "vector:knowledge:"
	SegmentVectorPrefix     = "vector:segment:"
)

// VectorMetadata contains metadata stored alongside the vector
type VectorMetadata struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"user_id,omitempty"`
	EntityType     string                 `json:"entity_type"`
	EmbeddingModel string                 `json:"embedding_model"`
	CreatedAt      time.Time              `json:"created_at"`
	Extra          map[string]interface{} `json:"extra,omitempty"`
}

// VectorSearchResult represents a single search result
type VectorSearchResult struct {
	ID       string         `json:"id"`
	Score    float64        `json:"score"`
	Metadata VectorMetadata `json:"metadata"`
	Vector   []float32      `json:"vector,omitempty"`
}

// RedisVectorStore manages vector storage and search operations in Redis
type RedisVectorStore struct {
	client *redis.Client
}

// NewRedisVectorStore creates a new Redis vector store instance
func NewRedisVectorStore(client *redis.Client) *RedisVectorStore {
	return &RedisVectorStore{
		client: client,
	}
}

// InitializeIndexes creates Redis Search indexes for vector similarity search
func (r *RedisVectorStore) InitializeIndexes(ctx context.Context) error {
	indexes := []struct {
		name   string
		prefix string
	}{
		{ContactVectorIndex, ContactVectorPrefix},
		{InteractionVectorIndex, InteractionVectorPrefix},
		{KnowledgeVectorIndex, KnowledgeVectorPrefix},
		{SegmentVectorIndex, SegmentVectorPrefix},
	}

	for _, idx := range indexes {
		if err := r.createVectorIndex(ctx, idx.name, idx.prefix); err != nil {
			// If index already exists, that's okay
			if err.Error() != "Index already exists" {
				return fmt.Errorf("failed to create index %s: %w", idx.name, err)
			}
		}
	}

	return nil
}

// createVectorIndex creates a single vector index using RediSearch
func (r *RedisVectorStore) createVectorIndex(ctx context.Context, indexName, prefix string) error {
	// FT.CREATE idx:contacts:vector
	//   ON HASH PREFIX 1 vector:contact:
	//   SCHEMA
	//     id TAG
	//     user_id TAG
	//     entity_type TAG
	//     embedding_model TAG
	//     created_at NUMERIC SORTABLE
	//     vector VECTOR HNSW 6 TYPE FLOAT32 DIM 1536 DISTANCE_METRIC COSINE

	args := []interface{}{
		indexName,
		"ON", "HASH",
		"PREFIX", "1", prefix,
		"SCHEMA",
		"id", "TAG",
		"user_id", "TAG",
		"entity_type", "TAG",
		"embedding_model", "TAG",
		"created_at", "NUMERIC", "SORTABLE",
		"vector", "VECTOR", "HNSW", "6",
		"TYPE", "FLOAT32",
		"DIM", strconv.Itoa(VectorDimensions),
		"DISTANCE_METRIC", "COSINE",
	}

	return r.client.Do(ctx, append([]interface{}{"FT.CREATE"}, args...)...).Err()
}

// StoreContactVector stores a contact embedding in Redis
func (r *RedisVectorStore) StoreContactVector(ctx context.Context, contactID uuid.UUID, userID uuid.UUID, vector []float32, metadata map[string]interface{}) error {
	key := ContactVectorPrefix + contactID.String()

	meta := VectorMetadata{
		ID:             contactID.String(),
		UserID:         userID.String(),
		EntityType:     "contact",
		EmbeddingModel: "text-embedding-ada-002",
		CreatedAt:      time.Now(),
		Extra:          metadata,
	}

	return r.storeVector(ctx, key, vector, meta)
}

// StoreInteractionVector stores an interaction embedding in Redis
func (r *RedisVectorStore) StoreInteractionVector(ctx context.Context, interactionID uuid.UUID, userID uuid.UUID, vector []float32, metadata map[string]interface{}) error {
	key := InteractionVectorPrefix + interactionID.String()

	meta := VectorMetadata{
		ID:             interactionID.String(),
		UserID:         userID.String(),
		EntityType:     "interaction",
		EmbeddingModel: "text-embedding-ada-002",
		CreatedAt:      time.Now(),
		Extra:          metadata,
	}

	return r.storeVector(ctx, key, vector, meta)
}

// StoreKnowledgeVector stores a knowledge chunk embedding in Redis
func (r *RedisVectorStore) StoreKnowledgeVector(ctx context.Context, knowledgeID uuid.UUID, userID uuid.UUID, chunkIndex int, vector []float32, chunkText string) error {
	// Use composite key: knowledge_id:chunk_index
	compositeID := fmt.Sprintf("%s:%d", knowledgeID.String(), chunkIndex)
	key := KnowledgeVectorPrefix + compositeID

	meta := VectorMetadata{
		ID:             compositeID,
		UserID:         userID.String(),
		EntityType:     "knowledge",
		EmbeddingModel: "text-embedding-ada-002",
		CreatedAt:      time.Now(),
		Extra: map[string]interface{}{
			"knowledge_id": knowledgeID.String(),
			"chunk_index":  chunkIndex,
			"chunk_text":   chunkText,
		},
	}

	return r.storeVector(ctx, key, vector, meta)
}

// StoreSegmentVector stores a segment aggregate embedding in Redis
func (r *RedisVectorStore) StoreSegmentVector(ctx context.Context, segmentID uuid.UUID, userID uuid.UUID, vector []float32, metadata map[string]interface{}) error {
	key := SegmentVectorPrefix + segmentID.String()

	meta := VectorMetadata{
		ID:             segmentID.String(),
		UserID:         userID.String(),
		EntityType:     "segment",
		EmbeddingModel: "text-embedding-ada-002",
		CreatedAt:      time.Now(),
		Extra:          metadata,
	}

	return r.storeVector(ctx, key, vector, meta)
}

// storeVector is a helper to store any vector with metadata
func (r *RedisVectorStore) storeVector(ctx context.Context, key string, vector []float32, metadata VectorMetadata) error {
	// Convert vector to bytes
	vectorBytes := make([]byte, len(vector)*4) // 4 bytes per float32
	for i, v := range vector {
		bits := floatToBits(v)
		vectorBytes[i*4] = byte(bits)
		vectorBytes[i*4+1] = byte(bits >> 8)
		vectorBytes[i*4+2] = byte(bits >> 16)
		vectorBytes[i*4+3] = byte(bits >> 24)
	}

	// Serialize metadata to JSON
	metaJSON, err := json.Marshal(metadata.Extra)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Remove hyphens from UUIDs for Redis TAG field compatibility
	cleanUserID := strings.ReplaceAll(metadata.UserID, "-", "")

	// Store as Redis hash
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key,
		"id", metadata.ID,
		"user_id", cleanUserID,
		"entity_type", metadata.EntityType,
		"embedding_model", metadata.EmbeddingModel,
		"created_at", metadata.CreatedAt.Unix(),
		"vector", vectorBytes,
		"metadata", string(metaJSON),
	)
	// No expiration - vectors should persist indefinitely

	_, err = pipe.Exec(ctx)
	if err != nil {
		fmt.Printf("[ERROR] Failed to store vector in Redis: %v\n", err)
		return err
	}
	fmt.Printf("[DEBUG] Successfully stored vector with key: %s\n", key)
	return nil
}

// SearchSimilarContacts finds similar contacts using vector similarity
func (r *RedisVectorStore) SearchSimilarContacts(ctx context.Context, queryVector []float32, userID uuid.UUID, limit int, threshold float64) ([]VectorSearchResult, error) {
	return r.searchVectors(ctx, ContactVectorIndex, queryVector, userID, limit, threshold)
}

// SearchSimilarInteractions finds similar interactions
func (r *RedisVectorStore) SearchSimilarInteractions(ctx context.Context, queryVector []float32, userID uuid.UUID, limit int, threshold float64) ([]VectorSearchResult, error) {
	return r.searchVectors(ctx, InteractionVectorIndex, queryVector, userID, limit, threshold)
}

// SearchKnowledge finds relevant knowledge chunks
func (r *RedisVectorStore) SearchKnowledge(ctx context.Context, queryVector []float32, userID uuid.UUID, limit int, threshold float64) ([]VectorSearchResult, error) {
	return r.searchVectors(ctx, KnowledgeVectorIndex, queryVector, userID, limit, threshold)
}

// SearchSimilarSegments finds similar segments
func (r *RedisVectorStore) SearchSimilarSegments(ctx context.Context, queryVector []float32, userID uuid.UUID, limit int, threshold float64) ([]VectorSearchResult, error) {
	return r.searchVectors(ctx, SegmentVectorIndex, queryVector, userID, limit, threshold)
}

// searchVectors performs vector similarity search using RediSearch
func (r *RedisVectorStore) searchVectors(ctx context.Context, indexName string, queryVector []float32, userID uuid.UUID, limit int, threshold float64) ([]VectorSearchResult, error) {
	fmt.Printf("[searchVectors] Index: %s, UserID: %s, Limit: %d, Threshold: %.2f\n", indexName, userID, limit, threshold)

	// Convert query vector to bytes
	vectorBytes := make([]byte, len(queryVector)*4)
	for i, v := range queryVector {
		bits := floatToBits(v)
		vectorBytes[i*4] = byte(bits)
		vectorBytes[i*4+1] = byte(bits >> 8)
		vectorBytes[i*4+2] = byte(bits >> 16)
		vectorBytes[i*4+3] = byte(bits >> 24)
	}

	// Use KNN query without user filter to avoid RediSearch tag-escaping issues.
	// Apply threshold filtering after parsing results.
	query := fmt.Sprintf("*=>[KNN %d @vector $vec]", limit)

	args := []interface{}{
		"FT.SEARCH", indexName,
		query,
		"PARAMS", "2", "vec", vectorBytes,
		"WITHSCORES",
		"SORTBY", "__vector_score",
		"DIALECT", "2",
		"LIMIT", "0", strconv.Itoa(limit),
		// Include __vector_score explicitly so RediSearch loads it and allows sorting
		"RETURN", "6", "__vector_score", "id", "user_id", "entity_type", "created_at", "metadata",
	}

	fmt.Printf("[searchVectors] Executing query: %s\n", query)

	result, err := r.client.Do(ctx, args...).Result()
	if err != nil {
		fmt.Printf("[searchVectors] Redis query ERROR: %v\n", err)
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Debug: print raw result
	fmt.Printf("[searchVectors] Raw Redis result type: %T\n", result)
	if arr, ok := result.([]interface{}); ok {
		fmt.Printf("[searchVectors] Raw Redis result length: %d\n", len(arr))
		if len(arr) > 0 {
			fmt.Printf("[searchVectors] First element (count): %v\n", arr[0])
		}
	} else if m, ok := result.(map[interface{}]interface{}); ok {
		fmt.Printf("[searchVectors] ERROR: Redis returned error map: %+v\n", m)
	} else {
		fmt.Printf("[searchVectors] ERROR: Unexpected result format: %+v\n", result)
	}

	results, err := r.parseSearchResults(result)
	if err != nil {
		fmt.Printf("[searchVectors] Parse results ERROR: %v\n", err)
		return nil, err
	}

	fmt.Printf("[searchVectors] Parsed %d results from Redis\n", len(results))

	// Tenant-scope results after search to avoid RediSearch tag-escaping issues in the query
	cleanUserID := strings.ReplaceAll(userID.String(), "-", "")
	filteredByUser := results[:0]
	for _, res := range results {
		if res.Metadata.UserID == "" || res.Metadata.UserID == cleanUserID {
			filteredByUser = append(filteredByUser, res)
		}
	}
	results = filteredByUser

	fmt.Printf("[searchVectors] After user filter (%s): %d results\n", cleanUserID, len(results))

	// Apply threshold filtering
	// __vector_score is a distance (lower is better). Keep results with score <= threshold.
	if threshold > 0 {
		fmt.Printf("[searchVectors] Applying threshold filter: distance <= %.2f\n", threshold)
		filtered := results[:0]
		for _, r := range results {
			keep := r.Score <= threshold
			fmt.Printf("[searchVectors] Result: ID=%s, Score=%.4f, Keep=%v\n", r.ID, r.Score, keep)
			if keep {
				filtered = append(filtered, r)
			}
		}
		results = filtered
		fmt.Printf("[searchVectors] After threshold filter: %d results\n", len(results))
	}

	// Sort by distance ascending (lowest = best)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score < results[j].Score
	})

	return results, nil
}

// parseSearchResults parses RediSearch results into VectorSearchResult structs
func (r *RedisVectorStore) parseSearchResults(result interface{}) ([]VectorSearchResult, error) {
	results := make([]VectorSearchResult, 0)

	// Try new map format first (redis v3 client)
	if m, ok := result.(map[interface{}]interface{}); ok {
		fmt.Printf("[parseSearchResults] Parsing map format\n")

		// Get results array from map
		resultsRaw, ok := m["results"]
		if !ok {
			fmt.Printf("[parseSearchResults] No 'results' key in map\n")
			return results, nil
		}

		resultsArr, ok := resultsRaw.([]interface{})
		if !ok {
			fmt.Printf("[parseSearchResults] 'results' is not array\n")
			return results, nil
		}

		fmt.Printf("[parseSearchResults] Found %d results in map\n", len(resultsArr))

		// Parse each result
		for _, item := range resultsArr {
			resultMap, ok := item.(map[interface{}]interface{})
			if !ok {
				continue
			}

			var res VectorSearchResult

			// Get ID from result map
			if id, ok := resultMap["id"]; ok {
				res.ID = fmt.Sprintf("%v", id)
			}

			// Get extra_attributes map
			if extraRaw, ok := resultMap["extra_attributes"]; ok {
				if extraMap, ok := extraRaw.(map[interface{}]interface{}); ok {
					// Parse score
					if score, ok := extraMap["__vector_score"]; ok {
						if scoreFloat, ok := score.(float64); ok {
							res.Score = scoreFloat
						} else if scoreStr, ok := score.(string); ok {
							res.Score, _ = strconv.ParseFloat(scoreStr, 64)
						}
					}

					// Parse metadata fields
					res.Metadata.ID = fmt.Sprintf("%v", extraMap["id"])
					res.Metadata.UserID = fmt.Sprintf("%v", extraMap["user_id"])
					res.Metadata.EntityType = fmt.Sprintf("%v", extraMap["entity_type"])

					// Parse metadata JSON
					if metaJSON, ok := extraMap["metadata"]; ok {
						metaStr := fmt.Sprintf("%v", metaJSON)
						var extra map[string]interface{}
						if err := json.Unmarshal([]byte(metaStr), &extra); err == nil {
							res.Metadata.Extra = extra
						}
					}
				}
			}

			results = append(results, res)
		}

		fmt.Printf("[parseSearchResults] Successfully parsed %d results from map\n", len(results))
		return results, nil
	}

	// Fall back to old array format
	arr, ok := result.([]interface{})
	if !ok || len(arr) == 0 {
		fmt.Printf("[parseSearchResults] Not array format or empty\n")
		return []VectorSearchResult{}, nil
	}

	fmt.Printf("[parseSearchResults] Parsing array format with %d elements\n", len(arr))

	// First element is total count
	// Parse results (skip first element which is count)
	for i := 1; i < len(arr); i += 2 {
		if i+1 >= len(arr) {
			break
		}

		// Document key
		// Fields array
		fields, ok := arr[i+1].([]interface{})
		if !ok {
			continue
		}

		var res VectorSearchResult
		var metaJSON string

		// Parse fields
		for j := 0; j < len(fields); j += 2 {
			if j+1 >= len(fields) {
				break
			}

			fieldName := fmt.Sprintf("%v", fields[j])
			fieldValue := fields[j+1]

			switch fieldName {
			case "id":
				res.ID = fmt.Sprintf("%v", fieldValue)
			case "__vector_score":
				if score, ok := fieldValue.(string); ok {
					res.Score, _ = strconv.ParseFloat(score, 64)
				}
			case "metadata":
				metaJSON = fmt.Sprintf("%v", fieldValue)
			case "user_id":
				res.Metadata.UserID = fmt.Sprintf("%v", fieldValue)
			case "entity_type":
				res.Metadata.EntityType = fmt.Sprintf("%v", fieldValue)
			}
		}

		// Parse metadata JSON
		if metaJSON != "" {
			var extra map[string]interface{}
			if err := json.Unmarshal([]byte(metaJSON), &extra); err == nil {
				res.Metadata.Extra = extra
			}
		}

		res.Metadata.ID = res.ID
		results = append(results, res)
	}

	fmt.Printf("[parseSearchResults] Successfully parsed %d results from array\n", len(results))
	return results, nil
}

// DeleteVector removes a vector from Redis
func (r *RedisVectorStore) DeleteVector(ctx context.Context, prefix, id string) error {
	key := prefix + id
	return r.client.Del(ctx, key).Err()
}

// DeleteContactVector removes a contact vector
func (r *RedisVectorStore) DeleteContactVector(ctx context.Context, contactID uuid.UUID) error {
	return r.DeleteVector(ctx, ContactVectorPrefix, contactID.String())
}

// floatToBits converts float32 to uint32 bits
func floatToBits(f float32) uint32 {
	return *(*uint32)(unsafe.Pointer(&f))
}
