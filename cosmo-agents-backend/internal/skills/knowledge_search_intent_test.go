package skills

import (
	"testing"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/pkg/vectorstore"
)

// The map is keyed by the intent strings the classifier produces; if those
// ever change, the preference would silently stop applying.
func TestPreferredTypes_KeysAreRealIntents(t *testing.T) {
	cases := map[domain.IntentType][]string{
		domain.IntentRequestForPricing: {"pricing"},
		domain.IntentRequestForInfo:    {"product", "faq"},
		domain.IntentInterested:        {"product", "case_study"},
	}
	for intent, want := range cases {
		got := PreferredTypes(string(intent))
		if len(got) != len(want) {
			t.Fatalf("%s: got %v, want %v", intent, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: got %v, want %v", intent, got, want)
			}
		}
	}
}

func TestPreferredTypes_NoPreferenceForOtherIntents(t *testing.T) {
	for _, intent := range []domain.IntentType{domain.IntentOutOfOffice, domain.IntentNurture, domain.IntentUnknown} {
		if got := PreferredTypes(string(intent)); got != nil {
			t.Fatalf("%s should have no preference, got %v", intent, got)
		}
	}
}

func TestToKnowledgeResults_CarriesTitleAndTypeAndDropsEmptyChunks(t *testing.T) {
	var withText, empty vectorstore.VectorSearchResult
	withText.Score = 0.2
	withText.Metadata.Extra = map[string]interface{}{
		"chunk_text": "Growth is $99", "title": "pricing.pdf", "knowledge_type": "pricing",
	}
	empty.Metadata.Extra = map[string]interface{}{"title": "blank.pdf"}

	got := toKnowledgeResults([]vectorstore.VectorSearchResult{withText, empty})
	if len(got) != 1 {
		t.Fatalf("expected the empty chunk to be dropped, got %d results", len(got))
	}
	if got[0].Title != "pricing.pdf" || got[0].Type != "pricing" || got[0].ChunkText != "Growth is $99" {
		t.Fatalf("got %+v", got[0])
	}
}
