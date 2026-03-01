package embedder

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// OpenAIEmbedder implements the Embedder interface using OpenAI's embedding API.
type OpenAIEmbedder struct {
	client     *openai.Client
	model      string
	dimensions int
}

// OpenAIConfig holds OpenAI-specific embedder configuration.
type OpenAIConfig struct {
	Model      string // Default: "text-embedding-3-small"
	Dimensions int    // Default: 768
	APIKey     string
}

// NewOpenAIEmbedder creates a new OpenAI embedder.
func NewOpenAIEmbedder(config OpenAIConfig) (*OpenAIEmbedder, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	// Set defaults
	if config.Model == "" {
		config.Model = "text-embedding-3-small"
	}
	if config.Dimensions == 0 {
		config.Dimensions = 768
	}

	client := openai.NewClient(config.APIKey)

	return &OpenAIEmbedder{
		client:     client,
		model:      config.Model,
		dimensions: config.Dimensions,
	}, nil
}

// Embed generates embeddings for multiple texts.
func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided for embedding")
	}

	req := openai.EmbeddingRequest{
		Input:      texts,
		Model:      openai.EmbeddingModel(e.model),
		Dimensions: e.dimensions,
	}

	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("expected %d embeddings, got %d", len(texts), len(resp.Data))
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// EmbedSingle generates an embedding for a single text.
func (e *OpenAIEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return embeddings[0], nil
}

// Dimensions returns the dimension size of embeddings.
func (e *OpenAIEmbedder) Dimensions() int {
	return e.dimensions
}

// ModelName returns the name of the embedding model.
func (e *OpenAIEmbedder) ModelName() string {
	return e.model
}
