package v1

import (
	"time"

	"github.com/google/uuid"
)

// CreateKnowledgeRequest represents the request to create a knowledge entry
type CreateKnowledgeRequest struct {
	Collection string                 `json:"collection" validate:"required"`
	Content    string                 `json:"content" validate:"required"`
	Source     *string                `json:"source,omitempty"`
	SourceID   *string                `json:"source_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateKnowledgeRequest represents the request to update a knowledge entry
type UpdateKnowledgeRequest struct {
	Collection *string                `json:"collection,omitempty"`
	Content    *string                `json:"content,omitempty"`
	Source     *string                `json:"source,omitempty"`
	SourceID   *string                `json:"source_id,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// KnowledgeRead mirrors machine/api/v1/knowledge/schema.py::KnowledgeRead
type KnowledgeRead struct {
	ID            uuid.UUID              `json:"id"`
	UserID        uuid.UUID              `json:"user_id"`
	SourceType    string                 `json:"source_type"`
	SummaryPair   []string               `json:"summary_pair"`
	EmbeddingGID  *string                `json:"embedding_gid,omitempty"`
	CozeDatasetID *string                `json:"coze_dataset_id,omitempty"`
	IsDeleted     bool                   `json:"is_deleted"`
	CMetadata     map[string]interface{} `json:"cmetadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// UploadKnowledgeResult mirrors UploadKnowledgeResult from Python API
type UploadKnowledgeResult struct {
	Msg       string         `json:"msg"`
	Knowledge *KnowledgeRead `json:"knowledge,omitempty"`
}

// UploadKnowledgeResponse mirrors UploadKnowledgeResponse from Python API
type UploadKnowledgeResponse struct {
	Success bool                  `json:"success"`
	Result  UploadKnowledgeResult `json:"result"`
}

// KnowledgeSearchRequest represents semantic search request
type KnowledgeSearchRequest struct {
	Query  string                 `json:"query" validate:"required"`
	TopK   *int                   `json:"top_k,omitempty"`
	Filter map[string]interface{} `json:"filter,omitempty"`
}

// KnowledgeSearchResult represents a single search result
type KnowledgeSearchResult struct {
	Document string                 `json:"document"`
	Metadata map[string]interface{} `json:"metadata"`
	Score    float64                `json:"score"`
}
