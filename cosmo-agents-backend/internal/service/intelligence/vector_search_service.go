package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// extractEmailFromProfile extracts email from profile JSONB
func extractEmailFromProfile(profile domain.JSONB) string {
	if len(profile) == 0 {
		return ""
	}
	var profileData map[string]interface{}
	if err := json.Unmarshal(profile, &profileData); err != nil {
		return ""
	}
	if email, ok := profileData["email"].(string); ok {
		return email
	}
	return ""
}

// SearchContactsByText performs semantic search for contacts using natural language.
// This enables use cases like "Find CTOs interested in AI and machine learning".
func (s *Service) SearchContactsByText(ctx context.Context, userID uuid.UUID, query string, limit int, threshold float64) (*v1.VectorSearchContactsResponse, error) {
	if s.vectorSearch == nil {
		return nil, fmt.Errorf("vector search not configured")
	}

	// Set defaults
	if limit == 0 {
		limit = 10
	}
	if threshold == 0 {
		threshold = 1.0 // Accept all results (no distance filtering)
	}

	// Perform vector search
	results, err := s.vectorSearch.SearchContactsByText(ctx, query, userID, limit, threshold)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Enrich results with contact details from database
	searchResults := make([]v1.ContactSearchResult, 0, len(results))
	for _, r := range results {
		// Extract UUID from Redis key format (e.g., "vector:contact:uuid" -> "uuid")
		contactIDStr := r.ContactID
		if strings.HasPrefix(contactIDStr, "vector:contact:") {
			contactIDStr = strings.TrimPrefix(contactIDStr, "vector:contact:")
		}

		contactID, err := uuid.Parse(contactIDStr)
		if err != nil {
			fmt.Printf("[SearchContactsByText] Failed to parse contact ID '%s': %v\n", r.ContactID, err)
			continue
		}

		// Get full contact details
		contact, err := s.contactRepo.GetByID(ctx, contactID)
		fmt.Printf("[SearchContactsByText] Contact %s: err=%v, nil=%v\n", r.ContactID, err, contact == nil)
		if contact != nil {
			fmt.Printf("[SearchContactsByText] Contact %s: IsDeleted=%v, Name=%s\n",
				r.ContactID, contact.IsDeleted, contact.Name)
		}
		if err != nil {
			fmt.Printf("[SearchContactsByText] Error fetching contact %s: %v\n", r.ContactID, err)
			continue
		}
		if contact == nil {
			fmt.Printf("[SearchContactsByText] Contact %s not found (nil)\n", r.ContactID)
			continue
		}
		if contact.IsDeleted {
			fmt.Printf("[SearchContactsByText] Skipping deleted contact: %s (IsDeleted=%v)\n", r.ContactID, contact.IsDeleted)
			continue
		}

		searchResults = append(searchResults, v1.ContactSearchResult{
			ContactID:  r.ContactID,
			Similarity: r.Similarity,
			Name:       contact.Name,
			Email:      extractEmailFromProfile(contact.Profile),
			Company:    contact.Company,
			JobTitle:   contact.JobTitle,
			Metadata:   r.Metadata,
		})
	}

	return &v1.VectorSearchContactsResponse{
		Results: searchResults,
		Query:   query,
		Count:   len(searchResults),
	}, nil
}

// FindSimilarContacts finds contacts similar to a given contact.
// Use case: "Show me more contacts like this high-value customer".
func (s *Service) FindSimilarContacts(ctx context.Context, userID uuid.UUID, contactID uuid.UUID, limit int, threshold float64) (*v1.VectorSearchContactsResponse, error) {
	if s.vectorSearch == nil {
		return nil, fmt.Errorf("vector search not configured")
	}

	// Set defaults
	if limit == 0 {
		limit = 10
	}
	if threshold == 0 {
		threshold = 1.0 // Accept all results (no distance filtering)
	}

	// Get the source contact for context
	sourceContact, err := s.contactRepo.GetByID(ctx, contactID)
	if err != nil || sourceContact == nil {
		return nil, fmt.Errorf("source contact not found: %w", err)
	}

	// Find similar contacts
	results, err := s.vectorSearch.FindSimilarContact(ctx, contactID, userID, limit, threshold)
	if err != nil {
		return nil, fmt.Errorf("similarity search failed: %w", err)
	}

	// Enrich with contact details
	searchResults := make([]v1.ContactSearchResult, 0, len(results))
	for _, r := range results {
		// Extract UUID from Redis key format
		contactIDStr := r.ContactID
		if strings.HasPrefix(contactIDStr, "vector:contact:") {
			contactIDStr = strings.TrimPrefix(contactIDStr, "vector:contact:")
		}

		cid, err := uuid.Parse(contactIDStr)
		if err != nil {
			continue
		}

		contact, err := s.contactRepo.GetByID(ctx, cid)
		if err != nil || contact == nil || contact.IsDeleted {
			continue
		}

		searchResults = append(searchResults, v1.ContactSearchResult{
			ContactID:  r.ContactID,
			Similarity: r.Similarity,
			Name:       contact.Name,
			Email:      extractEmailFromProfile(contact.Profile),
			Company:    contact.Company,
			JobTitle:   contact.JobTitle,
			Metadata:   r.Metadata,
		})
	}

	return &v1.VectorSearchContactsResponse{
		Results: searchResults,
		Query:   fmt.Sprintf("Similar to: %s (%s)", sourceContact.Name, sourceContact.Company),
		Count:   len(searchResults),
	}, nil
}

// SearchKnowledge performs semantic search across the knowledge base for RAG.
// Use case: "What information do we have about enterprise pricing strategies?"
func (s *Service) SearchKnowledge(ctx context.Context, userID uuid.UUID, query string, limit int, threshold float64) (*v1.SearchKnowledgeResponse, error) {
	if s.vectorSearch == nil {
		return nil, fmt.Errorf("vector search not configured")
	}

	// Set defaults
	if limit == 0 {
		limit = 5
	}
	if threshold == 0 {
		threshold = 1.0 // Accept all results (no distance filtering)
	}

	// Perform knowledge search
	results, err := s.vectorSearch.SearchKnowledge(ctx, query, userID, limit, threshold)
	if err != nil {
		return nil, fmt.Errorf("knowledge search failed: %w", err)
	}

	// Convert to API response
	searchResults := make([]v1.VectorKnowledgeSearchResult, len(results))
	for i, r := range results {
		searchResults[i] = v1.VectorKnowledgeSearchResult{
			KnowledgeID: r.KnowledgeID,
			ChunkIndex:  r.ChunkIndex,
			ChunkText:   r.ChunkText,
			Similarity:  r.Similarity,
		}
	}

	return &v1.SearchKnowledgeResponse{
		Results: searchResults,
		Query:   query,
		Count:   len(searchResults),
	}, nil
}

// SearchInteractions performs semantic search across interaction history.
// Use case: "Find all conversations discussing contract renewal or pricing".
func (s *Service) SearchInteractions(ctx context.Context, userID uuid.UUID, query string, limit int, threshold float64) (*v1.SearchInteractionsResponse, error) {
	if s.vectorSearch == nil {
		return nil, fmt.Errorf("vector search not configured")
	}

	// Set defaults
	if limit == 0 {
		limit = 20
	}
	if threshold == 0 {
		threshold = 0.7
	}

	// Perform interaction search
	results, err := s.vectorSearch.SearchInteractions(ctx, query, userID, limit, threshold)
	if err != nil {
		return nil, fmt.Errorf("interaction search failed: %w", err)
	}

	// Convert to API response with enriched metadata
	searchResults := make([]v1.InteractionSearchResult, len(results))
	for i, r := range results {
		interactionType := ""
		contactID := ""
		contentPreview := ""

		if r.Metadata != nil {
			if it, ok := r.Metadata["interaction_type"].(string); ok {
				interactionType = it
			}
			if cid, ok := r.Metadata["contact_id"].(string); ok {
				contactID = cid
			}
			if cp, ok := r.Metadata["content_preview"].(string); ok {
				contentPreview = cp
			}
		}

		searchResults[i] = v1.InteractionSearchResult{
			InteractionID:   r.InteractionID,
			ContactID:       contactID,
			InteractionType: interactionType,
			Similarity:      r.Similarity,
			ContentPreview:  contentPreview,
			Metadata:        r.Metadata,
		}
	}

	return &v1.SearchInteractionsResponse{
		Results: searchResults,
		Query:   query,
		Count:   len(searchResults),
	}, nil
}

// FindSimilarSegments finds segments that match a given profile or query.
// Use case: "Which segments would this contact profile fit into?"
func (s *Service) FindSimilarSegments(ctx context.Context, userID uuid.UUID, query string, limit int, threshold float64) (*v1.FindSimilarSegmentsResponse, error) {
	if s.vectorSearch == nil {
		return nil, fmt.Errorf("vector search not configured")
	}

	// Set defaults
	if limit == 0 {
		limit = 5
	}
	if threshold == 0 {
		threshold = 0.6
	}

	// Perform segment search
	results, err := s.vectorSearch.FindSimilarSegments(ctx, query, userID, limit, threshold)
	if err != nil {
		return nil, fmt.Errorf("segment search failed: %w", err)
	}

	// Convert to API response
	searchResults := make([]v1.SegmentSearchResult, len(results))
	for i, r := range results {
		contactCount := 0
		avgFitScore := 0.0

		if r.Metadata != nil {
			if cc, ok := r.Metadata["contact_count"].(float64); ok {
				contactCount = int(cc)
			}
			if afs, ok := r.Metadata["avg_fit_score"].(float64); ok {
				avgFitScore = afs
			}
		}

		searchResults[i] = v1.SegmentSearchResult{
			SegmentID:    r.SegmentID,
			SegmentName:  r.SegmentName,
			Similarity:   r.Similarity,
			ContactCount: contactCount,
			AvgFitScore:  avgFitScore,
		}
	}

	return &v1.FindSimilarSegmentsResponse{
		Results: searchResults,
		Query:   query,
		Count:   len(searchResults),
	}, nil
}

// HybridSearchContacts combines keyword and semantic search for best results.
// Use case: "VP of Engineering at SaaS companies" (exact title match + semantic similarity).
func (s *Service) HybridSearchContacts(ctx context.Context, userID uuid.UUID, query string, keywords []string, limit int) (*v1.HybridSearchResponse, error) {
	if s.vectorSearch == nil {
		return nil, fmt.Errorf("vector search not configured")
	}

	// Set defaults
	if limit == 0 {
		limit = 20
	}

	// Perform hybrid search
	results, err := s.vectorSearch.HybridSearchContacts(ctx, query, keywords, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("hybrid search failed: %w", err)
	}

	// Enrich with contact details
	searchResults := make([]v1.HybridSearchResult, 0, len(results))
	for _, r := range results {
		// Extract UUID from Redis key format
		contactIDStr := r.ContactID
		if strings.HasPrefix(contactIDStr, "vector:contact:") {
			contactIDStr = strings.TrimPrefix(contactIDStr, "vector:contact:")
		}

		contactID, err := uuid.Parse(contactIDStr)
		if err != nil {
			continue
		}

		contact, err := s.contactRepo.GetByID(ctx, contactID)
		if err != nil || contact == nil || contact.IsDeleted {
			continue
		}

		searchResults = append(searchResults, v1.HybridSearchResult{
			ContactID:     r.ContactID,
			KeywordScore:  r.KeywordScore,
			SemanticScore: r.SemanticScore,
			CombinedScore: r.CombinedScore,
			Name:          contact.Name,
			Email:         extractEmailFromProfile(contact.Profile),
			Company:       contact.Company,
			Metadata:      r.Metadata,
		})
	}

	return &v1.HybridSearchResponse{
		Results: searchResults,
		Query:   query,
		Count:   len(searchResults),
	}, nil
}
