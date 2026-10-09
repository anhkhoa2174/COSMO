// Package nextstep is the next-step decision engine of report Section 6.7.
//
// One pass runs five stages: observe (situation.go), filter (rules.go),
// select (llm.go, with the fixed cadence as fallback), approve (by the
// approval tier of the chosen action) and act (this file). Code holds the
// facts and the rules; the model chooses only among what the rules allow;
// a person approves whatever leaves the system.
package nextstep

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openai/openai-go"
	"github.com/rs/zerolog"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	ns "github.com/rockship/cosmo-agents-go/internal/domain/nextstep"
	outreachdomain "github.com/rockship/cosmo-agents-go/internal/domain/outreach"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	outreachRepo "github.com/rockship/cosmo-agents-go/internal/repository/outreach"
	outreachService "github.com/rockship/cosmo-agents-go/internal/service/outreach"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// ErrNotFound is returned when the contact or decision does not belong to the caller.
var ErrNotFound = errors.New("not found")

// Input is what starts a decision.
type Input struct {
	UserID    uuid.UUID
	ContactID uuid.UUID
	Trigger   ns.Trigger

	// Set when the trigger is an inbound reply.
	ConversationID *uuid.UUID
	EmailID        *uuid.UUID
	Intent         string
	ReplyText      string

	// DryRun takes the decision without storing or applying it, so the same
	// situation can be decided repeatedly to measure consistency.
	DryRun bool
}

// Engine takes next-step decisions.
type Engine struct {
	db           *gorm.DB
	contacts     *contactRepo.ContactRepository
	interactions *outreachRepo.InteractionLogRepository
	meetings     *outreachRepo.MeetingRepository
	outreach     *outreachService.Service
	client       *openai.Client
	model        string
	settings     func(ctx context.Context, userID uuid.UUID) ([]byte, error)
	logger       *zerolog.Logger
	now          func() time.Time
}

// New builds an engine. `settings` reads the organisation's outreach settings,
// where the on/off switch lives; the outreach service must carry the same
// loader so the cadence it computes is the organisation's.
func New(
	db *gorm.DB,
	contacts *contactRepo.ContactRepository,
	interactions *outreachRepo.InteractionLogRepository,
	meetings *outreachRepo.MeetingRepository,
	outreach *outreachService.Service,
	client *openai.Client,
	model string,
	settings func(ctx context.Context, userID uuid.UUID) ([]byte, error),
	log *zerolog.Logger,
) *Engine {
	return &Engine{
		db: db, contacts: contacts, interactions: interactions, meetings: meetings,
		outreach: outreach, client: client, model: model, settings: settings,
		logger: log, now: time.Now,
	}
}

func (e *Engine) log() *zerolog.Logger {
	if e.logger != nil {
		return e.logger
	}
	return &logger.Logger
}

// Enabled reports whether the user's organisation has turned the engine on.
func (e *Engine) Enabled(ctx context.Context, userID uuid.UUID) bool {
	if e == nil || e.settings == nil {
		return false
	}
	raw, err := e.settings(ctx, userID)
	if err != nil {
		return false
	}
	return outreachService.NextStepEngineFrom(raw)
}

// Decide runs one pass for one contact and applies the result.
func (e *Engine) Decide(ctx context.Context, in Input) (*ns.Decision, error) {
	c, err := e.contacts.FindByIDAndUserID(ctx, in.UserID, in.ContactID)
	if err != nil || c == nil {
		return nil, ErrNotFound
	}

	// 1. Observe.
	s, err := e.buildSituation(ctx, in, c)
	if err != nil {
		return nil, err
	}

	// 2. Filter.
	f := applyRules(s)
	if f.Hits == nil {
		f.Hits = []ns.RuleHit{}
	}

	// 3. Select.
	ch, selectedBy, cause := e.selectAction(ctx, s, f)

	// 4. Approve: the tier comes from the catalogue, never from the model.
	spec, _ := ns.SpecOf(ch.Action)
	status := ns.StatusPendingReview
	if spec.Approval == ns.ApprovalAutomatic {
		status = ns.StatusApplied
	}

	d := &ns.Decision{
		UserID:        in.UserID,
		ContactID:     c.ID,
		Trigger:       string(in.Trigger),
		Situation:     mustJSON(s),
		RulesFired:    mustJSON(f.Hits),
		Eligible:      mustJSON(f.Eligible),
		Action:        string(ch.Action),
		Args:          mustJSON(ch.Args),
		Reason:        ch.Reason,
		SelectedBy:    string(selectedBy),
		FallbackCause: cause,
		Approval:      string(spec.Approval),
		Status:        status,
		Model:         e.model,
		PromptVersion: PromptVersion,
	}
	if selectedBy == ns.SelectedByRule {
		d.Model = ""
	}

	if in.DryRun {
		d.CreatedAt = e.now()
		return d, nil
	}

	// 5. Act and record.
	if err := e.db.WithContext(ctx).Create(d).Error; err != nil {
		return nil, fmt.Errorf("store decision: %w", err)
	}
	if err := e.apply(ctx, in, c, s, d, ch); err != nil {
		e.log().Error().Err(err).Str("contact_id", c.ID.String()).Msg("next-step: decision stored but not applied")
		return d, err
	}

	e.log().Info().
		Str("contact_id", c.ID.String()).
		Str("trigger", d.Trigger).
		Str("action", d.Action).
		Str("selected_by", d.SelectedBy).
		Int("eligible", len(f.Eligible)).
		Str("reason", d.Reason).
		Msg("next-step: decision taken")
	return d, nil
}

// selectAction is stage 3.
func (e *Engine) selectAction(ctx context.Context, s *Situation, f filterResult) (*choice, ns.SelectedBy, string) {
	if len(f.Eligible) == 1 {
		a := f.Eligible[0]
		ch := &choice{Action: a, Args: f.Fixed[a], Reason: f.RuleReason}
		if ch.Reason == "" {
			ch.Reason = "Only one action is allowed in this situation"
		}
		return e.complete(s, f, ch), ns.SelectedByRule, ""
	}

	ch, err := e.choose(ctx, s, f.Eligible)
	if err == nil {
		// A rule-fixed argument (the end of a wait) overrides the model's.
		for k, v := range f.Fixed[ch.Action] {
			ch.Args[k] = v
		}
		return e.complete(s, f, ch), ns.SelectedByModel, ""
	}
	e.log().Warn().Err(err).Msg("next-step: model choice unusable; falling back to the cadence")
	return e.complete(s, f, e.fallback(s, f)), ns.SelectedByFallback, err.Error()
}

// fallback maps the fixed cadence's step onto the catalogue. If the rules
// removed that action, the first eligible action in catalogue order is used,
// so the engine is never worse than the sequence it replaces.
func (e *Engine) fallback(s *Situation, f filterResult) *choice {
	want := map[string]ns.Action{
		string(outreachdomain.NextStepSend):             ns.SendIntro,
		string(outreachdomain.NextStepFollowUp1):        ns.SendFollowUp,
		string(outreachdomain.NextStepFollowUp2):        ns.SendFollowUp,
		string(outreachdomain.NextStepSetMeeting):       ns.ProposeMeeting,
		string(outreachdomain.NextStepWait):             ns.Wait,
		string(outreachdomain.NextStepDrop):             ns.Nurture,
		string(outreachdomain.NextStepFollowUpMeeting1): ns.MeetingFollowUp,
		string(outreachdomain.NextStepFollowUpMeeting2): ns.MeetingFollowUp,
		string(outreachdomain.NextStepPrepareMeeting):   ns.MeetingFollowUp,
		string(outreachdomain.NextStepFollowUp):         ns.MeetingFollowUp,
	}[s.CadenceStep]
	if s.UnansweredReply && contains(f.Eligible, ns.AnswerReply) {
		want = ns.AnswerReply
	}
	a := want
	if !contains(f.Eligible, a) {
		a = f.Eligible[0]
	}
	return &choice{
		Action: a,
		Args:   copyArgs(f.Fixed[a]),
		Reason: fmt.Sprintf("Default cadence (step %s)", s.CadenceStep),
	}
}

// complete fills the arguments an action needs that the model left out, and
// drops ones that do not parse. Dates the prospect stated win over defaults.
func (e *Engine) complete(s *Situation, f filterResult, ch *choice) *choice {
	if ch.Args == nil {
		ch.Args = map[string]interface{}{}
	}
	args := ch.Args
	str := func(k string) string { v, _ := args[k].(string); return strings.TrimSpace(v) }
	date := func(k string, minDays, maxDays int) string {
		v := str(k)
		if v == "" {
			return ""
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.In(vietnam).Format("2006-01-02")
		}
		return dateWithin(&v, s.Now, minDays, maxDays)
	}
	at9 := func(day string) string {
		t, _ := time.ParseInLocation("2006-01-02", day, vietnam)
		return time.Date(t.Year(), t.Month(), t.Day(), 9, 0, 0, 0, vietnam).Format(time.RFC3339)
	}

	switch ch.Action {
	case ns.Wait:
		if _, fixed := f.Fixed[ns.Wait]; fixed {
			break
		}
		if d := date("until", 0, 120); d != "" {
			args["until"] = at9(d)
		} else {
			args["until"] = s.Now.Add(time.Duration(s.FollowUpDays) * 24 * time.Hour).Format(time.RFC3339)
		}
		if str("reason") == "" {
			args["reason"] = "waiting for a reply"
		}
	case ns.Nurture:
		revisit := ""
		if s.Details != nil && s.Details.RevisitDate != "" {
			revisit = s.Details.RevisitDate
		}
		if revisit == "" {
			revisit = date("revisit", 1, 400)
		}
		if revisit == "" {
			revisit = s.Now.AddDate(0, 0, s.ReEngageDays).In(vietnam).Format("2006-01-02")
		}
		args["revisit"] = revisit
		switch str("reason") {
		case "timing", "no_response", "not_now", "other":
		default:
			if s.Details != nil && s.Details.RevisitDate != "" {
				args["reason"] = "timing"
			} else if s.AwaitingReply {
				args["reason"] = "no_response"
			} else {
				args["reason"] = "other"
			}
		}
	case ns.AnswerReply:
		switch str("kind") {
		case "answer", "objection", "clarify":
		default:
			if s.Details != nil && s.Details.Objection != "" {
				args["kind"] = "objection"
			} else {
				args["kind"] = "answer"
			}
		}
		if s.Details != nil && s.Details.Objection != "" {
			args["objection"] = s.Details.Objection
		}
	case ns.MeetingFollowUp:
		switch str("kind") {
		case "confirm", "prepare", "after":
		default:
			if s.MeetingHeld && !s.MeetingScheduled {
				args["kind"] = "after"
			} else {
				args["kind"] = "confirm"
			}
		}
	case ns.SwitchChannel:
		if c := strings.ToLower(str("channel")); c != "linkedin" && c != "call" {
			args["channel"] = "linkedin"
		}
	case ns.ContactStakeholder:
		if s.Details != nil && (s.Details.ReferredName != "" || s.Details.ReferredEmail != "") {
			args["person"] = s.Details.ReferredName
			args["email"] = s.Details.ReferredEmail
			args["referrer"] = s.ContactName
		} else if str("person") == "" && len(s.Stakeholders) > 0 {
			args["person"] = s.Stakeholders[0].Name
			args["email"] = s.Stakeholders[0].Email
		}
		if e := emailOrEmpty(str("email")); e != "" {
			args["email"] = e
		} else {
			delete(args, "email")
		}
	case ns.FixData:
		args["field"] = "email"
	}
	if ch.Reason == "" {
		ch.Reason = "Chosen by the model"
	}
	return ch
}

// apply is stage 5: write the decision onto the contact and carry out the
// internal side effects. Outbound actions only become drafts for review.
func (e *Engine) apply(ctx context.Context, in Input, c *domain.Contact, s *Situation, d *ns.Decision, ch *choice) error {
	updates := map[string]interface{}{
		"next_action":             d.Action,
		"next_action_args":        d.Args,
		"next_action_reason":      d.Reason,
		"next_action_decision_id": d.ID,
		"next_action_due_at":      nil,
		"updated_at":              e.now(),
	}
	if due := dueAt(ch); due != nil {
		updates["next_action_due_at"] = *due
	}

	switch ch.Action {
	case ns.Suppress:
		// The request to stop is honoured at once and everywhere: the
		// do-not-contact flag is what every send path checks.
		updates["do_not_contact"] = true
		updates["outreach_stage"] = string(outreachdomain.StateDropped)
		updates["next_step"] = string(outreachdomain.NextStepDrop)
	case ns.ContactStakeholder:
		e.openStakeholder(ctx, in.UserID, c, ch)
	}

	if err := e.db.WithContext(ctx).Model(&domain.Contact{}).
		Where("id = ?", c.ID).Updates(updates).Error; err != nil {
		return err
	}

	if in.ConversationID != nil {
		e.annotateConversation(ctx, *in.ConversationID, d, updates["next_action_due_at"])
	}
	return nil
}

// openStakeholder creates the referred person as a lead whose own next step
// is an introduction naming the referrer. Without an address there is nobody
// to write to yet; the decision still records the name for the representative.
func (e *Engine) openStakeholder(ctx context.Context, userID uuid.UUID, c *domain.Contact, ch *choice) {
	email, _ := ch.Args["email"].(string)
	name, _ := ch.Args["person"].(string)
	if email == "" {
		return
	}
	if existing, err := e.contacts.FindByEmail(ctx, userID, email); err == nil && existing != nil {
		return
	}
	if name == "" {
		name = strings.Split(email, "@")[0]
	}
	referrer := c.Name
	reason := fmt.Sprintf("Referred by %s", referrer)
	action := string(ns.SendIntro)
	nc := &domain.Contact{
		UserID:             userID,
		OrganizationID:     c.OrganizationID,
		Name:               name,
		Company:            c.Company,
		ContactInformation: email,
		ContactChannel:     "Email",
		Source:             "referral",
		Industry:           c.Industry,
		NextAction:         &action,
		NextActionArgs:     mustJSON(map[string]interface{}{"angle": "introduction from " + referrer, "referrer": referrer}),
		NextActionReason:   &reason,
	}
	if err := e.contacts.Create(ctx, nc); err != nil {
		e.log().Warn().Err(err).Str("email", email).Msg("next-step: could not create the referred contact")
		return
	}
	ch.Args["contact_id"] = nc.ID.String()
}

// annotateConversation stores the decision on the conversation, where the
// reply drafter reads the goal of the message and the inbox shows it.
func (e *Engine) annotateConversation(ctx context.Context, conversationID uuid.UUID, d *ns.Decision, due interface{}) {
	var conv domain.Conversation
	if err := e.db.WithContext(ctx).Where("id = ?", conversationID).First(&conv).Error; err != nil {
		return
	}
	meta := map[string]interface{}{}
	_ = conv.CMetadata.Unmarshal(&meta)
	if meta == nil {
		meta = map[string]interface{}{}
	}
	var args map[string]interface{}
	_ = json.Unmarshal(d.Args, &args)
	entry := map[string]interface{}{
		"decision_id": d.ID.String(),
		"action":      d.Action,
		"args":        args,
		"reason":      d.Reason,
		"selected_by": d.SelectedBy,
		"status":      d.Status,
		"decided_at":  d.CreatedAt.Format(time.RFC3339),
	}
	if t, ok := due.(time.Time); ok {
		entry["due_at"] = t.Format(time.RFC3339)
	}
	meta["next_step"] = entry
	b, _ := json.Marshal(meta)
	e.db.WithContext(ctx).Model(&domain.Conversation{}).Where("id = ?", conversationID).
		Update("cmetadata", string(b))
}

// dueAt is when a waiting or nurtured contact should be decided again.
func dueAt(ch *choice) *time.Time {
	var v string
	switch ch.Action {
	case ns.Wait:
		v, _ = ch.Args["until"].(string)
	case ns.Nurture:
		v, _ = ch.Args["revisit"].(string)
	default:
		return nil
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return &t
	}
	if t, err := time.ParseInLocation("2006-01-02", v, vietnam); err == nil {
		t = t.Add(9 * time.Hour)
		return &t
	}
	return nil
}

// ---------------------------------------------------------------- reading

// History returns a contact's latest decisions, newest first.
func (e *Engine) History(ctx context.Context, userID, contactID uuid.UUID, limit int) (*domain.Contact, []ns.Decision, error) {
	c, err := e.contacts.FindByIDAndUserID(ctx, userID, contactID)
	if err != nil || c == nil {
		return nil, nil, ErrNotFound
	}
	var out []ns.Decision
	err = e.db.WithContext(ctx).Where("contact_id = ? AND user_id = ?", contactID, userID).
		Order("created_at DESC").Limit(limit).Find(&out).Error
	return c, out, err
}

// Review records the representative's verdict on a decision that waited for
// one. Confirming a disqualification closes the lead.
func (e *Engine) Review(ctx context.Context, userID, decisionID uuid.UUID, approve bool) (*ns.Decision, error) {
	var d ns.Decision
	if err := e.db.WithContext(ctx).Where("id = ? AND user_id = ?", decisionID, userID).First(&d).Error; err != nil {
		return nil, ErrNotFound
	}
	now := e.now()
	d.Status = ns.StatusRejected
	if approve {
		d.Status = ns.StatusApproved
	}
	d.ReviewedBy, d.ReviewedAt = &userID, &now
	if err := e.db.WithContext(ctx).Model(&d).Updates(map[string]interface{}{
		"status": d.Status, "reviewed_by": userID, "reviewed_at": now,
	}).Error; err != nil {
		return nil, err
	}
	if approve && d.Action == string(ns.Disqualify) {
		e.db.WithContext(ctx).Model(&domain.Contact{}).Where("id = ?", d.ContactID).Updates(map[string]interface{}{
			"outreach_stage": string(outreachdomain.StateDropped),
			"next_step":      string(outreachdomain.NextStepDrop),
		})
	}
	return &d, nil
}

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil || string(b) == "null" {
		return []byte("{}")
	}
	return b
}

func copyArgs(m map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range m {
		out[k] = v
	}
	return out
}
