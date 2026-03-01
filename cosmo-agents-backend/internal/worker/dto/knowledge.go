package dto

import "github.com/google/uuid"

// ProcessKnowledgePayload represents payload for processing knowledge base entry.
type ProcessKnowledgePayload struct {
	KnowledgeID uuid.UUID `json:"knowledge_id"`
	UserID      uuid.UUID `json:"user_id"`
	Action      string    `json:"action"` // "create", "update", "delete"
}

// GenerateEmbeddingPayload represents payload for generating text embeddings.
type GenerateEmbeddingPayload struct {
	KnowledgeID uuid.UUID `json:"knowledge_id"`
	Text        string    `json:"text"`
	UserID      uuid.UUID `json:"user_id"`
}
