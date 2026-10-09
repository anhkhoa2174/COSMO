package ai

import "testing"

// A model may or may not wrap structured output in a Markdown fence, and the
// same model does not always make the same choice. The reply body is taken
// straight from this parse, so a fence that is not stripped becomes the email.
func TestExtractReplyJSON_HandlesFencedAndBareOutput(t *testing.T) {
	want := struct{ subject, body string }{"Re: Pricing", "Growth is $99 per seat."}

	cases := map[string]string{
		"bare":                    `{"subject":"Re: Pricing","body":"Growth is $99 per seat."}`,
		"fenced with a language":  "```json\n{\"subject\":\"Re: Pricing\",\"body\":\"Growth is $99 per seat.\"}\n```",
		"fenced without one":      "```\n{\"subject\":\"Re: Pricing\",\"body\":\"Growth is $99 per seat.\"}\n```",
		"fenced with whitespace":  "  \n```json\n{\"subject\":\"Re: Pricing\",\"body\":\"Growth is $99 per seat.\"}\n```  \n",
		"prose then a raw object": "Here you go:\n{\"subject\":\"Re: Pricing\",\"body\":\"Growth is $99 per seat.\"}",
	}

	for name, in := range cases {
		subject, body := extractReplyJSON(in)
		if subject != want.subject || body != want.body {
			t.Errorf("%s: got (%q, %q)", name, subject, body)
		}
	}
}

func TestExtractReplyJSON_UnparseableYieldsNothing(t *testing.T) {
	// Returning empty is what lets the caller fall back to the raw text; a
	// half-parsed object would put a fragment of JSON in front of a prospect.
	for _, in := range []string{"", "not json at all", "```json\n{\"subject\": \n```"} {
		if s, b := extractReplyJSON(in); s != "" || b != "" {
			t.Errorf("%q gave (%q, %q)", in, s, b)
		}
	}
}

func TestStripCodeFence_LeavesUnfencedTextAlone(t *testing.T) {
	for _, in := range []string{"plain text", "{\"a\":1}", "a ``` in the middle"} {
		if got := stripCodeFence(in); got != in {
			t.Errorf("%q was altered to %q", in, got)
		}
	}
}
