package intent

import (
	"fmt"
	"strings"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// nextStepGoal reads the next-step decision the engine stored on the
// conversation (report Section 6.7) and turns it into the goal of the draft.
//
// draft is false for actions that send nothing — waiting, nurture,
// suppression, escalation — so no draft is written for them. Without a stored
// decision (the engine is off) it returns draft=true and no goal, which is the
// drafting behaviour that existed before the engine.
func nextStepGoal(meta base.JSONB) (draft bool, goal string) {
	var m map[string]interface{}
	if err := meta.Unmarshal(&m); err != nil || m == nil {
		return true, ""
	}
	ns, ok := m["next_step"].(map[string]interface{})
	if !ok {
		return true, ""
	}
	action, _ := ns["action"].(string)
	args, _ := ns["args"].(map[string]interface{})
	arg := func(k string) string { v, _ := args[k].(string); return strings.TrimSpace(v) }

	switch action {
	case "ANSWER_REPLY":
		switch arg("kind") {
		case "objection":
			obj := arg("objection")
			if obj == "" {
				obj = "an"
			}
			return true, fmt.Sprintf("The prospect raised %s objection. Acknowledge it, ask one short "+
				"clarifying question, and answer it with value and evidence from the knowledge provided. "+
				"Do not offer a discount that the knowledge does not state.", obj)
		case "clarify":
			return true, "Ask one short, polite question to understand whether they are truly not " +
				"interested or whether the timing or fit is the problem. Do not pitch again."
		}
		return true, "Answer every question the prospect asked, using only the knowledge provided, " +
			"then propose one concrete next step."
	case "PROPOSE_MEETING":
		return true, "Thank them and propose a short meeting: offer two time options or a booking link. Keep it brief."
	case "MEETING_FOLLOW_UP":
		switch arg("kind") {
		case "prepare":
			return true, "Confirm the agenda for the upcoming meeting and ask what they most want to cover."
		case "after":
			return true, "Follow up after the meeting: summarise what was agreed and the next step."
		}
		return true, "Confirm the meeting time and the attendees."
	case "CONTACT_STAKEHOLDER":
		person := arg("person")
		if person == "" {
			person = "the colleague they mentioned"
		}
		return true, fmt.Sprintf("Thank the prospect for pointing you to %s and say you will reach out to "+
			"them directly. Keep it to two or three sentences.", person)
	case "SEND_FOLLOW_UP", "SEND_INTRO":
		return true, ""
	case "":
		return true, ""
	}
	// WAIT, NURTURE, SUPPRESS, FIX_DATA, DISQUALIFY, ESCALATE, SWITCH_CHANNEL
	return false, ""
}

// DraftsFor reports whether a stored next-step decision calls for a written
// reply, so the reply worker can draft one even when the intent's own handler
// does not (an objection classified as "Not interested", for instance).
func DraftsFor(meta base.JSONB) bool {
	var m map[string]interface{}
	if err := meta.Unmarshal(&m); err != nil || m == nil {
		return false
	}
	ns, ok := m["next_step"].(map[string]interface{})
	if !ok {
		return false
	}
	switch ns["action"] {
	case "ANSWER_REPLY", "PROPOSE_MEETING", "MEETING_FOLLOW_UP", "CONTACT_STAKEHOLDER":
		return true
	}
	return false
}

// replyRoles tells the drafter who is writing and to whom.
func replyRoles(sender, recipient string) string {
	var b strings.Builder
	if sender != "" {
		fmt.Fprintf(&b, "You write as %s and sign the email with that name.", sender)
	}
	if recipient != "" {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		first := strings.Fields(recipient)[0]
		fmt.Fprintf(&b, "The recipient is %s; greet them by name (%s).", recipient, first)
	}
	return b.String()
}
