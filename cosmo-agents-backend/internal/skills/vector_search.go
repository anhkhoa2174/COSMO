package skills

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/vectorstore"
)

// VectorSearchSkill provides semantic search capabilities across contacts, interactions, knowledge, and segments.
type VectorSearchSkill struct {
	vectorStore *vectorstore.RedisVectorStore
	openAI      *ai.OpenAIClient
}

// NewVectorSearchSkill creates a new vector search skill.
func NewVectorSearchSkill(vectorStore *vectorstore.RedisVectorStore, openAI *ai.OpenAIClient) *VectorSearchSkill {
	return &VectorSearchSkill{
		vectorStore: vectorStore,
		openAI:      openAI,
	}
}

// SearchSimilarContactsResult represents a contact search result with additional context.
type SearchSimilarContactsResult struct {
	ContactID  string                 `json:"contact_id"`
	Similarity float64                `json:"similarity"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// SearchContactsByText performs semantic search using natural language query.
// Example: "Find contacts interested in AI and machine learning"
func (s *VectorSearchSkill) SearchContactsByText(ctx context.Context, query string, userID uuid.UUID, limit int, threshold float64) ([]SearchSimilarContactsResult, error) {
	fmt.Printf("[SearchContactsByText] Query: '%s', UserID: %s\n", query, userID)

	if s.openAI == nil {
		fmt.Printf("[SearchContactsByText] ERROR: OpenAI client is nil\n")
		return nil, fmt.Errorf("OpenAI client not configured")
	}

	// Generate embedding for the search query
	queryVector, err := s.openAI.GenerateEmbedding(ctx, query)
	if err != nil {
		fmt.Printf("[SearchContactsByText] Failed to generate embedding: %v\n", err)
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	fmt.Printf("[SearchContactsByText] Generated embedding vector with %d dimensions\n", len(queryVector))

	return s.SearchContactsByVector(ctx, queryVector, userID, limit, threshold)
}

// SearchContactsByVector performs semantic search using a pre-computed embedding vector.
func (s *VectorSearchSkill) SearchContactsByVector(ctx context.Context, queryVector []float32, userID uuid.UUID, limit int, threshold float64) ([]SearchSimilarContactsResult, error) {
	fmt.Printf("[SearchContactsByVector] UserID: %s, Vector dim: %d, Limit: %d, Threshold: %.2f\n", userID, len(queryVector), limit, threshold)

	results, err := s.vectorStore.SearchSimilarContacts(ctx, queryVector, userID, limit, threshold)
	if err != nil {
		fmt.Printf("[SearchContactsByVector] SearchSimilarContacts ERROR: %v\n", err)
		return nil, err
	}

	fmt.Printf("[SearchContactsByVector] Got %d results from vectorStore\n", len(results))

	searchResults := make([]SearchSimilarContactsResult, len(results))
	for i, r := range results {
		similarity := distanceToSimilarity(r.Score)

		searchResults[i] = SearchSimilarContactsResult{
			ContactID:  r.ID,
			Similarity: similarity,
			Metadata:   r.Metadata.Extra,
		}
	}

	return searchResults, nil
}

// FindSimilarContact finds contacts similar to a given contact.
// Use case: "Find more contacts like this high-value customer"
func (s *VectorSearchSkill) FindSimilarContact(ctx context.Context, contactID uuid.UUID, userID uuid.UUID, limit int, threshold float64) ([]SearchSimilarContactsResult, error) {
	// TODO: Implement GetVector method in RedisVectorStore to retrieve the contact's embedding
	// Then use that embedding to search for similar contacts
	// For now, return not implemented error
	_ = vectorstore.ContactVectorPrefix + contactID.String() // Will be used when GetVector is implemented
	return nil, fmt.Errorf("not implemented: need GetVector method in RedisVectorStore")
}

// SearchKnowledgeResult represents a knowledge search result.
type SearchKnowledgeResult struct {
	KnowledgeID string                 `json:"knowledge_id"`
	ChunkIndex  int                    `json:"chunk_index"`
	ChunkText   string                 `json:"chunk_text"`
	Similarity  float64                `json:"similarity"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SearchKnowledge performs semantic search across the knowledge base.
// Use case: RAG (Retrieval Augmented Generation) for AI enrichment
// Example: "What do we know about enterprise SaaS pricing?"
func (s *VectorSearchSkill) SearchKnowledge(ctx context.Context, query string, userID uuid.UUID, limit int, threshold float64) ([]SearchKnowledgeResult, error) {
	if s.openAI == nil {
		return nil, fmt.Errorf("OpenAI client not configured")
	}

	// Generate embedding for the search query
	queryVector, err := s.openAI.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search knowledge base
	results, err := s.vectorStore.SearchKnowledge(ctx, queryVector, userID, limit, threshold)
	if err != nil {
		return nil, err
	}

	searchResults := make([]SearchKnowledgeResult, len(results))
	for i, r := range results {
		chunkIndex := 0
		chunkText := ""
		knowledgeID := r.ID

		if r.Metadata.Extra != nil {
			if idx, ok := r.Metadata.Extra["chunk_index"].(float64); ok {
				chunkIndex = int(idx)
			}
			if text, ok := r.Metadata.Extra["chunk_text"].(string); ok {
				chunkText = text
			}
			if kid, ok := r.Metadata.Extra["knowledge_id"].(string); ok {
				knowledgeID = kid
			}
		}

		similarity := distanceToSimilarity(r.Score)

		searchResults[i] = SearchKnowledgeResult{
			KnowledgeID: knowledgeID,
			ChunkIndex:  chunkIndex,
			ChunkText:   chunkText,
			Similarity:  similarity,
			Metadata:    r.Metadata.Extra,
		}
	}

	return searchResults, nil
}

// SearchInteractionResult represents an interaction search result.
type SearchInteractionResult struct {
	InteractionID string                 `json:"interaction_id"`
	Similarity    float64                `json:"similarity"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// SearchInteractions performs semantic search across interaction history.
// Use case: "Find all interactions discussing contract renewal"
func (s *VectorSearchSkill) SearchInteractions(ctx context.Context, query string, userID uuid.UUID, limit int, threshold float64) ([]SearchInteractionResult, error) {
	if s.openAI == nil {
		return nil, fmt.Errorf("OpenAI client not configured")
	}

	// Generate embedding for the search query
	queryVector, err := s.openAI.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search interactions
	results, err := s.vectorStore.SearchSimilarInteractions(ctx, queryVector, userID, limit, threshold)
	if err != nil {
		return nil, err
	}

	searchResults := make([]SearchInteractionResult, len(results))
	for i, r := range results {
		similarity := distanceToSimilarity(r.Score)

		searchResults[i] = SearchInteractionResult{
			InteractionID: r.ID,
			Similarity:    similarity,
			Metadata:      r.Metadata.Extra,
		}
	}

	return searchResults, nil
}

// SearchSegmentResult represents a segment search result.
type SearchSegmentResult struct {
	SegmentID   string                 `json:"segment_id"`
	SegmentName string                 `json:"segment_name"`
	Similarity  float64                `json:"similarity"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// FindSimilarSegments finds segments similar to a given contact or query.
// Use case: "Which segments would this contact fit into based on similarity?"
func (s *VectorSearchSkill) FindSimilarSegments(ctx context.Context, query string, userID uuid.UUID, limit int, threshold float64) ([]SearchSegmentResult, error) {
	if s.openAI == nil {
		return nil, fmt.Errorf("OpenAI client not configured")
	}

	// Generate embedding for the search query
	queryVector, err := s.openAI.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search segments
	results, err := s.vectorStore.SearchSimilarSegments(ctx, queryVector, userID, limit, threshold)
	if err != nil {
		return nil, err
	}

	searchResults := make([]SearchSegmentResult, len(results))
	for i, r := range results {
		segmentName := ""
		if r.Metadata.Extra != nil {
			if name, ok := r.Metadata.Extra["name"].(string); ok {
				segmentName = name
			}
		}

		similarity := distanceToSimilarity(r.Score)

		searchResults[i] = SearchSegmentResult{
			SegmentID:   r.ID,
			SegmentName: segmentName,
			Similarity:  similarity,
			Metadata:    r.Metadata.Extra,
		}
	}

	return searchResults, nil
}

// HybridSearch combines keyword search with semantic search for best results.
// This is useful when you want both exact matches and semantic similarity.
type HybridSearchResult struct {
	ContactID     string                 `json:"contact_id"`
	KeywordScore  float64                `json:"keyword_score"`
	SemanticScore float64                `json:"semantic_score"`
	CombinedScore float64                `json:"combined_score"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// HybridSearchContacts performs hybrid search combining keywords and semantics.
// Use case: "CTO at SaaS companies" - finds exact title matches + semantically similar contacts
func (s *VectorSearchSkill) HybridSearchContacts(ctx context.Context, query string, keywords []string, userID uuid.UUID, limit int) ([]HybridSearchResult, error) {
	// This is a placeholder for hybrid search implementation
	// Would combine RediSearch keyword queries with vector similarity
	// For now, just do semantic search
	semanticResults, err := s.SearchContactsByText(ctx, query, userID, limit, 0.0)
	if err != nil {
		return nil, err
	}

	hybridResults := make([]HybridSearchResult, len(semanticResults))
	for i, r := range semanticResults {
		hybridResults[i] = HybridSearchResult{
			ContactID:     r.ContactID,
			KeywordScore:  0.0, // Not implemented yet
			SemanticScore: r.Similarity,
			CombinedScore: r.Similarity,
			Metadata:      r.Metadata,
		}
	}

	return hybridResults, nil
}

// distanceToSimilarity turns RediSearch's __vector_score — a COSINE DISTANCE,
// 0 for identical and up to 2 for opposite — into the 0..1, higher-is-better
// similarity the API reports. The knowledge, interaction and segment searches
// used to report the distance itself as the similarity, so the best match
// carried the lowest "similarity" of the page.
func distanceToSimilarity(distance float64) float64 {
	similarity := 1.0 - (distance / 2.0)
	if similarity < 0 {
		return 0
	}
	if similarity > 1 {
		return 1
	}
	return similarity
}
