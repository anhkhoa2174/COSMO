package vectorstore

import (
	"encoding/binary"
	"errors"
	"math"
	"strings"
	"testing"
)

// parseSearchResults has to read two wire shapes: the RESP3 map go-redis v9
// returns by default, and the RESP2 flat array an older server or a client
// pinned to protocol 2 returns. Both carry the tenant id that keepTenant then
// compares against, so a field read from the wrong slot is not a cosmetic
// error — it empties every search.

func TestParseSearchResults_MapFormat(t *testing.T) {
	s := &RedisVectorStore{}

	tests := []struct {
		name      string
		in        interface{}
		wantLen   int
		wantScore float64
		wantUser  string
		wantText  string
		wantID    string
	}{
		{
			name: "score as string, as RediSearch sends it",
			in: map[interface{}]interface{}{
				"results": []interface{}{
					map[interface{}]interface{}{
						"id": "vector:knowledge:abc:0",
						"extra_attributes": map[interface{}]interface{}{
							"__vector_score": "0.125",
							"id":             "abc:0",
							"user_id":        "aaaa1111",
							"entity_type":    "knowledge",
							"metadata":       `{"chunk_text":"Growth is $99"}`,
						},
					},
				},
			},
			// The row's id is the stored entity id, not the Redis key the
			// map carries at top level — the RESP2 path returns the former,
			// and callers put it in API responses and parse it as a UUID.
			wantLen: 1, wantScore: 0.125, wantUser: "aaaa1111", wantText: "Growth is $99", wantID: "abc:0",
		},
		{
			name: "score already a float",
			in: map[interface{}]interface{}{
				"results": []interface{}{
					map[interface{}]interface{}{
						"id": "k",
						"extra_attributes": map[interface{}]interface{}{
							"__vector_score": 0.25,
							"user_id":        "u",
						},
					},
				},
			},
			// No stored id: fall back to the key rather than to nothing.
			wantLen: 1, wantScore: 0.25, wantUser: "u", wantID: "k",
		},
		{
			name: "malformed metadata JSON leaves Extra empty but keeps the row",
			in: map[interface{}]interface{}{
				"results": []interface{}{
					map[interface{}]interface{}{
						"id": "k",
						"extra_attributes": map[interface{}]interface{}{
							"__vector_score": "0.3",
							"user_id":        "u",
							"metadata":       "{not json",
						},
					},
				},
			},
			wantLen: 1, wantScore: 0.3, wantUser: "u",
		},
		{
			name: "rows that are not maps are skipped",
			in: map[interface{}]interface{}{
				"results": []interface{}{"garbage", 42},
			},
			wantLen: 0,
		},
		{
			name:    "no results key",
			in:      map[interface{}]interface{}{"total_results": int64(0)},
			wantLen: 0,
		},
		{
			name:    "results is not an array",
			in:      map[interface{}]interface{}{"results": "nope"},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.parseSearchResults(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("got %d results, want %d: %+v", len(got), tt.wantLen, got)
			}
			if tt.wantLen == 0 {
				return
			}
			if got[0].Score != tt.wantScore {
				t.Errorf("score = %v, want %v", got[0].Score, tt.wantScore)
			}
			if got[0].Metadata.UserID != tt.wantUser {
				t.Errorf("user = %q, want %q", got[0].Metadata.UserID, tt.wantUser)
			}
			if tt.wantID != "" && got[0].ID != tt.wantID {
				t.Errorf("id = %q, want %q", got[0].ID, tt.wantID)
			}
			text, _ := got[0].Metadata.Extra["chunk_text"].(string)
			if text != tt.wantText {
				t.Errorf("chunk_text = %q, want %q", text, tt.wantText)
			}
		})
	}
}

func TestParseSearchResults_ArrayFormatWithoutScores(t *testing.T) {
	// [total, key, [field, value, ...], key, [field, value, ...]]
	in := []interface{}{
		int64(2),
		"vector:knowledge:a:0",
		[]interface{}{"__vector_score", "0.1", "id", "a:0", "user_id", "u1", "entity_type", "knowledge", "metadata", `{"chunk_text":"one"}`},
		"vector:knowledge:b:0",
		[]interface{}{"__vector_score", "0.2", "id", "b:0", "user_id", "u1", "entity_type", "knowledge", "metadata", `{"chunk_text":"two"}`},
	}
	got, err := (&RedisVectorStore{}).parseSearchResults(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d results, want 2: %+v", len(got), got)
	}
	if got[0].ID != "a:0" || got[0].Score != 0.1 || got[0].Metadata.UserID != "u1" || got[0].Metadata.Extra["chunk_text"] != "one" {
		t.Fatalf("first row misread: %+v", got[0])
	}
	if got[1].ID != "b:0" || got[1].Score != 0.2 {
		t.Fatalf("second row misread: %+v", got[1])
	}
}

// Every search is sent WITH WITHSCORES, and under RESP2 that inserts the
// document's text score between the key and its fields:
//
//	[total, key, score, [fields...], key, score, [fields...]]
//
// A parser that steps by two reads the score where it expects the fields,
// skips it, then reads the next *key* as a field list and skips that too —
// so every row is dropped and a RESP2 client never gets a single result.
func TestParseSearchResults_ArrayFormatWithScores(t *testing.T) {
	in := []interface{}{
		int64(2),
		"vector:knowledge:a:0", "1",
		[]interface{}{"__vector_score", "0.1", "id", "a:0", "user_id", "u1", "metadata", `{"chunk_text":"one"}`},
		"vector:knowledge:b:0", "1",
		[]interface{}{"__vector_score", "0.2", "id", "b:0", "user_id", "u1", "metadata", `{"chunk_text":"two"}`},
	}
	got, err := (&RedisVectorStore{}).parseSearchResults(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d results, want 2 — WITHSCORES layout misparsed: %+v", len(got), got)
	}
	if got[0].ID != "a:0" || got[0].Score != 0.1 || got[1].ID != "b:0" || got[1].Score != 0.2 {
		t.Fatalf("rows misread: %+v", got)
	}
}

func TestParseSearchResults_UnknownShapesAreEmpty(t *testing.T) {
	for _, in := range []interface{}{nil, "text", []interface{}{}, []interface{}{int64(0)}} {
		got, err := (&RedisVectorStore{}).parseSearchResults(in)
		if err != nil {
			t.Fatalf("%#v: unexpected error %v", in, err)
		}
		if len(got) != 0 {
			t.Fatalf("%#v: expected nothing, got %+v", in, got)
		}
	}
}

func TestIsDuplicateField(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"Duplicate field in schema - knowledge_type", true},
		{"DUPLICATE FIELD", true},
		{"Field already exists", true},
		{"Unknown index name", false},
		{"ERR wrong number of arguments", false},
	}
	for _, tt := range tests {
		if got := isDuplicateField(errors.New(tt.msg)); got != tt.want {
			t.Errorf("isDuplicateField(%q) = %v, want %v", tt.msg, got, tt.want)
		}
	}
}

func TestFloatToBits_MatchesIEEE754(t *testing.T) {
	// The index reads FLOAT32 little-endian, so the bits must be the standard
	// IEEE-754 encoding and not, say, a truncated integer.
	for _, f := range []float32{0, 1, -1, 0.5, 3.1415927, float32(math.Inf(1))} {
		if got, want := floatToBits(f), math.Float32bits(f); got != want {
			t.Errorf("floatToBits(%v) = %#x, want %#x", f, got, want)
		}
	}
}

// encodeVector mirrors storeVector's byte layout, used to check what reached
// Redis.
func decodeVector(b []byte) []float32 {
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

func TestTypeFilter_Table(t *testing.T) {
	tests := []struct {
		name  string
		types []string
		want  string
	}{
		{"single", []string{"pricing"}, "@knowledge_type:{pricing}"},
		// Underscore is not a TAG query separator, so case_study needs no
		// escaping; the integration test confirms the server agrees.
		{"underscore type", []string{"case_study"}, "@knowledge_type:{case_study}"},
		{"all known", []string{"pricing", "product", "case_study", "faq", "other"},
			"@knowledge_type:{pricing | product | case_study | faq | other}"},
		// Case matters: TAG values are compared as stored, and the known set is
		// lower-case, so an upper-case type is not silently widened.
		{"wrong case dropped", []string{"PRICING"}, ""},
		{"whitespace dropped", []string{" pricing"}, ""},
		{"injection dropped", []string{"pricing}) | (@user_id:{*"}, ""},
		{"empty strings dropped", []string{"", "faq"}, "@knowledge_type:{faq}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := typeFilter(tt.types)
			if got != tt.want {
				t.Fatalf("typeFilter(%q) = %q, want %q", tt.types, got, tt.want)
			}
			if strings.Count(got, "{") != strings.Count(got, "}") {
				t.Fatalf("unbalanced braces in %q", got)
			}
		})
	}
}
