package skills

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/pkg/vectorstore"
)

func chunkTexts(rs []KnowledgeSearchResult) string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.ChunkText)
	}
	return strings.Join(out, ",")
}

// seedKnowledge writes one chunk per entry for user and returns the store.
func seedKnowledge(t *testing.T, user uuid.UUID, chunks []struct {
	dist float64
	text string
	typ  string
}) *vectorstore.RedisVectorStore {
	t.Helper()
	c := appRedis(t)
	store := vectorstore.NewRedisVectorStore(c)
	for i, ch := range chunks {
		kid := uuid.New()
		deleteOnCleanup(t, c, fmt.Sprintf("%s%s:%d", vectorstore.KnowledgeVectorPrefix, kid, i))
		if err := store.StoreKnowledgeChunk(context.Background(), kid, user, i, vecAtDistance(ch.dist), ch.text, ch.text+".pdf", ch.typ); err != nil {
			t.Fatalf("seed %s: %v", ch.text, err)
		}
	}
	return store
}

// SearchForIntent prefers the document types that suit the intent, falls back
// to every document when those yield nothing, and in both cases admits only
// chunks within a cosine distance of 0.5 of the query.
func TestKnowledgeSearch_SearchForIntentPrefersTypesThenFallsBack(t *testing.T) {
	user := uuid.New()
	store := seedKnowledge(t, user, []struct {
		dist float64
		text string
		typ  string
	}{
		{0.05, "untyped-near", ""},
		{0.10, "pricing-near", "pricing"},
		{0.45, "pricing-edge", "pricing"},
		// Past the 0.5 cut-off: a matching type does not rescue a chunk that
		// is not about the question.
		{0.60, "faq-far", "faq"},
	})
	client, _ := fakeEmbeddings(t, http.StatusOK)
	s := NewKnowledgeSearchSkill(client, store)
	ctx := context.Background()

	eventually(t, func() bool {
		got, err := s.Search(ctx, user, "q", 10)
		return err == nil && len(got) == 3
	})

	everythingNear := "untyped-near,pricing-near,pricing-edge"
	tests := []struct {
		name   string
		intent domain.IntentType
		limit  int
		want   string
	}{
		{"pricing question gets only the pricing sheet", domain.IntentRequestForPricing, 5, "pricing-near,pricing-edge"},
		{"preferred type only beyond the cut-off falls back to everything", domain.IntentRequestForInfo, 5, everythingNear},
		{"preferred types with no documents fall back", domain.IntentInterested, 5, everythingNear},
		{"intent with no preference searches everything", domain.IntentNurture, 5, everythingNear},
		{"unknown intent string searches everything", domain.IntentType("SOMETHING_ELSE"), 5, everythingNear},
		{"non-positive limit defaults to five", domain.IntentRequestForPricing, 0, "pricing-near,pricing-edge"},
		{"limit applies to the typed search", domain.IntentRequestForPricing, 1, "pricing-near"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.SearchForIntent(ctx, user, "q", string(tt.intent), tt.limit)
			if err != nil {
				t.Fatal(err)
			}
			if chunkTexts(got) != tt.want {
				t.Fatalf("got %q, want %q", chunkTexts(got), tt.want)
			}
			for _, r := range got {
				if r.Score > 0.5 {
					t.Errorf("%s admitted at distance %.3f", r.ChunkText, r.Score)
				}
				if r.Title != r.ChunkText+".pdf" {
					t.Errorf("%s lost its title: %q", r.ChunkText, r.Title)
				}
			}
		})
	}

	// Another tenant with no documents gets nothing, not this tenant's chunks.
	other, err := s.SearchForIntent(ctx, uuid.New(), "q", string(domain.IntentRequestForPricing), 5)
	if err != nil || len(other) != 0 {
		t.Fatalf("another tenant got %q, %v", chunkTexts(other), err)
	}
}

func TestKnowledgeSearch_Search(t *testing.T) {
	user := uuid.New()
	store := seedKnowledge(t, user, []struct {
		dist float64
		text string
		typ  string
	}{{0.2, "a", "product"}, {0.3, "b", ""}, {0.9, "unrelated", ""}})
	client, _ := fakeEmbeddings(t, http.StatusOK)
	s := NewKnowledgeSearchSkill(client, store)

	var got []KnowledgeSearchResult
	eventually(t, func() bool {
		var err error
		got, err = s.Search(context.Background(), user, "q", 0)
		return err == nil && len(got) == 2
	})
	if chunkTexts(got) != "a,b" {
		t.Fatalf("got %q, want a,b", chunkTexts(got))
	}
	if got[0].Type != "product" || got[1].Type != "" {
		t.Fatalf("types lost: %+v", got)
	}
}

// Without a client or store the skill answers nothing rather than failing,
// so a deployment without embeddings still drafts replies from summaries.
func TestKnowledgeSearch_UnconfiguredReturnsNothing(t *testing.T) {
	client, calls := fakeEmbeddings(t, http.StatusOK)
	store := vectorstore.NewRedisVectorStore(nil)
	for name, s := range map[string]*KnowledgeSearchSkill{
		"nil skill":  nil,
		"no client":  NewKnowledgeSearchSkill(nil, store),
		"no store":   NewKnowledgeSearchSkill(client, nil),
		"zero value": {},
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := s.Search(context.Background(), uuid.New(), "q", 5); got != nil || err != nil {
				t.Fatalf("Search = %v, %v", got, err)
			}
			if got, err := s.SearchForIntent(context.Background(), uuid.New(), "q", string(domain.IntentRequestForPricing), 5); got != nil || err != nil {
				t.Fatalf("SearchForIntent = %v, %v", got, err)
			}
		})
	}
	if *calls != 0 {
		t.Fatalf("embedding API called %d times by an unconfigured skill", *calls)
	}
}

func TestKnowledgeSearch_EmbeddingFailureIsAnError(t *testing.T) {
	client, _ := fakeEmbeddings(t, http.StatusInternalServerError)
	s := NewKnowledgeSearchSkill(client, vectorstore.NewRedisVectorStore(nil))
	for _, intent := range []domain.IntentType{domain.IntentRequestForPricing, domain.IntentNurture} {
		_, err := s.SearchForIntent(context.Background(), uuid.New(), "q", string(intent), 5)
		if err == nil || !strings.Contains(err.Error(), "query embedding") {
			t.Fatalf("%s: expected an embedding error, got %v", intent, err)
		}
	}
}
