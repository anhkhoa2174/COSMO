package ai

import (
	"os"
	"strconv"
)

// The OpenAI clients in this project can be pointed at any API-compatible
// host. That is one environment variable rather than a code change because of
// how the account is shaped: a single OpenAI key serves intent classification,
// reply drafting, daily actions, contact enrichment and embeddings, so when it
// runs out of credit every AI feature fails together and there is no way to
// keep a demonstration running.
//
// Google's Gemini exposes an OpenAI-compatible endpoint, and its embedding
// model accepts an explicit output width, so it can produce the 1,536
// dimensions the existing vector index is built for. That makes it a usable
// stand-in without rebuilding the index.
//
// It is a stand-in and nothing more. The models are not the same, so the
// output is not the same, and any figure measured through this path describes
// a different system from the one the report evaluates. Anything published
// must say which provider produced it.

// BaseURL returns the API host to use, or "" for the provider's default.
func BaseURL() string {
	return os.Getenv("OPENAI_BASE_URL")
}

// EmbeddingModel returns the embedding model to use.
//
// It has to be settable alongside the base URL: a compatible host will not
// recognise OpenAI's model names, and silently falling back to one produces a
// confusing 404 rather than a clear misconfiguration.
func EmbeddingModel() string {
	if m := os.Getenv("EMBEDDING_MODEL"); m != "" {
		return m
	}
	return "text-embedding-3-small"
}

// EmbeddingDimensions returns the width to request from the embedding model.
//
// The vector index is created at a fixed width, so this must match it. A
// provider whose natural output is wider is asked to truncate; one that cannot
// is not usable without rebuilding the index.
func EmbeddingDimensions() int {
	if v := os.Getenv("EMBEDDING_DIMENSIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 1536
}
