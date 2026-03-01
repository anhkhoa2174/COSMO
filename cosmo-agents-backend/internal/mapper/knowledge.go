package mapper

import (
	"encoding/json"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// ToKnowledgeRead converts a domain Knowledge entity into the API schema.
func ToKnowledgeRead(k *domain.Knowledge) v1schema.KnowledgeRead {
	if k == nil {
		return v1schema.KnowledgeRead{}
	}

	metadata := make(map[string]interface{})
	if len(k.CMetadata) > 0 {
		if err := json.Unmarshal([]byte(k.CMetadata), &metadata); err != nil {
			metadata = make(map[string]interface{})
		}
	}

	summaryPair := append([]string(nil), k.SummaryPair...)

	return v1schema.KnowledgeRead{
		ID:            k.ID,
		UserID:        k.UserID,
		SourceType:    string(k.SourceType),
		SummaryPair:   summaryPair,
		EmbeddingGID:  k.EmbeddingGID,
		CozeDatasetID: k.CozeDatasetID,
		IsDeleted:     k.IsDeleted,
		CMetadata:     metadata,
		CreatedAt:     k.CreatedAt,
		UpdatedAt:     k.UpdatedAt,
	}
}
