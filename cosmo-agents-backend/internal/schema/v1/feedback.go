package v1

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

type CreateScoreFeedbackRequest struct {
	ContactID     uuid.UUID `json:"contact_id" validate:"required"`
	FieldPath     string    `json:"field_path" validate:"required"`
	OriginalScore int       `json:"original_score"`
	AdjustedScore int       `json:"adjusted_score" validate:"required"`
	Reason        string    `json:"reason"`
}

type CreateInsightFeedbackRequest struct {
	ContactID     uuid.UUID              `json:"contact_id" validate:"required"`
	InsightType   string                 `json:"insight_type" validate:"required"` // pain_point/goal/objection
	InsightText   string                 `json:"insight_text" validate:"required"`
	Validation    string                 `json:"validation" validate:"required"` // confirmed/rejected
	ConfirmedData map[string]interface{} `json:"confirmed_data"`                 // optional details
}

type CreateCustomFactRequest struct {
	ContactID uuid.UUID              `json:"contact_id" validate:"required"`
	FactType  string                 `json:"fact_type" validate:"required"`
	FactData  map[string]interface{} `json:"fact_data" validate:"required"`
}

type FeedbackResponse struct {
	ID           uuid.UUID              `json:"id"`
	UserID       uuid.UUID              `json:"user_id"`
	EntityType   string                 `json:"entity_type"`
	EntityID     uuid.UUID              `json:"entity_id"`
	FeedbackType string                 `json:"feedback_type"`
	FeedbackData map[string]interface{} `json:"feedback_data"`
	Applied      bool                   `json:"applied"`
	AppliedAt    *time.Time             `json:"applied_at,omitempty"`
	CreatedAt    string                 `json:"created_at"`
}

func ToFeedbackResponse(f *domain.UserFeedback) *FeedbackResponse {
	data := map[string]interface{}{}
	if len(f.FeedbackData) > 0 {
		_ = f.FeedbackData.Unmarshal(&data)
	}
	return &FeedbackResponse{
		ID:           f.ID,
		UserID:       f.UserID,
		EntityType:   f.EntityType,
		EntityID:     f.EntityID,
		FeedbackType: f.FeedbackType,
		FeedbackData: data,
		Applied:      f.Applied,
		AppliedAt:    f.AppliedAt,
		CreatedAt:    f.CreatedAt.Format(time.RFC3339Nano),
	}
}
