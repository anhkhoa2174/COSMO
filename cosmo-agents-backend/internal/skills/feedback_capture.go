package skills

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	fbRepo "github.com/rockship/cosmo-agents-go/internal/repository/feedback"
)

// FeedbackCaptureSkill stores user feedback records (score adjustments, insight validation, custom facts).
// It does not mutate contacts directly; handlers can apply changes via contact PATCH if desired.
type FeedbackCaptureSkill struct {
	repo *fbRepo.Repository
}

func NewFeedbackCaptureSkill(repo *fbRepo.Repository) *FeedbackCaptureSkill {
	return &FeedbackCaptureSkill{repo: repo}
}

func (s *FeedbackCaptureSkill) CaptureScoreAdjustment(ctx context.Context, userID, contactID uuid.UUID, fieldPath string, original, adjusted int, reason string) (*domain.UserFeedback, error) {
	data := map[string]any{
		"field_path":     fieldPath,
		"original_score": original,
		"adjusted_score": adjusted,
		"reason":         reason,
	}
	fb := &domain.UserFeedback{
		UserID:       userID,
		EntityType:   "contact",
		EntityID:     contactID,
		FeedbackType: "score_adjustment",
		FeedbackData: marshalJSONB(data),
		Applied:      true,
		AppliedAt:    ptr(time.Now()),
	}
	if _, err := s.repo.Create(ctx, fb); err != nil {
		return nil, err
	}
	return fb, nil
}

func (s *FeedbackCaptureSkill) CaptureInsightValidation(ctx context.Context, userID, contactID uuid.UUID, insightType, insightText, validation string, confirmedData map[string]any) (*domain.UserFeedback, error) {
	data := map[string]any{
		"insight_type":   insightType,
		"insight_text":   insightText,
		"validation":     validation,
		"confirmed_data": confirmedData,
	}
	fb := &domain.UserFeedback{
		UserID:       userID,
		EntityType:   "contact",
		EntityID:     contactID,
		FeedbackType: "insight_validation",
		FeedbackData: marshalJSONB(data),
		Applied:      true,
		AppliedAt:    ptr(time.Now()),
	}
	if _, err := s.repo.Create(ctx, fb); err != nil {
		return nil, err
	}
	return fb, nil
}

func (s *FeedbackCaptureSkill) CaptureCustomFact(ctx context.Context, userID, contactID uuid.UUID, factType string, factData map[string]any) (*domain.UserFeedback, error) {
	data := map[string]any{
		"fact_type": factType,
		"fact_data": factData,
	}
	fb := &domain.UserFeedback{
		UserID:       userID,
		EntityType:   "contact",
		EntityID:     contactID,
		FeedbackType: "custom_fact",
		FeedbackData: marshalJSONB(data),
		Applied:      true,
		AppliedAt:    ptr(time.Now()),
	}
	if _, err := s.repo.Create(ctx, fb); err != nil {
		return nil, err
	}
	return fb, nil
}

func marshalJSONB(m map[string]any) domain.JSONB {
	var jb domain.JSONB
	_ = jb.Marshal(m)
	return jb
}

func ptr(t time.Time) *time.Time { return &t }
