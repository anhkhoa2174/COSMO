// Package nextstep holds the vocabulary of the next-step decision engine
// (report Section 6.7): the catalogue of actions a decision can return, how
// each is approved, and the record every decision leaves behind.
package nextstep

import (
	"time"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// Action is one entry of the catalogue. A decision always returns exactly one.
type Action string

const (
	SendIntro          Action = "SEND_INTRO"
	SendFollowUp       Action = "SEND_FOLLOW_UP"
	AnswerReply        Action = "ANSWER_REPLY"
	ProposeMeeting     Action = "PROPOSE_MEETING"
	MeetingFollowUp    Action = "MEETING_FOLLOW_UP"
	ContactStakeholder Action = "CONTACT_STAKEHOLDER"
	SwitchChannel      Action = "SWITCH_CHANNEL"
	Wait               Action = "WAIT"
	Nurture            Action = "NURTURE"
	FixData            Action = "FIX_DATA"
	Suppress           Action = "SUPPRESS"
	Disqualify         Action = "DISQUALIFY"
	Escalate           Action = "ESCALATE"
)

// Approval says who has to agree before an action takes effect. It follows
// reversibility: what stays inside COSMO runs on its own, what leaves it waits
// for a person.
type Approval string

const (
	ApprovalAutomatic Approval = "automatic" // internal and reversible
	ApprovalApprove   Approval = "approve"   // an outbound message; a person approves the draft
	ApprovalTask      Approval = "task"      // becomes a task for the representative
	ApprovalConfirm   Approval = "confirm"   // a person confirms before it is final
	ApprovalDecide    Approval = "decide"    // handed to a person to decide
)

// Spec describes one catalogue entry.
type Spec struct {
	Action      Action
	Description string   // what the action means, shown to the model and the UI
	Args        []string // argument names the action takes
	Approval    Approval
	Sends       bool // whether the action produces an outbound message
}

// Catalogue is the full set of actions, in the fixed priority order used when
// a choice has to be made without the model.
var Catalogue = []Spec{
	{Suppress, "Stop all contact; the person asked not to be contacted.", nil, ApprovalAutomatic, false},
	{Escalate, "Hand the conversation to a person to decide.", []string{"reason"}, ApprovalDecide, false},
	{Wait, "Do nothing until a date, then decide again.", []string{"until", "reason"}, ApprovalAutomatic, false},
	{FixData, "Repair the contact's data (for example a bounced address).", []string{"field"}, ApprovalAutomatic, false},
	{MeetingFollowUp, "Confirm, prepare for, or follow up after a meeting.", []string{"kind"}, ApprovalApprove, true},
	{AnswerReply, "Reply to the prospect's message: answer it, address an objection, or ask one clarifying question.", []string{"kind"}, ApprovalApprove, true},
	{ProposeMeeting, "Propose a meeting.", nil, ApprovalApprove, true},
	{ContactStakeholder, "Open outreach to another person at the company, such as someone the prospect referred.", []string{"person", "email", "referrer"}, ApprovalApprove, true},
	{SendFollowUp, "Send a follow-up email from a new angle.", []string{"angle"}, ApprovalApprove, true},
	{SwitchChannel, "Try another channel: create a LinkedIn or call task for the representative.", []string{"channel"}, ApprovalTask, false},
	{SendIntro, "Send a first introduction email.", []string{"angle"}, ApprovalApprove, true},
	{Nurture, "Move to long-interval nurture with a reason and a date to come back.", []string{"reason", "revisit"}, ApprovalAutomatic, false},
	{Disqualify, "Mark the lead as not a fit, with a reason.", []string{"reason"}, ApprovalConfirm, false},
}

var specs = func() map[Action]Spec {
	m := make(map[Action]Spec, len(Catalogue))
	for _, s := range Catalogue {
		m[s.Action] = s
	}
	return m
}()

// SpecOf returns the catalogue entry for an action.
func SpecOf(a Action) (Spec, bool) {
	s, ok := specs[a]
	return s, ok
}

// AllActions returns every action in catalogue order.
func AllActions() []Action {
	out := make([]Action, 0, len(Catalogue))
	for _, s := range Catalogue {
		out = append(out, s.Action)
	}
	return out
}

// Trigger is the event that started a decision.
type Trigger string

const (
	TriggerReply   Trigger = "reply"   // an inbound reply was classified
	TriggerTimer   Trigger = "timer"   // a wait or nurture date was reached
	TriggerCadence Trigger = "cadence" // the default cadence moved on (e.g. the no-reply window ended)
	TriggerManual  Trigger = "manual"  // a person asked for a decision
)

// SelectedBy records how the action was picked.
type SelectedBy string

const (
	SelectedByRule     SelectedBy = "rule"     // only one action was eligible
	SelectedByModel    SelectedBy = "model"    // the model chose among eligible actions
	SelectedByFallback SelectedBy = "fallback" // the model failed; the default cadence was used
)

// Status of a decision.
const (
	StatusApplied       = "applied"
	StatusPendingReview = "pending_review"
	StatusApproved      = "approved"
	StatusRejected      = "rejected"
)

// RuleHit is one eligibility rule that applied.
type RuleHit struct {
	Rule    int      `json:"rule"`
	Name    string   `json:"name"`
	Effect  string   `json:"effect"`
	Removed []Action `json:"removed,omitempty"`
}

// Decision is the stored record of one pass of the engine.
type Decision struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	ContactID     uuid.UUID  `gorm:"type:uuid;not null" json:"contact_id"`
	Trigger       string     `gorm:"not null" json:"trigger"`
	Situation     base.JSONB `gorm:"type:jsonb" json:"situation"`
	RulesFired    base.JSONB `gorm:"type:jsonb" json:"rules_fired"`
	Eligible      base.JSONB `gorm:"type:jsonb" json:"eligible"`
	Action        string     `gorm:"not null" json:"action"`
	Args          base.JSONB `gorm:"type:jsonb" json:"args"`
	Reason        string     `json:"reason"`
	SelectedBy    string     `gorm:"not null" json:"selected_by"`
	FallbackCause string     `json:"fallback_cause,omitempty"`
	Approval      string     `gorm:"not null" json:"approval"`
	Status        string     `gorm:"not null" json:"status"`
	Model         string     `json:"model"`
	PromptVersion string     `json:"prompt_version"`
	ReviewedBy    *uuid.UUID `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"created_at"`
}

// TableName binds Decision to its table.
func (Decision) TableName() string { return "next_step_decisions" }
