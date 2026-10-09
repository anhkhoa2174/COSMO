package service

import "testing"

// The cleaner used [^[:print:]], which is ASCII-only in Go: every Vietnamese
// letter with a diacritic and every PDF ligature was deleted before the text
// reached the knowledge base.
func TestCleanTextKeepsEveryScript(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"vietnamese is kept", "Giá gói Scale là $249 một tháng", "Giá gói Scale là $249 một tháng"},
		{"pdf ligatures are split into letters", "the ﬁrst quarter, 30% oﬀ, ﬂow", "the first quarter, 30% off, flow"},
		{"typographic punctuation is kept", "“Quote” — dash … ellipsis", "“Quote” — dash ... ellipsis"},
		{"line breaks and tabs are kept", "a\nb\tc\r\n", "a\nb\tc\r\n"},
		{"control characters are dropped", "a\x00b\x07c\x1b", "abc"},
		{"undecodable bytes are dropped", "ok\xffok", "okok"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanText(tt.in); got != tt.want {
				t.Fatalf("cleanText(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExtractTextPlainKeepsVietnamese(t *testing.T) {
	got, err := extractText([]byte("Bảng giá: gói Growth 99$/người/tháng"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if want := "Bảng giá: gói Growth 99$/người/tháng"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
