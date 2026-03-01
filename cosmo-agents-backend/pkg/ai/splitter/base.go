package splitter

// Splitter is the interface for text splitting strategies.
type Splitter interface {
	// Split divides text into chunks based on the splitter's strategy.
	Split(text string) ([]string, error)

	// SplitDocuments splits multiple documents into chunks.
	SplitDocuments(documents []string) ([]string, error)
}

// Config represents base configuration for text splitters.
type Config struct {
	ChunkSize        int  // Maximum size of each chunk
	ChunkOverlap     int  // Number of characters to overlap between chunks
	IsSeparatorRegex bool // Whether the separator is a regex pattern
}
