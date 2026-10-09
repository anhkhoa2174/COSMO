package nextstep

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	ns "github.com/rockship/cosmo-agents-go/internal/domain/nextstep"
)

// ReplyDetails are the facts an action needs that only the wording of a reply
// carries: when someone is back, whom they pointed to, when to come back,
// what they objected to. They are extracted once per reply (extract.go).
type ReplyDetails struct {
	ReturnDate    string `json:"return_date,omitempty"`    // YYYY-MM-DD, from an out-of-office notice
	ReferredName  string `json:"referred_name,omitempty"`  // "talk to X"
	ReferredEmail string `json:"referred_email,omitempty"` //
	RevisitDate   string `json:"revisit_date,omitempty"`   // YYYY-MM-DD, "next quarter", "after Tet"
	Objection     string `json:"objection,omitempty"`      // price, competitor, timing, need, authority, other
	Complaint     bool   `json:"complaint,omitempty"`
	Unsubscribe   bool   `json:"unsubscribe,omitempty"`
	Bounce        bool   `json:"bounce,omitempty"`
}

// Person is another stakeholder the engine could contact.
type Person struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	Title string `json:"title,omitempty"`
}

// Situation is everything stage 1 knows about a lead. It is assembled from
// stored data only, and it is stored with the decision, so a decision can be
// re-read later exactly as the engine saw it.
type Situation struct {
	Now time.Time `json:"now"`

	ContactName string `json:"contact_name"`
	Company     string `json:"company,omitempty"`
	JobTitle    string `json:"job_title,omitempty"`
	Email       string `json:"email,omitempty"`

	OutreachStage string `json:"outreach_stage"`
	CadenceStep   string `json:"cadence_step"` // what the fixed cadence would do now

	SendsSoFar     int        `json:"sends_so_far"`
	FollowUpsSent  int        `json:"follow_ups_sent"`
	MaxFollowUps   int        `json:"max_follow_ups"`
	NoReplyHours   int        `json:"no_reply_hours"`
	FollowUpDays   int        `json:"follow_up_days"`
	ReEngageDays   int        `json:"re_engage_days"`
	LastOutgoingAt *time.Time `json:"last_outgoing_at,omitempty"`
	LastIncomingAt *time.Time `json:"last_incoming_at,omitempty"`

	// Elapsed time, computed here rather than left to the model: a small
	// model given two timestamps once read five days as eight hours.
	HoursSinceLastOutgoing *int `json:"hours_since_last_outgoing,omitempty"`
	DaysSinceLastOutgoing  *int `json:"days_since_last_outgoing,omitempty"`
	DaysSinceLastIncoming  *int `json:"days_since_last_incoming,omitempty"`
	FollowUpDue            bool `json:"follow_up_due"` // silent for at least follow_up_days

	// AwaitingReply: our message is the latest one. UnansweredReply: theirs is.
	AwaitingReply   bool `json:"awaiting_reply"`
	UnansweredReply bool `json:"unanswered_reply"`

	LatestIntent  string        `json:"latest_intent,omitempty"`
	LatestReply   string        `json:"latest_reply,omitempty"`
	LatestEmailID string        `json:"latest_email_id,omitempty"`
	Details       *ReplyDetails `json:"details,omitempty"`

	DoNotContact     bool     `json:"do_not_contact"`
	Bounced          bool     `json:"bounced"`
	MeetingScheduled bool     `json:"meeting_scheduled"`
	MeetingHeld      bool     `json:"meeting_held"`
	InActiveCampaign bool     `json:"in_active_campaign"`
	Stakeholders     []Person `json:"stakeholders,omitempty"`
	LeadScore        *float64 `json:"lead_score,omitempty"`

	// The previous decision, so a nurture that has come due can re-open
	// outreach and the model can see what was tried last.
	PreviousAction string `json:"previous_action,omitempty"`
	PreviousReason string `json:"previous_reason,omitempty"`
	RevisitDue     bool   `json:"revisit_due"`
}

// RequestedStop is true when the contact has asked not to be contacted.
func (s *Situation) RequestedStop() bool {
	return s.DoNotContact ||
		s.LatestIntent == string(domain.IntentDoNotContact) ||
		(s.Details != nil && s.Details.Unsubscribe)
}

// StakeholderKnown is true when there is someone else to contact.
func (s *Situation) StakeholderKnown() bool {
	if s.Details != nil && (s.Details.ReferredName != "" || s.Details.ReferredEmail != "") {
		return true
	}
	return len(s.Stakeholders) > 0
}

// buildSituation assembles stage 1. `in` may carry the reply that triggered
// the decision; otherwise the latest stored reply is used.
func (e *Engine) buildSituation(ctx context.Context, in Input, c *domain.Contact) (*Situation, error) {
	cfg := e.outreach.ConfigFor(ctx, in.UserID)
	now := e.now()

	s := &Situation{
		Now:           now,
		ContactName:   c.Name,
		Company:       clean(c.Company),
		JobTitle:      clean(c.JobTitle),
		Email:         c.ContactInformation,
		OutreachStage: c.OutreachStage,
		MaxFollowUps:  cfg.MaxFollowups,
		NoReplyHours:  cfg.NoReplyHours,
		FollowUpDays:  cfg.FollowUp1MinDays,
		ReEngageDays:  cfg.ReEngageThresholdDays,
		DoNotContact:  c.DoNotContact,
	}
	if s.FollowUpDays <= 0 {
		s.FollowUpDays = 4
	}

	// The fixed cadence's view: kept as the fallback and shown to the model.
	cad, err := e.outreach.DetermineConversationState(ctx, c, in.UserID)
	if err != nil {
		return nil, fmt.Errorf("cadence state: %w", err)
	}
	s.CadenceStep = string(cad.NextStep)
	s.FollowUpsSent = cad.FollowupCount
	if cad.LastOutgoing != nil {
		t := cad.LastOutgoing.Timestamp
		s.LastOutgoingAt = &t
	}
	if cad.LastIncoming != nil {
		t := cad.LastIncoming.Timestamp
		s.LastIncomingAt = &t
	}
	if s.LastOutgoingAt != nil {
		h := int(now.Sub(*s.LastOutgoingAt).Hours())
		d := h / 24
		s.HoursSinceLastOutgoing, s.DaysSinceLastOutgoing = &h, &d
	}
	if s.LastIncomingAt != nil {
		d := int(now.Sub(*s.LastIncomingAt).Hours()) / 24
		s.DaysSinceLastIncoming = &d
	}
	switch {
	case s.LastIncomingAt != nil && (s.LastOutgoingAt == nil || s.LastIncomingAt.After(*s.LastOutgoingAt)):
		s.UnansweredReply = true
	case s.LastOutgoingAt != nil:
		s.AwaitingReply = true
		s.FollowUpDue = s.DaysSinceLastOutgoing != nil && *s.DaysSinceLastOutgoing >= s.FollowUpDays
	}

	if n, err := e.interactions.CountOutgoingByUserID(ctx, c.ID, in.UserID); err == nil {
		s.SendsSoFar = n
	}
	if ok, err := e.meetings.HasScheduledMeeting(ctx, c.ID); err == nil {
		s.MeetingScheduled = ok
	}
	if ok, err := e.meetings.HasCompletedMeeting(ctx, c.ID); err == nil {
		s.MeetingHeld = ok
	}

	s.InActiveCampaign = e.inActiveCampaign(ctx, c.ID)
	s.Bounced = e.hasBounce(ctx, c.ID)
	s.Stakeholders = e.stakeholders(ctx, in.UserID, c)
	s.LeadScore = leadScore(c)

	if c.NextAction != nil {
		s.PreviousAction = *c.NextAction
		if c.NextActionReason != nil {
			s.PreviousReason = *c.NextActionReason
		}
		s.RevisitDue = *c.NextAction == string(ns.Nurture) &&
			c.NextActionDueAt != nil && !c.NextActionDueAt.After(now)
	}

	// The reply: the one that triggered this decision, or the latest stored.
	if in.ReplyText != "" {
		s.LatestIntent = in.Intent
		s.LatestReply = truncate(in.ReplyText, 1500)
		if in.EmailID != nil {
			s.LatestEmailID = in.EmailID.String()
		}
	} else if s.UnansweredReply || s.LastIncomingAt != nil {
		e.loadLatestReply(ctx, in.UserID, c, s)
	}

	if s.LatestReply != "" {
		s.Details = e.detailsFor(ctx, c.ID, s)
		if s.Details != nil && s.Details.Bounce {
			s.Bounced = true
		}
	}
	return s, nil
}

// loadLatestReply reads the newest inbound email from the contact.
func (e *Engine) loadLatestReply(ctx context.Context, userID uuid.UUID, c *domain.Contact, s *Situation) {
	if c.ContactInformation == "" {
		return
	}
	var row struct {
		ID      uuid.UUID
		Content string
		Intents string
	}
	err := e.db.WithContext(ctx).Raw(`
		SELECT id, content, array_to_string(intents, ',') AS intents
		FROM emails
		WHERE user_id = ? AND lower(from_email) = lower(?) AND is_deleted = false
		ORDER BY created_at DESC LIMIT 1`, userID, c.ContactInformation).Scan(&row).Error
	if err != nil || row.ID == uuid.Nil {
		return
	}
	s.LatestEmailID = row.ID.String()
	s.LatestReply = truncate(row.Content, 1500)
	if row.Intents != "" {
		s.LatestIntent = strings.Split(row.Intents, ",")[0]
	}
}

// detailsFor reuses the details extracted for this reply by an earlier
// decision, and extracts them only for a reply not seen before: a wait that
// comes due should not pay for reading the same email again.
func (e *Engine) detailsFor(ctx context.Context, contactID uuid.UUID, s *Situation) *ReplyDetails {
	if s.LatestEmailID != "" {
		var prev ns.Decision
		err := e.db.WithContext(ctx).
			Where("contact_id = ? AND situation->>'latest_email_id' = ?", contactID, s.LatestEmailID).
			Order("created_at DESC").Limit(1).Find(&prev).Error
		if err == nil && prev.ID != uuid.Nil {
			var old Situation
			if json.Unmarshal(prev.Situation, &old) == nil && old.Details != nil {
				return old.Details
			}
		}
	}
	d, err := e.extract(ctx, s.LatestReply, s.Now)
	if err != nil {
		e.log().Warn().Err(err).Msg("next-step: reply details could not be extracted; deciding without them")
		return nil
	}
	return d
}

func (e *Engine) inActiveCampaign(ctx context.Context, contactID uuid.UUID) bool {
	var n int64
	e.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM tasks t
		JOIN campaigns c ON c.id = t.campaign_id
		WHERE t.contact_id = ? AND t.status = 'pending'
		  AND c.status = 'active' AND c.is_deleted = false`, contactID).Scan(&n)
	return n > 0
}

func (e *Engine) hasBounce(ctx context.Context, contactID uuid.UUID) bool {
	var n int64
	e.db.WithContext(ctx).Raw(`
		SELECT count(*) FROM interactions
		WHERE contact_id = ? AND interaction_type = 'bounce'`, contactID).Scan(&n)
	return n > 0
}

// stakeholders lists up to three other people at the same company.
func (e *Engine) stakeholders(ctx context.Context, userID uuid.UUID, c *domain.Contact) []Person {
	company := clean(c.Company)
	if company == "" {
		return nil
	}
	var rows []struct {
		Name               string
		ContactInformation string
		JobTitle           string
	}
	e.db.WithContext(ctx).Raw(`
		SELECT name, contact_information, job_title FROM contacts
		WHERE user_id = ? AND id <> ? AND is_deleted = false AND do_not_contact = false
		  AND lower(btrim(company)) = lower(btrim(?))
		ORDER BY created_at LIMIT 3`, userID, c.ID, company).Scan(&rows)
	out := make([]Person, 0, len(rows))
	for _, r := range rows {
		out = append(out, Person{Name: r.Name, Email: r.ContactInformation, Title: clean(r.JobTitle)})
	}
	return out
}

func leadScore(c *domain.Contact) *float64 {
	if len(c.Scores) == 0 {
		return nil
	}
	var m map[string]interface{}
	if json.Unmarshal(c.Scores, &m) != nil {
		return nil
	}
	if v, ok := m["priority_score"].(float64); ok {
		return &v
	}
	return nil
}

func clean(v string) string {
	v = strings.TrimSpace(v)
	if strings.EqualFold(v, "N/A") {
		return ""
	}
	return v
}

func truncate(v string, n int) string {
	r := []rune(strings.TrimSpace(v))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}
