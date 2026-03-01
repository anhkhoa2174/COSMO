package v2

import (
	"time"

	"github.com/google/uuid"
)

// KnowledgeRead represents a knowledge item
type KnowledgeRead struct {
	ID            uuid.UUID              `json:"id"`
	Name          string                 `json:"name,omitempty"`
	UserID        uuid.UUID              `json:"user_id"`
	SourceType    string                 `json:"source_type"`
	SummaryPair   *string                `json:"summary_pair,omitempty"`
	EmbeddingGID  string                 `json:"embedding_gid,omitempty"`
	Collection    string                 `json:"collection,omitempty"`
	CozeDatasetID *string                `json:"coze_dataset_id,omitempty"`
	IsDeleted     bool                   `json:"is_deleted"`
	CMetadata     map[string]interface{} `json:"cmetadata,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// UploadKnowledgeResult represents the result of a knowledge upload
type UploadKnowledgeResult struct {
	Msg       string         `json:"msg"`
	Knowledge *KnowledgeRead `json:"knowledge"`
}

// UploadKnowledgeResponse represents the response for POST /v2/knowledge/upload
type UploadKnowledgeResponse struct {
	Result UploadKnowledgeResult `json:"result"`
}
