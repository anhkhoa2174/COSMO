package intent

import (
	"regexp"
	"strings"
)

// draftNamePlaceholders are the sign-off stand-ins a model writes instead of
// the sender's name.
var draftNamePlaceholders = []string{
	"[Your Name]", "[your name]", "[YOUR NAME]", "[Your name]", "[Your Full Name]",
	"[Sender Name]", "[sender name]", "[Name]",
	"[Tên bạn]", "[tên bạn]", "[Tên Bạn]", "[Tên của bạn]", "[tên của bạn]", "[Tên người gửi]",
}

// draftLeftoverPlaceholder matches a short bracketed stand-in such as
// "[Your Position]" or "[Company]": at most three words, so a bracketed
// phrase of prose or a markdown link label is left alone.
var draftLeftoverPlaceholder = regexp.MustCompile(`\[[\p{L}\p{N}._/'-]+(?: [\p{L}\p{N}._/'-]+){0,2}\]`)

// tidyDraftPlaceholders fills the sender's name into a draft's sign-off and
// removes the placeholders no data can fill. A line that held nothing but a
// placeholder goes with it, so the sign-off closes up instead of leaving a
// hole. With no name available the name placeholder is removed too: an
// unsigned draft reads better than one signed "[Your Name]".
func tidyDraftPlaceholders(draft, senderName string) string {
	if !strings.Contains(draft, "[") {
		return draft
	}
	lines := strings.Split(draft, "\n")
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		if senderName != "" {
			for _, ph := range draftNamePlaceholders {
				l = strings.ReplaceAll(l, ph, senderName)
			}
		}
		cleaned := strings.TrimRight(draftLeftoverPlaceholder.ReplaceAllString(l, ""), " \t")
		if cleaned == "" && strings.TrimSpace(l) != "" {
			continue // the line was only a placeholder
		}
		out = append(out, cleaned)
	}
	return strings.Join(out, "\n")
}
