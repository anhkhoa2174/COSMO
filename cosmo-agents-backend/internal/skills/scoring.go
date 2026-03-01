package skills

import (
	"context"

	"github.com/google/uuid"
	segRepo "github.com/rockship/cosmo-agents-go/internal/repository/segmentation"
	"github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// ScoringSkill persists segment scores using the existing repository/API layer.
type ScoringSkill struct {
	scoreRepo *segRepo.ScoreRepository
}

func NewScoringSkill(scoreRepo *segRepo.ScoreRepository) *ScoringSkill {
	return &ScoringSkill{scoreRepo: scoreRepo}
}

// SaveScore upserts a fit score for a contact + segment.
func (s *ScoringSkill) SaveScore(ctx context.Context, contactID, segID uuid.UUID, fit FitResult) error {
	req := v1.UpsertSegmentScoreRequest{
		FitScore:       fit.FitScore,
		ScoreBreakdown: map[string]interface{}{},
		Status:         "qualified",
	}
	for k, v := range fit.ScoreBreakdown {
		req.ScoreBreakdown[k] = v
	}
	model := v1.ToSegmentScoreModel(req, contactID, segID)
	return s.scoreRepo.UpsertScore(ctx, model)
}

// MarkAsEnrolled updates the contact_segment_score to mark contact as enrolled
func (s *ScoringSkill) MarkAsEnrolled(ctx context.Context, contactID, segID uuid.UUID) error {
	return s.scoreRepo.MarkAsEnrolled(ctx, contactID, segID)
}
