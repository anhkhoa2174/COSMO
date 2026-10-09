package skills

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
)

// SegmentationLogicSkill performs simple fit calculation from criteria/icp_definition JSONB.
// This is a lightweight, rule-based scorer; it does NOT persist (use ScoringSkill to save).
type SegmentationLogicSkill struct {
	segRepo *segmentation.SegmentationRepository
}

func NewSegmentationLogicSkill(segRepo *segmentation.SegmentationRepository) *SegmentationLogicSkill {
	return &SegmentationLogicSkill{segRepo: segRepo}
}

// EvaluateContactFit calculates fit_score based on segment criteria and contact data.
func (s *SegmentationLogicSkill) EvaluateContactFit(ctx context.Context, contact map[string]any, segID uuid.UUID) (*FitResult, error) {
	seg, err := s.segRepo.GetByID(ctx, segID)
	if err != nil || seg == nil {
		return nil, err
	}

	criteria := unmarshalMap(seg.Criteria)
	icp := unmarshalMap(seg.ICPDefinition)

	filterPass, _ := evalFilters(contact, criteria["filters"])
	scoreBreakdown := map[string]int{
		"title_match":        scoreTitle(contact, icp["ideal_titles"]),
		"company_size_match": scoreCompanySize(contact, icp["ideal_company_size"]),
		"industry_match":     scoreIndustry(contact, icp["ideal_industries"]),
		"engagement_level":   scoreEngagement(contact),
		"user_defined_score": scoreUserRules(contact, criteria["scoring_rules"]),
	}
	fitScore := weightedScore(scoreBreakdown)

	return &FitResult{
		FitScore:       fitScore,
		ScoreBreakdown: scoreBreakdown,
		PassesFilters:  filterPass,
	}, nil
}

func unmarshalMap(src []byte) map[string]any {
	dst := map[string]any{}
	if len(src) == 0 {
		return dst
	}
	_ = json.Unmarshal(src, &dst)
	return dst
}

func evalFilters(contact map[string]any, raw any) (bool, []map[string]any) {
	filters, ok := raw.([]any)
	if !ok {
		return true, nil
	}
	failed := []map[string]any{}
	for _, f := range filters {
		rule, ok := f.(map[string]any)
		if !ok {
			continue
		}
		// Criteria are user-edited JSON; a rule missing its field or operator
		// fails rather than panicking the request that evaluates it.
		field, _ := rule["field"].(string)
		op, _ := rule["operator"].(string)
		val := rule["value"]
		actual := extract(contact, field)
		if !applyOp(actual, op, val) {
			failed = append(failed, rule)
		}
	}
	return len(failed) == 0, failed
}

func extract(obj map[string]any, path string) any {
	parts := strings.Split(path, ".")
	curr := any(obj)
	for _, p := range parts {
		m, ok := curr.(map[string]any)
		if !ok {
			return nil
		}
		curr = m[p]
	}
	return curr
}

func applyOp(actual any, op string, expected any) bool {
	if actual == nil {
		return false
	}
	switch op {
	// == on interfaces panics when both hold the same uncomparable type (a
	// JSON array or object), so equality is structural.
	case "=":
		return reflect.DeepEqual(actual, expected)
	case "!=":
		return !reflect.DeepEqual(actual, expected)
	case ">=", ">":
		av, okA := toFloat(actual)
		ev, okE := toFloat(expected)
		if !okA || !okE {
			return false
		}
		if op == ">=" {
			return av >= ev
		}
		return av > ev
	case "<=", "<":
		av, okA := toFloat(actual)
		ev, okE := toFloat(expected)
		if !okA || !okE {
			return false
		}
		if op == "<=" {
			return av <= ev
		}
		return av < ev
	case "contains":
		as, ok := actual.(string)
		es, okE := expected.(string)
		return ok && okE && strings.Contains(strings.ToLower(as), strings.ToLower(es))
	}
	return false
}

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case float64:
		return t, true
	case float32:
		return float64(t), true
	}
	return 0, false
}

func scoreTitle(contact map[string]any, ideal any) int {
	idealList, _ := ideal.([]any)
	title := strings.ToLower(getString(contact, "job_title"))
	if title == "" {
		return 50
	}
	for _, v := range idealList {
		if s, ok := v.(string); ok && strings.ToLower(s) == title {
			return 100
		}
		if s, ok := v.(string); ok && strings.Contains(title, strings.ToLower(s)) {
			return 80
		}
	}
	if strings.Contains(title, "vp") || strings.Contains(title, "director") || strings.Contains(title, "head") || strings.Contains(title, "chief") {
		return 70
	}
	return 30
}

func scoreCompanySize(contact map[string]any, ideal any) int {
	idealList, _ := ideal.([]any)
	size := getString(contact, "profile.company_data.company_size")
	if size == "" || len(idealList) == 0 {
		return 50
	}
	for _, v := range idealList {
		if s, ok := v.(string); ok && s == size {
			return 100
		}
	}
	return 50
}

func scoreIndustry(contact map[string]any, ideal any) int {
	idealList, _ := ideal.([]any)
	industry := strings.ToLower(getString(contact, "profile.company_data.industry"))
	if industry == "" || len(idealList) == 0 {
		return 50
	}
	for _, v := range idealList {
		if s, ok := v.(string); ok && strings.Contains(industry, strings.ToLower(s)) {
			return 100
		}
	}
	return 20
}

func scoreEngagement(contact map[string]any) int {
	// Placeholder: read from scores.engagement if available
	if scores, ok := contact["scores"].(map[string]any); ok {
		if v, ok := toFloat(scores["engagement"]); ok {
			return int(v)
		}
	}
	return 50
}

func scoreUserRules(contact map[string]any, rules any) int {
	ruleList, ok := rules.([]any)
	if !ok || len(ruleList) == 0 {
		return 0
	}
	total := 0
	max := 0
	for _, r := range ruleList {
		rule, ok := r.(map[string]any)
		if !ok {
			continue
		}
		points := intFrom(rule["points"])
		max += points
		field := getString(rule, "field")
		op := getString(rule, "operator")
		val := rule["value"]
		actual := extract(contact, field)
		if applyOp(actual, op, val) {
			total += points
		}
	}
	if max == 0 {
		return 0
	}
	return int(float64(total) / float64(max) * 100)
}

func weightedScore(b map[string]int) int {
	weights := map[string]float64{
		"title_match":        0.25,
		"company_size_match": 0.15,
		"industry_match":     0.15,
		"engagement_level":   0.30,
		"user_defined_score": 0.15,
	}
	sum := 0.0
	for k, w := range weights {
		sum += float64(b[k]) * w
	}
	return int(sum)
}

func getString(m map[string]any, path string) string {
	val := extract(m, path)
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

func intFrom(v any) int {
	if f, ok := toFloat(v); ok {
		return int(f)
	}
	return 0
}
