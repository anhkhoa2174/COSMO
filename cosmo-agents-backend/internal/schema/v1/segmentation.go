package v1

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

type CreateSegmentationRequest struct {
	Name          string                 `json:"name" validate:"required"`
	Description   string                 `json:"description"`
	Priority      int                    `json:"priority" validate:"min=1,max=10"`
	Criteria      map[string]interface{} `json:"criteria"`
	ICPDefinition map[string]interface{} `json:"icp_definition"`
	IsActive      *bool                  `json:"is_active"`
}

type SegmentationResponse struct {
	ID            uuid.UUID              `json:"id"`
	UserID        uuid.UUID              `json:"user_id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description,omitempty"`
	Priority      int                    `json:"priority"`
	Criteria      map[string]interface{} `json:"criteria,omitempty"`
	ICPDefinition map[string]interface{} `json:"icp_definition,omitempty"`
	IsActive      bool                   `json:"is_active"`
	CreatedAt     string                 `json:"created_at"`
	UpdatedAt     string                 `json:"updated_at"`
}

type UpsertSegmentScoreRequest struct {
	FitScore           int                    `json:"fit_score" validate:"min=0,max=100"`
	ScoreBreakdown     map[string]interface{} `json:"score_breakdown"`
	ScoreType          string                 `json:"score_type"`
	Status             string                 `json:"status"`
	PassesFilters      *bool                  `json:"passes_filters"`
	EnrolledInCampaign *bool                  `json:"enrolled_in_campaign"`
	CurrentCampaignID  *uuid.UUID             `json:"current_campaign_id"`
}

type SegmentScoreResponse struct {
	ID                 uuid.UUID              `json:"id"`
	ContactID          uuid.UUID              `json:"contact_id"`
	SegmentationID     uuid.UUID              `json:"segmentation_id"`
	FitScore           int                    `json:"fit_score"`
	ScoreBreakdown     map[string]interface{} `json:"score_breakdown,omitempty"`
	ScoreType          string                 `json:"score_type"`
	Status             string                 `json:"status"`
	PassesFilters      bool                   `json:"passes_filters"`
	EnrolledInCampaign bool                   `json:"enrolled_in_campaign"`
	CurrentCampaignID  *uuid.UUID             `json:"current_campaign_id,omitempty"`
	CreatedAt          string                 `json:"created_at"`
	UpdatedAt          string                 `json:"updated_at"`
}

func ToSegmentationModel(req CreateSegmentationRequest, userID uuid.UUID) *domain.Segmentation {
	seg := &domain.Segmentation{
		UserID:        userID,
		Name:          req.Name,
		Description:   req.Description,
		Priority:      req.Priority,
		Criteria:      marshalMapToJSONB(req.Criteria),
		ICPDefinition: marshalMapToJSONB(req.ICPDefinition),
		IsActive:      true,
	}
	if req.IsActive != nil {
		seg.IsActive = *req.IsActive
	}
	return seg
}

func ToSegmentationResponse(seg *domain.Segmentation) *SegmentationResponse {
	return &SegmentationResponse{
		ID:            seg.ID,
		UserID:        seg.UserID,
		Name:          seg.Name,
		Description:   seg.Description,
		Priority:      seg.Priority,
		Criteria:      unmarshalJSONB(seg.Criteria),
		ICPDefinition: unmarshalJSONB(seg.ICPDefinition),
		IsActive:      seg.IsActive,
		CreatedAt:     seg.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:     seg.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func ToSegmentScoreModel(req UpsertSegmentScoreRequest, contactID, segID uuid.UUID) *domain.SegmentationScore {
	score := &domain.SegmentationScore{
		ContactID:          contactID,
		SegmentationID:     segID,
		FitScore:           req.FitScore,
		ScoreBreakdown:     marshalMapToJSONB(req.ScoreBreakdown),
		ScoreType:          req.ScoreType,
		Status:             req.Status,
		PassesFilters:      true,
		EnrolledInCampaign: false,
		CurrentCampaignID:  req.CurrentCampaignID,
	}
	if req.PassesFilters != nil {
		score.PassesFilters = *req.PassesFilters
	}
	if req.EnrolledInCampaign != nil {
		score.EnrolledInCampaign = *req.EnrolledInCampaign
	}
	if score.ScoreType == "" {
		score.ScoreType = "auto"
	}
	if score.Status == "" {
		score.Status = "qualified"
	}
	return score
}

func ToSegmentScoreResponse(score *domain.SegmentationScore) *SegmentScoreResponse {
	return &SegmentScoreResponse{
		ID:                 score.ID,
		ContactID:          score.ContactID,
		SegmentationID:     score.SegmentationID,
		FitScore:           score.FitScore,
		ScoreBreakdown:     unmarshalJSONB(score.ScoreBreakdown),
		ScoreType:          score.ScoreType,
		Status:             score.Status,
		PassesFilters:      score.PassesFilters,
		EnrolledInCampaign: score.EnrolledInCampaign,
		CurrentCampaignID:  score.CurrentCampaignID,
		CreatedAt:          score.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:          score.UpdatedAt.Format(time.RFC3339Nano),
	}
}

// marshalMapToJSONB converts a map into domain.JSONB with safe defaults.
func marshalMapToJSONB(data map[string]interface{}) domain.JSONB {
	var jb domain.JSONB
	if len(data) == 0 {
		_ = jb.Marshal(map[string]interface{}{})
		return jb
	}
	_ = jb.Marshal(data)
	return jb
}

// unmarshalJSONB converts domain.JSONB into map[string]interface{} with empty map fallback.
func unmarshalJSONB(jb domain.JSONB) map[string]interface{} {
	result := make(map[string]interface{})
	if len(jb) == 0 {
		return result
	}
	_ = jb.Unmarshal(&result)
	return result
}
