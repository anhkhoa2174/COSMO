package contact

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

type validateInsightRequest struct {
	InsightType   string                 `json:"insight_type"`   // pain_point|goal|objection
	InsightText   string                 `json:"insight_text"`   // text to match
	Validation    string                 `json:"validation"`     // confirmed|rejected
	ConfirmedData map[string]interface{} `json:"confirmed_data"` // optional extra fields when confirming
}

// ValidateInsight moves/removes AI insights and records feedback.
// Route: POST /v1/contacts/:id/insights/validate
func (h *Handler) ValidateInsight(c fiber.Ctx) error {
	user, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	contactID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "invalid contact id", err)
	}

	var req validateInsightRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "invalid request body", err)
	}
	if req.InsightType == "" || req.InsightText == "" || req.Validation == "" {
		return h.responseHelper.BadRequest(c, "insight_type, insight_text, validation are required", nil)
	}

	// Access check + load contact
	contactModel, err := h.repo.GetByID(c.Context(), contactID)
	if err != nil || contactModel == nil {
		return h.responseHelper.NotFound(c, "contact not found", err)
	}
	if contactModel.UserID != user.ID {
		if contactModel.OrganizationID == nil || *contactModel.OrganizationID != orgID {
			return h.responseHelper.HandleAuthError(c, fmt.Errorf("unauthorized contact access"))
		}
	}

	// Unmarshal ai_insights and confirmed_facts
	aiMap := map[string]interface{}{}
	_ = contactModel.AIInsights.Unmarshal(&aiMap)
	confirmed := map[string]interface{}{}
	_ = contactModel.ConfirmedFacts.Unmarshal(&confirmed)

	// Remove from ai_insights list and optionally append to confirmed_facts
	updateAI := func(key string, textKey string) {
		list, _ := aiMap[key].([]interface{})
		newList := make([]interface{}, 0, len(list))
		for _, item := range list {
			if m, ok := item.(map[string]interface{}); ok {
				if val, ok := m[textKey].(string); ok && val == req.InsightText {
					// match found, skip (remove) and optionally confirm
					if req.Validation == "confirmed" {
						confirmedKey := keyToConfirmed(key)
						entry := map[string]interface{}{
							textKey:        req.InsightText,
							"confirmed_at": time.Now().UTC().Format(time.RFC3339Nano),
						}
						for k, v := range req.ConfirmedData {
							entry[k] = v
						}
						if confirmedEntry, ok := confirmed[confirmedKey].([]interface{}); ok {
							confirmedEntry = append(confirmedEntry, entry)
							confirmed[confirmedKey] = confirmedEntry
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

	switch req.InsightType {
	case "pain_point":
		updateAI("suspected_pain_points", "pain_point")
	case "goal":
		updateAI("suspected_goals", "goal")
	case "objection":
		updateAI("anticipated_objections", "objection")
	default:
		updateAI("suspected_pain_points", "pain_point")
	}

	// Persist updates
	updates := map[string]interface{}{}
	var aiJB domain.JSONB
	_ = aiJB.Marshal(aiMap)
	updates["ai_insights"] = aiJB

	var cfJB domain.JSONB
	_ = cfJB.Marshal(confirmed)
	updates["confirmed_facts"] = cfJB

	if _, err := h.repo.UpdateFields(c.Context(), contactID, updates, user.ID, orgID); err != nil {
		return h.responseHelper.InternalServerError(c, "failed to update contact insights", err)
	}

	// Record feedback
	data := map[string]interface{}{
		"insight_type":   req.InsightType,
		"insight_text":   req.InsightText,
		"validation":     req.Validation,
		"confirmed_data": req.ConfirmedData,
	}
	fb := &domain.UserFeedback{
		UserID:       user.ID,
		EntityType:   "contact",
		EntityID:     contactID,
		FeedbackType: "insight_validation",
	}
	if fbData, err := marshalJSONB(data); err == nil {
		fb.FeedbackData = fbData
	}
	fb.Applied = true
	fb.AppliedAt = ptrTime(time.Now())
	if h.feedbackRepo != nil {
		_, _ = h.feedbackRepo.Create(c.Context(), fb) // best-effort
	}

	return h.responseHelper.Success(c, fiber.Map{
		"status":         "ok",
		"insight_type":   req.InsightType,
		"validation":     req.Validation,
		"contact_id":     contactID,
		"confirmed_data": req.ConfirmedData,
	})
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

func ptrTime(t time.Time) *time.Time { return &t }
