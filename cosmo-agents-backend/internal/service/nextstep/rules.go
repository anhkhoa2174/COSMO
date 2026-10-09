package nextstep

import (
	"fmt"
	"time"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	ns "github.com/rockship/cosmo-agents-go/internal/domain/nextstep"
)

// vietnam is where COSMO's users work; dates a prospect states ("back on the
// 15th") are read in this zone and a wait ends at 09:00 local time.
var vietnam = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Ho_Chi_Minh"); err == nil {
		return loc
	}
	return time.FixedZone("ICT", 7*3600)
}()

// filterResult is the outcome of stage 2.
type filterResult struct {
	Eligible []ns.Action
	Hits     []ns.RuleHit
	// Arguments a rule fixed, e.g. the date a wait must last until.
	Fixed map[ns.Action]map[string]interface{}
	// The reason to give when a rule leaves exactly one action.
	RuleReason string
}

// applicable removes actions whose arguments cannot be filled in this
// situation — a meeting follow-up with no meeting, data repair with nothing
// broken. It runs before the ten rules and is recorded as rule 0, so the
// decision record still explains every action that was not offered.
func applicable(s *Situation) (keep []ns.Action, removed []ns.Action) {
	ok := func(a ns.Action) bool {
		switch a {
		case ns.Suppress:
			return s.RequestedStop()
		case ns.FixData:
			return s.Bounced
		case ns.MeetingFollowUp:
			return s.MeetingScheduled || s.MeetingHeld
		case ns.SendIntro:
			return s.SendsSoFar == 0 || s.RevisitDue
		case ns.SendFollowUp, ns.SwitchChannel:
			return s.SendsSoFar > 0
		case ns.Disqualify:
			return s.LastIncomingAt != nil
		}
		return true
	}
	for _, a := range ns.AllActions() {
		if ok(a) {
			keep = append(keep, a)
		} else {
			removed = append(removed, a)
		}
	}
	return keep, removed
}

// applyRules is stage 2: the eligibility rules of report Table 78, evaluated
// in order. A rule either keeps only some actions or removes some. A "keep
// only" rule that would leave nothing is recorded as not applied rather than
// emptying the set, so the engine always has something to decide among.
func applyRules(s *Situation) filterResult {
	set, removed := applicable(s)
	res := filterResult{Fixed: map[ns.Action]map[string]interface{}{}}
	if len(removed) > 0 {
		res.Hits = append(res.Hits, ns.RuleHit{
			Rule: 0, Name: "applicability",
			Effect:  "Actions whose arguments cannot be filled in this situation are not offered",
			Removed: removed,
		})
	}

	only := func(rule int, name, effect string, allowed ...ns.Action) bool {
		next := intersect(set, allowed)
		if len(next) == 0 {
			res.Hits = append(res.Hits, ns.RuleHit{Rule: rule, Name: name,
				Effect: effect + " (not applied: it would leave no action)"})
			return false
		}
		res.Hits = append(res.Hits, ns.RuleHit{Rule: rule, Name: name, Effect: effect, Removed: minus(set, next)})
		set = next
		res.RuleReason = effect
		return true
	}
	remove := func(rule int, name, effect string, drop ...ns.Action) {
		next := minus(set, drop)
		gone := minus(set, next)
		if len(gone) == 0 {
			return
		}
		res.Hits = append(res.Hits, ns.RuleHit{Rule: rule, Name: name, Effect: effect, Removed: gone})
		set = next
	}
	wait := func(until time.Time, reason string) {
		res.Fixed[ns.Wait] = map[string]interface{}{"until": until.Format(time.RFC3339), "reason": reason}
	}

	// 1. Asked to stop: nothing but suppression, and nothing else is weighed.
	if s.RequestedStop() {
		only(1, "opt-out", "The contact asked not to be contacted", ns.Suppress)
		res.Eligible = set
		return res
	}

	// 2. Bounced address.
	if s.Bounced {
		only(2, "bounce", "The address bounced; repair it or reach someone else", ns.FixData, ns.ContactStakeholder)
	}

	// 3. An active campaign sequence owns the proactive messages.
	if s.InActiveCampaign && !s.UnansweredReply {
		remove(3, "campaign", "An active campaign sequence sends the next message", ns.SendIntro, ns.SendFollowUp)
	}

	// 4. Away until a stated date.
	if s.Details != nil && s.Details.ReturnDate != "" {
		if back, err := time.ParseInLocation("2006-01-02", s.Details.ReturnDate, vietnam); err == nil && back.After(s.Now) {
			until := time.Date(back.Year(), back.Month(), back.Day()+1, 9, 0, 0, 0, vietnam)
			effect := fmt.Sprintf("The contact is away until %s", s.Details.ReturnDate)
			if only(4, "away", effect, ns.Wait) {
				wait(until, "away until "+s.Details.ReturnDate)
			}
		}
	}

	// 5. Inside the no-reply window after our last message.
	if s.AwaitingReply && s.LastOutgoingAt != nil {
		end := s.LastOutgoingAt.Add(time.Duration(s.NoReplyHours) * time.Hour)
		if end.After(s.Now) {
			effect := fmt.Sprintf("Still inside the %dh no-reply window", s.NoReplyHours)
			if only(5, "no-reply window", effect, ns.Wait) {
				if _, fixed := res.Fixed[ns.Wait]; !fixed {
					wait(end, "no-reply window")
				}
			}
		}
	}

	// 6. A meeting is booked: no sales email, only meeting follow-up or waiting.
	if s.MeetingScheduled {
		only(6, "meeting booked", "A meeting is scheduled", ns.MeetingFollowUp, ns.Wait)
	}

	// 7. The follow-up cap.
	if s.FollowUpsSent >= s.MaxFollowUps {
		remove(7, "follow-up cap",
			fmt.Sprintf("%d of %d follow-ups already sent", s.FollowUpsSent, s.MaxFollowUps), ns.SendFollowUp)
	}

	// 8. Nothing to answer.
	if !s.UnansweredReply {
		remove(8, "no reply", "There is no reply to answer", ns.AnswerReply)
	}

	// 9. Nobody else to contact.
	if !s.StakeholderKnown() {
		remove(9, "no stakeholder", "No referred person or second stakeholder is known", ns.ContactStakeholder)
	}

	// 10. A reply the classifier could not read, or a complaint.
	if s.UnansweredReply && (s.LatestIntent == string(domain.IntentUnknown) ||
		(s.Details != nil && s.Details.Complaint)) {
		only(10, "unclear or complaint", "The reply is unclear or a complaint; a person decides", ns.Escalate)
	}

	res.Eligible = set
	return res
}

func intersect(set, allowed []ns.Action) []ns.Action {
	keep := map[ns.Action]bool{}
	for _, a := range allowed {
		keep[a] = true
	}
	var out []ns.Action
	for _, a := range set {
		if keep[a] {
			out = append(out, a)
		}
	}
	return out
}

func minus(set, drop []ns.Action) []ns.Action {
	gone := map[ns.Action]bool{}
	for _, a := range drop {
		gone[a] = true
	}
	var out []ns.Action
	for _, a := range set {
		if !gone[a] {
			out = append(out, a)
		}
	}
	return out
}

func contains(set []ns.Action, a ns.Action) bool {
	for _, x := range set {
		if x == a {
			return true
		}
	}
	return false
}
