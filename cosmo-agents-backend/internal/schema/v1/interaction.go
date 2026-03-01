package v1

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

type CreateInteractionRequest struct {
	ContactID       uuid.UUID              `json:"contact_id" validate:"required"`
	CampaignID      *uuid.UUID             `json:"campaign_id,omitempty"`
	SegmentationID  *uuid.UUID             `json:"segmentation_id,omitempty"`
	InteractionType string                 `json:"interaction_type" validate:"required"`
	Channel         string                 `json:"channel"`
	Direction       string                 `json:"direction"`
	Content         map[string]interface{} `json:"content"`
	AIAnalysis      map[string]interface{} `json:"ai_analysis"`
	OccurredAt      *time.Time             `json:"occurred_at"`
}

type InteractionResponse struct {
	ID              uuid.UUID              `json:"id"`
	ContactID       uuid.UUID              `json:"contact_id"`
	CampaignID      *uuid.UUID             `json:"campaign_id,omitempty"`
	SegmentationID  *uuid.UUID             `json:"segmentation_id,omitempty"`
	InteractionType string                 `json:"interaction_type"`
	Channel         string                 `json:"channel"`
	Direction       string                 `json:"direction"`
	Content         map[string]interface{} `json:"content,omitempty"`
	AIAnalysis      map[string]interface{} `json:"ai_analysis,omitempty"`
	OccurredAt      string                 `json:"occurred_at"`
	CreatedAt       string                 `json:"created_at"`
	UpdatedAt       string                 `json:"updated_at"`
}

func ToInteractionModel(req CreateInteractionRequest) *domain.Interaction {
	i := &domain.Interaction{
		ContactID:       req.ContactID,
		CampaignID:      req.CampaignID,
		SegmentationID:  req.SegmentationID,
		InteractionType: req.InteractionType,
		Channel:         req.Channel,
		Direction:       req.Direction,
		Content:         marshalMapToJSONB(req.Content),
		AIAnalysis:      marshalMapToJSONB(req.AIAnalysis),
		OccurredAt:      time.Now(),
	}
	if req.OccurredAt != nil {
		i.OccurredAt = *req.OccurredAt
	}
	return i
}

func ToInteractionResponse(i *domain.Interaction) *InteractionResponse {
	return &InteractionResponse{
		ID:              i.ID,
		ContactID:       i.ContactID,
		CampaignID:      i.CampaignID,
		SegmentationID:  i.SegmentationID,
		InteractionType: i.InteractionType,
		Channel:         i.Channel,
		Direction:       i.Direction,
		Content:         unmarshalJSONB(i.Content),
		AIAnalysis:      unmarshalJSONB(i.AIAnalysis),
		OccurredAt:      i.OccurredAt.Format(time.RFC3339Nano),
		CreatedAt:       i.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:       i.UpdatedAt.Format(time.RFC3339Nano),
	}
}
