package conversation

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// Correcting an intent is not a cosmetic relabel: the classifier's label
// decided which handler ran, and one of those handlers suppressed the contact.
// So the correction has to undo what the wrong label did, not merely overwrite
// the word shown on the badge.
//
// Two things are deliberately *not* done here:
//
//   - The AI draft is not regenerated or deleted. It may already carry the
//     representative's edits, and throwing that away to chase a label they just
//     fixed is the wrong default. The response reports `draft_stale` instead and
//     lets the caller decide.
//   - The classifier's quoted evidence is not kept as the new label's evidence.
//     Those sentences argued for the old intent; re-colouring them under the new
//     one would present the machine's reasoning as support for a conclusion it
//     never reached. They move to `original_reasoning` for audit, and the
//     highlight disappears.

// canonicalIntents maps every spelling the front end might send — the display
// string the backend stores ("Do not contact") or the SCREAMING_SNAKE key the
// UI works in ("DO_NOT_CONTACT") — onto the stored form.
var canonicalIntents = map[string]domain.IntentType{
	"INTERESTED":              domain.IntentInterested,
	"NOT_INTERESTED":          domain.IntentNotInterested,
	"REFERRAL":                domain.IntentReferral,
	"REQUEST_FOR_PRICING":     domain.IntentRequestForPricing,
	"REQUEST_FOR_INFORMATION": domain.IntentRequestForInfo,
	"NURTURE":                 domain.IntentNurture,
	"DO_NOT_CONTACT":          domain.IntentDoNotContact,
	"OUT_OF_OFFICE":           domain.IntentOutOfOffice,
	"UNKNOWN_INTENT":          domain.IntentUnknown,
	"OTHER":                   domain.IntentUnknown,
}

// normalizeIntent folds "Do not contact", "do_not_contact" and
// "DO NOT CONTACT" onto the same key.
func normalizeIntent(raw string) (domain.IntentType, bool) {
	key := strings.ToUpper(strings.TrimSpace(raw))
	key = strings.NewReplacer(" ", "_", "-", "_").Replace(key)
	v, ok := canonicalIntents[key]
	return v, ok
}

type correctIntentRequest struct {
	// Intent is the label the human says is correct.
	Intent string `json:"intent"`
	// Note is optional free text: why the classifier was wrong. It is stored
	// with the feedback row, which is what makes the corrections usable later
	// as calibration data rather than just an audit trail.
	Note string `json:"note,omitempty"`
}

// CorrectIntent overrides a classified reply intent with a human decision.
// Route: POST /v1/conversations/:id/intent
//
// @Summary Correct the classified intent of a conversation
// @Tags Conversations
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 200 {object} schema.APIResponse[any]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 403 {object} schema.APIResponse[any]
// @Failure 404 {object} schema.APIResponse[any]
// @Router /v1/conversations/{id}/intent [post]
func (h *ConversationHandler) CorrectIntent(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	conversationID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid conversation ID format", err.Error(),
		))
	}

	var req correctIntentRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	newIntent, valid := normalizeIntent(req.Intent)
	if !valid {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Unknown intent. Expected one of: Interested, Not interested, Referral, "+
				"Request for pricing, Request for information, Nurture, Do not contact, "+
				"Out of office, Unknown intent", req.Intent,
		))
	}

	conversation, err := h.conversationRepo.FindByID(c.Context(), conversationID)
	if err != nil || conversation == nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Conversation not found", "",
		))
	}
	if conversation.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "You don't have permission to edit this conversation", "",
		))
	}

	meta := map[string]interface{}{}
	if err := conversation.CMetadata.Unmarshal(&meta); err != nil || meta == nil {
		meta = map[string]interface{}{}
	}
	detail, _ := meta["intent_detail"].(map[string]interface{})
	if detail == nil {
		detail = map[string]interface{}{}
	}

	oldIntent := ""
	if len(conversation.Intents) > 0 {
		oldIntent = conversation.Intents[0]
	} else if s, ok := detail["intent"].(string); ok {
		oldIntent = s
	}

	if oldIntent == string(newIntent) {
		return c.JSON(schema.SuccessResponse(fiber.Map{
			"conversation_id": conversationID,
			"intent":          string(newIntent),
			"changed":         false,
			"draft_stale":     false,
		}))
	}

	// The evidence belonged to the old label; retire it rather than re-point it.
	if reasoning, ok := detail["reasoning"]; ok {
		if _, already := detail["original_reasoning"]; !already {
			detail["original_reasoning"] = reasoning
		}
		delete(detail, "reasoning")
	}
	if _, already := detail["original_intent"]; !already && oldIntent != "" {
		detail["original_intent"] = oldIntent
	}
	// The machine's confidence describes a label no longer in force.
	if conf, ok := detail["confidence"]; ok {
		if _, already := detail["original_confidence"]; !already {
			detail["original_confidence"] = conf
		}
		delete(detail, "confidence")
	}
	detail["intent"] = string(newIntent)
	detail["corrected_by_user"] = true
	detail["corrected_by"] = userID.String()
	detail["corrected_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	if req.Note != "" {
		detail["correction_note"] = req.Note
	}
	meta["intent_detail"] = detail

	conversation.Intents = []string{string(newIntent)}
	if err := conversation.CMetadata.Marshal(meta); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to encode conversation metadata", err.Error(),
		))
	}
	if err := h.conversationRepo.Update(c.Context(), conversation.ID, conversation); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to update conversation intent", err.Error(),
		))
	}

	// The badge in the thread reads from the email row, so leaving it behind
	// would show the old label next to the corrected one.
	sourceEmail := h.findClassifiedEmail(c, conversationID, detail)
	if sourceEmail != nil {
		sourceEmail.Intents = []string{string(newIntent)}
		if err := h.emailRepo.Update(c.Context(), sourceEmail.ID, sourceEmail); err != nil {
			// The conversation is already corrected; a stale email row is worth
			// reporting through the feedback record, not worth failing over.
			sourceEmail = nil
		}
	}

	// Undo the one handler side effect a human cannot otherwise reverse.
	suppressionChanged := h.reconcileSuppression(c, userID, conversation, sourceEmail, oldIntent, newIntent)

	// Record the correction: an audit trail, and the only signal that says the
	// classifier was wrong on this reply.
	if h.feedbackRepo != nil {
		data := map[string]interface{}{
			"from_intent":         oldIntent,
			"to_intent":           string(newIntent),
			"note":                req.Note,
			"suppression_changed": suppressionChanged,
		}
		if sourceEmail != nil {
			data["email_id"] = sourceEmail.ID.String()
		}
		fb := &domain.UserFeedback{
			UserID:       userID,
			EntityType:   "conversation",
			EntityID:     conversationID,
			FeedbackType: "intent_correction",
			Applied:      true,
			AppliedAt:    ptrNow(),
		}
		var payload domain.JSONB
		if err := payload.Marshal(data); err == nil {
			fb.FeedbackData = payload
		}
		_, _ = h.feedbackRepo.Create(c.Context(), fb) // best-effort
	}

	// A draft written for the old intent is probably wrong for the new one, but
	// it may also carry the representative's edits — so say so and let them choose.
	_, hasDraft := meta["ai_reply"]

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"conversation_id":     conversationID,
		"intent":              string(newIntent),
		"previous_intent":     oldIntent,
		"changed":             true,
		"draft_stale":         hasDraft,
		"suppression_changed": suppressionChanged,
	}))
}

// findClassifiedEmail returns the email row whose label the correction has to
// follow. Normally the classifier recorded its id in intent_detail; for
// conversations labelled before that field existed, the reply it looked at is
// recoverable because it is the only email in the thread carrying an intent.
func (h *ConversationHandler) findClassifiedEmail(
	c fiber.Ctx,
	conversationID uuid.UUID,
	detail map[string]interface{},
) *domain.Email {
	if idStr, ok := detail["email_id"].(string); ok && idStr != "" {
		if emailID, err := uuid.Parse(idStr); err == nil {
			if e, err := h.emailRepo.FindByID(c.Context(), emailID); err == nil {
				return e
			}
		}
	}

	emails, err := h.emailRepo.FindByConversationID(c.Context(), conversationID)
	if err != nil {
		return nil
	}
	var labelled, newest *domain.Email
	for _, e := range emails {
		if newest == nil || e.CreatedAt.After(newest.CreatedAt) {
			newest = e
		}
		if len(e.Intents) == 0 {
			continue
		}
		if labelled == nil || e.CreatedAt.After(labelled.CreatedAt) {
			labelled = e
		}
	}
	if labelled != nil {
		return labelled
	}
	// Older threads exist where the label was written to the conversation but
	// never to any message. Returning nothing there would quietly skip the
	// suppression reconciliation, which is the one consequence a person cannot
	// undo from the interface — so the newest message stands in, purely as a
	// way to reach the contact behind the thread.
	return newest
}

// reconcileSuppression applies or lifts do_not_contact to match the corrected
// label. Every other handler effect is either idempotent or reversible from the
// UI; suppression is neither, which is why it is singled out.
//
// Returns "suppressed", "unsuppressed" or "" so the caller can report what
// actually happened rather than what was intended.
func (h *ConversationHandler) reconcileSuppression(
	c fiber.Ctx,
	userID uuid.UUID,
	conversation *domain.Conversation,
	sourceEmail *domain.Email,
	oldIntent string,
	newIntent domain.IntentType,
) string {
	wasDNC := oldIntent == string(domain.IntentDoNotContact)
	isDNC := newIntent == domain.IntentDoNotContact
	if wasDNC == isDNC || h.contactRepo == nil {
		return ""
	}

	// The contact is found the same way the handler found it: by the address
	// that sent the reply.
	if sourceEmail == nil {
		return ""
	}
	// Which of the two addresses belongs to the prospect depends on whether the
	// message was inbound or outbound, and this handler has no way to know the
	// agent's own address. Trying both is cheaper than threading that through,
	// and the agent's address matches no contact anyway.
	var contact *domain.Contact
	for _, addr := range []string{sourceEmail.FromEmail, sourceEmail.ToEmail} {
		if addr == "" {
			continue
		}
		if found, err := h.contactRepo.FindByEmail(c.Context(), userID, addr); err == nil && found != nil {
			contact = found
			break
		}
	}
	if contact == nil {
		return ""
	}

	orgID := uuid.Nil
	if contact.OrganizationID != nil {
		orgID = *contact.OrganizationID
	}
	if _, err := h.contactRepo.UpdateFields(c.Context(), contact.ID,
		map[string]interface{}{"do_not_contact": isDNC}, userID, orgID); err != nil {
		return ""
	}
	if isDNC {
		return "suppressed"
	}
	return "unsuppressed"
}

func ptrNow() *time.Time {
	t := time.Now()
	return &t
}
