package skills

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/vectorstore"
)

// KnowledgeSearchResult represents a single knowledge search result with text and score.
type KnowledgeSearchResult struct {
	ChunkText string
	Score     float64

	// Title is the name of the document the chunk came from, and Type its
	// knowledge type. Both are empty for chunks indexed before they were
	// recorded.
	Title string
	Type  string
}

// preferredTypes maps an inbound reply's intent to the knowledge types most
// likely to answer it. Intents not listed have no preference.
var preferredTypes = map[string][]string{
	"Request for pricing":     {"pricing"},
	"Request for information": {"product", "faq"},
	"Interested":              {"product", "case_study"},
}

// PreferredTypes returns the knowledge types retrieval should try first for an
// intent, or nil when there is no preference.
func PreferredTypes(intent string) []string {
	return preferredTypes[intent]
}

// KnowledgeSearchSkill searches the knowledge vector store using semantic similarity.
type KnowledgeSearchSkill struct {
	client      *ai.OpenAIClient
	vectorStore *vectorstore.RedisVectorStore
}

// NewKnowledgeSearchSkill creates a new knowledge search skill.
func NewKnowledgeSearchSkill(client *ai.OpenAIClient, vectorStore *vectorstore.RedisVectorStore) *KnowledgeSearchSkill {
	return &KnowledgeSearchSkill{
		client:      client,
		vectorStore: vectorStore,
	}
}

// maxChunkDistance is the furthest a chunk may sit from the query and still be
// used as grounding, measured as cosine distance — 0 is identical, 1 is
// unrelated.
//
// This was 0.8, described in a comment as "fairly relevant". It is not: a
// distance of 0.8 is a similarity of 0.2, which admits essentially any chunk in
// the store. The effect was that the five nearest chunks were always injected
// into the prompt whether or not they had anything to do with the question,
// which is the failure mode grounding is supposed to prevent — the model is
// handed irrelevant text and asked to answer from it.
//
// 0.5 keeps chunks at a similarity of 0.5 or better and drops the clearly
// unrelated. It is a starting point rather than a tuned figure: the right value
// depends on the documents a team actually uploads, and choosing it properly
// needs a labelled retrieval set, which the project does not yet have. Erring
// strict is the safer direction — too few chunks makes the model say it does not
// know, too many makes it answer confidently from the wrong passage.
const maxChunkDistance = 0.5

// Search finds relevant knowledge chunks for the given query text, scoped to a
// user.
func (s *KnowledgeSearchSkill) Search(ctx context.Context, userID uuid.UUID, query string, limit int) ([]KnowledgeSearchResult, error) {
	if s == nil || s.client == nil || s.vectorStore == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 5
	}

	// Generate embedding for the query
	queryVector, err := s.client.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	results, err := s.vectorStore.SearchKnowledge(ctx, queryVector, userID, limit, maxChunkDistance)
	if err != nil {
		return nil, fmt.Errorf("knowledge search failed: %w", err)
	}
	return toKnowledgeResults(results), nil
}

// SearchForIntent searches the documents whose type suits the reply's intent
// first — the pricing sheet for a pricing question — and falls back to every
// document when those yield nothing. The fallback matters: most documents
// are uploaded without a type, and a preference must never turn into
// "answer from nothing".
func (s *KnowledgeSearchSkill) SearchForIntent(ctx context.Context, userID uuid.UUID, query, intent string, limit int) ([]KnowledgeSearchResult, error) {
	if s == nil || s.client == nil || s.vectorStore == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}

	types := PreferredTypes(intent)
	if len(types) == 0 {
		return s.Search(ctx, userID, query, limit)
	}

	queryVector, err := s.client.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	typed, err := s.vectorStore.SearchKnowledgeOfTypes(ctx, queryVector, userID, types, limit, maxChunkDistance)
	if err == nil {
		if found := toKnowledgeResults(typed); len(found) > 0 {
			return found, nil
		}
	}

	all, err := s.vectorStore.SearchKnowledge(ctx, queryVector, userID, limit, maxChunkDistance)
	if err != nil {
		return nil, fmt.Errorf("knowledge search failed: %w", err)
	}
	return toKnowledgeResults(all), nil
}

// toKnowledgeResults keeps the chunks that carry text, with their document
// title and type.
func toKnowledgeResults(results []vectorstore.VectorSearchResult) []KnowledgeSearchResult {
	out := make([]KnowledgeSearchResult, 0, len(results))
	for _, r := range results {
		extra := r.Metadata.Extra
		text, _ := extra["chunk_text"].(string)
		if text == "" {
			continue
		}
		title, _ := extra["title"].(string)
		typ, _ := extra["knowledge_type"].(string)
		out = append(out, KnowledgeSearchResult{ChunkText: text, Score: r.Score, Title: title, Type: typ})
	}
	return out
}
