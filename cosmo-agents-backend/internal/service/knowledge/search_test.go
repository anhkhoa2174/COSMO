package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/skills"
)

// Search used to be a stub that discarded every argument and returned an empty
// slice, so the endpoint reported "no results" for every query and nobody could
// tell the difference between an empty knowledge base and a feature that did
// nothing. These tests pin the behaviour that replaced it.

type fakeSearcher struct {
	chunks []skills.KnowledgeSearchResult
	err    error

	gotQuery string
	gotLimit int
	gotUser  uuid.UUID
	calls    int
}

func (f *fakeSearcher) Search(
	_ context.Context, userID uuid.UUID, query string, limit int,
) ([]skills.KnowledgeSearchResult, error) {
	f.calls++
	f.gotUser, f.gotQuery, f.gotLimit = userID, query, limit
	return f.chunks, f.err
}

func TestSearch_UnconfiguredReportsItRatherThanReturningNothing(t *testing.T) {
	// The distinction this makes is the reason the stub was a problem: an empty
	// list is a valid answer, so returning one hid the absence of the feature.
	svc := &KnowledgeService{}
	_, err := svc.Search(context.Background(), uuid.New(), "pricing", 5, nil)

	if err == nil {
		t.Fatal("an unconfigured search must report that, not answer emptily")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("unhelpful error: %v", err)
	}
}

func TestSearch_PassesTheQueryAndLimitThrough(t *testing.T) {
	// topK was validated by the handler and then discarded by the service.
	f := &fakeSearcher{chunks: []skills.KnowledgeSearchResult{
		{ChunkText: "Growth is $99 per seat.", Score: 0.12},
	}}
	user := uuid.New()
	svc := (&KnowledgeService{}).WithSearch(f)

	got, err := svc.Search(context.Background(), user, "how much", 3, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.gotQuery != "how much" || f.gotLimit != 3 || f.gotUser != user {
		t.Fatalf("arguments not passed through: q=%q limit=%d user=%v",
			f.gotQuery, f.gotLimit, f.gotUser)
	}
	if len(got) != 1 || got[0].Document != "Growth is $99 per seat." {
		t.Fatalf("chunk text not mapped: %+v", got)
	}
}

func TestSearch_DefaultsTheLimitWhenUnset(t *testing.T) {
	f := &fakeSearcher{}
	svc := (&KnowledgeService{}).WithSearch(f)

	for _, in := range []int{0, -1} {
		if _, err := svc.Search(context.Background(), uuid.New(), "q", in, nil); err != nil {
			t.Fatal(err)
		}
		if f.gotLimit != 10 {
			t.Fatalf("limit %d should have defaulted to 10, got %d", in, f.gotLimit)
		}
	}
}

func TestSearch_BlankQueryDoesNotReachTheEmbeddingCall(t *testing.T) {
	// Embedding an empty string costs a request and returns a vector that
	// matches nothing in particular, so the whitespace case is refused here.
	f := &fakeSearcher{}
	svc := (&KnowledgeService{}).WithSearch(f)

	got, err := svc.Search(context.Background(), uuid.New(), "   \n ", 5, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no results, got %d", len(got))
	}
	if f.calls != 0 {
		t.Fatalf("the searcher was called %d times for a blank query", f.calls)
	}
}

func TestSearch_ReportsRetrievalFailure(t *testing.T) {
	// A failed lookup must not be flattened into "no matches": the caller is
	// deciding whether the knowledge base can answer something.
	svc := (&KnowledgeService{}).WithSearch(
		&fakeSearcher{err: errors.New("redis unreachable")})

	if _, err := svc.Search(context.Background(), uuid.New(), "q", 5, nil); err == nil {
		t.Fatal("a retrieval error must surface")
	}
}

func TestSearch_ScoreIsLabelledAsADistance(t *testing.T) {
	// The score is a cosine distance, where smaller is closer. Returned bare
	// under the name "score" it reads as a relevance rating, which is the
	// misreading that left a 0.8 cut-off in place for so long.
	f := &fakeSearcher{chunks: []skills.KnowledgeSearchResult{
		{ChunkText: "a", Score: 0.2},
	}}
	svc := (&KnowledgeService{}).WithSearch(f)

	got, err := svc.Search(context.Background(), uuid.New(), "q", 5, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Metadata["cosine_distance"] != 0.2 {
		t.Fatalf("distance not reported in metadata: %+v", got[0].Metadata)
	}
}

func TestSearch_NoChunksIsAnEmptyListNotNil(t *testing.T) {
	// The handler serialises this straight to JSON, and nil marshals to `null`
	// where the documented shape is an array.
	svc := (&KnowledgeService{}).WithSearch(&fakeSearcher{})
	got, err := svc.Search(context.Background(), uuid.New(), "q", 5, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected an empty slice, got nil")
	}
}
