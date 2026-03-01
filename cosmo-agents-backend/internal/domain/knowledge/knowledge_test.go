package knowledge

import (
	"testing"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func TestKnowledgeBeforeCreateDefaults(t *testing.T) {
	knowledge := Knowledge{}

	if err := knowledge.BeforeCreate(nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if knowledge.ID == uuid.Nil {
		t.Fatal("expected knowledge ID to be set")
	}
	if knowledge.SourceType != KnowledgeSourceUpload {
		t.Fatalf("expected default source type %q, got %q", KnowledgeSourceUpload, knowledge.SourceType)
	}
	if knowledge.SummaryPair == nil || len(knowledge.SummaryPair) != 0 {
		t.Fatalf("expected summary pair to default to empty array, got %v", knowledge.SummaryPair)
	}
	if string(knowledge.CMetadata) != "{}" {
		t.Fatalf("expected cmetadata '{}', got %s", string(knowledge.CMetadata))
	}
}

func TestKnowledgeBeforeCreateKeepsProvidedValues(t *testing.T) {
	source := KnowledgeSourceWebsite
	summary := pq.StringArray{"a", "b"}
	metadata := base.JSONB([]byte(`{"foo":"bar"}`))
	knowledge := Knowledge{
		SourceType:  source,
		SummaryPair: summary,
		CMetadata:   metadata,
	}

	if err := knowledge.BeforeCreate(nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if knowledge.SourceType != source {
		t.Fatalf("expected source type to remain %q, got %q", source, knowledge.SourceType)
	}
	if len(knowledge.SummaryPair) != len(summary) {
		t.Fatalf("expected summary pair to remain length %d, got %d", len(summary), len(knowledge.SummaryPair))
	}
	if string(knowledge.CMetadata) != string(metadata) {
		t.Fatalf("expected metadata to remain %s, got %s", string(metadata), string(knowledge.CMetadata))
	}
}
