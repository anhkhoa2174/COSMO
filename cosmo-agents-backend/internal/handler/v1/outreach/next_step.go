package outreach

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	nsdomain "github.com/rockship/cosmo-agents-go/internal/domain/nextstep"
	nextstepService "github.com/rockship/cosmo-agents-go/internal/service/nextstep"
)

// WithNextStep exposes the next-step engine's decisions over HTTP.
func (h *Handler) WithNextStep(e *nextstepService.Engine) *Handler {
	h.nextStep = e
	return h
}

// nextStepView is a contact's current next step and its recent decisions.
type nextStepView struct {
	Enabled   bool             `json:"enabled"`
	Current   *currentNextStep `json:"current"`
	Decisions []decisionView   `json:"decisions"`
	Catalogue []catalogueEntry `json:"catalogue"`
}

type currentNextStep struct {
	Action     string      `json:"action"`
	Args       interface{} `json:"args,omitempty"`
	Reason     string      `json:"reason,omitempty"`
	DueAt      interface{} `json:"due_at,omitempty"`
	DecisionID string      `json:"decision_id,omitempty"`
}

// decisionView renders a stored decision with its JSON columns as JSON (the
// JSONB type would otherwise be encoded as base64).
type decisionView struct {
	nsdomain.Decision
	Situation  json.RawMessage `json:"situation"`
	RulesFired json.RawMessage `json:"rules_fired"`
	Eligible   json.RawMessage `json:"eligible"`
	Args       json.RawMessage `json:"args"`
}

func viewOf(d nsdomain.Decision) decisionView {
	raw := func(b []byte, empty string) json.RawMessage {
		if len(b) == 0 {
			return json.RawMessage(empty)
		}
		return json.RawMessage(b)
	}
	return decisionView{
		Decision:   d,
		Situation:  raw(d.Situation, "{}"),
		RulesFired: raw(d.RulesFired, "[]"),
		Eligible:   raw(d.Eligible, "[]"),
		Args:       raw(d.Args, "{}"),
	}
}

type catalogueEntry struct {
	Action      nsdomain.Action   `json:"action"`
	Description string            `json:"description"`
	Approval    nsdomain.Approval `json:"approval"`
	Sends       bool              `json:"sends"`
}

func catalogue() []catalogueEntry {
	out := make([]catalogueEntry, 0, len(nsdomain.Catalogue))
	for _, s := range nsdomain.Catalogue {
		out = append(out, catalogueEntry{s.Action, s.Description, s.Approval, s.Sends})
	}
	return out
}

// GetNextStep returns a contact's next step and decision history.
// GET /v1/outreach/contacts/:contact_id/next-step?limit=10
func (h *Handler) GetNextStep(c fiber.Ctx) error {
	if h.nextStep == nil {
		return h.responseHelper.NotFound(c, "Next-step engine is not configured", nil)
	}
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	contactID, err := uuid.Parse(c.Params("contact_id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	contact, decisions, err := h.nextStep.History(c.Context(), userID, contactID, limit)
	if errors.Is(err, nextstepService.ErrNotFound) {
		return h.responseHelper.NotFound(c, "Contact not found", nil)
	}
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to load next step", err)
	}

	views := make([]decisionView, 0, len(decisions))
	for _, d := range decisions {
		views = append(views, viewOf(d))
	}
	view := nextStepView{
		Enabled:   h.nextStep.Enabled(c.Context(), userID),
		Decisions: views,
		Catalogue: catalogue(),
	}
	if contact.NextAction != nil {
		cur := &currentNextStep{Action: *contact.NextAction}
		if len(contact.NextActionArgs) > 0 {
			cur.Args = json.RawMessage(contact.NextActionArgs)
		}
		if contact.NextActionReason != nil {
			cur.Reason = *contact.NextActionReason
		}
		if contact.NextActionDueAt != nil {
			cur.DueAt = contact.NextActionDueAt
		}
		if contact.NextActionDecisionID != nil {
			cur.DecisionID = contact.NextActionDecisionID.String()
		}
		view.Current = cur
	}
	return h.responseHelper.Success(c, view)
}

// DecideNextStep runs the engine for one contact now.
// POST /v1/outreach/contacts/:contact_id/next-step/decide
func (h *Handler) DecideNextStep(c fiber.Ctx) error {
	if h.nextStep == nil {
		return h.responseHelper.NotFound(c, "Next-step engine is not configured", nil)
	}
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	contactID, err := uuid.Parse(c.Params("contact_id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}
	// ?dry_run=true decides without storing or applying the result.
	d, err := h.nextStep.Decide(c.Context(), nextstepService.Input{
		UserID: userID, ContactID: contactID, Trigger: nsdomain.TriggerManual,
		DryRun: c.Query("dry_run") == "true",
	})
	if errors.Is(err, nextstepService.ErrNotFound) {
		return h.responseHelper.NotFound(c, "Contact not found", nil)
	}
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to decide the next step", err)
	}
	return h.responseHelper.Success(c, viewOf(*d))
}

// ReviewNextStep records a person's verdict on a decision.
// POST /v1/outreach/next-step/decisions/:decision_id/review  {"approve": true}
func (h *Handler) ReviewNextStep(c fiber.Ctx) error {
	if h.nextStep == nil {
		return h.responseHelper.NotFound(c, "Next-step engine is not configured", nil)
	}
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}
	decisionID, err := uuid.Parse(c.Params("decision_id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid decision ID", err)
	}
	var body struct {
		Approve *bool `json:"approve"`
	}
	if err := c.Bind().JSON(&body); err != nil || body.Approve == nil {
		return h.responseHelper.BadRequest(c, `Body must be {"approve": true|false}`, err)
	}
	d, err := h.nextStep.Review(c.Context(), userID, decisionID, *body.Approve)
	if errors.Is(err, nextstepService.ErrNotFound) {
		return h.responseHelper.NotFound(c, "Decision not found", nil)
	}
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to record the review", err)
	}
	return h.responseHelper.Success(c, viewOf(*d))
}
