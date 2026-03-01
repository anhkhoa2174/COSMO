package agents

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactrepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	fbRepo "github.com/rockship/cosmo-agents-go/internal/repository/feedback"
	"github.com/rockship/cosmo-agents-go/internal/skills"
)

// InsightValidatorAgent confirms or rejects AI insights and logs feedback.
type InsightValidatorAgent struct {
	contactRepo   *contactrepo.ContactRepository
	feedbackSkill *skills.FeedbackCaptureSkill
}

func NewInsightValidatorAgent(contactRepo *contactrepo.ContactRepository, feedbackRepo *fbRepo.Repository) *InsightValidatorAgent {
	return &InsightValidatorAgent{
		contactRepo:   contactRepo,
		feedbackSkill: skills.NewFeedbackCaptureSkill(feedbackRepo),
	}
}

type InsightValidationResult struct {
	ContactID     uuid.UUID              `json:"contact_id"`
	InsightType   string                 `json:"insight_type"`
	Validation    string                 `json:"validation"`
	ConfirmedData map[string]interface{} `json:"confirmed_data,omitempty"`
	UpdatedAI     map[string]interface{} `json:"ai_insights,omitempty"`
	Confirmed     map[string]interface{} `json:"confirmed_facts,omitempty"`
}

func (a *InsightValidatorAgent) Run(
	ctx context.Context,
	userID, orgID, contactID uuid.UUID,
	insightType, insightText, validation string,
	confirmedData map[string]interface{},
) (*InsightValidationResult, error) {
	contactModel, err := a.contactRepo.GetByID(ctx, contactID)
	if err != nil {
		return nil, err
	}
	if contactModel == nil {
		return nil, ErrNotFound
	}
	if contactModel.UserID != userID {
		if contactModel.OrganizationID == nil || *contactModel.OrganizationID != orgID {
			return nil, ErrUnauthorized
		}
	}

	aiMap := map[string]interface{}{}
	_ = contactModel.AIInsights.Unmarshal(&aiMap)
	confirmed := map[string]interface{}{}
	_ = contactModel.ConfirmedFacts.Unmarshal(&confirmed)

	updateAI := func(key string, textKey string) {
		list, _ := aiMap[key].([]interface{})
		newList := make([]interface{}, 0, len(list))
		for _, item := range list {
			if m, ok := item.(map[string]interface{}); ok {
				if val, ok := m[textKey].(string); ok && val == insightText {
					if validation == "confirmed" {
						confirmedKey := keyToConfirmed(key)
						entry := map[string]interface{}{
							textKey:        insightText,
							"confirmed_at": time.Now().UTC().Format(time.RFC3339Nano),
						}
						for k, v := range confirmedData {
							entry[k] = v
						}
						if existing, ok := confirmed[confirmedKey].([]interface{}); ok {
							confirmed[confirmedKey] = append(existing, entry)
						} else {
							confirmed[confirmedKey] = []interface{}{entry}
						}
					}
					continue
				}
			}
			newList = append(newList, item)
		}
		aiMap[key] = newList
	}

	switch insightType {
	case "pain_point":
		updateAI("suspected_pain_points", "pain_point")
	case "goal":
		updateAI("suspected_goals", "goal")
	case "objection":
		updateAI("anticipated_objections", "objection")
	default:
		updateAI("suspected_pain_points", "pain_point")
	}

	var aiJB domain.JSONB
	_ = aiJB.Marshal(aiMap)
	var cfJB domain.JSONB
	_ = cfJB.Marshal(confirmed)

	if _, err := a.contactRepo.UpdateFields(ctx, contactID, map[string]interface{}{
		"ai_insights":     aiJB,
		"confirmed_facts": cfJB,
	}, userID, orgID); err != nil {
		return nil, err
	}

	if a.feedbackSkill != nil {
		_, _ = a.feedbackSkill.CaptureInsightValidation(ctx, userID, contactID, insightType, insightText, validation, confirmedData)
	}

	return &InsightValidationResult{
		ContactID:     contactID,
		InsightType:   insightType,
		Validation:    validation,
		ConfirmedData: confirmedData,
		UpdatedAI:     aiMap,
		Confirmed:     confirmed,
	}, nil
}

func keyToConfirmed(aiKey string) string {
	switch aiKey {
	case "suspected_pain_points":
		return "pain_points"
	case "suspected_goals":
		return "goals"
	case "anticipated_objections":
		return "objections"
	default:
		return aiKey
	}
}
