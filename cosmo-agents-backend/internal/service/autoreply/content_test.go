package autoreply

import (
	"strings"
	"testing"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

func TestRenderFillsEveryMergeField(t *testing.T) {
	got := Render("Hi {{first_name}} ({{name}}), {{job_title}} at {{company}}. — {{sender_name}}",
		MergeValues{FirstName: "Rachel", Name: "Rachel Kim", Company: "Northwind",
			JobTitle: "COO", SenderName: "Alex"})
	want := "Hi Rachel (Rachel Kim), COO at Northwind. — Alex"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRenderToleratesSpacesInsideTheBraces(t *testing.T) {
	if got := Render("At {{ company }}.", MergeValues{Company: "Northwind"}); got != "At Northwind." {
		t.Fatalf("got %q", got)
	}
}

// A contact with no name must not receive "Hi ," — the empty field takes the
// space in front of it with it.
func TestRenderDropsAnEmptyFieldWithItsLeadingSpace(t *testing.T) {
	if got := Render("Hi {{first_name}}, thanks.", MergeValues{}); got != "Hi, thanks." {
		t.Fatalf("got %q", got)
	}
}

// Validation refuses unknown fields at save time, but if one ever reaches a
// send it must not arrive in a prospect's inbox as literal braces.
func TestRenderNeverLeavesBracesBehind(t *testing.T) {
	got := Render("Hi {{nickname}}!", MergeValues{FirstName: "Rachel"})
	if strings.Contains(got, "{{") || got != "Hi!" {
		t.Fatalf("got %q", got)
	}
}

func TestNewMergeValuesTreatsTheColumnDefaultAsMissing(t *testing.T) {
	v := NewMergeValues("N/A", "N/A", "", "Alex")
	if v.FirstName != "" || v.Name != "" || v.Company != "" || v.SenderName != "Alex" {
		t.Fatalf("placeholder values leaked through: %+v", v)
	}
	v = NewMergeValues("  Rachel   Kim ", "Northwind", "COO", "")
	if v.FirstName != "Rachel" || v.Name != "Rachel Kim" {
		t.Fatalf("name not split: %+v", v)
	}
}

func TestContentValidation(t *testing.T) {
	long := func(n int) string { return strings.Repeat("a", n) }
	cases := []struct {
		name    string
		content Content
		wantErr string
	}{
		{"template ok", Content{Mode: ModeTemplate, Body: "Hi {{first_name}}"}, ""},
		{"ai ok", Content{Mode: ModeAI, Guidance: "Mention the 14-day trial."}, ""},
		{"template needs a body", Content{Mode: ModeTemplate, Body: "   "}, "needs a message"},
		{"ai needs guidance", Content{Mode: ModeAI}, "needs guidance"},
		{"unknown mode", Content{Mode: "magic", Body: "x"}, "mode"},
		{"unknown merge field in body", Content{Mode: ModeTemplate, Body: "Hi {{frist_name}}"}, "{{frist_name}}"},
		{"unknown merge field in subject", Content{Mode: ModeTemplate, Subject: "{{deal}}", Body: "x"}, "{{deal}}"},
		{"subject too long", Content{Mode: ModeTemplate, Subject: long(maxSubjectLen + 1), Body: "x"}, "subject"},
		{"body too long", Content{Mode: ModeTemplate, Body: long(maxBodyLen + 1)}, "message"},
		{"guidance too long", Content{Mode: ModeAI, Guidance: long(maxGuidanceLen + 1)}, "guidance"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			problems := tc.content.validate("OUT_OF_OFFICE")
			if tc.wantErr == "" {
				if len(problems) != 0 {
					t.Fatalf("unexpected problems: %v", problems)
				}
				return
			}
			joined := strings.Join(problems, "; ")
			if !strings.Contains(joined, tc.wantErr) {
				t.Fatalf("problems %q do not mention %q", joined, tc.wantErr)
			}
		})
	}
}

func TestContentsAreRefusedForBannedAndUnknownIntents(t *testing.T) {
	p := Policy{Contents: map[string]Content{
		string(domain.IntentDoNotContact): {Mode: ModeTemplate, Body: "Sorry to see you go"},
		"CLOSED_WON":                      {Mode: ModeTemplate, Body: "x"},
	}}
	joined := strings.Join(p.Validate(), "; ")
	if !strings.Contains(joined, "DO NOT CONTACT") && !strings.Contains(joined, "DO_NOT_CONTACT") {
		t.Fatalf("banned intent content accepted: %q", joined)
	}
	if !strings.Contains(joined, "CLOSED_WON") {
		t.Fatalf("unknown intent content accepted: %q", joined)
	}
}

func TestContentsResolveUnderEitherSpelling(t *testing.T) {
	r := Policy{
		Enabled: b(true),
		Intents: []string{"OUT_OF_OFFICE"},
		Contents: map[string]Content{
			"OUT_OF_OFFICE": {Mode: ModeTemplate, Body: "Thanks, talk soon."},
		},
	}.Resolve()
	c, ok := r.ContentFor(domain.IntentOutOfOffice)
	if !ok || c.Body != "Thanks, talk soon." {
		t.Fatalf("content not resolved: %+v %v", c, ok)
	}
}

// Content is part of auto-reply: with the feature off, or the intent not
// selected, COSMO behaves exactly as it did before the content existed.
func TestContentOnlyAppliesToEnabledSelectedIntents(t *testing.T) {
	contents := map[string]Content{
		string(domain.IntentOutOfOffice): {Mode: ModeTemplate, Body: "x"},
	}
	off := Policy{Enabled: b(false), Intents: []string{string(domain.IntentOutOfOffice)}, Contents: contents}.Resolve()
	if _, ok := off.ContentFor(domain.IntentOutOfOffice); ok {
		t.Fatal("content applied with auto-reply off")
	}
	unselected := Policy{Enabled: b(true), Intents: []string{string(domain.IntentInterested)}, Contents: contents}.Resolve()
	if _, ok := unselected.ContentFor(domain.IntentOutOfOffice); ok {
		t.Fatal("content applied to an intent that is not selected")
	}
}

func TestGuidanceIsOnlyReturnedForAIMode(t *testing.T) {
	r := Policy{
		Enabled: b(true),
		Intents: []string{string(domain.IntentRequestForPricing), string(domain.IntentOutOfOffice)},
		Contents: map[string]Content{
			string(domain.IntentRequestForPricing): {Mode: ModeAI, Guidance: "Always mention the free trial."},
			string(domain.IntentOutOfOffice):       {Mode: ModeTemplate, Body: "x"},
		},
	}.Resolve()
	if g := r.GuidanceFor(domain.IntentRequestForPricing); g != "Always mention the free trial." {
		t.Fatalf("guidance = %q", g)
	}
	if g := r.GuidanceFor(domain.IntentOutOfOffice); g != "" {
		t.Fatalf("template intent returned guidance %q", g)
	}
	if g := r.GuidanceFor(domain.IntentInterested); g != "" {
		t.Fatalf("unconfigured intent returned guidance %q", g)
	}
}

func TestResolveSettingsReadsTheStoredOrganisationRow(t *testing.T) {
	raw := []byte(`{"no_reply_hours":8,"auto_reply":{"enabled":true,"intents":["Out of office"],
		"contents":{"Out of office":{"mode":"template","body":"Back soon"}}}}`)
	r, err := ResolveSettings(raw)
	if err != nil {
		t.Fatal(err)
	}
	if c, ok := r.ContentFor(domain.IntentOutOfOffice); !ok || c.Body != "Back soon" {
		t.Fatalf("content lost: %+v", r)
	}
	if r, err := ResolveSettings(nil); err != nil || r.Enabled {
		t.Fatalf("empty settings must resolve to disabled, got %+v %v", r, err)
	}
	if _, err := ResolveSettings([]byte("{not json")); err == nil {
		t.Fatal("malformed settings must report an error")
	}
}
