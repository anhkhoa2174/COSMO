package v1

// VectorSearchContactsRequest represents a request to search contacts by semantic similarity
type VectorSearchContactsRequest struct {
	Query     string  `json:"query" validate:"required,min=3"`
	Limit     int     `json:"limit" validate:"min=1,max=50"`
	Threshold float64 `json:"threshold" validate:"min=0,max=1"`
}

// VectorSearchContactsResponse represents the response from contact vector search
type VectorSearchContactsResponse struct {
	Results []ContactSearchResult `json:"results"`
	Query   string                `json:"query"`
	Count   int                   `json:"count"`
}

// ContactSearchResult represents a single contact search result
type ContactSearchResult struct {
	ContactID  string                 `json:"contact_id"`
	Similarity float64                `json:"similarity"`
	Name       string                 `json:"name,omitempty"`
	Email      string                 `json:"email,omitempty"`
	Company    string                 `json:"company,omitempty"`
	JobTitle   string                 `json:"job_title,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// FindSimilarContactRequest represents a request to find contacts similar to a given contact
type FindSimilarContactRequest struct {
	ContactID string  `json:"contact_id" validate:"required,uuid"`
	Limit     int     `json:"limit" validate:"min=1,max=50"`
	Threshold float64 `json:"threshold" validate:"min=0,max=1"`
}

// SearchKnowledgeRequest represents a request to search the knowledge base
type SearchKnowledgeRequest struct {
	Query     string  `json:"query" validate:"required,min=3"`
	Limit     int     `json:"limit" validate:"min=1,max=20"`
	Threshold float64 `json:"threshold" validate:"min=0,max=1"`
}

// SearchKnowledgeResponse represents the response from knowledge search
type SearchKnowledgeResponse struct {
	Results []VectorKnowledgeSearchResult `json:"results"`
	Query   string                        `json:"query"`
	Count   int                           `json:"count"`
}

// VectorKnowledgeSearchResult represents a single knowledge chunk result from vector search
type VectorKnowledgeSearchResult struct {
	KnowledgeID string  `json:"knowledge_id"`
	ChunkIndex  int     `json:"chunk_index"`
	ChunkText   string  `json:"chunk_text"`
	Similarity  float64 `json:"similarity"`
}

// SearchInteractionsRequest represents a request to search interactions
type SearchInteractionsRequest struct {
	Query     string  `json:"query" validate:"required,min=3"`
	Limit     int     `json:"limit" validate:"min=1,max=50"`
	Threshold float64 `json:"threshold" validate:"min=0,max=1"`
}

// SearchInteractionsResponse represents the response from interaction search
type SearchInteractionsResponse struct {
	Results []InteractionSearchResult `json:"results"`
	Query   string                    `json:"query"`
	Count   int                       `json:"count"`
}

// InteractionSearchResult represents a single interaction search result
type InteractionSearchResult struct {
	InteractionID   string                 `json:"interaction_id"`
	ContactID       string                 `json:"contact_id,omitempty"`
	InteractionType string                 `json:"interaction_type,omitempty"`
	Similarity      float64                `json:"similarity"`
	ContentPreview  string                 `json:"content_preview,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// FindSimilarSegmentsRequest represents a request to find similar segments
type FindSimilarSegmentsRequest struct {
	Query     string  `json:"query" validate:"required,min=3"`
	Limit     int     `json:"limit" validate:"min=1,max=10"`
	Threshold float64 `json:"threshold" validate:"min=0,max=1"`
}

// FindSimilarSegmentsResponse represents the response from segment search
type FindSimilarSegmentsResponse struct {
	Results []SegmentSearchResult `json:"results"`
	Query   string                `json:"query"`
	Count   int                   `json:"count"`
}

// SegmentSearchResult represents a single segment search result
type SegmentSearchResult struct {
	SegmentID    string  `json:"segment_id"`
	SegmentName  string  `json:"segment_name"`
	Similarity   float64 `json:"similarity"`
	ContactCount int     `json:"contact_count,omitempty"`
	AvgFitScore  float64 `json:"avg_fit_score,omitempty"`
}

// HybridSearchRequest combines keyword and semantic search
type HybridSearchRequest struct {
	Query    string   `json:"query" validate:"required,min=3"`
	Keywords []string `json:"keywords" validate:"max=10"`
	Limit    int      `json:"limit" validate:"min=1,max=50"`
}

// HybridSearchResponse represents the response from hybrid search
type HybridSearchResponse struct {
	Results []HybridSearchResult `json:"results"`
	Query   string               `json:"query"`
	Count   int                  `json:"count"`
}

// HybridSearchResult combines keyword and semantic scores
type HybridSearchResult struct {
	ContactID     string                 `json:"contact_id"`
	KeywordScore  float64                `json:"keyword_score"`
	SemanticScore float64                `json:"semantic_score"`
	CombinedScore float64                `json:"combined_score"`
	Name          string                 `json:"name,omitempty"`
	Email         string                 `json:"email,omitempty"`
	Company       string                 `json:"company,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}
