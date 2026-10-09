package outreach

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	contact "github.com/rockship/cosmo-agents-go/internal/domain/contact"
)

func jsonb(t *testing.T, v interface{}) base.JSONB {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return base.JSONB(b)
}

func TestBuildContactFactsBlock_MarksConfirmedAndSuspected(t *testing.T) {
	c := &contact.Contact{
		ConfirmedFacts: jsonb(t, map[string]interface{}{
			"pain_points": []interface{}{
				map[string]interface{}{"pain_point": "Manual CRM data entry", "confirmed_at": "2026-08-01T00:00:00Z"},
			},
		}),
		AIInsights: jsonb(t, map[string]interface{}{
			"suspected_goals": []interface{}{
				map[string]interface{}{"goal": "Cut ramp time for new reps"},
			},
			"buying_signals": []interface{}{
				map[string]interface{}{"signal": "Visited pricing page twice"},
			},
		}),
	}

	block := buildContactFactsBlock(c, "en")

	if !strings.Contains(block, "[VERIFIED] Pain point: Manual CRM data entry") {
		t.Fatalf("confirmed fact missing or unlabelled:\n%s", block)
	}
	if !strings.Contains(block, "[UNVERIFIED] Goal: Cut ramp time for new reps") {
		t.Fatalf("suspected goal missing or unlabelled:\n%s", block)
	}
	if !strings.Contains(block, "[UNVERIFIED] Buying signal: Visited pricing page twice") {
		t.Fatalf("buying signal missing:\n%s", block)
	}
	if !strings.Contains(block, "<contact_facts>") || !strings.Contains(block, "</contact_facts>") {
		t.Fatalf("facts must sit in a delimited block:\n%s", block)
	}
	if !strings.Contains(block, "SECURITY:") {
		t.Fatalf("untrusted-data warning missing:\n%s", block)
	}
}

func TestBuildContactFactsBlock_EmptyWhenNothingKnown(t *testing.T) {
	if got := buildContactFactsBlock(&contact.Contact{}, "en"); got != "" {
		t.Fatalf("a contact with no facts must add nothing to the prompt, got %q", got)
	}
	if got := buildContactFactsBlock(nil, "en"); got != "" {
		t.Fatalf("nil contact must add nothing, got %q", got)
	}
}

func TestBuildContactFactsBlock_CapsPerCategory(t *testing.T) {
	entries := make([]interface{}, 0, 6)
	for _, text := range []string{"p1", "p2", "p3", "p4", "p5", "p6"} {
		entries = append(entries, map[string]interface{}{"pain_point": text})
	}
	c := &contact.Contact{
		AIInsights: jsonb(t, map[string]interface{}{"suspected_pain_points": entries}),
	}

	block := buildContactFactsBlock(c, "en")

	if n := strings.Count(block, "Pain point:"); n != maxFactsPerCategory {
		t.Fatalf("expected %d pain points in the prompt, got %d:\n%s", maxFactsPerCategory, n, block)
	}
	// Newest entries win.
	if !strings.Contains(block, "p6") || strings.Contains(block, "p1") {
		t.Fatalf("expected the newest entries to be kept:\n%s", block)
	}
}

func TestBuildContactFactsBlock_VietnameseNote(t *testing.T) {
	c := &contact.Contact{
		ConfirmedFacts: jsonb(t, map[string]interface{}{
			"goals": []interface{}{map[string]interface{}{"goal": "Tăng doanh số quý 4"}},
		}),
	}

	block := buildContactFactsBlock(c, "vi")

	if !strings.Contains(block, "NHỮNG GÌ ĐÃ BIẾT") {
		t.Fatalf("expected the Vietnamese header:\n%s", block)
	}
	if !strings.Contains(block, "Tăng doanh số quý 4") {
		t.Fatalf("expected the confirmed goal:\n%s", block)
	}
}

func TestExtractFactTexts_ToleratesPlainStrings(t *testing.T) {
	got := extractFactTexts([]interface{}{"older", "newer"}, "pain_point")
	if len(got) != 2 || got[0] != "newer" {
		t.Fatalf("expected newest-first plain strings, got %v", got)
	}
	if extractFactTexts("not a list", "pain_point") != nil {
		t.Fatal("a non-list value must yield no facts")
	}
}

func TestBuildFullContextPrompt_MarksCompanyAsTheirs(t *testing.T) {
	// A prospect at Microsoft once received "At Microsoft, we've been actively
	// enhancing our offerings" — the model had only one company name in the
	// prompt and assumed it was its own.
	s := &Service{}
	ctx := &DraftContext{
		Contact:  &contact.Contact{Name: "Trần Anh", Company: "Microsoft", JobTitle: "Software Engineer"},
		State:    &ConversationStateResult{},
		UserName: "Nguyen Van A",
	}

	for _, lang := range []string{"en", "vi"} {
		prompt := s.buildFullContextPromptWithLanguage(ctx, nil, lang)
		if !strings.Contains(prompt, "Microsoft") {
			t.Fatalf("[%s] the company should still be in the prompt", lang)
		}
		if !strings.Contains(prompt, "NƠI HỌ LÀM VIỆC") && !strings.Contains(prompt, "where THEY work") {
			t.Fatalf("[%s] the company must be labelled as the contact's employer:\n%s", lang, prompt)
		}
		if !strings.Contains(prompt, "KHÔNG làm ở Microsoft") && !strings.Contains(prompt, "do NOT work at Microsoft") {
			t.Fatalf("[%s] the prompt must state the sender does not work there:\n%s", lang, prompt)
		}
	}
}
