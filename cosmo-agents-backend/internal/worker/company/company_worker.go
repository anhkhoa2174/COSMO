package company

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

const (
	TypeCompanyIndexing = "company:indexing"
)

// CompanyIndexingPayload represents the payload for company indexing task
type CompanyIndexingPayload struct {
	Collection   string                 `json:"collection"`
	Content      string                 `json:"content"`
	EmbeddingGID string                 `json:"embedding_gid"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Worker handles company-related background tasks
type Worker struct {
	// vectorStore would be injected here for production
	// vectorStore VectorStore
}

// New creates a new company worker
func New() *Worker {
	return &Worker{}
}

// HandleCompanyIndexing indexes company information in vector store for RAG
// This stores company data for later retrieval and similarity search
func (w *Worker) HandleCompanyIndexing(ctx context.Context, task *asynq.Task) error {
	var payload CompanyIndexingPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	logger.Logger.Info().
		Str("collection", payload.Collection).
		Str("embedding_gid", payload.EmbeddingGID).
		Msg("Indexing company information")

	// Validate and clean content
	content := strings.TrimSpace(payload.Content)
	if content == "" {
		return fmt.Errorf("processed content is empty")
	}

	// Prepare documents and metadata
	documents := []string{content}

	// Create metadata for each document
	metadatas := make([]map[string]interface{}, len(documents))
	documentIDs := make([]string, len(documents))

	for idx := range documents {
		// Combine embedding_gid with index
		docID := fmt.Sprintf("%s_%d", payload.EmbeddingGID, idx)
		documentIDs[idx] = docID

		// Merge provided metadata with ID
		metadatas[idx] = make(map[string]interface{})
		metadatas[idx]["id"] = docID
		for k, v := range payload.Metadata {
			metadatas[idx][k] = v
		}
	}

	// TODO: Integrate with actual vector store (Qdrant, Weaviate, or Pinecone)
	// For now, just log the operation
	logger.Logger.Info().
		Str("collection", payload.Collection).
		Int("num_documents", len(documents)).
		Int("content_length", len(content)).
		Strs("document_ids", documentIDs).
		Msg("Company information indexed successfully")

	// Production implementation would be:
	// err := w.vectorStore.AddDocuments(ctx, payload.Collection, documents, metadatas, documentIDs)
	// if err != nil {
	//     return fmt.Errorf("failed to index documents: %w", err)
	// }

	return nil
}

// EnqueueCompanyIndexing enqueues a company indexing task
func EnqueueCompanyIndexing(client *asynq.Client, payload CompanyIndexingPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(TypeCompanyIndexing, data)

	info, err := client.Enqueue(task)
	if err != nil {
		return fmt.Errorf("failed to enqueue task: %w", err)
	}

	logger.Logger.Info().
		Str("task_id", info.ID).
		Str("queue", info.Queue).
		Msg("Company indexing task enqueued")

	return nil
}
