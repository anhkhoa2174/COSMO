package knowledge

import (
	"strings"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
)

func mapDomainKnowledgeToV2(k *domain.Knowledge) v2schema.KnowledgeRead {
	if k == nil {
		return v2schema.KnowledgeRead{}
	}

	metadata := parseMetadataMap(k.CMetadata)
	name := deriveKnowledgeName(k.ID.String(), metadata)
	summary := joinSummaryPair(k.SummaryPair)

	embeddingGID := ""
	if k.EmbeddingGID != nil {
		embeddingGID = *k.EmbeddingGID
	}

	result := v2schema.KnowledgeRead{
		ID:           k.ID,
		UserID:       k.UserID,
		Name:         name,
		SourceType:   string(k.SourceType),
		EmbeddingGID: embeddingGID,
		Collection:   "",
		CMetadata:    metadata,
		SummaryPair:  summary,
		IsDeleted:    k.IsDeleted,
		CreatedAt:    k.CreatedAt,
		UpdatedAt:    k.UpdatedAt,
	}
	if k.CozeDatasetID != nil {
		result.CozeDatasetID = k.CozeDatasetID
	}
	return result
}

func parseMetadataMap(data domain.JSONB) map[string]interface{} {
	result := make(map[string]interface{})
	if len(data) == 0 {
		return result
	}
	if err := data.Unmarshal(&result); err != nil {
		return make(map[string]interface{})
	}
	return result
}

func joinSummaryPair(summary []string) *string {
	if len(summary) == 0 {
		return nil
	}
	val := strings.Join(summary, "\n")
	if strings.TrimSpace(val) == "" {
		return nil
	}
	return &val
}

func deriveKnowledgeName(fallback string, metadata map[string]interface{}) string {
	if origin, ok := metadata["origin"].(map[string]interface{}); ok {
		if filename, ok := origin["filename"].(string); ok && filename != "" {
			return filename
		}
	}
	return fallback
}
