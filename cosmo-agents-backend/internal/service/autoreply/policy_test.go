package autoreply

import (
	"strings"
	"testing"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

func b(v bool) *bool       { return &v }
func f(v float64) *float64 { return &v }
func i(v int) *int         { return &v }

// A permissive policy, used as the starting point for the tests that check
// what still gets refused even when everything is switched on.
func allOn(intents ...domain.IntentType) Resolved {
	names := make([]string, len(intents))
	for k, in := range intents {
		names[k] = string(in)
	}
	return Policy{
		Enabled:       b(true),
		Intents:       names,
		MinConfidence: f(0.5),
		DailyCap:      i(100),
	}.Resolve()
}

func TestDisabledByDefault(t *testing.T) {
	// The default has to be "hold for review". An organisation that never
	// opens the settings page must never have an email sent on its behalf.
	d := Disabled().Decide(Request{
		Intent:     domain.IntentInterested,
		Confidence: 1.0,
	})
	if d.Send {
		t.Fatal("auto-reply must be off until switched on")
	}
	if !strings.Contains(d.Reason, "off") {
		t.Fatalf("the reason should say it is off, got %q", d.Reason)
	}
}

func TestEmptyPolicyResolvesToDisabled(t *testing.T) {
	if (Policy{}).Resolve().Enabled {
		t.Fatal("an unset policy is a disabled policy")
	}
}

func TestEnabledWithNoIntentsSendsNothing(t *testing.T) {
	// Half-finished configuration is common: the switch gets flipped before
	// anyone decides which replies to hand over. That must send nothing.
	r := Policy{Enabled: b(true)}.Resolve()
	for _, in := range Selectable() {
		if r.Decide(Request{Intent: in, Confidence: 1.0}).Send {
			t.Fatalf("%s was auto-sent with no intent enabled", in)
		}
	}
}

func TestOnlyNamedIntentsAreSent(t *testing.T) {
	r := allOn(domain.IntentOutOfOffice)

	if !r.Decide(Request{Intent: domain.IntentOutOfOffice, Confidence: 0.95}).Send {
		t.Fatal("the enabled intent should be sent")
	}
	d := r.Decide(Request{Intent: domain.IntentInterested, Confidence: 0.99})
	if d.Send {
		t.Fatal("an intent that was not enabled must not be sent")
	}
	if !strings.Contains(d.Reason, "not one of the intents") {
		t.Fatalf("unhelpful reason: %q", d.Reason)
	}
}

func TestBannedIntentsAreNeverSent(t *testing.T) {
	// This is the test that matters most. Even with the intent explicitly
	// named, the switch on, no cap and a floor of zero, these must not go out.
	for banned, why := range neverAutoSend {
		r := allOn(banned)
		d := r.Decide(Request{Intent: banned, Confidence: 1.0})
		if d.Send {
			t.Fatalf("%s was auto-sent despite being banned (%s)", banned, why)
		}
		if !strings.Contains(d.Reason, "never auto-sent") {
			t.Fatalf("%s refused for the wrong reason: %q", banned, d.Reason)
		}
	}
}

func TestBannedIntentsAreRejectedAtSaveTime(t *testing.T) {
	// Refusing at decision time is the safety net. Refusing at save time is
	// what tells the admin their choice will not take effect, instead of
	// letting them believe it was stored.
	problems := Policy{
		Enabled: b(true),
		Intents: []string{string(domain.IntentDoNotContact)},
	}.Validate()

	if len(problems) == 0 {
		t.Fatal("saving a banned intent must be rejected")
	}
	if !strings.Contains(strings.ToLower(problems[0]), "never") {
		t.Fatalf("the message should explain it can never happen: %q", problems[0])
	}
}

func TestSelectableExcludesTheBannedIntents(t *testing.T) {
	// The settings UI is built from this list, so a banned intent must not be
	// offered in the first place.
	for _, in := range Selectable() {
		if _, banned := neverAutoSend[in]; banned {
			t.Fatalf("%s must not be offered to administrators", in)
		}
	}
	if len(Selectable()) != len(domain.AllIntents())-len(neverAutoSend) {
		t.Fatalf("expected %d selectable intents, got %d",
			len(domain.AllIntents())-len(neverAutoSend), len(Selectable()))
	}
}

func TestConfidenceFloorHolds(t *testing.T) {
	r := allOn(domain.IntentRequestForPricing)
	r.MinConfidence = 0.9

	if r.Decide(Request{Intent: domain.IntentRequestForPricing, Confidence: 0.89}).Send {
		t.Fatal("below the floor must be held")
	}
	if !r.Decide(Request{Intent: domain.IntentRequestForPricing, Confidence: 0.9}).Send {
		t.Fatal("exactly at the floor is good enough")
	}
}

func TestUnreportedConfidenceIsTreatedAsNoConfidence(t *testing.T) {
	// The classifier returns zero when it omits a confidence value. That must
	// read as "unknown", which fails the floor, rather than slipping through.
	r := allOn(domain.IntentRequestForPricing)
	r.MinConfidence = 0.9

	if r.Decide(Request{Intent: domain.IntentRequestForPricing, Confidence: 0}).Send {
		t.Fatal("a missing confidence must not pass the floor")
	}
}

func TestDailyCapStopsSending(t *testing.T) {
	r := allOn(domain.IntentOutOfOffice)
	r.DailyCap = 3

	req := Request{Intent: domain.IntentOutOfOffice, Confidence: 1.0}
	req.SentToday = 2
	if !r.Decide(req).Send {
		t.Fatal("under the cap should send")
	}
	req.SentToday = 3
	d := r.Decide(req)
	if d.Send {
		t.Fatal("at the cap should stop")
	}
	if !strings.Contains(d.Reason, "cap") {
		t.Fatalf("the reason should name the cap: %q", d.Reason)
	}
}

func TestOptedOutContactIsNeverAnswered(t *testing.T) {
	// The intent of this particular reply may be harmless, but the contact
	// asked to be left alone at some earlier point. That flag outranks it.
	r := allOn(domain.IntentInterested)
	d := r.Decide(Request{
		Intent:          domain.IntentInterested,
		Confidence:      1.0,
		ContactOptedOut: true,
	})
	if d.Send {
		t.Fatal("a do-not-contact contact must never be auto-answered")
	}
	if !strings.Contains(d.Reason, "do-not-contact") {
		t.Fatalf("unhelpful reason: %q", d.Reason)
	}
}

func TestEmptyDraftIsNeverSent(t *testing.T) {
	r := allOn(domain.IntentInterested)
	if r.Decide(Request{
		Intent: domain.IntentInterested, Confidence: 1.0, DraftEmpty: true,
	}).Send {
		t.Fatal("an empty body must not be sent")
	}
}

func TestValidateReportsEveryProblemAtOnce(t *testing.T) {
	problems := Policy{
		MinConfidence: f(0.1),  // below the allowed floor
		DailyCap:      i(9999), // above the allowed cap
		Intents:       []string{"NOT_AN_INTENT"},
	}.Validate()

	if len(problems) != 3 {
		t.Fatalf("expected all three problems together, got %v", problems)
	}
}

func TestInvalidPolicyResolvesToDisabledRatherThanPartlyApplied(t *testing.T) {
	// A stored policy that fails validation means a bad write got through
	// somewhere. Falling back to "hold everything" is the only safe reading —
	// applying the half that parsed could enable sending the admin never
	// confirmed.
	r := Policy{
		Enabled:       b(true),
		Intents:       []string{string(domain.IntentInterested)},
		MinConfidence: f(0.01),
	}.Resolve()

	if r.Enabled {
		t.Fatal("an invalid policy must resolve to disabled")
	}
}

func TestIntentNamesAcceptBothSpellings(t *testing.T) {
	// The API stores canonical UPPER_SNAKE, the UI sends the display form.
	// Both have to resolve, or a policy saved by one is ignored by the other.
	for _, raw := range []string{"OUT_OF_OFFICE", "Out of office", "out of office", " OUT_OF_OFFICE "} {
		r := Policy{Enabled: b(true), Intents: []string{raw}}.Resolve()
		if !r.Intents[domain.IntentOutOfOffice] {
			t.Fatalf("%q did not resolve to the out-of-office intent", raw)
		}
	}
}

func TestDescribeTellsAnAdminWhatWillHappen(t *testing.T) {
	if got := Disabled().Describe(); !strings.Contains(got, "review") {
		t.Fatalf("a disabled policy should say replies are reviewed: %q", got)
	}

	// On, but nothing selected: the summary must not imply anything is sent.
	got := Policy{Enabled: b(true)}.Resolve().Describe()
	if !strings.Contains(got, "no reply kind chosen") {
		t.Fatalf("should call out the empty selection: %q", got)
	}

	r := allOn(domain.IntentOutOfOffice)
	r.MinConfidence = 0.9
	r.DailyCap = 20
	got = r.Describe()
	for _, want := range []string{"OUT_OF_OFFICE", "90%", "20"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary missing %q: %q", want, got)
		}
	}
}

func TestOverridingOneFieldLeavesTheOthersOnDefaults(t *testing.T) {
	r := Policy{Enabled: b(true), DailyCap: i(5),
		Intents: []string{string(domain.IntentOutOfOffice)}}.Resolve()

	if r.DailyCap != 5 {
		t.Fatalf("cap not applied: %d", r.DailyCap)
	}
	if r.MinConfidence != DefaultMinConfidence {
		t.Fatalf("changing the cap must not move the confidence floor: %v",
			r.MinConfidence)
	}
}

// TestDescribeReadsAsAPredicate guards the contract the settings page depends
// on: it renders "COSMO " + Describe(), so every branch has to be a verb phrase
// agreeing with that subject.
//
// Two branches once returned complete sentences instead — "every AI reply is
// held for review" — and the page displayed "COSMO every AI reply is held for
// review". Asserting the wording would have missed it, because each string was
// perfectly good English on its own; what was wrong was the shape. A
// third-person singular verb ends in "s", and none of "every" or "auto-reply"
// does, so that is what this checks.
func TestDescribeReadsAsAPredicate(t *testing.T) {
	on := allOn(domain.IntentOutOfOffice)
	for _, r := range []Resolved{
		Disabled(),
		Policy{Enabled: b(true)}.Resolve(),
		on,
	} {
		got := r.Describe()
		first, _, _ := strings.Cut(got, " ")
		if !strings.HasSuffix(first, "s") {
			t.Fatalf("%q does not read after \"COSMO\": first word %q is not a verb",
				got, first)
		}
	}
}
