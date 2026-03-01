package skills

// FitResult captures a simple segmentation fit computation.
type FitResult struct {
	FitScore       int            `json:"fit_score"`
	ScoreBreakdown map[string]int `json:"score_breakdown"`
	PassesFilters  bool           `json:"passes_filters"`
}
