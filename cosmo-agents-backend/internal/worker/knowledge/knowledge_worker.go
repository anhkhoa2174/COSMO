package knowledge

import (
	"context"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/lib/pq"
	"gorm.io/gorm"

	knowledgeRepo "github.com/rockship/cosmo-agents-go/internal/repository/knowledge"
	"github.com/rockship/cosmo-agents-go/internal/service/summary"
	"github.com/rockship/cosmo-agents-go/pkg/ai/splitter"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	queueworker "github.com/rockship/cosmo-agents-go/pkg/worker"
)

const (
	TypeKnowledgeIndexing   = "knowledge:indexing"
	TypeKnowledgesPruning   = "knowledge:pruning"
	TypeKnowledgesSummarize = "knowledge:summarize"
)

// KnowledgeIndexingPayload represents the payload for knowledge indexing.
type KnowledgeIndexingPayload struct {
	Collection   string                 `json:"collection"`
	Content      string                 `json:"content"`
	EmbeddingGID string                 `json:"embedding_gid"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// KnowledgesPruningPayload represents the payload for pruning embeddings.
type KnowledgesPruningPayload struct {
	Collection    string   `json:"collection"`
	EmbeddingGIDs []string `json:"embedding_gids,omitempty"`
}

// KnowledgesSummarizePayload represents the payload for summarizing knowledge.
type KnowledgesSummarizePayload struct {
	Content        string                 `json:"content"`
	EmbeddingGID   string                 `json:"embedding_gid"`
	Attributes     map[string]interface{} `json:"attributes,omitempty"`
	ForceSummarize bool                   `json:"force_summarize,omitempty"`
}

// Worker handles knowledge-related background tasks.
type Worker struct {
	db            *gorm.DB
	knowledgeRepo *knowledgeRepo.KnowledgeRepository
	vectorStore   VectorStore
	summarizer    Summarizer
	textSplitter  splitter.Splitter
}

// VectorStore interface for vector database operations.
type VectorStore interface {
	AddDocuments(ctx context.Context, collection string, documents []string, metadatas []map[string]interface{}, ids []string) error
	DeleteDocuments(ctx context.Context, collection string, filter map[string]interface{}) error
}

// Summarizer interface for text summarization.
type Summarizer interface {
	Summarize(ctx context.Context, content string, detail float64, recursive bool) ([]summary.SummaryPair, error)
}

// New creates a new knowledge worker.
func New(db *gorm.DB, vectorStore VectorStore, summarizer Summarizer) *Worker {
	// Create text splitter with default config
	textSplitter := splitter.NewRecursiveCharacterSplitter(splitter.RecursiveConfig{
		ChunkSize:    300,
		ChunkOverlap: 100,
	})

	return &Worker{
		db:            db,
		knowledgeRepo: knowledgeRepo.NewKnowledgeRepository(db),
		vectorStore:   vectorStore,
		summarizer:    summarizer,
		textSplitter:  textSplitter,
	}
}

// HandleKnowledgeIndexing processes knowledge indexing tasks.
func (w *Worker) HandleKnowledgeIndexing(ctx context.Context, task *asynq.Task) error {
	var payload KnowledgeIndexingPayload
	if err := queueworker.ParsePayload(task, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("collection", payload.Collection).
		Str("embedding_gid", payload.EmbeddingGID).
		Msg("Processing knowledge indexing")

	// Validate content
	content := strings.TrimSpace(payload.Content)
	if content == "" {
		return fmt.Errorf("processed content is empty")
	}

	// Split content into chunks
	documents, err := w.textSplitter.Split(content)
	if err != nil {
		return fmt.Errorf("failed to split content: %w", err)
	}

	if len(documents) == 0 {
		logger.FromContext(ctx).Info().Msg("Skipping indexing, no documents found")
		return nil
	}

	// Prepare metadata for each chunk
	metadatas := make([]map[string]interface{}, len(documents))
	documentIDs := make([]string, len(documents))

	for idx := range documents {
		chunkID := fmt.Sprintf("%s_%d", payload.EmbeddingGID, idx)
		documentIDs[idx] = chunkID

		// Merge base metadata with chunk-specific metadata
		metadatas[idx] = make(map[string]interface{})
		for k, v := range payload.Metadata {
			metadatas[idx][k] = v
		}
		metadatas[idx]["id"] = chunkID
		metadatas[idx]["embedding_gid"] = payload.EmbeddingGID
		metadatas[idx]["chunk_index"] = idx
	}

	// Add documents to vector store (skip if not implemented)
	if w.vectorStore != nil {
		if err := w.vectorStore.AddDocuments(ctx, payload.Collection, documents, metadatas, documentIDs); err != nil {
			return fmt.Errorf("failed to add documents to vector store: %w", err)
		}
	} else {
		logger.FromContext(ctx).Info().Msg("Vector store not configured - skipping indexing")
	}

	logger.FromContext(ctx).Info().
		Str("collection", payload.Collection).
		Int("chunks", len(documents)).
		Msg("Successfully indexed knowledge")

	return nil
}

// HandleKnowledgesPruning processes knowledge pruning tasks.
func (w *Worker) HandleKnowledgesPruning(ctx context.Context, task *asynq.Task) error {
	var payload KnowledgesPruningPayload
	if err := queueworker.ParsePayload(task, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("collection", payload.Collection).
		Int("embedding_gids_count", len(payload.EmbeddingGIDs)).
		Msg("Processing knowledge pruning")

	// Build filter
	filter := make(map[string]interface{})
	if len(payload.EmbeddingGIDs) > 0 {
		filter["must"] = []map[string]interface{}{
			{
				"key": "gid",
				"match": map[string]interface{}{
					"any": payload.EmbeddingGIDs,
				},
			},
		}
	}

	// Delete documents from vector store (skip if not implemented)
	if w.vectorStore != nil {
		if err := w.vectorStore.DeleteDocuments(ctx, payload.Collection, filter); err != nil {
			return fmt.Errorf("failed to delete documents: %w", err)
		}
	} else {
		logger.FromContext(ctx).Info().Msg("Vector store not configured - skipping pruning")
	}

	logger.FromContext(ctx).Info().
		Str("collection", payload.Collection).
		Int("count", len(payload.EmbeddingGIDs)).
		Msg("Successfully pruned knowledge")

	return nil
}

// HandleKnowledgesSummarize processes knowledge summarization tasks.
func (w *Worker) HandleKnowledgesSummarize(ctx context.Context, task *asynq.Task) error {
	var payload KnowledgesSummarizePayload
	if err := queueworker.ParsePayload(task, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("embedding_gid", payload.EmbeddingGID).
		Msg("Processing knowledge summarization")

	// Find knowledge by embedding GID
	knowledge, err := w.knowledgeRepo.GetByEmbeddingGID(ctx, payload.EmbeddingGID)
	if err != nil {
		return fmt.Errorf("failed to find knowledge: %w", err)
	}

	if knowledge == nil {
		logger.FromContext(ctx).Info().
			Str("embedding_gid", payload.EmbeddingGID).
			Msg("Skipping summary, knowledge not found")
		return nil
	}

	// Check if already summarized
	if !payload.ForceSummarize && len(knowledge.SummaryPair) > 0 {
		logger.FromContext(ctx).Info().
			Str("embedding_gid", payload.EmbeddingGID).
			Msg("Skipping summary, already summarized")
		return nil
	}

	// Check if summarizer is initialized
	if w.summarizer == nil {
		return fmt.Errorf("summarizer not initialized")
	}

	// Generate summary
	summaryPairs, err := w.summarizer.Summarize(ctx, payload.Content, 0.1, true)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Convert SummaryPair to string array with proper indexing
	summaryArray := make([]string, len(summaryPairs)*2)
	for i, pair := range summaryPairs {
		summaryArray[i*2] = pair.Compressed // Even indices: compressed
		summaryArray[i*2+1] = pair.Raw      // Odd indices: raw
	}

	// Prepare update attributes with map copy to prevent concurrent modification
	updates := make(map[string]interface{}, len(payload.Attributes))
	for k, v := range payload.Attributes {
		updates[k] = v
	}
	updates["summary_pair"] = pq.StringArray(summaryArray)

	logger.FromContext(ctx).Info().
		Str("embedding_gid", payload.EmbeddingGID).
		Interface("updates", updates).
		Msg("Attempting to update knowledge")

	// Update knowledge with race condition mitigation
	if err := w.knowledgeRepo.UpdateByEmbeddingGID(ctx, payload.EmbeddingGID, updates); err != nil {
		logger.FromContext(ctx).Error().
			Err(err).
			Str("embedding_gid", payload.EmbeddingGID).
			Msg("Failed to update knowledge")
		return fmt.Errorf("failed to update knowledge: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("embedding_gid", payload.EmbeddingGID).
		Int("summary_length", len(summaryArray)).
		Msg("Successfully summarized knowledge")

	return nil
}

// RegisterKnowledgeTasks registers all knowledge-related tasks.
func (w *Worker) RegisterKnowledgeTasks(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeKnowledgeIndexing, w.HandleKnowledgeIndexing)
	mux.HandleFunc(TypeKnowledgesPruning, w.HandleKnowledgesPruning)
	mux.HandleFunc(TypeKnowledgesSummarize, w.HandleKnowledgesSummarize)
}

// EnqueueKnowledgeIndexing enqueues a knowledge indexing task.
func EnqueueKnowledgeIndexing(client *asynq.Client, payload KnowledgeIndexingPayload) error {
	task, err := queueworker.NewTask(TypeKnowledgeIndexing, payload)
	if err != nil {
		return err
	}

	_, err = client.Enqueue(task)
	return err
}

// EnqueueKnowledgesPruning enqueues a knowledge pruning task.
func EnqueueKnowledgesPruning(client *asynq.Client, payload KnowledgesPruningPayload) error {
	task, err := queueworker.NewTask(TypeKnowledgesPruning, payload)
	if err != nil {
		return err
	}

	_, err = client.Enqueue(task)
	return err
}

// EnqueueKnowledgesSummarize enqueues a knowledge summarization task.
func EnqueueKnowledgesSummarize(client *asynq.Client, payload KnowledgesSummarizePayload) error {
	task, err := queueworker.NewTask(TypeKnowledgesSummarize, payload)
	if err != nil {
		return err
	}

	_, err = client.Enqueue(task)
	return err
}
