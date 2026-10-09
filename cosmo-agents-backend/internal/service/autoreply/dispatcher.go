package autoreply

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// The dispatcher runs after an intent handler has saved its draft, and decides
// whether that draft goes out now or waits for a person.
//
// Doing it here rather than inside the handler has two consequences worth
// stating. The draft is always persisted first, so there is a record of what
// was sent even if the send itself fails; and the decision lives in exactly
// one place, so auditing "why did this go out" does not mean reading six
// handlers.

// SettingsLoader returns the raw organisation settings JSON for a user.
type SettingsLoader func(ctx context.Context, userID uuid.UUID) ([]byte, error)

// Counter reports how many replies have already been auto-sent for this user
// today, so the daily cap can be enforced.
type Counter interface {
	CountAutoSentToday(ctx context.Context, userID uuid.UUID) (int, error)
}

// Enqueuer hands the send to the same queue a human "approve and send" uses.
// Auto-reply deliberately reuses that path rather than sending inline: retries,
// rate limits, and logging then behave identically whoever pressed the button.
type Enqueuer interface {
	EnqueueSend(ctx context.Context, in SendRequest) error
}

// SendRequest is what the mail worker needs.
type SendRequest struct {
	AgentID    uuid.UUID
	ContactID  uuid.UUID
	CampaignID *uuid.UUID
	To         string
	Subject    string
	Body       string
	InReplyTo  string
}

// Draft is the reply an intent handler just produced.
type Draft struct {
	Subject   string
	Body      string
	InReplyTo string
}

// Target identifies who the reply would go to.
type Target struct {
	UserID     uuid.UUID
	AgentID    uuid.UUID
	ContactID  uuid.UUID
	CampaignID *uuid.UUID
	ToEmail    string
	OptedOut   bool

	// Merge fills the merge fields of an organisation's fixed message.
	Merge MergeValues
}

type Dispatcher struct {
	load    SettingsLoader
	counter Counter
	queue   Enqueuer
	logger  *zerolog.Logger
}

func NewDispatcher(
	load SettingsLoader,
	counter Counter,
	queue Enqueuer,
	logger *zerolog.Logger,
) *Dispatcher {
	return &Dispatcher{load: load, counter: counter, queue: queue, logger: logger}
}

// Outcome is what the caller records on the conversation.
type Outcome struct {
	Decision Decision
	SentAt   *time.Time

	// Draft is the reply this decision was about. When FromTemplate is set it
	// is the organisation's rendered fixed message rather than the draft the
	// caller passed in, and the caller must record it: a held template is the
	// message a person now has to review, and a sent one is what went out.
	Draft        Draft
	FromTemplate bool
}

// Status is the value written to the draft's status field, which is what the
// inbox reads to decide whether to show a review prompt.
func (o Outcome) Status() string {
	if o.Decision.Send {
		return "auto_sent"
	}
	return "pending_review"
}

// Consider resolves the organisation's policy and acts on it.
//
// Any failure resolves to "hold for review". A settings lookup that errors, a
// counter that cannot answer, an enqueue that fails — none of those are
// reasons to send an unreviewed email, and all of them are reasons not to.
func (d *Dispatcher) Consider(
	ctx context.Context,
	intent string,
	confidence float64,
	target Target,
	draft Draft,
) Outcome {
	if d == nil || d.queue == nil {
		return Outcome{Decision: hold("auto-reply is not configured on this deployment")}
	}

	raw, err := d.load(ctx, target.UserID)
	if err != nil {
		d.log().Warn().Err(err).Msg("auto-reply: settings unreadable, holding draft for review")
		return Outcome{Decision: hold("the organisation's settings could not be read")}
	}

	policy, err := ResolveSettings(raw)
	if err != nil {
		d.log().Warn().Err(err).Msg("auto-reply: settings malformed, holding draft for review")
		return Outcome{Decision: hold("the organisation's settings are malformed")}
	}

	// A fixed message replaces the handler's draft before the decision is
	// made, so the empty-draft check sees the message that would actually go
	// out. It changes what is sent, never whether.
	parsed := parseIntent(intent)
	fromTemplate := false
	if content, ok := policy.ContentFor(parsed); ok && content.Mode == ModeTemplate {
		draft.Body = Render(content.Body, target.Merge)
		if subject := strings.TrimSpace(Render(content.Subject, target.Merge)); subject != "" {
			draft.Subject = subject
		}
		fromTemplate = true
	}
	outcome := func(dec Decision, sentAt *time.Time) Outcome {
		return Outcome{Decision: dec, SentAt: sentAt, Draft: draft, FromTemplate: fromTemplate}
	}

	// The cap is only consulted once the policy might actually send, so a
	// disabled organisation never pays for the query.
	sentToday := 0
	if policy.Enabled {
		n, err := d.counter.CountAutoSentToday(ctx, target.UserID)
		if err != nil {
			d.log().Warn().Err(err).
				Msg("auto-reply: cannot count today's sends, holding draft for review")
			return outcome(hold("today's auto-reply count is unavailable"), nil)
		}
		sentToday = n
	}

	decision := policy.Decide(Request{
		Intent:          parsed,
		Confidence:      confidence,
		ContactOptedOut: target.OptedOut,
		SentToday:       sentToday,
		DraftEmpty:      strings.TrimSpace(draft.Body) == "",
	})

	// Out-of-office, nurture and referral replies get no AI draft, so an
	// administrator who selects one without writing a message would otherwise
	// see nothing happen and no useful reason why.
	if !decision.Send && strings.TrimSpace(draft.Body) == "" &&
		policy.Enabled && policy.Intents[parsed] {
		decision = hold("%s replies have no AI-written draft to send; give it a fixed message to answer it automatically",
			canonical(parsed))
	}

	if !decision.Send {
		d.log().Info().
			Str("contact_id", target.ContactID.String()).
			Str("intent", intent).
			Str("reason", decision.Reason).
			Msg("auto-reply: holding draft for review")
		return outcome(decision, nil)
	}

	// A missing agent is a configuration error, not a reason to improvise a
	// sender address.
	if target.AgentID == uuid.Nil || target.ToEmail == "" {
		return outcome(hold("no sending agent or recipient is configured"), nil)
	}

	if err := d.queue.EnqueueSend(ctx, SendRequest{
		AgentID:    target.AgentID,
		ContactID:  target.ContactID,
		CampaignID: target.CampaignID,
		To:         target.ToEmail,
		Subject:    draft.Subject,
		Body:       draft.Body,
		InReplyTo:  draft.InReplyTo,
	}); err != nil {
		d.log().Error().Err(err).
			Str("contact_id", target.ContactID.String()).
			Msg("auto-reply: enqueue failed, draft left for review")
		return outcome(hold("the send could not be queued: %v", err), nil)
	}

	now := time.Now()
	d.log().Info().
		Str("contact_id", target.ContactID.String()).
		Str("intent", intent).
		Float64("confidence", confidence).
		Str("reason", decision.Reason).
		Bool("from_template", fromTemplate).
		Msg("auto-reply: reply sent without human review")

	return outcome(decision, &now)
}

func (d *Dispatcher) log() *zerolog.Logger {
	if d.logger != nil {
		return d.logger
	}
	nop := zerolog.Nop()
	return &nop
}
