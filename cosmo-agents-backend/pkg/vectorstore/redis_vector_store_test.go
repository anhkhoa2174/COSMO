package vectorstore

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// These cover the retrieval boundary rather than Redis itself: which vectors a
// search is allowed to see, and which are close enough to be used as grounding.
// Both were wrong in ways that only show up with more than one tenant or more
// than one document, which is to say not during a demo.

func TestTenantTag_IsPureHexSoItNeedsNoEscaping(t *testing.T) {
	// RediSearch TAG treats "-" as a token separator, so a raw UUID would be
	// read as five tokens and the filter would not match. Stripping the hyphens
	// is what makes the tag usable in a query at all.
	id := uuid.MustParse("5436326f-2b66-40be-a425-373f7797d14d")
	got := tenantTag(id)

	if strings.Contains(got, "-") {
		t.Fatalf("tag still contains a separator: %q", got)
	}
	if got != "5436326f2b6640bea425373f7797d14d" {
		t.Fatalf("unexpected tag: %q", got)
	}
	for _, r := range got {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Fatalf("tag has a character that would need escaping: %q in %q", r, got)
		}
	}
}

func TestKnnQuery_FiltersTheTenantBeforeTheKnnStage(t *testing.T) {
	// The order matters and is the whole point of the change. A pre-filter
	// returns the nearest K among this tenant's vectors; a post-filter returns
	// the nearest K overall and then discards most of them, which starves the
	// caller when another tenant holds closer vectors.
	q := knnQuery("5436326f2b6640bea425373f7797d14d", 5)

	if !strings.HasPrefix(q, "(@user_id:{") {
		t.Fatalf("the tenant filter must come first: %q", q)
	}
	tagEnd := strings.Index(q, "})=>")
	knn := strings.Index(q, "[KNN")
	if tagEnd == -1 || knn == -1 || tagEnd > knn {
		t.Fatalf("filter is not applied before the KNN stage: %q", q)
	}
	if !strings.Contains(q, "[KNN 5 @vector $vec]") {
		t.Fatalf("limit not carried into the query: %q", q)
	}
	if strings.HasPrefix(q, "*=>") {
		t.Fatal("the query is unscoped — every tenant's vectors are searched")
	}
}

func result(userID string, score float64) VectorSearchResult {
	r := VectorSearchResult{Score: score}
	r.Metadata.UserID = userID
	return r
}

func TestKeepTenant_DropsOtherTenants(t *testing.T) {
	me := "aaaa1111"
	them := "bbbb2222"

	got := keepTenant([]VectorSearchResult{
		result(me, 0.1), result(them, 0.05), result(me, 0.2),
	}, me)

	if len(got) != 2 {
		t.Fatalf("expected my two rows, got %d", len(got))
	}
	for _, r := range got {
		if r.Metadata.UserID != me {
			t.Fatalf("kept another tenant's row: %q", r.Metadata.UserID)
		}
	}
}

func TestKeepTenant_DropsRowsWithNoOwner(t *testing.T) {
	// The previous filter read `UserID == "" || UserID == mine`, so a vector
	// stored without an owner was returned to every user. Nothing writes such a
	// vector today, but "shared with everyone" is the wrong default for the one
	// check standing between two customers' documents.
	got := keepTenant([]VectorSearchResult{
		result("", 0.01), result("aaaa1111", 0.3),
	}, "aaaa1111")

	if len(got) != 1 {
		t.Fatalf("expected only the owned row, got %d", len(got))
	}
	if got[0].Metadata.UserID == "" {
		t.Fatal("an ownerless vector must not be returned")
	}
}

func TestKeepTenant_EmptyInputIsEmptyOutput(t *testing.T) {
	if got := keepTenant(nil, "aaaa1111"); len(got) != 0 {
		t.Fatalf("expected nothing, got %d", len(got))
	}
}

func TestKeepWithinDistance_SmallerIsCloser(t *testing.T) {
	// The score is a cosine distance. Treating it as a similarity is the
	// mistake that made the old 0.8 "threshold" admit everything, so the
	// direction of the comparison is worth pinning.
	got := keepWithinDistance([]VectorSearchResult{
		result("u", 0.10), // near
		result("u", 0.50), // exactly at the limit
		result("u", 0.51), // just past it
		result("u", 0.90), // unrelated
	}, 0.5)

	if len(got) != 2 {
		t.Fatalf("expected the two within 0.5, got %d", len(got))
	}
	for _, r := range got {
		if r.Score > 0.5 {
			t.Fatalf("kept a result at distance %.2f", r.Score)
		}
	}
}

func TestKeepWithinDistance_TheOldValueWouldHaveKeptAlmostEverything(t *testing.T) {
	// A regression guard written as a comparison: at 0.8 a chunk with a
	// similarity of 0.2 survives, which is what "no filter" looks like.
	rows := []VectorSearchResult{
		result("u", 0.2), result("u", 0.55), result("u", 0.79),
	}
	if len(keepWithinDistance(rows, 0.8)) != 3 {
		t.Fatal("expected the old value to admit all three")
	}
	if got := len(keepWithinDistance(rows, 0.5)); got != 1 {
		t.Fatalf("the current value should admit only the closest, got %d", got)
	}
}

func TestKeepWithinDistance_ReusesTheBackingArray(t *testing.T) {
	// The filters write into the slice they were given, which is fine because
	// the caller does not use the original afterwards — but it means the
	// returned slice must not be longer than what was kept.
	rows := []VectorSearchResult{result("u", 0.9), result("u", 0.1)}
	got := keepWithinDistance(rows, 0.5)
	if len(got) != 1 || got[0].Score != 0.1 {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestKnnQueryWhere_AddsTheTypeFilterBeforeTheKnnStage(t *testing.T) {
	q := knnQueryWhere("abc123", typeFilter([]string{"pricing", "faq"}), 5)
	want := "(@user_id:{abc123} @knowledge_type:{pricing | faq})=>[KNN 5 @vector $vec]"
	if q != want {
		t.Fatalf("got %q, want %q", q, want)
	}
}

func TestKnnQueryWhere_NoFilterIsTheTenantOnlyQuery(t *testing.T) {
	if got, want := knnQueryWhere("abc123", "", 5), knnQuery("abc123", 5); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// Type names are interpolated into the query, so anything outside the known
// set must be dropped rather than passed through.
func TestTypeFilter_AcceptsOnlyKnownTypes(t *testing.T) {
	if got := typeFilter([]string{"pricing", "} | @user_id:{*", "faq"}); got != "@knowledge_type:{pricing | faq}" {
		t.Fatalf("unknown type reached the query: %q", got)
	}
	if got := typeFilter([]string{"nonsense"}); got != "" {
		t.Fatalf("a filter of only unknown types must be empty, got %q", got)
	}
	if got := typeFilter(nil); got != "" {
		t.Fatalf("no types must mean no filter, got %q", got)
	}
}
