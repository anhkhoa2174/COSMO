package outreach

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func ptr(v int) *int { return &v }

func TestConfigFrom_EmptyKeepsDefaults(t *testing.T) {
	// An organisation that never opened the settings page must behave exactly
	// as it did before the column existed.
	got := ConfigFrom(nil)
	if got != DefaultConfig() {
		t.Fatalf("no settings should mean defaults, got %+v", got)
	}
	if got := ConfigFrom([]byte("{}")); got != DefaultConfig() {
		t.Fatalf("empty settings should mean defaults, got %+v", got)
	}
}

func TestConfigFrom_OverridesOnlyWhatIsSet(t *testing.T) {
	raw, _ := json.Marshal(OutreachSettings{NoReplyHours: ptr(24)})
	got := ConfigFrom(raw)

	if got.NoReplyHours != 24 {
		t.Fatalf("no-reply window not applied: %d", got.NoReplyHours)
	}
	def := DefaultConfig()
	if got.MaxFollowups != def.MaxFollowups ||
		got.FollowUp1MinDays != def.FollowUp1MinDays ||
		got.ReEngageThresholdDays != def.ReEngageThresholdDays {
		t.Fatalf("changing one field must not pin the others: %+v", got)
	}
}

func TestConfigFrom_MalformedFallsBackInsteadOfBreakingOutreach(t *testing.T) {
	for _, raw := range []string{`{`, `[]`, `{"no_reply_hours":"eight"}`} {
		if got := ConfigFrom([]byte(raw)); got != DefaultConfig() {
			t.Fatalf("%q should fall back to defaults, got %+v", raw, got)
		}
	}
}

func TestConfigFrom_OutOfRangeIsIgnored(t *testing.T) {
	// A stored value outside the bounds means a bad write got through
	// somewhere; running on defaults beats running on zero.
	raw, _ := json.Marshal(OutreachSettings{NoReplyHours: ptr(0)})
	if got := ConfigFrom(raw); got != DefaultConfig() {
		t.Fatalf("a zero no-reply window must not be honoured, got %+v", got)
	}
}

func TestValidate_ReportsEveryProblemAtOnce(t *testing.T) {
	s := OutreachSettings{
		NoReplyHours: ptr(0),  // below the minimum
		MaxFollowups: ptr(50), // above the maximum
	}
	problems := s.Validate()
	if len(problems) != 2 {
		t.Fatalf("expected both problems reported together, got %v", problems)
	}
}

func TestValidate_RejectsWindowThatNeverOpens(t *testing.T) {
	s := OutreachSettings{FollowUp1MinDays: ptr(9), FollowUp1MaxDays: ptr(4)}
	problems := s.Validate()
	if len(problems) == 0 {
		t.Fatal("a window starting after it ends must be rejected")
	}
	if !strings.Contains(strings.ToLower(problems[0]), "follow-up 1") {
		t.Fatalf("the message should name the field, got %q", problems[0])
	}
}

func TestValidate_AcceptsAValidCadence(t *testing.T) {
	s := OutreachSettings{
		NoReplyHours:     ptr(48),
		FollowUp1MinDays: ptr(3),
		FollowUp1MaxDays: ptr(5),
		MaxFollowups:     ptr(3),
	}
	if p := s.Validate(); len(p) != 0 {
		t.Fatalf("expected no complaints, got %v", p)
	}
}

func TestOrgConfigLoader_CachesAndInvalidates(t *testing.T) {
	user := uuid.New()
	calls := 0
	stored := `{"no_reply_hours":12}`

	loader := newOrgConfigLoader(func(context.Context, uuid.UUID) ([]byte, error) {
		calls++
		return []byte(stored), nil
	})

	ctx := context.Background()
	base := DefaultConfig()

	if got := loader.Get(ctx, user, base); got.NoReplyHours != 12 {
		t.Fatalf("first read wrong: %d", got.NoReplyHours)
	}
	if got := loader.Get(ctx, user, base); got.NoReplyHours != 12 {
		t.Fatalf("second read wrong: %d", got.NoReplyHours)
	}
	if calls != 1 {
		t.Fatalf("the second read should come from cache, hit the DB %d times", calls)
	}

	// A save must take effect straight away, not after the TTL.
	stored = `{"no_reply_hours":36}`
	loader.Invalidate(user)
	if got := loader.Get(ctx, user, base); got.NoReplyHours != 36 {
		t.Fatalf("invalidate did not force a re-read: %d", got.NoReplyHours)
	}
}

func TestOrgConfigLoader_FailureKeepsOutreachRunning(t *testing.T) {
	loader := newOrgConfigLoader(func(context.Context, uuid.UUID) ([]byte, error) {
		return nil, errors.New("database unreachable")
	})
	got := loader.Get(context.Background(), uuid.New(), DefaultConfig())
	if got != DefaultConfig() {
		t.Fatalf("a failed lookup must fall back to defaults, got %+v", got)
	}
}

func TestOrgConfigLoader_NilIsSafe(t *testing.T) {
	var loader *orgConfigLoader
	if got := loader.Get(context.Background(), uuid.New(), DefaultConfig()); got != DefaultConfig() {
		t.Fatal("an unconfigured loader must return the base config")
	}
	loader.Invalidate(uuid.New()) // must not panic
}

func TestDescribe_MentionsTheNumbersAnAdminSet(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NoReplyHours = 48
	got := cfg.Describe()
	if !strings.Contains(got, "48h") {
		t.Fatalf("the summary should show the configured window: %q", got)
	}
}
