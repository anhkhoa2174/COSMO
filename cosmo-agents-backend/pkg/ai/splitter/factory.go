package splitter

import (
	"fmt"
	"strings"
)

// SplitterType represents supported splitter types.
type SplitterType string

const (
	SplitterTypeRecursiveCharacter SplitterType = "recursive_character"
)

// GetSplitter creates a splitter instance based on the provided configuration.
func GetSplitter(splitterType SplitterType, config interface{}) (Splitter, error) {
	switch strings.ToLower(string(splitterType)) {
	case "recursive_character":
		cfg, ok := config.(RecursiveConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for RecursiveCharacterSplitter")
		}
		return NewRecursiveCharacterSplitter(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported splitter type: %s", splitterType)
	}
}

// GetSupportedSplitters returns a list of supported splitter types.
func GetSupportedSplitters() []SplitterType {
	return []SplitterType{
		SplitterTypeRecursiveCharacter,
	}
}

// GetSplitterSchema returns configuration schema information for splitters.
func GetSplitterSchema(splitterType SplitterType) (map[string]interface{}, error) {
	switch splitterType {
	case SplitterTypeRecursiveCharacter:
		return map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"chunk_size": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum size of each chunk",
					"default":     200,
				},
				"chunk_overlap": map[string]interface{}{
					"type":        "integer",
					"description": "Number of characters to overlap between chunks",
					"default":     30,
				},
				"separators": map[string]interface{}{
					"type":        "array",
					"description": "List of separators to use for splitting",
					"default":     []string{"\n\n", "\n", " ", ""},
				},
				"is_separator_regex": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether separators are regex patterns",
					"default":     false,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported splitter type: %s", splitterType)
	}
}
