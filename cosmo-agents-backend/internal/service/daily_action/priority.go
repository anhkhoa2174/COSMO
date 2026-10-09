package daily_action

import (
	"math"
	"time"

	contact "github.com/rockship/cosmo-agents-go/internal/domain/contact"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	outreach "github.com/rockship/cosmo-agents-go/internal/domain/outreach"
)

// Priority scoring weights (must sum to 100).
const (
	weightRecency      = 30
	weightCadence      = 25
	weightMeeting      = 25
	weightCompleteness = 12
	weightLifecycle    = 8
)

// ComputePriority calculates a priority score for an action.
// Score 1 = highest urgency, 100 = lowest. Returns the score and factor breakdown.
func ComputePriority(c *contact.Contact, state *outreach.OutreachState, meeting *outreach.Meeting) (int, []domain.PriorityFactor) {
	recency, recencyDesc := scoreRecency(state)
	cadence, cadenceDesc := scoreCadence(state)
	meetingScore, meetingDesc := scoreMeetingProximity(meeting)
	completeness, completenessDesc := scoreCompleteness(c)
	lifecycle, lifecycleDesc := scoreLifecycle(c)

	// Weighted urgency on 0-100 scale
	urgency := float64(weightRecency)*float64(recency)/100 +
		float64(weightCadence)*float64(cadence)/100 +
		float64(weightMeeting)*float64(meetingScore)/100 +
		float64(weightCompleteness)*float64(completeness)/100 +
		float64(weightLifecycle)*float64(lifecycle)/100

	// Invert: high urgency → low score number (1=highest priority)
	score := int(math.Round(100 - urgency))
	if score < 1 {
		score = 1
	}
	if score > 100 {
		score = 100
	}

	factors := []domain.PriorityFactor{
		{Factor: "response_recency", Value: recency, Description: recencyDesc},
		{Factor: "cadence_deadline", Value: cadence, Description: cadenceDesc},
		{Factor: "meeting_proximity", Value: meetingScore, Description: meetingDesc},
		{Factor: "data_completeness", Value: completeness, Description: completenessDesc},
		{Factor: "lifecycle_stage", Value: lifecycle, Description: lifecycleDesc},
	}

	return score, factors
}

// scoreRecency scores based on how recently the contact responded.
// REPLIED + <1h = 100, <4h = 80, <24h = 60, <72h = 40, else = 20
func scoreRecency(state *outreach.OutreachState) (int, string) {
	if state == nil || state.ConversationState != "REPLIED" || state.LastInteractionAt == nil {
		return 0, "No recent reply"
	}
	hours := time.Since(*state.LastInteractionAt).Hours()
	switch {
	case hours < 1:
		return 100, "Replied less than 1 hour ago"
	case hours < 4:
		return 80, "Replied within 4 hours"
	case hours < 24:
		return 60, "Replied within 24 hours"
	case hours < 72:
		return 40, "Replied within 3 days"
	default:
		return 20, "Replied more than 3 days ago"
	}
}

// scoreCadence scores based on follow-up deadline proximity.
// In FU window = 100, slightly overdue = 70, well overdue = 40, not due = 10
func scoreCadence(state *outreach.OutreachState) (int, string) {
	if state == nil {
		return 10, "No outreach state"
	}

	nextStep := state.NextStep
	days := state.DaysSinceLastInteraction

	switch nextStep {
	case "FOLLOW_UP_1", "FOLLOW_UP_2", "FOLLOW_UP":
		switch {
		case days >= 4 && days <= 5:
			return 100, "Follow-up due today (in window)"
		case days > 5 && days <= 7:
			return 70, "Follow-up slightly overdue"
		case days > 7:
			return 40, "Follow-up well overdue"
		default:
			return 10, "Follow-up not yet due"
		}
	case "SEND":
		return 50, "Initial outreach pending"
	case "DROP":
		return 20, "Contact marked for drop"
	case "WAIT":
		return 10, "Waiting for response"
	default:
		return 10, "No follow-up cadence"
	}
}

// scoreMeetingProximity scores based on how close an upcoming meeting is.
// <2h = 100, <4h = 90, <8h = 75, <24h = 50, <48h = 30, else = 5
func scoreMeetingProximity(meeting *outreach.Meeting) (int, string) {
	if meeting == nil || meeting.Status != "scheduled" {
		return 0, "No upcoming meeting"
	}
	hours := time.Until(meeting.Time).Hours()
	if hours < 0 {
		return 0, "Meeting already passed"
	}
	switch {
	case hours < 2:
		return 100, "Meeting in less than 2 hours"
	case hours < 4:
		return 90, "Meeting in less than 4 hours"
	case hours < 8:
		return 75, "Meeting in less than 8 hours"
	case hours < 24:
		return 50, "Meeting within 24 hours"
	case hours < 48:
		return 30, "Meeting within 48 hours"
	default:
		return 5, "Meeting more than 48 hours away"
	}
}

// scoreCompleteness scores based on how many required fields are missing.
// 4+ missing = 100, 3 = 50, 2 = 30, 1 = 10, 0 = 0
func scoreCompleteness(c *contact.Contact) (int, string) {
	missing := c.GetMissingFields()
	count := len(missing)
	switch {
	case count >= 4:
		return 100, "4+ fields missing — urgent enrichment needed"
	case count == 3:
		return 50, "3 fields missing"
	case count == 2:
		return 30, "2 fields missing"
	case count == 1:
		return 10, "1 field missing"
	default:
		return 0, "Profile complete"
	}
}

// scoreLifecycle scores based on contact's business lifecycle stage.
// Pre-sales = 100, sales = 50, post-sales = 10. Migration 000050 rewrote the
// stored values to the lifecycle names (PRE_SALES→LEAD, SALES→OPPORTUNITY,
// POST_SALES→CUSTOMER), so both vocabularies are accepted.
func scoreLifecycle(c *contact.Contact) (int, string) {
	switch c.BusinessStage {
	case "PRE_SALES", "SUBSCRIBER", "LEAD":
		return 100, "Pre-sales pipeline"
	case "SALES", "QUALIFIED", "OPPORTUNITY":
		return 50, "Active sales"
	case "POST_SALES", "CUSTOMER", "ADVOCATE":
		return 10, "Post-sales"
	default:
		return 50, "Unknown stage"
	}
}
