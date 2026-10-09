package autoreply

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// What an auto-reply says, as opposed to whether it may be sent.
//
// An administrator who hands a kind of reply to the system usually has an
// opinion about its wording: an out-of-office acknowledgement is the same two
// sentences every time, and a pricing answer should always mention the trial.
// Each selected intent can therefore carry one of two kinds of content:
//
//   - a fixed message, sent as written after its merge fields are filled in.
//     This is also the only way to answer intents that never get an AI draft
//     — out-of-office, nurture, referral — automatically.
//   - guidance for the AI, added to the prompt that writes the draft, so the
//     reply stays personal but follows the organisation's rules.
//
// Content never changes whether a reply may be sent. The banned intents, the
// confidence floor, the daily cap and the opt-out flag are all decided before
// and independently of it.

// ContentMode says how an intent's reply is produced.
type ContentMode string

const (
	ModeTemplate ContentMode = "template"
	ModeAI       ContentMode = "ai"
)

// Content is the admin-written part of one intent's reply.
type Content struct {
	Mode ContentMode `json:"mode"`

	// Template mode. An empty subject keeps the default "Re: <their subject>",
	// which is what keeps the reply in the prospect's thread.
	Subject string `json:"subject,omitempty"`
	Body    string `json:"body,omitempty"`

	// AI mode.
	Guidance string `json:"guidance,omitempty"`
}

const (
	maxSubjectLen  = 200
	maxBodyLen     = 5000
	maxGuidanceLen = 1000
)

// MergeTags lists the fields a fixed message may use. It is served to the
// settings page so the editor offers exactly what the renderer fills in.
var MergeTags = []string{"first_name", "name", "company", "job_title", "sender_name"}

// MergeValues are the values those fields are filled with.
type MergeValues struct {
	FirstName  string
	Name       string
	Company    string
	JobTitle   string
	SenderName string
}

// NewMergeValues builds the values from a contact and the sending agent.
//
// The contacts table defaults name, company and job title to "N/A". That is a
// storage placeholder, not a value, and "Hi N/A" is worse in a prospect's
// inbox than "Hi" — so it is treated as missing.
func NewMergeValues(name, company, jobTitle, senderName string) MergeValues {
	clean := func(s string) string {
		s = strings.Join(strings.Fields(s), " ")
		if strings.EqualFold(s, "N/A") {
			return ""
		}
		return s
	}
	v := MergeValues{
		Name:       clean(name),
		Company:    clean(company),
		JobTitle:   clean(jobTitle),
		SenderName: clean(senderName),
	}
	if parts := strings.Fields(v.Name); len(parts) > 0 {
		v.FirstName = parts[0]
	}
	return v
}

func (v MergeValues) lookup(tag string) string {
	switch tag {
	case "first_name":
		return v.FirstName
	case "name":
		return v.Name
	case "company":
		return v.Company
	case "job_title":
		return v.JobTitle
	case "sender_name":
		return v.SenderName
	}
	return ""
}

var mergeTag = regexp.MustCompile(`( ?)\{\{\s*([A-Za-z_]+)\s*\}\}`)

// Render fills a fixed message's merge fields.
//
// A field with no value is removed together with the space in front of it, so
// "Hi {{first_name}}," becomes "Hi," rather than "Hi ,". An unknown field is
// treated the same way: validation refuses them at save time, and a message
// that somehow still carries one must not reach a prospect as literal braces.
func Render(text string, v MergeValues) string {
	return mergeTag.ReplaceAllStringFunc(text, func(match string) string {
		parts := mergeTag.FindStringSubmatch(match)
		value := v.lookup(strings.ToLower(parts[2]))
		if value == "" {
			return ""
		}
		return parts[1] + value
	})
}

func isMergeTag(tag string) bool {
	for _, t := range MergeTags {
		if t == tag {
			return true
		}
	}
	return false
}

// validate checks one intent's content. label names the intent in messages.
func (c Content) validate(label string) []string {
	var problems []string
	tooLong := func(field, value string, max int) {
		if n := utf8.RuneCountInString(value); n > max {
			problems = append(problems, fmt.Sprintf(
				"%s: the %s is %d characters; the limit is %d", label, field, n, max))
		}
	}
	unknownTags := func(value string) {
		for _, m := range mergeTag.FindAllStringSubmatch(value, -1) {
			if tag := strings.ToLower(m[2]); !isMergeTag(tag) {
				problems = append(problems, fmt.Sprintf(
					"%s: {{%s}} is not a merge field; use one of {{%s}}",
					label, m[2], strings.Join(MergeTags, "}}, {{")))
			}
		}
	}

	switch c.Mode {
	case ModeTemplate:
		if strings.TrimSpace(c.Body) == "" {
			problems = append(problems, fmt.Sprintf(
				"%s: a fixed reply needs a message to send", label))
		}
		tooLong("subject", c.Subject, maxSubjectLen)
		tooLong("message", c.Body, maxBodyLen)
		unknownTags(c.Subject)
		unknownTags(c.Body)
	case ModeAI:
		if strings.TrimSpace(c.Guidance) == "" {
			problems = append(problems, fmt.Sprintf(
				"%s: an AI-written reply needs guidance; remove the entry to use the default draft", label))
		}
		tooLong("guidance", c.Guidance, maxGuidanceLen)
	default:
		problems = append(problems, fmt.Sprintf(
			"%s: reply mode must be %q or %q (got %q)", label, ModeTemplate, ModeAI, c.Mode))
	}
	return problems
}

// ContentFor returns the content that applies to an intent right now. Content
// only applies while auto-reply is on and the intent is selected, so switching
// either off returns COSMO to the behaviour it had before content existed.
func (r Resolved) ContentFor(intent domain.IntentType) (Content, bool) {
	if !r.Enabled || !r.Intents[intent] {
		return Content{}, false
	}
	c, ok := r.Contents[intent]
	return c, ok
}

// GuidanceFor returns the administrator's instructions for AI-written replies
// to an intent, or "" when there are none.
func (r Resolved) GuidanceFor(intent domain.IntentType) string {
	c, ok := r.ContentFor(intent)
	if !ok || c.Mode != ModeAI {
		return ""
	}
	return strings.TrimSpace(c.Guidance)
}
