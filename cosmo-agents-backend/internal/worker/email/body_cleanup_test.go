package email

import (
	"strings"
	"testing"
)

// The email a prospect actually received, reproduced verbatim: three blank
// lines between every paragraph and an unfilled sign-off.
const brokenDraft = "Hi Trần Anh,\n" +
	"\n\n\n" +
	"I hope you've been doing well.\n" +
	"\n\n\n" +
	"It's been a little while since we last connected.\n" +
	"\n\n\n" +
	"Best,\n" +
	"[Your Name]\n"

func TestPrepareEmailBody_FixesTheDraftThatShipped(t *testing.T) {
	filled := applyTemplateData(brokenDraft, map[string]string{
		"sender_name": "Nguyen Van A",
	})
	body, _ := prepareEmailBody(filled, false)

	if strings.Contains(body, "[Your Name]") {
		t.Fatalf("sign-off placeholder reached the prospect:\n%s", body)
	}
	if !strings.Contains(body, "Best,\nNguyen Van A") {
		t.Fatalf("expected the sign-off to carry the sender's name:\n%s", body)
	}
	if strings.Contains(body, "\n\n\n") {
		t.Fatalf("paragraphs still separated by more than one blank line:\n%s", body)
	}
	if !strings.Contains(body, "well.\n\nIt's been") {
		t.Fatalf("paragraphs should stay separated by exactly one blank line:\n%s", body)
	}
}

func TestFillNamePlaceholders_Variants(t *testing.T) {
	for _, ph := range []string{"[Your Name]", "[your name]", "[Tên bạn]", "[Tên của bạn]"} {
		got := fillNamePlaceholders("Best,\n"+ph, "Mai")
		if strings.Contains(got, "[") {
			t.Fatalf("%s was not replaced: %q", ph, got)
		}
	}
}

func TestFillNamePlaceholders_NoSenderNameLeavesItForStripping(t *testing.T) {
	// Without a name the placeholder must survive this step so the stripper
	// removes it; substituting an empty string here would hide the gap.
	got := fillNamePlaceholders("Best,\n[Your Name]", "")
	if !strings.Contains(got, "[Your Name]") {
		t.Fatalf("expected the placeholder to be left for the stripper, got %q", got)
	}

	body, _ := prepareEmailBody(got, false)
	if strings.Contains(body, "[Your Name]") {
		t.Fatalf("stripper should have removed it, got %q", body)
	}
}

func TestStripUnfilledPlaceholders_KeepsRealBracketedProse(t *testing.T) {
	// A long bracketed aside is content, not a merge field.
	in := "See the pricing sheet [attached to this message for your review]."
	if got := stripUnfilledPlaceholders(in); got != in {
		t.Fatalf("real prose in brackets was stripped:\n  before: %q\n  after:  %q", in, got)
	}
}

func TestCollapseBlankLines(t *testing.T) {
	cases := []struct{ in, want string }{
		{"a\n\n\n\nb", "a\n\nb"},
		{"a\nb", "a\nb"},
		{"\n\n\na\n\n\n", "a"},
		{"a\n   \n   \nb", "a\n\nb"},
	}
	for _, c := range cases {
		if got := collapseBlankLines(c.in); got != c.want {
			t.Fatalf("collapseBlankLines(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestPrepareEmailBody_MarkdownStillConvertsAfterCleanup(t *testing.T) {
	body, isHTML := prepareEmailBody("Hi **there**\n\n\n\nBye", false)
	if !isHTML {
		t.Fatal("markdown body should be sent as HTML")
	}
	if !strings.Contains(body, "<strong>there</strong>") {
		t.Fatalf("bold markup lost: %s", body)
	}
	if strings.Contains(body, "<br/><br/><br/>") {
		t.Fatalf("blank lines should be collapsed before the <br/> conversion: %s", body)
	}
}
