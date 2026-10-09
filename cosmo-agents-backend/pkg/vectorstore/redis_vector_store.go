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

	// An index created before knowledge_type existed does not have the field,
	// and FT.CREATE on an existing index does nothing, so it is added here.
	// Chunks stored before then carry no type and simply never match a type
	// filter; retrieval falls back to an unfiltered search for them.
	if err := r.client.Do(ctx, "FT.ALTER", KnowledgeVectorIndex, "SCHEMA", "ADD",
		"knowledge_type", "TAG").Err(); err != nil && !isDuplicateField(err) {
		return fmt.Errorf("failed to add knowledge_type to %s: %w", KnowledgeVectorIndex, err)
	}

	return nil
}

// isDuplicateField reports whether FT.ALTER failed only because the field is
// already in the schema, which is the normal case on every start after the
// first.
func isDuplicateField(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "already exists")
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
		"knowledge_type", "TAG",
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

// StoreKnowledgeVector stores a knowledge chunk embedding in Redis, with no
// document title or type.
func (r *RedisVectorStore) StoreKnowledgeVector(ctx context.Context, knowledgeID uuid.UUID, userID uuid.UUID, chunkIndex int, vector []float32, chunkText string) error {
	return r.StoreKnowledgeChunk(ctx, knowledgeID, userID, chunkIndex, vector, chunkText, "", "")
}

// StoreKnowledgeChunk stores a knowledge chunk embedding together with the
// title of the document it came from and the document's type.
//
// The title is what the reply prompt labels each retrieved passage with, so a
// reader of the prompt — and the model — can tell which document a fact came
// from. The type is also written as its own hash field, indexed as a TAG, so a
// search can be restricted to, say, pricing documents.
func (r *RedisVectorStore) StoreKnowledgeChunk(ctx context.Context, knowledgeID uuid.UUID, userID uuid.UUID, chunkIndex int, vector []float32, chunkText, title, knowledgeType string) error {
	// Use composite key: knowledge_id:chunk_index
	compositeID := fmt.Sprintf("%s:%d", knowledgeID.String(), chunkIndex)
	key := KnowledgeVectorPrefix + compositeID

	extra := map[string]interface{}{
		"knowledge_id": knowledgeID.String(),
		"chunk_index":  chunkIndex,
		"chunk_text":   chunkText,
	}
	if title != "" {
		extra["title"] = title
	}
	if knowledgeType != "" {
		extra["knowledge_type"] = knowledgeType
	}

	meta := VectorMetadata{
		ID:             compositeID,
		UserID:         userID.String(),
		EntityType:     "knowledge",
		EmbeddingModel: "text-embedding-ada-002",
		CreatedAt:      time.Now(),
		Extra:          extra,
	}

	if err := r.storeVector(ctx, key, vector, meta); err != nil {
		return err
	}
	if knowledgeType == "" {
		return nil
	}
	return r.client.HSet(ctx, key, "knowledge_type", knowledgeType).Err()
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
func (r *RedisVectorStore) SearchSimilarContacts(ctx context.Context, queryVector []float32, userID uuid.UUID, limit int, maxDistance float64) ([]VectorSearchResult, error) {
	return r.searchVectors(ctx, ContactVectorIndex, queryVector, userID, limit, maxDistance)
}

// SearchSimilarInteractions finds similar interactions
func (r *RedisVectorStore) SearchSimilarInteractions(ctx context.Context, queryVector []float32, userID uuid.UUID, limit int, maxDistance float64) ([]VectorSearchResult, error) {
	return r.searchVectors(ctx, InteractionVectorIndex, queryVector, userID, limit, maxDistance)
}

// SearchKnowledge finds relevant knowledge chunks
func (r *RedisVectorStore) SearchKnowledge(ctx context.Context, queryVector []float32, userID uuid.UUID, limit int, maxDistance float64) ([]VectorSearchResult, error) {
	return r.searchVectors(ctx, KnowledgeVectorIndex, queryVector, userID, limit, maxDistance)
}

// SearchKnowledgeOfTypes finds relevant knowledge chunks, restricted to
// documents of the given types. With no types it is SearchKnowledge.
func (r *RedisVectorStore) SearchKnowledgeOfTypes(ctx context.Context, queryVector []float32, userID uuid.UUID, types []string, limit int, maxDistance float64) ([]VectorSearchResult, error) {
	return r.searchVectorsWhere(ctx, KnowledgeVectorIndex, queryVector, userID, typeFilter(types), limit, maxDistance)
}

// SearchSimilarSegments finds similar segments
func (r *RedisVectorStore) SearchSimilarSegments(ctx context.Context, queryVector []float32, userID uuid.UUID, limit int, maxDistance float64) ([]VectorSearchResult, error) {
	return r.searchVectors(ctx, SegmentVectorIndex, queryVector, userID, limit, maxDistance)
}

// searchVectors performs vector similarity search using RediSearch
// searchVectors runs a KNN search scoped to one tenant.
//
// `maxDistance` is a COSINE DISTANCE, not a similarity: 0 is identical and 1 is
// unrelated, so a smaller number is stricter. The distinction is spelled out
// because callers had been passing a value that reads like a similarity — 0.8
// was described as "fairly relevant" but admits everything down to a
// similarity of 0.2, which is no filter at all.
func (r *RedisVectorStore) searchVectors(ctx context.Context, indexName string, queryVector []float32, userID uuid.UUID, limit int, maxDistance float64) ([]VectorSearchResult, error) {
	return r.searchVectorsWhere(ctx, indexName, queryVector, userID, "", limit, maxDistance)
}

// searchVectorsWhere is searchVectors with an extra RediSearch pre-filter,
// applied alongside the tenant filter before the KNN stage.
func (r *RedisVectorStore) searchVectorsWhere(ctx context.Context, indexName string, queryVector []float32, userID uuid.UUID, where string, limit int, maxDistance float64) ([]VectorSearchResult, error) {
	fmt.Printf("[searchVectors] Index: %s, UserID: %s, Limit: %d, MaxDistance: %.2f\n", indexName, userID, limit, maxDistance)

	// Convert query vector to bytes
	vectorBytes := make([]byte, len(queryVector)*4)
	for i, v := range queryVector {
		bits := floatToBits(v)
		vectorBytes[i*4] = byte(bits)
		vectorBytes[i*4+1] = byte(bits >> 8)
		vectorBytes[i*4+2] = byte(bits >> 16)
		vectorBytes[i*4+3] = byte(bits >> 24)
	}

	// The tenant filter belongs in the query, not after it.
	//
	// This used to fetch the global nearest neighbours and then drop the rows
	// belonging to other users in Go, on the grounds that a UUID's hyphens are
	// awkward to escape in RediSearch TAG syntax. But the stored user_id
	// already has its hyphens removed, so it is pure hex and needs no escaping
	// at all — the workaround outlived its reason.
	//
	// Filtering afterwards had two real consequences. Another tenant's
	// documents could occupy the top K and leave this user with fewer results
	// than asked for, or none, so retrieval quality decayed as unrelated data
	// grew. And a vector stored without a user id passed the post-hoc check and
	// was returned to everyone. Pre-filtering removes both.
	cleanUserID := tenantTag(userID)
	query := knnQueryWhere(cleanUserID, where, limit)

	args := []interface{}{
		"FT.SEARCH", indexName,
		query,
		"PARAMS", "2", "vec", vectorBytes,
		"WITHSCORES",
		"SORTBY", "__vector_score",
		"DIALECT", "2",
		"LIMIT", "0", strconv.Itoa(limit),
		// __vector_score is listed explicitly so RediSearch loads it and allows
		// sorting. `user_id` must stay in this list: keepTenant compares
		// against it and now fails closed, so dropping it here would silently
		// return nothing rather than returning too much.
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

	// The query already scoped this to one tenant. Re-checking is defence in
	// depth on a boundary where a mistake means showing one customer another's
	// documents; note that a row with no user id is now dropped rather than
	// shared, which is the opposite of what the old post-hoc filter did.
	results = keepTenant(results, cleanUserID)

	fmt.Printf("[searchVectors] After user filter (%s): %d results\n", cleanUserID, len(results))

	// __vector_score is a cosine distance: lower is better.
	if maxDistance > 0 {
		fmt.Printf("[searchVectors] Applying distance filter: <= %.2f\n", maxDistance)
		for _, r := range results {
			fmt.Printf("[searchVectors] Result: ID=%s, Score=%.4f, Keep=%v\n",
				r.ID, r.Score, r.Score <= maxDistance)
		}
		filtered := keepWithinDistance(results, maxDistance)
		results = filtered
		fmt.Printf("[searchVectors] After distance filter: %d results\n", len(results))
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

					// Parse metadata fields. The top-level "id" is the Redis
					// key ("vector:contact:<uuid>"); the entity's own id is the
					// stored field, which is what the RESP2 path returns too.
					res.Metadata.ID = fmt.Sprintf("%v", extraMap["id"])
					if storedID, ok := extraMap["id"]; ok {
						res.ID = fmt.Sprintf("%v", storedID)
					}
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

	// First element is total count; then each document is its key, followed —
	// because the query asks WITHSCORES — by its text score, then its fields.
	// The score slot is detected rather than assumed, so a reply without
	// WITHSCORES still parses.
	for i := 1; i+1 < len(arr); {
		next := i + 1
		if _, isFields := arr[next].([]interface{}); !isFields && next+1 < len(arr) {
			next++ // skip the WITHSCORES score
		}
		i = next + 1

		fields, ok := arr[next].([]interface{})
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

// tenantTag renders a user id the way it is stored in the index.
//
// The hyphens are removed so the value is pure hexadecimal, which is what lets
// it be used in a RediSearch TAG filter without escaping — TAG treats `-` as a
// token separator, so an unmodified UUID would be read as several tokens.
func tenantTag(userID uuid.UUID) string {
	return strings.ReplaceAll(userID.String(), "-", "")
}

// knnQuery builds a nearest-neighbour query scoped to one tenant.
//
// The tag filter is applied before the KNN stage, so the K returned are the K
// nearest *among this tenant's vectors*. Filtering afterwards instead would let
// another tenant's documents consume the K and leave this one short.
func knnQuery(tenant string, limit int) string {
	return knnQueryWhere(tenant, "", limit)
}

// knnQueryWhere scopes a KNN query to one tenant and, optionally, to an extra
// pre-filter such as a knowledge type.
func knnQueryWhere(tenant, where string, limit int) string {
	if where == "" {
		return fmt.Sprintf("(@user_id:{%s})=>[KNN %d @vector $vec]", tenant, limit)
	}
	return fmt.Sprintf("(@user_id:{%s} %s)=>[KNN %d @vector $vec]", tenant, where, limit)
}

// typeFilter builds a knowledge_type TAG filter. Only the known type names are
// accepted, so no caller-supplied text reaches the query unescaped.
func typeFilter(types []string) string {
	allowed := map[string]bool{"pricing": true, "product": true, "case_study": true, "faq": true, "other": true}
	var kept []string
	for _, t := range types {
		if allowed[t] {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		return ""
	}
	return "@knowledge_type:{" + strings.Join(kept, " | ") + "}"
}

// keepTenant drops anything not belonging to the given tenant.
//
// The query has already scoped the search; this is a second check on a boundary
// where being wrong means showing one customer another's documents. A row whose
// user id is empty is dropped rather than shared.
func keepTenant(results []VectorSearchResult, tenant string) []VectorSearchResult {
	kept := results[:0]
	for _, res := range results {
		if res.Metadata.UserID == tenant {
			kept = append(kept, res)
		}
	}
	return kept
}

// keepWithinDistance drops results further from the query than maxDistance.
// The score is a cosine distance, so smaller is closer.
func keepWithinDistance(results []VectorSearchResult, maxDistance float64) []VectorSearchResult {
	kept := results[:0]
	for _, r := range results {
		if r.Score <= maxDistance {
			kept = append(kept, r)
		}
	}
	return kept
}
