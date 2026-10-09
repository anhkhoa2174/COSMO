package autoreply

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

type fakeCounter struct {
	n   int
	err error
}

func (f fakeCounter) CountAutoSentToday(context.Context, uuid.UUID) (int, error) {
	return f.n, f.err
}

type fakeQueue struct {
	sent []SendRequest
	err  error
}

func (f *fakeQueue) EnqueueSend(_ context.Context, in SendRequest) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, in)
	return nil
}

// settingsJSON builds the stored organisation settings for a policy that
// enables one intent.
func settingsJSON(t *testing.T, intent domain.IntentType) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"no_reply_hours": 8,
		"auto_reply": map[string]any{
			"enabled":        true,
			"intents":        []string{string(intent)},
			"min_confidence": 0.9,
			"daily_cap":      10,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func newTarget() Target {
	return Target{
		UserID:    uuid.New(),
		AgentID:   uuid.New(),
		ContactID: uuid.New(),
		ToEmail:   "prospect@example.com",
	}
}

func loaderReturning(raw []byte, err error) SettingsLoader {
	return func(context.Context, uuid.UUID) ([]byte, error) { return raw, err }
}

const goodBody = "Thanks — here are the details you asked for."

func TestConsider_SendsWhenThePolicyAllows(t *testing.T) {
	q := &fakeQueue{}
	d := NewDispatcher(
		loaderReturning(settingsJSON(t, domain.IntentRequestForPricing), nil),
		fakeCounter{n: 0}, q, nil)

	target := newTarget()
	out := d.Consider(context.Background(),
		string(domain.IntentRequestForPricing), 0.95, target,
		Draft{Subject: "Re: pricing", Body: goodBody})

	if !out.Decision.Send {
		t.Fatalf("expected a send, got %q", out.Decision.Reason)
	}
	if out.SentAt == nil {
		t.Fatal("a send must be timestamped")
	}
	if out.Status() != "auto_sent" {
		t.Fatalf("status = %q", out.Status())
	}
	if len(q.sent) != 1 {
		t.Fatalf("expected one queued send, got %d", len(q.sent))
	}
	if q.sent[0].To != target.ToEmail || q.sent[0].Body != goodBody {
		t.Fatalf("wrong payload queued: %+v", q.sent[0])
	}
}

func TestConsider_HoldsWhenNothingIsConfigured(t *testing.T) {
	// An organisation that never opened the settings page.
	q := &fakeQueue{}
	d := NewDispatcher(loaderReturning(nil, nil), fakeCounter{}, q, nil)

	out := d.Consider(context.Background(),
		string(domain.IntentInterested), 1.0, newTarget(),
		Draft{Body: goodBody})

	if out.Decision.Send || len(q.sent) != 0 {
		t.Fatal("no settings must mean no automatic send")
	}
	if out.Status() != "pending_review" {
		t.Fatalf("status = %q", out.Status())
	}
}

// Every one of these is a way the surrounding system can fail. None of them is
// a reason to put an unreviewed email in a prospect's inbox, so each must hold
// the draft rather than fall through to sending.
func TestConsider_EveryFailureHoldsTheDraft(t *testing.T) {
	intent := string(domain.IntentRequestForPricing)
	good := settingsJSON(t, domain.IntentRequestForPricing)

	cases := []struct {
		name       string
		loader     SettingsLoader
		counter    Counter
		queueErr   error
		wantReason string
	}{
		{
			name:       "settings lookup fails",
			loader:     loaderReturning(nil, errors.New("database down")),
			counter:    fakeCounter{},
			wantReason: "could not be read",
		},
		{
			name:       "settings are malformed",
			loader:     loaderReturning([]byte("{not json"), nil),
			counter:    fakeCounter{},
			wantReason: "malformed",
		},
		{
			name:       "the daily count is unavailable",
			loader:     loaderReturning(good, nil),
			counter:    fakeCounter{err: errors.New("timeout")},
			wantReason: "unavailable",
		},
		{
			name:       "the queue rejects the send",
			loader:     loaderReturning(good, nil),
			counter:    fakeCounter{},
			queueErr:   errors.New("redis unreachable"),
			wantReason: "could not be queued",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := &fakeQueue{err: tc.queueErr}
			d := NewDispatcher(tc.loader, tc.counter, q, nil)

			out := d.Consider(context.Background(), intent, 0.99, newTarget(),
				Draft{Body: goodBody})

			if out.Decision.Send {
				t.Fatal("a failure must never resolve to sending")
			}
			if out.Status() != "pending_review" {
				t.Fatalf("status = %q", out.Status())
			}
			if !strings.Contains(out.Decision.Reason, tc.wantReason) {
				t.Fatalf("reason %q does not mention %q",
					out.Decision.Reason, tc.wantReason)
			}
		})
	}
}

func TestConsider_NilDispatcherIsSafe(t *testing.T) {
	// Deployments that never wire auto-reply must behave exactly as before.
	var d *Dispatcher
	out := d.Consider(context.Background(), string(domain.IntentInterested), 1.0,
		newTarget(), Draft{Body: goodBody})
	if out.Decision.Send {
		t.Fatal("an unwired dispatcher must not send")
	}
}

func TestConsider_RespectsTheDailyCap(t *testing.T) {
	q := &fakeQueue{}
	d := NewDispatcher(
		loaderReturning(settingsJSON(t, domain.IntentRequestForPricing), nil),
		fakeCounter{n: 10}, q, nil) // cap in settingsJSON is 10

	out := d.Consider(context.Background(),
		string(domain.IntentRequestForPricing), 1.0, newTarget(),
		Draft{Body: goodBody})

	if out.Decision.Send || len(q.sent) != 0 {
		t.Fatal("at the cap nothing may be sent")
	}
	if !strings.Contains(out.Decision.Reason, "cap") {
		t.Fatalf("reason should name the cap: %q", out.Decision.Reason)
	}
}

func TestConsider_RefusesAnOptedOutContactEvenWhenEnabled(t *testing.T) {
	q := &fakeQueue{}
	d := NewDispatcher(
		loaderReturning(settingsJSON(t, domain.IntentRequestForPricing), nil),
		fakeCounter{}, q, nil)

	target := newTarget()
	target.OptedOut = true

	out := d.Consider(context.Background(),
		string(domain.IntentRequestForPricing), 1.0, target,
		Draft{Body: goodBody})

	if out.Decision.Send || len(q.sent) != 0 {
		t.Fatal("a do-not-contact contact must never be auto-answered")
	}
}

func TestConsider_RefusesWithoutASendingAgent(t *testing.T) {
	// Reaching a send decision with no agent is a misconfiguration. Improvising
	// a sender address would put mail out from the wrong mailbox.
	q := &fakeQueue{}
	d := NewDispatcher(
		loaderReturning(settingsJSON(t, domain.IntentRequestForPricing), nil),
		fakeCounter{}, q, nil)

	target := newTarget()
	target.AgentID = uuid.Nil

	out := d.Consider(context.Background(),
		string(domain.IntentRequestForPricing), 1.0, target,
		Draft{Body: goodBody})

	if out.Decision.Send || len(q.sent) != 0 {
		t.Fatal("no agent must mean no send")
	}
	if !strings.Contains(out.Decision.Reason, "agent") {
		t.Fatalf("reason should name the agent: %q", out.Decision.Reason)
	}
}

func TestConsider_DoesNotCountWhenDisabled(t *testing.T) {
	// The cap query is skipped for organisations that have auto-reply off, so
	// the common case costs nothing. A counter that would error proves it was
	// never called.
	q := &fakeQueue{}
	d := NewDispatcher(loaderReturning([]byte(`{"no_reply_hours":8}`), nil),
		fakeCounter{err: errors.New("must not be called")}, q, nil)

	out := d.Consider(context.Background(),
		string(domain.IntentInterested), 1.0, newTarget(),
		Draft{Body: goodBody})

	if out.Decision.Send {
		t.Fatal("disabled means no send")
	}
	if !strings.Contains(out.Decision.Reason, "off for this organisation") {
		t.Fatalf("expected the disabled reason, got %q", out.Decision.Reason)
	}
}

func TestConsider_BannedIntentIsRefusedEvenIfStoredAsEnabled(t *testing.T) {
	// A policy naming DO_NOT_CONTACT should never have been saved, but if one
	// reaches the database by any route the dispatcher still refuses it. The
	// stored policy fails validation, so it resolves to disabled, and the hard
	// exclusion catches it regardless.
	raw, _ := json.Marshal(map[string]any{
		"auto_reply": map[string]any{
			"enabled": true,
			"intents": []string{string(domain.IntentDoNotContact)},
		},
	})
	q := &fakeQueue{}
	d := NewDispatcher(loaderReturning(raw, nil), fakeCounter{}, q, nil)

	out := d.Consider(context.Background(),
		string(domain.IntentDoNotContact), 1.0, newTarget(),
		Draft{Body: goodBody})

	if out.Decision.Send || len(q.sent) != 0 {
		t.Fatal("DO_NOT_CONTACT must never be auto-answered")
	}
}

// settingsWithContents enables the given intents with a permissive floor and
// attaches per-intent content.
func settingsWithContents(t *testing.T, enabled bool, intents []domain.IntentType,
	contents map[domain.IntentType]Content) []byte {
	t.Helper()
	names := make([]string, len(intents))
	for k, in := range intents {
		names[k] = string(in)
	}
	stored := map[string]Content{}
	for in, c := range contents {
		stored[string(in)] = c
	}
	raw, err := json.Marshal(map[string]any{
		"auto_reply": map[string]any{
			"enabled":        enabled,
			"intents":        names,
			"min_confidence": 0.9,
			"daily_cap":      10,
			"contents":       stored,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// Out-of-office replies have no AI draft at all. A fixed message is what lets
// them be answered automatically.
func TestConsider_TemplateSuppliesTheReplyAndSends(t *testing.T) {
	q := &fakeQueue{}
	raw := settingsWithContents(t, true, []domain.IntentType{domain.IntentOutOfOffice},
		map[domain.IntentType]Content{domain.IntentOutOfOffice: {
			Mode: ModeTemplate, Body: "Hi {{first_name}}, thanks — we'll follow up when you're back.",
		}})
	d := NewDispatcher(loaderReturning(raw, nil), fakeCounter{}, q, nil)

	target := newTarget()
	target.Merge = MergeValues{FirstName: "Mike"}
	out := d.Consider(context.Background(), string(domain.IntentOutOfOffice), 0.97,
		target, Draft{Subject: "Re: Quick intro"})

	if !out.Decision.Send {
		t.Fatalf("expected a send, got %q", out.Decision.Reason)
	}
	if !out.FromTemplate {
		t.Fatal("outcome must report that the organisation's template was used")
	}
	want := "Hi Mike, thanks — we'll follow up when you're back."
	if len(q.sent) != 1 || q.sent[0].Body != want || q.sent[0].Subject != "Re: Quick intro" {
		t.Fatalf("wrong payload queued: %+v", q.sent)
	}
	if out.Draft.Body != want {
		t.Fatalf("outcome draft = %q", out.Draft.Body)
	}
}

func TestConsider_TemplateSubjectReplacesTheDefault(t *testing.T) {
	q := &fakeQueue{}
	raw := settingsWithContents(t, true, []domain.IntentType{domain.IntentRequestForPricing},
		map[domain.IntentType]Content{domain.IntentRequestForPricing: {
			Mode: ModeTemplate, Subject: "Pricing for {{company}}", Body: "Attached.",
		}})
	d := NewDispatcher(loaderReturning(raw, nil), fakeCounter{}, q, nil)

	target := newTarget()
	target.Merge = MergeValues{Company: "TechCorp"}
	d.Consider(context.Background(), string(domain.IntentRequestForPricing), 0.95,
		target, Draft{Subject: "Re: hello", Body: "AI text"})

	if len(q.sent) != 1 || q.sent[0].Subject != "Pricing for TechCorp" || q.sent[0].Body != "Attached." {
		t.Fatalf("template did not replace the AI draft: %+v", q.sent)
	}
}

// Content decides what is sent, never whether. A low-confidence reply is held
// even with a template — but the rendered template is still handed back so a
// person reviews the message that would have gone out.
func TestConsider_TemplateDoesNotBypassTheConfidenceFloor(t *testing.T) {
	q := &fakeQueue{}
	raw := settingsWithContents(t, true, []domain.IntentType{domain.IntentOutOfOffice},
		map[domain.IntentType]Content{domain.IntentOutOfOffice: {Mode: ModeTemplate, Body: "Back soon"}})
	d := NewDispatcher(loaderReturning(raw, nil), fakeCounter{}, q, nil)

	out := d.Consider(context.Background(), string(domain.IntentOutOfOffice), 0.4,
		newTarget(), Draft{Subject: "Re: x"})

	if out.Decision.Send || len(q.sent) != 0 {
		t.Fatal("a template must not lower the confidence floor")
	}
	if !out.FromTemplate || out.Draft.Body != "Back soon" {
		t.Fatalf("held template not returned for review: %+v", out)
	}
}

func TestConsider_TemplateIsIgnoredWhenAutoReplyIsOff(t *testing.T) {
	q := &fakeQueue{}
	raw := settingsWithContents(t, false, []domain.IntentType{domain.IntentRequestForPricing},
		map[domain.IntentType]Content{domain.IntentRequestForPricing: {Mode: ModeTemplate, Body: "Fixed"}})
	d := NewDispatcher(loaderReturning(raw, nil), fakeCounter{}, q, nil)

	out := d.Consider(context.Background(), string(domain.IntentRequestForPricing), 0.99,
		newTarget(), Draft{Body: "AI text"})

	if out.FromTemplate || out.Draft.Body != "AI text" {
		t.Fatalf("switched-off auto-reply still rewrote the draft: %+v", out)
	}
}

// Selecting an intent that produces no AI draft used to hold silently. The
// reason now tells the administrator what would make it work.
func TestConsider_MissingDraftForASelectedIntentExplainsTheFix(t *testing.T) {
	q := &fakeQueue{}
	raw := settingsWithContents(t, true, []domain.IntentType{domain.IntentOutOfOffice}, nil)
	d := NewDispatcher(loaderReturning(raw, nil), fakeCounter{}, q, nil)

	out := d.Consider(context.Background(), string(domain.IntentOutOfOffice), 0.99,
		newTarget(), Draft{Subject: "Re: x"})

	if out.Decision.Send {
		t.Fatal("an empty reply must never be sent")
	}
	if !strings.Contains(out.Decision.Reason, "fixed message") {
		t.Fatalf("reason does not explain the fix: %q", out.Decision.Reason)
	}
}
