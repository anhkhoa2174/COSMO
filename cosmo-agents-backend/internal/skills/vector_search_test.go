package skills

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/pkg/vectorstore"
)

func TestDistanceToSimilarity(t *testing.T) {
	tests := []struct {
		distance, want float64
	}{
		{0, 1},      // identical
		{0.5, 0.75}, // the grounding cut-off
		{1, 0.5},    // orthogonal
		{2, 0},      // opposite
		{2.5, 0},    // clamped
		{-0.1, 1},   // float noise below zero clamps to 1
	}
	for _, tt := range tests {
		if got := distanceToSimilarity(tt.distance); math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("distanceToSimilarity(%v) = %v, want %v", tt.distance, got, tt.want)
		}
	}
}

// Every search on this skill reports a similarity, where higher is better.
// The store returns results nearest first, so the first similarity must be
// the highest. Knowledge, interaction and segment searches used to copy the
// cosine distance straight into the field, which put the best match at the
// bottom of any list sorted by similarity and made a "threshold 0.8" filter
// on similarity keep exactly the wrong rows.
func TestVectorSearchSkill_ReportsSimilarityNotDistance(t *testing.T) {
	c := appRedis(t)
	client, _ := fakeEmbeddings(t, http.StatusOK)
	store := vectorstore.NewRedisVectorStore(c)
	s := NewVectorSearchSkill(store, client)
	ctx := context.Background()
	user := uuid.New()

	// Two entities per kind: one at distance 0.1, one at 0.4.
	type seeded struct{ near, far uuid.UUID }
	kinds := map[string]seeded{}
	for _, kind := range []string{"contact", "knowledge", "interaction", "segment"} {
		near, far := uuid.New(), uuid.New()
		kinds[kind] = seeded{near, far}
		for id, d := range map[uuid.UUID]float64{near: 0.1, far: 0.4} {
			var err error
			meta := map[string]interface{}{"name": kind + "-" + fmt.Sprint(d)}
			switch kind {
			case "contact":
				deleteOnCleanup(t, c, vectorstore.ContactVectorPrefix+id.String())
				err = store.StoreContactVector(ctx, id, user, vecAtDistance(d), meta)
			case "knowledge":
				deleteOnCleanup(t, c, vectorstore.KnowledgeVectorPrefix+id.String()+":2")
				err = store.StoreKnowledgeVector(ctx, id, user, 2, vecAtDistance(d), "chunk "+fmt.Sprint(d))
			case "interaction":
				deleteOnCleanup(t, c, vectorstore.InteractionVectorPrefix+id.String())
				err = store.StoreInteractionVector(ctx, id, user, vecAtDistance(d), meta)
			case "segment":
				deleteOnCleanup(t, c, vectorstore.SegmentVectorPrefix+id.String())
				err = store.StoreSegmentVector(ctx, id, user, vecAtDistance(d), meta)
			}
			if err != nil {
				t.Fatalf("seed %s: %v", kind, err)
			}
		}
	}

	check := func(t *testing.T, sims []float64) {
		t.Helper()
		if len(sims) != 2 {
			t.Fatalf("expected 2 results, got %d", len(sims))
		}
		if sims[0] <= sims[1] {
			t.Fatalf("nearest result has similarity %.3f, not above the farther one's %.3f", sims[0], sims[1])
		}
		if math.Abs(sims[0]-0.95) > 0.01 || math.Abs(sims[1]-0.8) > 0.01 {
			t.Fatalf("similarities %v, want ~[0.95 0.80] for distances 0.1 and 0.4", sims)
		}
	}

	t.Run("contacts", func(t *testing.T) {
		var got []SearchSimilarContactsResult
		eventually(t, func() bool {
			got, _ = s.SearchContactsByText(ctx, "q", user, 5, 0.5)
			return len(got) == 2
		})
		check(t, []float64{got[0].Similarity, got[1].Similarity})
		if got[0].ContactID != kinds["contact"].near.String() {
			t.Errorf("nearest contact is %s", got[0].ContactID)
		}
	})
	t.Run("hybrid contacts", func(t *testing.T) {
		got, err := s.HybridSearchContacts(ctx, "q", []string{"cto"}, user, 5)
		if err != nil || len(got) != 2 {
			t.Fatalf("got %v, %v", got, err)
		}
		if got[0].SemanticScore != got[0].CombinedScore || got[0].KeywordScore != 0 {
			t.Errorf("hybrid scores: %+v", got[0])
		}
		check(t, []float64{got[0].SemanticScore, got[1].SemanticScore})
	})
	t.Run("knowledge", func(t *testing.T) {
		var got []SearchKnowledgeResult
		eventually(t, func() bool {
			got, _ = s.SearchKnowledge(ctx, "q", user, 5, 0.5)
			return len(got) == 2
		})
		check(t, []float64{got[0].Similarity, got[1].Similarity})
		if got[0].KnowledgeID != kinds["knowledge"].near.String() || got[0].ChunkIndex != 2 || got[0].ChunkText != "chunk 0.1" {
			t.Errorf("knowledge fields: %+v", got[0])
		}
	})
	t.Run("interactions", func(t *testing.T) {
		var got []SearchInteractionResult
		eventually(t, func() bool {
			got, _ = s.SearchInteractions(ctx, "q", user, 5, 0.5)
			return len(got) == 2
		})
		check(t, []float64{got[0].Similarity, got[1].Similarity})
	})
	t.Run("segments", func(t *testing.T) {
		var got []SearchSegmentResult
		eventually(t, func() bool {
			got, _ = s.FindSimilarSegments(ctx, "q", user, 5, 0.5)
			return len(got) == 2
		})
		check(t, []float64{got[0].Similarity, got[1].Similarity})
		if got[0].SegmentName != "segment-0.1" {
			t.Errorf("segment name = %q", got[0].SegmentName)
		}
	})
}

func TestVectorSearchSkill_WithoutAClientEveryTextSearchFails(t *testing.T) {
	s := NewVectorSearchSkill(vectorstore.NewRedisVectorStore(nil), nil)
	ctx := context.Background()
	u := uuid.New()
	errs := map[string]error{}
	_, errs["contacts"] = s.SearchContactsByText(ctx, "q", u, 5, 0.5)
	_, errs["hybrid"] = s.HybridSearchContacts(ctx, "q", nil, u, 5)
	_, errs["knowledge"] = s.SearchKnowledge(ctx, "q", u, 5, 0.5)
	_, errs["interactions"] = s.SearchInteractions(ctx, "q", u, 5, 0.5)
	_, errs["segments"] = s.FindSimilarSegments(ctx, "q", u, 5, 0.5)
	for name, err := range errs {
		if err == nil || !strings.Contains(err.Error(), "not configured") {
			t.Errorf("%s: expected a not-configured error, got %v", name, err)
		}
	}
}

func TestVectorSearchSkill_EmbeddingFailuresPropagate(t *testing.T) {
	client, _ := fakeEmbeddings(t, http.StatusInternalServerError)
	s := NewVectorSearchSkill(vectorstore.NewRedisVectorStore(nil), client)
	ctx := context.Background()
	u := uuid.New()
	errs := map[string]error{}
	_, errs["contacts"] = s.SearchContactsByText(ctx, "q", u, 5, 0.5)
	_, errs["knowledge"] = s.SearchKnowledge(ctx, "q", u, 5, 0.5)
	_, errs["interactions"] = s.SearchInteractions(ctx, "q", u, 5, 0.5)
	_, errs["segments"] = s.FindSimilarSegments(ctx, "q", u, 5, 0.5)
	for name, err := range errs {
		if err == nil || !strings.Contains(err.Error(), "query embedding") {
			t.Errorf("%s: expected an embedding error, got %v", name, err)
		}
	}
}

func TestVectorSearchSkill_FindSimilarContactIsNotImplemented(t *testing.T) {
	_, err := NewVectorSearchSkill(nil, nil).FindSimilarContact(context.Background(), uuid.New(), uuid.New(), 5, 0.5)
	if err == nil || !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("expected not implemented, got %v", err)
	}
}

func TestEmbeddingSkill_StoresAndDeletes(t *testing.T) {
	c := appRedis(t)
	client, _ := fakeEmbeddings(t, http.StatusOK)
	store := vectorstore.NewRedisVectorStore(c)
	s := NewEmbeddingSkill(client, store)
	ctx := context.Background()
	user := uuid.New()

	contact, interaction, knowledge := uuid.New(), uuid.New(), uuid.New()
	keys := []string{
		vectorstore.ContactVectorPrefix + contact.String(),
		vectorstore.InteractionVectorPrefix + interaction.String(),
		vectorstore.KnowledgeVectorPrefix + knowledge.String() + ":0",
	}
	deleteOnCleanup(t, c, keys...)

	v, err := s.GenerateAndStore(ctx, contact, user, "text", map[string]interface{}{"k": "v"})
	if err != nil || len(v) != vectorstore.VectorDimensions {
		t.Fatalf("GenerateAndStore = %d dims, %v", len(v), err)
	}
	if _, err := s.GenerateInteractionEmbedding(ctx, interaction, user, "text", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GenerateKnowledgeEmbedding(ctx, knowledge, user, 0, "chunk"); err != nil {
		t.Fatal(err)
	}
	for _, k := range keys {
		if n, _ := c.Exists(ctx, k).Result(); n != 1 {
			t.Errorf("%s not stored", k)
		}
	}

	if err := s.DeleteContactVector(ctx, contact); err != nil {
		t.Fatal(err)
	}
	if n, _ := c.Exists(ctx, keys[0]).Result(); n != 0 {
		t.Error("contact vector survived DeleteContactVector")
	}
}

func TestEmbeddingSkill_WithoutClientIsANoOp(t *testing.T) {
	ctx := context.Background()
	for name, s := range map[string]*EmbeddingSkill{"nil": nil, "no client": NewEmbeddingSkill(nil, nil)} {
		t.Run(name, func(t *testing.T) {
			if v, err := s.GenerateAndStore(ctx, uuid.New(), uuid.New(), "t", nil); v != nil || err != nil {
				t.Fatalf("GenerateAndStore = %v, %v", v, err)
			}
			if v, err := s.GenerateInteractionEmbedding(ctx, uuid.New(), uuid.New(), "t", nil); v != nil || err != nil {
				t.Fatalf("GenerateInteractionEmbedding = %v, %v", v, err)
			}
			if v, err := s.GenerateKnowledgeEmbedding(ctx, uuid.New(), uuid.New(), 0, "t"); v != nil || err != nil {
				t.Fatalf("GenerateKnowledgeEmbedding = %v, %v", v, err)
			}
			if err := s.DeleteContactVector(ctx, uuid.New()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEmbeddingSkill_WithoutStoreStillReturnsTheVector(t *testing.T) {
	client, _ := fakeEmbeddings(t, http.StatusOK)
	s := NewEmbeddingSkill(client, nil)
	ctx := context.Background()
	for name, f := range map[string]func() ([]float32, error){
		"contact": func() ([]float32, error) { return s.GenerateAndStore(ctx, uuid.New(), uuid.New(), "t", nil) },
		"interaction": func() ([]float32, error) {
			return s.GenerateInteractionEmbedding(ctx, uuid.New(), uuid.New(), "t", nil)
		},
		"knowledge": func() ([]float32, error) { return s.GenerateKnowledgeEmbedding(ctx, uuid.New(), uuid.New(), 0, "t") },
	} {
		if v, err := f(); err != nil || len(v) != vectorstore.VectorDimensions {
			t.Errorf("%s: %d dims, %v", name, len(v), err)
		}
	}
}

func TestEmbeddingSkill_EmbeddingFailuresPropagate(t *testing.T) {
	client, _ := fakeEmbeddings(t, http.StatusInternalServerError)
	s := NewEmbeddingSkill(client, nil)
	ctx := context.Background()
	for name, f := range map[string]func() ([]float32, error){
		"contact": func() ([]float32, error) { return s.GenerateAndStore(ctx, uuid.New(), uuid.New(), "t", nil) },
		"interaction": func() ([]float32, error) {
			return s.GenerateInteractionEmbedding(ctx, uuid.New(), uuid.New(), "t", nil)
		},
		"knowledge": func() ([]float32, error) { return s.GenerateKnowledgeEmbedding(ctx, uuid.New(), uuid.New(), 0, "t") },
	} {
		if _, err := f(); err == nil || !strings.Contains(err.Error(), "failed to generate embedding") {
			t.Errorf("%s: expected an embedding error, got %v", name, err)
		}
	}
}
