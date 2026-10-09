package intent

import (
	"strings"
	"testing"
)

// Wrong-recipient replies were read as a request to stop (3 of 3 in the paper
// set, 6 of 6 held out) and polite declines sometimes as opt-outs. The prompt
// now states both boundaries.
func TestClassifierPrompt_StatesTheOptOutAndWrongRecipientRules(t *testing.T) {
	p := (&IntentClassifier{}).constructPrompt()
	for _, want := range []string{
		"DO_NOT_CONTACT only when the sender explicitly asks to stop",
		"is NOT_INTERESTED",
		"Wrong recipient (misdirected email)",
		"classify UNKNOWN_INTENT so that a person can re-route it",
		"NOT REFERRAL\n       unless they name or point to a specific person or team",
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt is missing %q", want)
		}
	}
}
