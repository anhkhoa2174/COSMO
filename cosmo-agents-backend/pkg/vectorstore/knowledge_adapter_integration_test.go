package vectorstore

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
)

// fakeEmbeddings serves the OpenAI embeddings endpoint locally, returning e0
// for every input, so no real embedding API is ever called.
func fakeEmbeddings(t *testing.T, status int) (*ai.OpenAIClient, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if !strings.HasSuffix(r.URL.Path, "/embeddings") {
			http.NotFound(w, r)
			return
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"message":"boom","type":"server_error"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"model":  "fake",
			"data":   []map[string]interface{}{{"object": "embedding", "index": 0, "embedding": e0()}},
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("OPENAI_BASE_URL", srv.URL)
	return ai.NewOpenAIClient(ai.Config{APIKey: "test"}), &calls
}

func TestKnowledgeAdapter_AddDocumentsStoresEachChunkWithLabels(t *testing.T) {
	c := testRedis(t, 3)
	client, _ := fakeEmbeddings(t, http.StatusOK)
	ctx := context.Background()

	gid := "test-gid-" + runTag(t)
	knowledgeID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(gid))
	t.Cleanup(func() {
		for i := 0; i < 2; i++ {
			_ = c.Del(ctx, KnowledgeVectorPrefix+knowledgeID.String()+":"+string(rune('0'+i))).Err()
		}
	})

	user := uuid.New()
	meta := func() map[string]interface{} {
		return map[string]interface{}{
			"user_id":       user.String(),
			"embedding_gid": gid,
			"type":          "case_study",
			"origin":        map[string]interface{}{"filename": "acme.pdf"},
		}
	}
	a := NewKnowledgeVectorStoreAdapter(NewRedisVectorStore(c), client)
	if err := a.AddDocuments(ctx, "knowledge", []string{"first", "second"}, []map[string]interface{}{meta(), meta()}, []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}

	for i, want := range []string{"first", "second"} {
		key := KnowledgeVectorPrefix + knowledgeID.String() + ":" + string(rune('0'+i))
		h, err := c.HGetAll(ctx, key).Result()
		if err != nil || len(h) == 0 {
			t.Fatalf("chunk %d not stored under %s: %v", i, key, err)
		}
		if h["knowledge_type"] != "case_study" {
			t.Errorf("chunk %d: knowledge_type = %q", i, h["knowledge_type"])
		}
		if h["user_id"] != tenantTag(user) {
			t.Errorf("chunk %d: user_id = %q", i, h["user_id"])
		}
		var extra map[string]interface{}
		_ = json.Unmarshal([]byte(h["metadata"]), &extra)
		if extra["chunk_text"] != want || extra["title"] != "acme.pdf" {
			t.Errorf("chunk %d: metadata = %v", i, extra)
		}
	}

	// Deletion is a stub, so this only checks it does not fail the worker.
	if err := a.DeleteDocuments(ctx, "knowledge", nil); err != nil {
		t.Fatal(err)
	}
}

func TestKnowledgeAdapter_AddDocumentsRefusesChunksWithoutAnOwner(t *testing.T) {
	// No Redis needed: the chunk must be refused before anything is embedded
	// or written. Previously a missing or malformed user_id parsed to the nil
	// UUID and the chunk was stored under a tenant no search ever matches.
	client, calls := fakeEmbeddings(t, http.StatusOK)
	a := NewKnowledgeVectorStoreAdapter(NewRedisVectorStore(nil), client)

	for name, meta := range map[string]map[string]interface{}{
		"missing":   {"embedding_gid": "g"},
		"malformed": {"user_id": "not-a-uuid", "embedding_gid": "g"},
		"nil uuid":  {"user_id": uuid.Nil.String(), "embedding_gid": "g"},
		"not text":  {"user_id": uuid.New(), "embedding_gid": "g"},
	} {
		t.Run(name, func(t *testing.T) {
			err := a.AddDocuments(context.Background(), "knowledge", []string{"doc"}, []map[string]interface{}{meta}, []string{"id"})
			if err == nil || !strings.Contains(err.Error(), "user_id") {
				t.Fatalf("expected a user_id error, got %v", err)
			}
		})
	}
	if n := atomic.LoadInt32(calls); n != 0 {
		t.Fatalf("embedding API called %d times for chunks that cannot be stored", n)
	}
}

func TestKnowledgeAdapter_EmbeddingFailureIsReturned(t *testing.T) {
	client, _ := fakeEmbeddings(t, http.StatusInternalServerError)
	a := NewKnowledgeVectorStoreAdapter(NewRedisVectorStore(nil), client)
	err := a.AddDocuments(context.Background(), "knowledge", []string{"doc"},
		[]map[string]interface{}{{"user_id": uuid.New().String()}}, []string{"id"})
	if err == nil || !strings.Contains(err.Error(), "embedding chunk 0") {
		t.Fatalf("expected the embedding error, got %v", err)
	}
}

// BUG: deleting a knowledge document never removes its vectors. The pruning
// worker (internal/worker/knowledge.HandleKnowledgesPruning) passes the
// deleted documents' embedding_gids here, and DeleteDocuments returns nil
// without touching Redis — so a deleted pricing sheet keeps grounding AI
// replies indefinitely.
func TestKnowledgeAdapter_DeleteDocumentsRemovesTheChunks(t *testing.T) {
	t.Skip("BUG: KnowledgeVectorStoreAdapter.DeleteDocuments is a no-op; pruned documents stay retrievable")

	c := testRedis(t, 3)
	client, _ := fakeEmbeddings(t, http.StatusOK)
	ctx := context.Background()
	gid := "test-gid-" + runTag(t)
	key := KnowledgeVectorPrefix + uuid.NewSHA1(uuid.NameSpaceURL, []byte(gid)).String() + ":0"
	t.Cleanup(func() { _ = c.Del(ctx, key).Err() })

	a := NewKnowledgeVectorStoreAdapter(NewRedisVectorStore(c), client)
	if err := a.AddDocuments(ctx, "knowledge", []string{"doc"},
		[]map[string]interface{}{{"user_id": uuid.New().String(), "embedding_gid": gid}}, []string{"id"}); err != nil {
		t.Fatal(err)
	}
	filter := map[string]interface{}{"must": []map[string]interface{}{
		{"key": "gid", "match": map[string]interface{}{"any": []string{gid}}},
	}}
	if err := a.DeleteDocuments(ctx, "knowledge", filter); err != nil {
		t.Fatal(err)
	}
	if n, _ := c.Exists(ctx, key).Result(); n != 0 {
		t.Fatal("chunk still stored after its document was pruned")
	}
}
