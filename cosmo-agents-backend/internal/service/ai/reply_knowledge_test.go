package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/skills"
)

type fakeRetriever struct {
	results []skills.KnowledgeSearchResult
	err     error

	gotQuery, gotIntent string
	gotLimit            int
}

func (f *fakeRetriever) SearchForIntent(_ context.Context, _ uuid.UUID, query, intent string, limit int) ([]skills.KnowledgeSearchResult, error) {
	f.gotQuery, f.gotIntent, f.gotLimit = query, intent, limit
	return f.results, f.err
}

// The reply is grounded in the passages retrieved for the message being
// answered, queried with that message and its intent.
func TestBuildKnowledgeContext_UsesRetrievedPassages(t *testing.T) {
	r := &fakeRetriever{results: []skills.KnowledgeSearchResult{
		{ChunkText: "Growth is $99 per seat per month.", Title: "pricing.pdf", Type: "pricing"},
		{ChunkText: "Every plan starts with a 14-day trial.", Title: ""},
	}}
	svc := (&AIEmailService{}).WithKnowledgeSearch(r)

	got, err := svc.buildKnowledgeContext(context.Background(), uuid.New(),
		"How much does it cost?", "Request for pricing")
	if err != nil {
		t.Fatal(err)
	}
	if r.gotQuery != "How much does it cost?" || r.gotIntent != "Request for pricing" || r.gotLimit != replyChunks {
		t.Fatalf("retriever called with query=%q intent=%q limit=%d", r.gotQuery, r.gotIntent, r.gotLimit)
	}
	want := "<doc title=\"pricing.pdf\">\nGrowth is $99 per seat per month.\n</doc>\n" +
		"<doc title=\"Company knowledge\">\nEvery plan starts with a 14-day trial.\n</doc>"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// Retrieval failing, or finding nothing close enough, must not leave the reply
// without grounding: it falls back to summaries.
func TestBuildKnowledgeContext_FallsBackWhenRetrievalGivesNothing(t *testing.T) {
	for name, r := range map[string]*fakeRetriever{
		"no results": {},
		"error":      {err: errors.New("redis down")},
	} {
		t.Run(name, func(t *testing.T) {
			svc := (&AIEmailService{}).WithKnowledgeSearch(r)
			got, err := svc.buildKnowledgeContext(context.Background(), uuid.New(), "hi", "Interested")
			if err != nil {
				t.Fatal(err)
			}
			if got != "No internal knowledge available." {
				t.Fatalf("got %q", got)
			}
		})
	}
}

func TestKnowledgeDoc_QuotesInTitlesCannotEndTheAttribute(t *testing.T) {
	got := knowledgeDoc(`price "list".pdf`, "x")
	if strings.Count(got, `"`) != 2 {
		t.Fatalf("title quote leaked into the attribute: %q", got)
	}
}

func TestKnowledgeTitle_ReadsTheUploadedFileName(t *testing.T) {
	meta := []byte(`{"origin":{"filename":"faq.md","file_size":10}}`)
	if got := knowledgeTitle(meta); got != "faq.md" {
		t.Fatalf("got %q", got)
	}
	if got := knowledgeTitle([]byte("not json")); got != "" {
		t.Fatalf("malformed metadata must give no title, got %q", got)
	}
}
