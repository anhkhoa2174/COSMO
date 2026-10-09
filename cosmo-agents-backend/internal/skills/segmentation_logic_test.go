package skills

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"gorm.io/gorm"
)

// jsonMap decodes a JSON literal the way segment criteria reach the scorer:
// numbers as float64, arrays as []any, objects as map[string]any.
func jsonMap(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("bad fixture %s: %v", s, err)
	}
	return m
}

func TestApplyOp(t *testing.T) {
	tests := []struct {
		name     string
		actual   any
		op       string
		expected any
		want     bool
	}{
		{"equal strings", "saas", "=", "saas", true},
		{"unequal strings", "saas", "=", "fintech", false},
		{"not equal", "saas", "!=", "fintech", true},
		{"equal JSON numbers", 50.0, "=", 50.0, true},
		// Both sides decoded from JSON arrays: == on interfaces holding slices
		// panics at run time, which took down the request evaluating a segment.
		{"equal arrays do not panic", []any{"a"}, "=", []any{"a"}, true},
		{"unequal objects do not panic", map[string]any{"a": 1.0}, "!=", map[string]any{"a": 2.0}, true},
		{">= true", 10.0, ">=", 10.0, true},
		{"> false at equality", 10.0, ">", 10.0, false},
		{"<= with int actual", 3, "<=", 5.0, true},
		{"< with int64 and float32", int64(2), "<", float32(2.5), true},
		{"numeric op on a string fails", "10", ">=", 5.0, false},
		{"numeric op on a string expectation fails", 10.0, "<", "20", false},
		{"contains is case-insensitive", "VP of Sales", "contains", "vp", true},
		{"contains needs strings", 5.0, "contains", "5", false},
		{"unknown operator", "x", "~=", "x", false},
		{"missing field never matches", nil, "!=", "x", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := applyOp(tt.actual, tt.op, tt.expected); got != tt.want {
				t.Fatalf("applyOp(%v %s %v) = %v, want %v", tt.actual, tt.op, tt.expected, got, tt.want)
			}
		})
	}
}

func TestEvalFilters(t *testing.T) {
	contact := jsonMap(t, `{"job_title":"CTO","profile":{"company_data":{"employees":120}}}`)
	tests := []struct {
		name       string
		filters    string
		wantPass   bool
		wantFailed int
	}{
		{"no filters passes", `null`, true, 0},
		{"filters not a list passes", `{"field":"x"}`, true, 0},
		{"all match", `[{"field":"job_title","operator":"=","value":"CTO"},
			{"field":"profile.company_data.employees","operator":">=","value":100}]`, true, 0},
		{"one fails", `[{"field":"job_title","operator":"=","value":"CEO"},
			{"field":"profile.company_data.employees","operator":">=","value":100}]`, false, 1},
		{"non-object rules are ignored", `["junk", 3]`, true, 0},
		// User-edited criteria missing a key used to hit an unchecked type
		// assertion and panic; the rule now simply fails.
		{"rule without field fails instead of panicking", `[{"operator":"=","value":"CTO"}]`, false, 1},
		{"rule without operator fails instead of panicking", `[{"field":"job_title","value":"CTO"}]`, false, 1},
		{"numeric field name fails instead of panicking", `[{"field":7,"operator":"=","value":"CTO"}]`, false, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw any
			if err := json.Unmarshal([]byte(tt.filters), &raw); err != nil {
				t.Fatal(err)
			}
			pass, failed := evalFilters(contact, raw)
			if pass != tt.wantPass || len(failed) != tt.wantFailed {
				t.Fatalf("pass=%v failed=%d, want pass=%v failed=%d", pass, len(failed), tt.wantPass, tt.wantFailed)
			}
		})
	}
}

func TestExtractAndGetString(t *testing.T) {
	m := jsonMap(t, `{"a":{"b":{"c":"deep"}},"n":3}`)
	if got := extract(m, "a.b.c"); got != "deep" {
		t.Errorf("extract a.b.c = %v", got)
	}
	if got := extract(m, "a.x.c"); got != nil {
		t.Errorf("missing path = %v", got)
	}
	if got := extract(m, "n.x"); got != nil {
		t.Errorf("path through a scalar = %v", got)
	}
	if got := getString(m, "n"); got != "" {
		t.Errorf("getString of a number = %q", got)
	}
	if intFrom(7.9) != 7 || intFrom("7") != 0 {
		t.Error("intFrom mis-converts")
	}
}

func TestScoreTitle(t *testing.T) {
	ideal := []any{"CTO", "Head of Engineering"}
	tests := []struct {
		title string
		want  int
	}{
		{"", 50},
		{"cto", 100},                     // exact, case-insensitive
		{"Head of Engineering EMEA", 80}, // contains an ideal title
		{"VP Marketing", 70},             // seniority keyword
		{"Chief of Staff", 70},
		{"Engineer", 30},
	}
	for _, tt := range tests {
		if got := scoreTitle(map[string]any{"job_title": tt.title}, ideal); got != tt.want {
			t.Errorf("scoreTitle(%q) = %d, want %d", tt.title, got, tt.want)
		}
	}
}

func TestScoreCompanySizeAndIndustry(t *testing.T) {
	contact := jsonMap(t, `{"profile":{"company_data":{"company_size":"51-200","industry":"B2B SaaS"}}}`)
	blank := map[string]any{}

	if got := scoreCompanySize(contact, []any{"51-200"}); got != 100 {
		t.Errorf("matching size = %d", got)
	}
	if got := scoreCompanySize(contact, []any{"1000+"}); got != 50 {
		t.Errorf("non-matching size = %d", got)
	}
	if got := scoreCompanySize(blank, []any{"51-200"}); got != 50 {
		t.Errorf("unknown size = %d", got)
	}
	if got := scoreCompanySize(contact, nil); got != 50 {
		t.Errorf("no ideal size = %d", got)
	}

	if got := scoreIndustry(contact, []any{"saas"}); got != 100 {
		t.Errorf("matching industry = %d", got)
	}
	if got := scoreIndustry(contact, []any{"retail"}); got != 20 {
		t.Errorf("non-matching industry = %d", got)
	}
	if got := scoreIndustry(blank, []any{"saas"}); got != 50 {
		t.Errorf("unknown industry = %d", got)
	}
}

func TestScoreEngagementAndUserRules(t *testing.T) {
	if got := scoreEngagement(jsonMap(t, `{"scores":{"engagement":72}}`)); got != 72 {
		t.Errorf("engagement = %d", got)
	}
	if got := scoreEngagement(map[string]any{}); got != 50 {
		t.Errorf("default engagement = %d", got)
	}

	contact := jsonMap(t, `{"job_title":"CTO","company":"Acme"}`)
	rules := func(s string) any {
		var v any
		_ = json.Unmarshal([]byte(s), &v)
		return v
	}
	tests := []struct {
		name  string
		rules string
		want  int
	}{
		{"no rules", `[]`, 0},
		{"not a list", `{}`, 0},
		{"all points earned", `[{"field":"job_title","operator":"=","value":"CTO","points":10}]`, 100},
		{"a quarter earned", `[{"field":"job_title","operator":"=","value":"CTO","points":10},
			{"field":"company","operator":"=","value":"Other","points":30}]`, 25},
		{"zero points in total", `[{"field":"job_title","operator":"=","value":"CTO","points":0}]`, 0},
		{"junk rules skipped", `["x",{"field":"company","operator":"contains","value":"ac","points":5}]`, 100},
	}
	for _, tt := range tests {
		if got := scoreUserRules(contact, rules(tt.rules)); got != tt.want {
			t.Errorf("%s: scoreUserRules = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestWeightedScore(t *testing.T) {
	all := func(v int) map[string]int {
		return map[string]int{"title_match": v, "company_size_match": v, "industry_match": v, "engagement_level": v, "user_defined_score": v}
	}
	if got := weightedScore(all(100)); got != 100 {
		t.Errorf("all 100 = %d; the weights must sum to 1", got)
	}
	if got := weightedScore(all(0)); got != 0 {
		t.Errorf("all 0 = %d", got)
	}
	// Engagement carries the largest weight (0.30).
	if got := weightedScore(map[string]int{"engagement_level": 100}); got != 30 {
		t.Errorf("engagement only = %d", got)
	}
}

func TestSegmentationLogic_EvaluateContactFit(t *testing.T) {
	db, err := pgtest.Open(t, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.Segmentation{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := segmentation.NewSegmentationRepository(db)
	seg := &domain.Segmentation{
		UserID:        uuid.New(),
		Name:          "CTOs",
		Criteria:      domain.JSONB(`{"filters":[{"field":"job_title","operator":"=","value":"CTO"}]}`),
		ICPDefinition: domain.JSONB(`{"ideal_titles":["CTO"]}`),
	}
	if _, err := repo.Create(ctx, seg); err != nil {
		t.Fatal(err)
	}
	s := NewSegmentationLogicSkill(repo)

	fit, err := s.EvaluateContactFit(ctx, map[string]any{"job_title": "CTO"}, seg.ID)
	if err != nil || fit == nil {
		t.Fatalf("fit = %v, %v", fit, err)
	}
	if !fit.PassesFilters || fit.ScoreBreakdown["title_match"] != 100 {
		t.Fatalf("unexpected fit: %+v", fit)
	}
	// 100*.25 + 50*.15 + 50*.15 + 50*.30 + 0*.15
	if fit.FitScore != 55 {
		t.Fatalf("fit score = %d, want 55", fit.FitScore)
	}

	miss, err := s.EvaluateContactFit(ctx, map[string]any{"job_title": "Intern"}, seg.ID)
	if err != nil || miss.PassesFilters {
		t.Fatalf("an intern should fail the CTO filter: %+v, %v", miss, err)
	}

	// An unknown segment yields no result and no error; the caller skips it.
	none, err := s.EvaluateContactFit(ctx, map[string]any{}, uuid.New())
	if none != nil || err != nil {
		t.Fatalf("missing segment = %v, %v", none, err)
	}
}

func TestUnmarshalMap(t *testing.T) {
	if got := unmarshalMap(nil); len(got) != 0 {
		t.Errorf("nil = %v", got)
	}
	if got := unmarshalMap([]byte(`not json`)); len(got) != 0 {
		t.Errorf("bad json = %v", got)
	}
	if got := unmarshalMap([]byte(`{"a":1}`)); got["a"] != 1.0 {
		t.Errorf("good json = %v", got)
	}
}
