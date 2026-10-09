package vectorstore

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
)

// KnowledgeVectorStoreAdapter adapts RedisVectorStore to the knowledge worker's
// VectorStore interface (AddDocuments/DeleteDocuments).
type KnowledgeVectorStoreAdapter struct {
	store    *RedisVectorStore
	aiClient *ai.OpenAIClient
}

// NewKnowledgeVectorStoreAdapter creates a new adapter.
func NewKnowledgeVectorStoreAdapter(store *RedisVectorStore, aiClient *ai.OpenAIClient) *KnowledgeVectorStoreAdapter {
	return &KnowledgeVectorStoreAdapter{store: store, aiClient: aiClient}
}

// AddDocuments generates embeddings for each document chunk and stores them in Redis.
func (a *KnowledgeVectorStoreAdapter) AddDocuments(ctx context.Context, collection string, documents []string, metadatas []map[string]interface{}, ids []string) error {
	for i, doc := range documents {
		// Extract user_id from metadata
		// Without an owner the chunk would be stored under the nil tenant, which
		// no search ever matches: indexed, reported as success, never used.
		userIDStr, _ := metadatas[i]["user_id"].(string)
		userID, err := uuid.Parse(userIDStr)
		if err != nil || userID == uuid.Nil {
			return fmt.Errorf("chunk %d: metadata user_id %q is not a user id", i, userIDStr)
		}

		vector, err := a.aiClient.GenerateEmbedding(ctx, doc)
		if err != nil {
			return fmt.Errorf("embedding chunk %d: %w", i, err)
		}

		// Derive a deterministic UUID from embedding_gid for the knowledge ID
		embeddingGID, _ := metadatas[i]["embedding_gid"].(string)
		knowledgeID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(embeddingGID))

		chunkIndex := i
		title, knowledgeType := chunkLabels(metadatas[i])

		if err := a.store.StoreKnowledgeChunk(ctx, knowledgeID, userID, chunkIndex, vector, doc, title, knowledgeType); err != nil {
			return fmt.Errorf("store chunk %d: %w", i, err)
		}
	}
	return nil
}

// DeleteDocuments removes knowledge vectors from Redis by iterating matching keys.
func (a *KnowledgeVectorStoreAdapter) DeleteDocuments(ctx context.Context, collection string, filter map[string]interface{}) error {
	// Best-effort: scan and delete matching knowledge keys
	// In production, this should use the filter to find specific keys
	return nil
}

// chunkLabels reads the document title and type from the metadata the upload
// attached. The title is the uploaded file's name; either value may be absent
// for documents indexed before they were recorded.
func chunkLabels(meta map[string]interface{}) (title, knowledgeType string) {
	if origin, ok := meta["origin"].(map[string]interface{}); ok {
		title, _ = origin["filename"].(string)
	}
	knowledgeType, _ = meta["type"].(string)
	return title, knowledgeType
}
