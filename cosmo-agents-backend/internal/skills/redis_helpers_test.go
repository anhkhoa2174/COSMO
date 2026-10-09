package skills

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/vectorstore"
)

// The search skills call the store's exported methods, which are bound to the
// application's index names, so these tests use those indexes on a real Redis
// Stack. Isolation comes from the tenant: every test writes under a fresh
// random user id, which the store's pre-filter confines every search to, and
// deletes the keys it wrote when it ends. Nothing is created or dropped.

func redisAddr() string {
	if a := os.Getenv("REDIS_TEST_ADDR"); a != "" {
		return a
	}
	return "localhost:6381"
}

// appRedis returns a client, skipping when Redis is unreachable or the
// application's vector indexes have never been created on it.
func appRedis(t *testing.T) *redis.Client {
	t.Helper()
	c := redis.NewClient(&redis.Options{Addr: redisAddr()})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		t.Skipf("Redis not reachable at %s: %v", redisAddr(), err)
	}
	for _, idx := range []string{vectorstore.KnowledgeVectorIndex, vectorstore.ContactVectorIndex,
		vectorstore.InteractionVectorIndex, vectorstore.SegmentVectorIndex} {
		if err := c.Do(ctx, "FT.INFO", idx).Err(); err != nil {
			_ = c.Close()
			t.Skipf("index %s missing on %s: %v", idx, redisAddr(), err)
		}
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// deleteOnCleanup removes the given keys when the test ends.
func deleteOnCleanup(t *testing.T, c *redis.Client, keys ...string) {
	t.Cleanup(func() { _ = c.Del(context.Background(), keys...).Err() })
}

// vecAtDistance returns a unit vector whose cosine distance from e0 is d.
func vecAtDistance(d float64) []float32 {
	v := make([]float32, vectorstore.VectorDimensions)
	c := 1 - d
	v[0] = float32(c)
	v[1] = float32(math.Sqrt(math.Max(0, 1-c*c)))
	return v
}

// fakeEmbeddings serves the embeddings endpoint locally and answers every
// input with e0, so a query "is" the unit vector the stored chunks are placed
// around. status other than 200 makes every call fail.
func fakeEmbeddings(t *testing.T, status int) (*ai.OpenAIClient, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if !strings.HasSuffix(r.URL.Path, "/embeddings") || status != http.StatusOK {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"message":"boom","type":"server_error"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"model":  "fake",
			"data":   []map[string]interface{}{{"object": "embedding", "index": 0, "embedding": vecAtDistance(0)}},
		})
	}))
	t.Cleanup(srv.Close)
	t.Setenv("OPENAI_BASE_URL", srv.URL)
	return ai.NewOpenAIClient(ai.Config{APIKey: "test"}), &calls
}

// eventually retries f until it reports done, for the index's asynchronous
// catch-up after a write.
func eventually(t *testing.T, f func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if f() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}
