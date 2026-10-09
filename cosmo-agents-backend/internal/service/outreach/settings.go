package outreach

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/service/autoreply"
)

// OutreachSettings is the admin-editable half of Config.
//
// Every field is a pointer so "not set" is distinguishable from "set to zero".
// A missing field falls back to the service default, which is what lets an
// organisation change only the one number it cares about — usually the
// no-reply window — without pinning the rest of the cadence to whatever the
// defaults happened to be on the day it saved.
type OutreachSettings struct {
	NoReplyHours          *int `json:"no_reply_hours,omitempty"`
	FollowUp1MinDays      *int `json:"follow_up1_min_days,omitempty"`
	FollowUp1MaxDays      *int `json:"follow_up1_max_days,omitempty"`
	FollowUp2MinDays      *int `json:"follow_up2_min_days,omitempty"`
	FollowUp2MaxDays      *int `json:"follow_up2_max_days,omitempty"`
	MeetingConfirmMinDays *int `json:"meeting_confirm_min_days,omitempty"`
	ReEngageThresholdDays *int `json:"re_engage_threshold_days,omitempty"`
	MaxFollowups          *int `json:"max_followups,omitempty"`

	// AutoReply governs whether an AI-written reply may be sent without a
	// person reading it first. It lives beside the cadence because both are
	// "how this organisation runs outreach", but it is resolved separately:
	// Config stays a comparable value type, and a reply policy carries a set.
	AutoReply *autoreply.Policy `json:"auto_reply,omitempty"`

	// NextStepEngine turns on the next-step decision engine for the
	// organisation. Off (or unset) keeps the fixed cadence as the only source
	// of next steps, which is how outreach behaved before the engine existed.
	NextStepEngine *bool `json:"next_step_engine,omitempty"`
}

// bound describes the range a setting may take.
type bound struct {
	min, max int
	label    string
}

// Bounds keep a typo from quietly breaking outreach. A no-reply window of 0
// would fire a follow-up the instant the first mail is sent; one of 10,000
// hours would silently retire the campaign.
var bounds = map[string]bound{
	"no_reply_hours":           {1, 720, "No-reply window (hours)"},
	"follow_up1_min_days":      {0, 90, "Follow-up 1 earliest (days)"},
	"follow_up1_max_days":      {0, 90, "Follow-up 1 latest (days)"},
	"follow_up2_min_days":      {0, 90, "Follow-up 2 earliest (days)"},
	"follow_up2_max_days":      {0, 90, "Follow-up 2 latest (days)"},
	"meeting_confirm_min_days": {0, 30, "Meeting confirmation delay (days)"},
	"re_engage_threshold_days": {1, 730, "Re-engage after (days)"},
	"max_followups":            {0, 10, "Maximum follow-ups"},
}

// Validate reports every problem at once rather than the first, so an admin
// fixing a form is not sent back three times.
func (s OutreachSettings) Validate() []string {
	var problems []string

	check := func(key string, v *int) {
		if v == nil {
			return
		}
		b := bounds[key]
		if *v < b.min || *v > b.max {
			problems = append(problems,
				fmt.Sprintf("%s must be between %d and %d (got %d)",
					b.label, b.min, b.max, *v))
		}
	}

	check("no_reply_hours", s.NoReplyHours)
	check("follow_up1_min_days", s.FollowUp1MinDays)
	check("follow_up1_max_days", s.FollowUp1MaxDays)
	check("follow_up2_min_days", s.FollowUp2MinDays)
	check("follow_up2_max_days", s.FollowUp2MaxDays)
	check("meeting_confirm_min_days", s.MeetingConfirmMinDays)
	check("re_engage_threshold_days", s.ReEngageThresholdDays)
	check("max_followups", s.MaxFollowups)

	// A window whose start is after its end never opens, so the follow-up it
	// governs would never be sent.
	if s.FollowUp1MinDays != nil && s.FollowUp1MaxDays != nil &&
		*s.FollowUp1MinDays > *s.FollowUp1MaxDays {
		problems = append(problems,
			"Follow-up 1 earliest must not be later than its latest")
	}
	if s.FollowUp2MinDays != nil && s.FollowUp2MaxDays != nil &&
		*s.FollowUp2MinDays > *s.FollowUp2MaxDays {
		problems = append(problems,
			"Follow-up 2 earliest must not be later than its latest")
	}

	if s.AutoReply != nil {
		problems = append(problems, s.AutoReply.Validate()...)
	}

	return problems
}

// AutoReplyFrom resolves the stored auto-reply policy.
//
// Unreadable settings yield the disabled policy rather than an error. That is
// the only safe direction to fail in: a malformed row must never be the reason
// an email goes out unreviewed.
func AutoReplyFrom(raw []byte) autoreply.Resolved {
	if len(raw) == 0 {
		return autoreply.Disabled()
	}
	var s OutreachSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		return autoreply.Disabled()
	}
	if s.AutoReply == nil {
		return autoreply.Disabled()
	}
	return s.AutoReply.Resolve()
}

// NextStepEngineFrom reports whether the organisation has turned the
// next-step engine on. Unreadable settings read as off: the cadence is the
// behaviour the organisation had before it chose anything.
func NextStepEngineFrom(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	var s OutreachSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		return false
	}
	return s.NextStepEngine != nil && *s.NextStepEngine
}

// Apply overlays the settings onto a base config, leaving unset fields alone.
func (s OutreachSettings) Apply(base Config) Config {
	out := base
	set := func(dst *int, v *int) {
		if v != nil {
			*dst = *v
		}
	}
	set(&out.NoReplyHours, s.NoReplyHours)
	set(&out.FollowUp1MinDays, s.FollowUp1MinDays)
	set(&out.FollowUp1MaxDays, s.FollowUp1MaxDays)
	set(&out.FollowUp2MinDays, s.FollowUp2MinDays)
	set(&out.FollowUp2MaxDays, s.FollowUp2MaxDays)
	set(&out.MeetingConfirmMinDays, s.MeetingConfirmMinDays)
	set(&out.ReEngageThresholdDays, s.ReEngageThresholdDays)
	set(&out.MaxFollowups, s.MaxFollowups)
	return out
}

// ConfigFrom builds the effective config from stored JSON.
//
// Unreadable or absent settings yield the defaults rather than an error: a
// malformed row must not stop outreach from running, and the defaults are the
// behaviour the organisation had before it ever saved anything.
func ConfigFrom(raw []byte) Config {
	base := DefaultConfig()
	if len(raw) == 0 {
		return base
	}
	var s OutreachSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		return base
	}
	if len(s.Validate()) > 0 {
		return base
	}
	return s.Apply(base)
}

// Describe renders the effective cadence as a sentence, for the settings page
// and for the audit log entry written when an admin changes it.
func (c Config) Describe() string {
	return fmt.Sprintf(
		"waits %dh for a reply, follows up on days %d–%d then %d–%d, "+
			"at most %d follow-ups, re-engages after %d days",
		c.NoReplyHours,
		c.FollowUp1MinDays, c.FollowUp1MaxDays,
		c.FollowUp2MinDays, c.FollowUp2MaxDays,
		c.MaxFollowups, c.ReEngageThresholdDays,
	)
}

// --------------------------------------------------------------- lookup

// orgConfigLoader resolves a user to their organisation's cadence.
//
// The state machine consults the cadence several times per contact, and a
// recalculation sweep touches every contact in the account, so the settings
// are cached briefly rather than re-read per call. The TTL is short because
// an admin who changes the cadence expects it to take effect, not to wait out
// a long cache.
type orgConfigLoader struct {
	mu    sync.RWMutex
	cache map[uuid.UUID]cachedConfig
	load  func(ctx context.Context, userID uuid.UUID) ([]byte, error)
}

type cachedConfig struct {
	cfg Config
	at  time.Time
}

const orgConfigTTL = 30 * time.Second

func newOrgConfigLoader(
	load func(ctx context.Context, userID uuid.UUID) ([]byte, error),
) *orgConfigLoader {
	return &orgConfigLoader{cache: map[uuid.UUID]cachedConfig{}, load: load}
}

// Get returns the cadence for a user's organisation, falling back to the
// defaults whenever the lookup cannot answer. Outreach must keep running even
// if the settings row is unreachable.
func (l *orgConfigLoader) Get(ctx context.Context, userID uuid.UUID, base Config) Config {
	if l == nil || l.load == nil {
		return base
	}

	l.mu.RLock()
	hit, ok := l.cache[userID]
	l.mu.RUnlock()
	if ok && time.Since(hit.at) < orgConfigTTL {
		return hit.cfg
	}

	raw, err := l.load(ctx, userID)
	if err != nil {
		return base
	}
	cfg := ConfigFrom(raw)

	l.mu.Lock()
	l.cache[userID] = cachedConfig{cfg: cfg, at: time.Now()}
	l.mu.Unlock()
	return cfg
}

// Invalidate drops a cached entry so a save takes effect immediately.
func (l *orgConfigLoader) Invalidate(userID uuid.UUID) {
	if l == nil {
		return
	}
	l.mu.Lock()
	delete(l.cache, userID)
	l.mu.Unlock()
}
