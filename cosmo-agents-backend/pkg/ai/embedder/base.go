package embedder

import (
	"context"
)

// Embedder is the interface for text embedding models.
type Embedder interface {
	// Embed generates embeddings for the given texts.
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedSingle generates an embedding for a single text.
	EmbedSingle(ctx context.Context, text string) ([]float32, error)

	// Dimensions returns the dimension size of embeddings.
	Dimensions() int

	// ModelName returns the name of the embedding model.
	ModelName() string
}

// Config represents base configuration for embedders.
type Config struct {
	Model      string
	Dimensions int
	APIKey     string
}

// Factory creates embedder instances based on configuration.
type Factory interface {
	CreateEmbedder(config Config) (Embedder, error)
}
