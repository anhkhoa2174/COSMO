package outreach

import (
	"encoding/json"
	"testing"
)

// TestConfigSerialisesWithTheKeysTheSettingsPageReads guards a contract that
// spans two repositories and so had nothing checking it.
//
// Config is returned as the `effective` and `defaults` objects of the outreach
// settings endpoint. The settings page indexes both by the same snake_case keys
// it posts back in OutreachSettings. Config carried no json tags, so Go emitted
// "FollowUp1MinDays" while the page looked for "follow_up1_min_days": every
// lookup returned undefined, every timing input rendered NaN, and the form
// refused to save on fields the administrator had never touched — including
// saves that only changed the auto-reply policy.
//
// Nothing about that is visible from either side alone. The Go type was
// reasonable, the TypeScript interface was reasonable, and they disagreed.
func TestConfigSerialisesWithTheKeysTheSettingsPageReads(t *testing.T) {
	raw, err := json.Marshal(DefaultConfig())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Exactly the keys src/network/client/outreach-settings.ts declares on
	// EffectiveOutreachConfig.
	want := []string{
		"no_reply_hours",
		"follow_up1_min_days",
		"follow_up1_max_days",
		"follow_up2_min_days",
		"follow_up2_max_days",
		"meeting_confirm_min_days",
		"re_engage_threshold_days",
		"max_followups",
	}
	for _, k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("missing %q; the settings page reads that key. Got: %v", k, keysOf(got))
		}
	}
	if len(got) != len(want) {
		t.Errorf("unexpected keys: got %v, want exactly %v", keysOf(got), want)
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
