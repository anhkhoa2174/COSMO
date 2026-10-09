package vectorstore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// These run against a real Redis Stack, because the parts most likely to be
// wrong are the parts only the server can judge: whether a TAG filter matches
// what was stored, what shape a reply comes back in, and which error strings
// the startup code compares against.
//
// Isolation: every test builds its own index over a key prefix no other test
// or dev data shares. The prefix is a random first UUID segment, and the ids
// written are minted with that segment, so the store's hard-coded key layout
// (vector:knowledge:<id>:<chunk>) lands under it. The index is dropped and the
// keys deleted when the test ends.

func redisAddr() string {
	if a := os.Getenv("REDIS_TEST_ADDR"); a != "" {
		return a
	}
	return "localhost:6381"
}

// testRedis connects with the given RESP protocol and skips only when no
// server is reachable or it lacks RediSearch.
func testRedis(t *testing.T, protocol int) *redis.Client {
	t.Helper()
	c := redis.NewClient(&redis.Options{Addr: redisAddr(), Protocol: protocol})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		_ = c.Close()
		t.Skipf("Redis not reachable at %s: %v", redisAddr(), err)
	}
	if err := c.Do(ctx, "FT._LIST").Err(); err != nil {
		_ = c.Close()
		t.Skipf("Redis at %s has no RediSearch: %v", redisAddr(), err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// runTag is 8 random hex characters: a UUID's first segment.
func runTag(t *testing.T) string {
	t.Helper()
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}

// taggedID returns a fresh UUID whose first segment is tag, so its key falls
// under this test's prefix.
func taggedID(t *testing.T, tag string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	return uuid.MustParse(tag + id.String()[8:])
}

// testIndex creates an index named uniquely over prefix+tag and removes it and
// its keys afterwards.
func testIndex(t *testing.T, c *redis.Client, prefix, tag string) string {
	t.Helper()
	ctx := context.Background()
	name := "idx:test:" + tag + ":" + strings.TrimSuffix(strings.TrimPrefix(prefix, "vector:"), ":")
	s := NewRedisVectorStore(c)
	if err := s.createVectorIndex(ctx, name, prefix+tag); err != nil {
		t.Fatalf("create test index: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Do(ctx, "FT.DROPINDEX", name).Err()
		iter := c.Scan(ctx, 0, prefix+tag+"*", 100).Iterator()
		for iter.Next(ctx) {
			_ = c.Del(ctx, iter.Val()).Err()
		}
	})
	return name
}

// vecAtDistance returns a unit vector whose cosine distance from e0 is d.
func vecAtDistance(d float64) []float32 {
	v := make([]float32, VectorDimensions)
	c := 1 - d
	v[0] = float32(c)
	v[1] = float32(math.Sqrt(math.Max(0, 1-c*c)))
	return v
}

func e0() []float32 { return vecAtDistance(0) }

// waitIndexed waits for the background indexer to catch up with n documents.
func waitIndexed(t *testing.T, c *redis.Client, index string, n int) {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		res, err := c.Do(ctx, "FT.SEARCH", index, "*", "LIMIT", "0", "0").Result()
		if err == nil {
			total := int64(-1)
			switch r := res.(type) {
			case []interface{}:
				if len(r) > 0 {
					total, _ = r[0].(int64)
				}
			case map[interface{}]interface{}:
				total, _ = r["total_results"].(int64)
			}
			if total >= int64(n) {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("index %s did not reach %d documents", index, n)
}

func texts(rs []VectorSearchResult) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		s, _ := r.Metadata.Extra["chunk_text"].(string)
		out = append(out, s)
	}
	return out
}

func TestRedis_KnowledgeSearchScopesByTenantTypeAndDistance(t *testing.T) {
	for _, protocol := range []int{2, 3} {
		t.Run(fmt.Sprintf("RESP%d", protocol), func(t *testing.T) {
			c := testRedis(t, protocol)
			ctx := context.Background()
			tag := runTag(t)
			index := testIndex(t, c, KnowledgeVectorPrefix, tag)
			s := NewRedisVectorStore(c)

			me, them := uuid.New(), uuid.New()
			seed := []struct {
				owner uuid.UUID
				dist  float64
				text  string
				typ   string
			}{
				{me, 0.05, "untyped-near", ""},
				{me, 0.10, "pricing-near", "pricing"},
				{me, 0.20, "case-study-near", "case_study"},
				{me, 0.30, "faq-near", "faq"},
				{me, 0.70, "pricing-far", "pricing"},
				// Another tenant's chunk is the closest of all; it must never
				// surface, and must not use up this tenant's K.
				{them, 0.00, "their-pricing", "pricing"},
			}
			for i, d := range seed {
				if err := s.StoreKnowledgeChunk(ctx, taggedID(t, tag), d.owner, i, vecAtDistance(d.dist), d.text, d.text+".pdf", d.typ); err != nil {
					t.Fatalf("store %s: %v", d.text, err)
				}
			}
			waitIndexed(t, c, index, len(seed))

			tests := []struct {
				name  string
				types []string
				limit int
				want  []string
			}{
				{"no type filter keeps the near chunks in distance order", nil, 10,
					[]string{"untyped-near", "pricing-near", "case-study-near", "faq-near"}},
				{"pricing only", []string{"pricing"}, 10, []string{"pricing-near"}},
				{"underscore tag value matches", []string{"case_study"}, 10, []string{"case-study-near"}},
				{"union of types", []string{"faq", "case_study"}, 10, []string{"case-study-near", "faq-near"}},
				{"type with no documents", []string{"product"}, 10, []string{}},
				{"limit is honoured after the tenant filter", nil, 2, []string{"untyped-near", "pricing-near"}},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					got, err := s.searchVectorsWhere(ctx, index, e0(), me, typeFilter(tt.types), tt.limit, 0.5)
					if err != nil {
						t.Fatalf("search: %v", err)
					}
					if g := texts(got); strings.Join(g, ",") != strings.Join(tt.want, ",") {
						t.Fatalf("got %v, want %v", g, tt.want)
					}
					for i, r := range got {
						if r.Score > 0.5 {
							t.Errorf("kept a result at distance %.3f", r.Score)
						}
						if i > 0 && r.Score < got[i-1].Score {
							t.Errorf("results not sorted by distance: %v", got)
						}
						if r.Metadata.UserID != tenantTag(me) {
							t.Errorf("row from another tenant: %q", r.Metadata.UserID)
						}
					}
				})
			}

			// Title and type travel with the chunk for the prompt to label it.
			got, err := s.searchVectorsWhere(ctx, index, e0(), me, typeFilter([]string{"case_study"}), 5, 0.5)
			if err != nil || len(got) != 1 {
				t.Fatalf("got %v, %v", got, err)
			}
			if got[0].Metadata.Extra["title"] != "case-study-near.pdf" || got[0].Metadata.Extra["knowledge_type"] != "case_study" {
				t.Fatalf("labels lost: %+v", got[0].Metadata.Extra)
			}

			// The other tenant sees only their own chunk.
			theirs, err := s.searchVectorsWhere(ctx, index, e0(), them, "", 10, 0.5)
			if err != nil {
				t.Fatal(err)
			}
			if g := texts(theirs); len(g) != 1 || g[0] != "their-pricing" {
				t.Fatalf("other tenant got %v", g)
			}

			// maxDistance 0 means "no distance filter": the far chunk returns.
			all, err := s.searchVectorsWhere(ctx, index, e0(), me, "", 10, 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(all) != 5 {
				t.Fatalf("unfiltered search returned %v", texts(all))
			}
		})
	}
}

func TestRedis_StoredHashLayout(t *testing.T) {
	c := testRedis(t, 3)
	ctx := context.Background()
	tag := runTag(t)
	_ = testIndex(t, c, KnowledgeVectorPrefix, tag)
	s := NewRedisVectorStore(c)

	user := uuid.MustParse("5436326f-2b66-40be-a425-373f7797d14d")
	kid := taggedID(t, tag)
	vec := vecAtDistance(0.25)
	if err := s.StoreKnowledgeVector(ctx, kid, user, 3, vec, "hello"); err != nil {
		t.Fatal(err)
	}
	key := KnowledgeVectorPrefix + kid.String() + ":3"

	h, err := c.HGetAll(ctx, key).Result()
	if err != nil {
		t.Fatal(err)
	}
	if h["user_id"] != "5436326f2b6640bea425373f7797d14d" {
		t.Errorf("user_id stored as %q; the TAG filter expects it without hyphens", h["user_id"])
	}
	if h["id"] != kid.String()+":3" || h["entity_type"] != "knowledge" {
		t.Errorf("unexpected id/entity_type: %q %q", h["id"], h["entity_type"])
	}
	// An untyped chunk must not carry a knowledge_type field at all — an empty
	// TAG value would be a type of its own.
	if _, ok := h["knowledge_type"]; ok {
		t.Errorf("untyped chunk has a knowledge_type field: %q", h["knowledge_type"])
	}
	got := decodeVector([]byte(h["vector"]))
	if len(got) != len(vec) || got[0] != vec[0] || got[1] != vec[1] {
		t.Errorf("vector bytes do not round-trip as little-endian FLOAT32")
	}
}

func TestRedis_ContactVectorStoreSearchDelete(t *testing.T) {
	c := testRedis(t, 3)
	ctx := context.Background()
	tag := runTag(t)
	index := testIndex(t, c, ContactVectorPrefix, tag)
	s := NewRedisVectorStore(c)

	user := uuid.New()
	contact := taggedID(t, tag)
	if err := s.StoreContactVector(ctx, contact, user, vecAtDistance(0.1), map[string]interface{}{"name": "Ada"}); err != nil {
		t.Fatal(err)
	}
	waitIndexed(t, c, index, 1)

	got, err := s.searchVectors(ctx, index, e0(), user, 5, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Metadata.Extra["name"] != "Ada" || got[0].Metadata.EntityType != "contact" {
		t.Fatalf("got %+v", got)
	}
	if got[0].ID != contact.String() {
		t.Errorf("result id = %q, want the contact id %q", got[0].ID, contact)
	}
	if math.Abs(got[0].Score-0.1) > 1e-3 {
		t.Errorf("distance = %v, want ~0.1 (the score is a cosine distance)", got[0].Score)
	}

	if err := s.DeleteContactVector(ctx, contact); err != nil {
		t.Fatal(err)
	}
	if n, _ := c.Exists(ctx, ContactVectorPrefix+contact.String()).Result(); n != 0 {
		t.Fatal("contact vector still present after delete")
	}
}

func TestRedis_InteractionAndSegmentVectorsAreStoredUnderTheirPrefixes(t *testing.T) {
	c := testRedis(t, 3)
	ctx := context.Background()
	tag := runTag(t)
	_ = testIndex(t, c, InteractionVectorPrefix, tag)
	_ = testIndex(t, c, SegmentVectorPrefix, tag)
	s := NewRedisVectorStore(c)

	iid, sid := taggedID(t, tag), taggedID(t, tag)
	if err := s.StoreInteractionVector(ctx, iid, uuid.New(), e0(), nil); err != nil {
		t.Fatal(err)
	}
	if err := s.StoreSegmentVector(ctx, sid, uuid.New(), e0(), map[string]interface{}{"name": "SaaS"}); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		InteractionVectorPrefix + iid.String(): "interaction",
		SegmentVectorPrefix + sid.String():     "segment",
	} {
		if got, _ := c.HGet(ctx, key, "entity_type").Result(); got != want {
			t.Errorf("%s: entity_type = %q, want %q", key, got, want)
		}
	}
}

// InitializeIndexes treats FT.CREATE failing with exactly "Index already
// exists" as success, and FT.ALTER failing with a duplicate-field message as
// success. Both strings come from the server, so they are checked against it.
func TestRedis_IndexErrorStringsMatchWhatStartupExpects(t *testing.T) {
	c := testRedis(t, 3)
	ctx := context.Background()
	tag := runTag(t)
	index := testIndex(t, c, KnowledgeVectorPrefix, tag)
	s := NewRedisVectorStore(c)

	err := s.createVectorIndex(ctx, index, KnowledgeVectorPrefix+tag)
	if err == nil || err.Error() != "Index already exists" {
		t.Fatalf("re-creating an index returned %v; InitializeIndexes compares against \"Index already exists\"", err)
	}

	err = c.Do(ctx, "FT.ALTER", index, "SCHEMA", "ADD", "knowledge_type", "TAG").Err()
	if err == nil || !isDuplicateField(err) {
		t.Fatalf("re-adding knowledge_type returned %v, which isDuplicateField does not recognise", err)
	}

	if err := c.Do(ctx, "FT.SEARCH", "idx:test:missing:"+tag, "*").Err(); err == nil {
		t.Fatal("searching a missing index should fail")
	} else if _, serr := s.searchVectors(ctx, "idx:test:missing:"+tag, e0(), uuid.New(), 1, 0.5); serr == nil {
		t.Fatal("searchVectors must surface a missing index as an error")
	}
}

// The Initializer's read-only checks, against both reply protocols. They do
// not create or drop anything.
func TestRedis_InitializerReadOnlyChecks(t *testing.T) {
	for _, protocol := range []int{2, 3} {
		t.Run(fmt.Sprintf("RESP%d", protocol), func(t *testing.T) {
			c := testRedis(t, protocol)
			init := NewInitializer(c)
			if init.GetVectorStore() == nil {
				t.Fatal("no vector store")
			}
			if err := init.verifyRedisStack(context.Background()); err != nil {
				t.Fatalf("verifyRedisStack on a Redis Stack server: %v", err)
			}

			// The stats and health check read the application's own indexes;
			// on a server that has never run the app there is nothing to read.
			if err := init.HealthCheck(context.Background()); err != nil {
				t.Skipf("application indexes absent on this server: %v", err)
			}
			stats, err := init.GetIndexStats(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			for _, idx := range []string{ContactVectorIndex, InteractionVectorIndex, KnowledgeVectorIndex, SegmentVectorIndex} {
				st, ok := stats[idx]
				if !ok {
					t.Fatalf("no stats for %s (got %v)", idx, stats)
				}
				// A count is a number, never blank: blank is what the parser
				// left when it expected a string and the server sent an int.
				if _, err := fmt.Sscanf(st.DocumentCount, "%d", new(int)); err != nil {
					t.Errorf("%s: document count %q is not a number", idx, st.DocumentCount)
				}
				if _, err := fmt.Sscanf(st.RecordCount, "%d", new(int)); err != nil {
					t.Errorf("%s: record count %q is not a number", idx, st.RecordCount)
				}
			}
		})
	}
}
