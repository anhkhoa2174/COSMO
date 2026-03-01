package splitter

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// RecursiveCharacterSplitter implements recursive character-based text splitting.
type RecursiveCharacterSplitter struct {
	chunkSize        int
	chunkOverlap     int
	separators       []string
	isSeparatorRegex bool
}

// RecursiveConfig holds configuration for RecursiveCharacterSplitter.
type RecursiveConfig struct {
	ChunkSize        int      // Default: 200
	ChunkOverlap     int      // Default: 30
	Separators       []string // Default: ["\n\n", "\n", " ", ""]
	IsSeparatorRegex bool     // Default: false
}

// NewRecursiveCharacterSplitter creates a new RecursiveCharacterSplitter.
func NewRecursiveCharacterSplitter(config RecursiveConfig) *RecursiveCharacterSplitter {
	// Set defaults
	if config.ChunkSize == 0 {
		config.ChunkSize = 200
	}
	if config.ChunkOverlap == 0 {
		config.ChunkOverlap = 30
	}
	if len(config.Separators) == 0 {
		config.Separators = []string{"\n\n", "\n", " ", ""}
	}

	return &RecursiveCharacterSplitter{
		chunkSize:        config.ChunkSize,
		chunkOverlap:     config.ChunkOverlap,
		separators:       config.Separators,
		isSeparatorRegex: config.IsSeparatorRegex,
	}
}

// Split divides text into chunks using recursive character splitting.
func (s *RecursiveCharacterSplitter) Split(text string) ([]string, error) {
	if text == "" {
		return []string{}, nil
	}

	return s.splitText(text, s.separators), nil
}

// SplitDocuments splits multiple documents into chunks.
func (s *RecursiveCharacterSplitter) SplitDocuments(documents []string) ([]string, error) {
	var allChunks []string

	for _, doc := range documents {
		chunks, err := s.Split(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to split document: %w", err)
		}
		allChunks = append(allChunks, chunks...)
	}

	return allChunks, nil
}

// splitText recursively splits text using the provided separators.
func (s *RecursiveCharacterSplitter) splitText(text string, separators []string) []string {
	var finalChunks []string

	// Base case: if text is small enough, return it
	if utf8.RuneCountInString(text) <= s.chunkSize {
		return []string{text}
	}

	// No separators left, split by character count
	if len(separators) == 0 {
		return s.splitBySize(text)
	}

	// Try current separator
	separator := separators[0]
	var splits []string

	if s.isSeparatorRegex {
		re, err := regexp.Compile(separator)
		if err == nil {
			splits = re.Split(text, -1)
		} else {
			// Fallback to literal split if regex fails
			splits = strings.Split(text, separator)
		}
	} else {
		if separator == "" {
			// Split by characters
			splits = s.splitBySize(text)
		} else {
			splits = strings.Split(text, separator)
		}
	}

	// Merge splits into chunks
	var currentChunk strings.Builder
	var chunks []string

	for _, split := range splits {
		if split == "" {
			continue
		}

		splitLen := utf8.RuneCountInString(split)
		currentLen := utf8.RuneCountInString(currentChunk.String())

		if currentLen+splitLen+1 <= s.chunkSize {
			// Add to current chunk
			if currentChunk.Len() > 0 && separator != "" {
				currentChunk.WriteString(separator)
			}
			currentChunk.WriteString(split)
		} else {
			// Current chunk is full, save it
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
			}

			// If split is still too large, recursively split it
			if splitLen > s.chunkSize {
				subChunks := s.splitText(split, separators[1:])
				chunks = append(chunks, subChunks...)
			} else {
				currentChunk.WriteString(split)
			}
		}
	}

	// Add remaining chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	// Apply overlap
	finalChunks = s.applyOverlap(chunks)

	return finalChunks
}

// splitBySize splits text into chunks of specified size.
func (s *RecursiveCharacterSplitter) splitBySize(text string) []string {
	var chunks []string
	runes := []rune(text)

	for i := 0; i < len(runes); i += s.chunkSize {
		end := i + s.chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}

	return chunks
}

// applyOverlap applies chunk overlap to the chunks.
func (s *RecursiveCharacterSplitter) applyOverlap(chunks []string) []string {
	if s.chunkOverlap == 0 || len(chunks) <= 1 {
		return chunks
	}

	var result []string

	for i, chunk := range chunks {
		if i == 0 {
			result = append(result, chunk)
			continue
		}

		// Get overlap from previous chunk
		prevChunk := chunks[i-1]
		prevRunes := []rune(prevChunk)

		var overlap string
		if len(prevRunes) > s.chunkOverlap {
			overlap = string(prevRunes[len(prevRunes)-s.chunkOverlap:])
		} else {
			overlap = prevChunk
		}

		// Prepend overlap to current chunk
		combined := overlap + chunk
		result = append(result, combined)
	}

	return result
}
