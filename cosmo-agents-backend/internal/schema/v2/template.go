package v2

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// TemplateUpdateRequest represents the request for PATCH /v2/templates/{template_id}
type TemplateUpdateRequest struct {
	Subject   *string `json:"subject"`
	Content   *string `json:"content"`
	SendAfter *int    `json:"send_after"`
}

// TemplateUpdateResponse represents the response for PATCH /v2/templates/{template_id}
type TemplateUpdateResponse struct {
	ID      uuid.UUID `json:"id"`
	Subject *string   `json:"subject"`
	Content *string   `json:"content"`
}

// TemplateDeleteResponse represents the response for DELETE /v2/templates/{template_id}
type TemplateDeleteResponse struct {
	ID uuid.UUID `json:"id"`
}

// TemplateKnowledgeEntity represents a knowledge entity attached to a template
type TemplateKnowledgeEntity struct {
	ID           uuid.UUID              `json:"id"`
	EmbeddingGID string                 `json:"embedding_gid"`
	CMetadata    map[string]interface{} `json:"cmetadata,omitempty"`
}

// TemplateGetResponse represents the response for GET /v2/templates/{template_id}
type TemplateGetResponse struct {
	ID         uuid.UUID                 `json:"id"`
	Type       string                    `json:"type"`
	Subject    string                    `json:"subject"`
	Content    string                    `json:"content"`
	SendAfter  *int                      `json:"send_after"`
	Knowledges []TemplateKnowledgeEntity `json:"knowledges"`
}

// KnowledgeUploadResult represents a single uploaded knowledge result
type KnowledgeUploadResult struct {
	ID       uuid.UUID              `json:"id"`
	Metadata map[string]interface{} `json:"cmetadata"`
}

// AddKnowledgeRequest represents the request for POST /v2/templates/{template_id}/knowledges
type AddKnowledgeRequest struct {
	KnowledgeIDs []string `json:"knowledge_ids"`
}

// AddKnowledgeResponse represents the response for POST /v2/templates/{template_id}/knowledges
type AddKnowledgeResponse struct {
	Message string `json:"message"`
}

// KnowledgeUploadResponse represents the response for POST /v2/templates/{template_id}/knowledges
type KnowledgeUploadResponse = schema.PaginatedResponse[KnowledgeUploadResult]
