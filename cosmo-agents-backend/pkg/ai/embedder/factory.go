package embedder

import (
	"fmt"
	"strings"
)

// EmbedderType represents supported embedder types.
type EmbedderType string

const (
	EmbedderTypeOpenAI EmbedderType = "openai"
)

// GetEmbedder creates an embedder instance based on the provided configuration.
func GetEmbedder(embedderType EmbedderType, config interface{}) (Embedder, error) {
	switch strings.ToLower(string(embedderType)) {
	case "openai":
		cfg, ok := config.(OpenAIConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for OpenAI embedder")
		}
		return NewOpenAIEmbedder(cfg)
	default:
		return nil, fmt.Errorf("unsupported embedder type: %s", embedderType)
	}
}

// GetSupportedEmbedders returns a list of supported embedder types.
func GetSupportedEmbedders() []EmbedderType {
	return []EmbedderType{
		EmbedderTypeOpenAI,
	}
}

// GetEmbedderSchema returns configuration schema information for embedders.
func GetEmbedderSchema(embedderType EmbedderType) (map[string]interface{}, error) {
	switch embedderType {
	case EmbedderTypeOpenAI:
		return map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"model": map[string]interface{}{
					"type":        "string",
					"description": "OpenAI embedding model name",
					"default":     "text-embedding-3-small",
				},
				"dimensions": map[string]interface{}{
					"type":        "integer",
					"description": "Dimension size of embeddings",
					"default":     768,
				},
				"api_key": map[string]interface{}{
					"type":        "string",
					"description": "OpenAI API key",
				},
			},
			"required": []string{"api_key"},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported embedder type: %s", embedderType)
	}
}
