package skills

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/vectorstore"
)

// EmbeddingSkill generates embeddings and stores them in Redis vector store.
type EmbeddingSkill struct {
	client      *ai.OpenAIClient
	vectorStore *vectorstore.RedisVectorStore
}

// NewEmbeddingSkill creates a new embedding skill. If client is nil, GenerateAndStore becomes a no-op.
func NewEmbeddingSkill(client *ai.OpenAIClient, vectorStore *vectorstore.RedisVectorStore) *EmbeddingSkill {
	return &EmbeddingSkill{
		client:      client,
		vectorStore: vectorStore,
	}
}

// GenerateAndStore creates an embedding for the given text and stores it in Redis.
func (s *EmbeddingSkill) GenerateAndStore(ctx context.Context, contactID uuid.UUID, userID uuid.UUID, text string, metadata map[string]interface{}) ([]float32, error) {
	if s == nil || s.client == nil {
		// No client configured; skip without error so agents remain functional in offline/dev mode.
		return nil, nil
	}

	// Generate embedding using OpenAI
	vector, err := s.client.GenerateEmbedding(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Store in Redis vector store
	if s.vectorStore != nil {
		if err := s.vectorStore.StoreContactVector(ctx, contactID, userID, vector, metadata); err != nil {
			return nil, fmt.Errorf("failed to store vector: %w", err)
		}
	}

	return vector, nil
}

// GenerateInteractionEmbedding creates and stores an interaction embedding
func (s *EmbeddingSkill) GenerateInteractionEmbedding(ctx context.Context, interactionID uuid.UUID, userID uuid.UUID, text string, metadata map[string]interface{}) ([]float32, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}

	vector, err := s.client.GenerateEmbedding(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if s.vectorStore != nil {
		if err := s.vectorStore.StoreInteractionVector(ctx, interactionID, userID, vector, metadata); err != nil {
			return nil, fmt.Errorf("failed to store interaction vector: %w", err)
		}
	}

	return vector, nil
}

// GenerateKnowledgeEmbedding creates and stores a knowledge chunk embedding
func (s *EmbeddingSkill) GenerateKnowledgeEmbedding(ctx context.Context, knowledgeID uuid.UUID, userID uuid.UUID, chunkIndex int, chunkText string) ([]float32, error) {
	if s == nil || s.client == nil {
		return nil, nil
	}

	vector, err := s.client.GenerateEmbedding(ctx, chunkText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if s.vectorStore != nil {
		if err := s.vectorStore.StoreKnowledgeVector(ctx, knowledgeID, userID, chunkIndex, vector, chunkText); err != nil {
			return nil, fmt.Errorf("failed to store knowledge vector: %w", err)
		}
	}

	return vector, nil
}
