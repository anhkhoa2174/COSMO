// Package autoreply decides whether an AI-written reply may be sent without a
// person reading it first.
//
// Every reply COSMO writes is held as a draft for review. That is the right
// default and it stays the default, but it is the wrong answer for a team whose
// inbox fills with replies that need an acknowledgement and nothing more. This
// package lets an administrator hand specific intents over to the system while
// keeping the rest under review.
//
// The design is deliberately restrictive, because the failure mode is not a
// wrong pixel — it is a wrong email in a prospect's inbox that cannot be
// recalled:
//
//   - Off unless switched on, and then only for intents named explicitly.
//     There is no "all intents" option.
//   - Some intents can never be auto-sent, whatever the settings say. Those
//     are not configuration; they are in the code below.
//   - A confidence floor, because an auto-send decision inherits the
//     classifier's uncertainty. A misread intent sends a confident reply to
//     the wrong question.
//   - A daily cap, so a misconfiguration or a reply storm costs a bounded
//     number of emails rather than an afternoon's worth.
//
// Every refusal carries a reason, which is what makes the behaviour auditable
// after the fact instead of merely quiet.
package autoreply

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// neverAutoSend lists the intents that stay under human review no matter how
// the organisation is configured.
//
// DO_NOT_CONTACT is the important one: the prospect has asked to be left
// alone, so an automatic reply is precisely the thing they objected to, and in
// several jurisdictions continuing to mail them after an opt-out is not merely
// rude. NOT_INTERESTED is close enough to an opt-out that a machine should not
// be the one to answer it. UNKNOWN_INTENT means the classifier could not tell
// what it was reading, which is the worst possible basis for sending mail.
var neverAutoSend = map[domain.IntentType]string{
	domain.IntentDoNotContact:  "the prospect asked not to be contacted",
	domain.IntentNotInterested: "a decline is answered by a person, not a machine",
	domain.IntentUnknown:       "the intent could not be classified",
}

// Defaults for an organisation that switches auto-reply on without tuning it.
const (
	DefaultMinConfidence = 0.9
	DefaultDailyCap      = 20

	minAllowedConfidence = 0.5 // below this the floor is not a floor
	maxDailyCap          = 500
)

// Policy is the admin-editable part. Pointers distinguish "not set" from "set
// to zero", so an organisation can change the cap without pinning the
// confidence floor to whatever the default was on the day it saved.
type Policy struct {
	Enabled *bool `json:"enabled,omitempty"`

	// Intents named here may be answered automatically. Empty means none:
	// enabling auto-reply without naming an intent changes nothing, which is
	// the safe reading of an incomplete configuration.
	Intents []string `json:"intents,omitempty"`

	MinConfidence *float64 `json:"min_confidence,omitempty"`
	DailyCap      *int     `json:"daily_cap,omitempty"`

	// Contents holds what each intent's reply says, keyed by intent. It may
	// name intents that are not currently selected, so unticking an intent
	// does not throw away the message an administrator wrote for it.
	Contents map[string]Content `json:"contents,omitempty"`
}

// Resolved is a Policy with the defaults filled in.
type Resolved struct {
	Enabled       bool                          `json:"enabled"`
	Intents       map[domain.IntentType]bool    `json:"-"`
	MinConfidence float64                       `json:"min_confidence"`
	DailyCap      int                           `json:"daily_cap"`
	Contents      map[domain.IntentType]Content `json:"-"`
}

// Disabled is the state every organisation starts in.
func Disabled() Resolved {
	return Resolved{
		Enabled:       false,
		Intents:       map[domain.IntentType]bool{},
		MinConfidence: DefaultMinConfidence,
		DailyCap:      DefaultDailyCap,
		Contents:      map[domain.IntentType]Content{},
	}
}

// Selectable lists the intents an administrator is allowed to choose, so the
// settings UI and the validator agree on one source of truth.
func Selectable() []domain.IntentType {
	var out []domain.IntentType
	for _, i := range domain.AllIntents() {
		if _, banned := neverAutoSend[i]; !banned {
			out = append(out, i)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out
}

// Validate reports every problem at once rather than the first, so an admin
// fixing a form is not sent back three times.
func (p Policy) Validate() []string {
	var problems []string

	if p.MinConfidence != nil {
		if *p.MinConfidence < minAllowedConfidence || *p.MinConfidence > 1 {
			problems = append(problems, fmt.Sprintf(
				"Auto-reply confidence floor must be between %.2f and 1.00 (got %.2f)",
				minAllowedConfidence, *p.MinConfidence))
		}
	}
	if p.DailyCap != nil && (*p.DailyCap < 1 || *p.DailyCap > maxDailyCap) {
		problems = append(problems, fmt.Sprintf(
			"Auto-reply daily cap must be between 1 and %d (got %d)",
			maxDailyCap, *p.DailyCap))
	}

	for _, raw := range p.Intents {
		intent := parseIntent(raw)
		if intent == "" {
			problems = append(problems,
				fmt.Sprintf("%q is not an intent COSMO recognises", raw))
			continue
		}
		// Rejecting a banned intent loudly beats silently dropping it: an
		// admin who ticked DO_NOT_CONTACT needs to be told it will not happen,
		// not left believing it was saved.
		if why, banned := neverAutoSend[intent]; banned {
			problems = append(problems, fmt.Sprintf(
				"%s can never be answered automatically: %s",
				strings.ToUpper(raw), why))
		}
	}

	// Keys are sorted so the problems come back in the same order every time.
	keys := make([]string, 0, len(p.Contents))
	for k := range p.Contents {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, raw := range keys {
		intent := parseIntent(raw)
		if intent == "" {
			problems = append(problems,
				fmt.Sprintf("reply content for %q: not an intent COSMO recognises", raw))
			continue
		}
		if why, banned := neverAutoSend[intent]; banned {
			problems = append(problems, fmt.Sprintf(
				"%s can never be answered automatically, so it cannot have reply content: %s",
				canonical(intent), why))
			continue
		}
		problems = append(problems, p.Contents[raw].validate(canonical(intent))...)
	}

	return problems
}

// Resolve fills in the defaults. A policy that fails validation resolves to
// disabled rather than to something partly applied — auto-sending is not a
// feature to fall back into.
func (p Policy) Resolve() Resolved {
	out := Disabled()
	if len(p.Validate()) > 0 {
		return out
	}

	if p.Enabled != nil {
		out.Enabled = *p.Enabled
	}
	if p.MinConfidence != nil {
		out.MinConfidence = *p.MinConfidence
	}
	if p.DailyCap != nil {
		out.DailyCap = *p.DailyCap
	}
	for _, raw := range p.Intents {
		if intent := parseIntent(raw); intent != "" {
			out.Intents[intent] = true
		}
	}
	for raw, content := range p.Contents {
		if intent := parseIntent(raw); intent != "" {
			out.Contents[intent] = content
		}
	}
	return out
}

// ResolveSettings reads the auto-reply policy out of an organisation's stored
// outreach settings. Empty settings resolve to disabled; unreadable ones
// return an error, which every caller treats as "hold for review".
func ResolveSettings(raw []byte) (Resolved, error) {
	if len(raw) == 0 {
		return Disabled(), nil
	}
	var settings struct {
		AutoReply *Policy `json:"auto_reply"`
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return Disabled(), err
	}
	if settings.AutoReply == nil {
		return Disabled(), nil
	}
	return settings.AutoReply.Resolve(), nil
}

// parseIntent accepts the canonical UPPER_SNAKE identifier and the display
// spelling, so a value stored by either the API or the UI resolves.
func parseIntent(raw string) domain.IntentType {
	norm := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(raw), "_", " "))
	for _, i := range domain.AllIntents() {
		if strings.ToLower(string(i)) == norm {
			return i
		}
	}
	return ""
}

// Decision is the answer, with the reason attached. The reason is recorded
// whichever way the decision goes, so a draft left for review can be explained
// without re-deriving the policy.
type Decision struct {
	Send   bool
	Reason string
}

func hold(format string, args ...any) Decision {
	return Decision{Send: false, Reason: fmt.Sprintf(format, args...)}
}

// Request is what the caller knows at the moment the draft is ready.
type Request struct {
	Intent     domain.IntentType
	Confidence float64

	// ContactOptedOut is the contact's stored do-not-contact flag. It is
	// checked here as well as at the intent level because the flag may have
	// been set by an earlier message, or by hand, rather than by the reply
	// being answered now.
	ContactOptedOut bool

	// SentToday is how many replies have already been auto-sent for this
	// account today.
	SentToday int

	// DraftEmpty guards the case where generation silently produced nothing.
	DraftEmpty bool
}

// Decide answers whether this reply may be sent without review.
//
// The checks run cheapest-and-most-absolute first, so the reason returned is
// the most fundamental one rather than whichever happened to be tested last.
func (r Resolved) Decide(req Request) Decision {
	if req.DraftEmpty {
		return hold("the generated reply was empty")
	}
	if req.ContactOptedOut {
		return hold("the contact is marked do-not-contact")
	}
	if why, banned := neverAutoSend[req.Intent]; banned {
		return hold("%s is never auto-sent: %s", canonical(req.Intent), why)
	}
	if !r.Enabled {
		return hold("auto-reply is off for this organisation")
	}
	if !r.Intents[req.Intent] {
		return hold("%s is not one of the intents enabled for auto-reply",
			canonical(req.Intent))
	}
	if req.Confidence < r.MinConfidence {
		return hold("classifier confidence %.2f is below the %.2f floor",
			req.Confidence, r.MinConfidence)
	}
	if req.SentToday >= r.DailyCap {
		return hold("the daily auto-reply cap of %d has been reached", r.DailyCap)
	}

	return Decision{
		Send: true,
		Reason: fmt.Sprintf("%s auto-reply enabled, confidence %.2f meets the %.2f floor",
			canonical(req.Intent), req.Confidence, r.MinConfidence),
	}
}

func canonical(i domain.IntentType) string {
	return strings.ReplaceAll(strings.ToUpper(string(i)), " ", "_")
}

// Describe renders the policy for the settings page and for the audit entry
// written when an administrator changes it.
//
// Every branch must return a predicate — something that reads correctly after
// "COSMO", which is how the settings page presents it. Two of these used to be
// complete sentences carrying their own subject, and the page rendered "COSMO
// every AI reply is held for review": a caller cannot repair that, because the
// grammar of the result is the callee's contract to keep.
func (r Resolved) Describe() string {
	if !r.Enabled {
		return "holds every AI reply for review"
	}
	if len(r.Intents) == 0 {
		return "has automatic replies switched on but no reply kind chosen, so it still holds every reply for review"
	}
	names := make([]string, 0, len(r.Intents))
	for i := range r.Intents {
		names = append(names, canonical(i))
	}
	sort.Strings(names)
	return fmt.Sprintf(
		"sends %s replies automatically above %.0f%% confidence, up to %d a day; everything else is held for review",
		strings.Join(names, ", "), r.MinConfidence*100, r.DailyCap)
}
