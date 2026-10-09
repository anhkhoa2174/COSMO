package reply

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/service/autoreply"
)

type stubContactLookup struct{ contact *domain.Contact }

func (s *stubContactLookup) FindByEmail(context.Context, uuid.UUID, string) (*domain.Contact, error) {
	return s.contact, nil
}

type stubAgentLookup struct{ agent *domain.Agent }

func (s *stubAgentLookup) FindByID(context.Context, uuid.UUID) (*domain.Agent, error) {
	return s.agent, nil
}

// recordingDispatcher captures what the worker asked and answers with a fixed
// outcome.
type recordingDispatcher struct {
	target  autoreply.Target
	draft   autoreply.Draft
	outcome autoreply.Outcome
	calls   int
}

func (r *recordingDispatcher) Consider(_ context.Context, _ string, _ float64,
	target autoreply.Target, draft autoreply.Draft) autoreply.Outcome {
	r.calls++
	r.target, r.draft = target, draft
	return r.outcome
}

func autoReplyFixture(t *testing.T, meta string, d *recordingDispatcher) (*Worker, *stubConversationRepo, *domain.Campaign, *domain.Email) {
	t.Helper()
	convID := uuid.New()
	agentID := uuid.New()
	conv := &domain.Conversation{Base: domain.Base{ID: convID}, CMetadata: base.JSONB([]byte(meta))}
	convRepo := &stubConversationRepo{conversation: conv}
	contacts := &stubContactLookup{contact: &domain.Contact{
		Base: domain.Base{ID: uuid.New()}, Name: "Mike Chen", Company: "ScaleUp", JobTitle: "N/A",
	}}
	log := zerolog.Nop()
	w := New(nil, convRepo, nil, contacts, nil, nil, nil, &log).
		WithAutoReply(d, &stubAgentLookup{agent: &domain.Agent{Base: domain.Base{ID: agentID}, Name: "Alex"}})
	campaign := &domain.Campaign{Base: domain.Base{ID: uuid.New()}, UserID: uuid.New(), AgentID: &agentID}
	email := &domain.Email{ConversationID: &convID, FromEmail: "mike@scaleup.io",
		Subject: "Out of office", GmailMessageID: "msg-1"}
	return w, convRepo, campaign, email
}

func aiReplyBlock(t *testing.T, conv *domain.Conversation) map[string]interface{} {
	t.Helper()
	var meta map[string]interface{}
	if err := json.Unmarshal(conv.CMetadata, &meta); err != nil {
		t.Fatal(err)
	}
	block, _ := meta["ai_reply"].(map[string]interface{})
	return block
}

// An out-of-office reply has no AI draft. With a fixed message configured the
// worker must still ask the dispatcher, and must record the message it sent.
func TestConsiderAutoReply_RecordsTheOrganisationsFixedMessage(t *testing.T) {
	d := &recordingDispatcher{outcome: autoreply.Outcome{
		Decision:     autoreply.Decision{Send: true, Reason: "ok"},
		Draft:        autoreply.Draft{Subject: "Re: Out of office", Body: "Hi Mike, talk soon."},
		FromTemplate: true,
	}}
	w, convRepo, campaign, email := autoReplyFixture(t, `{}`, d)

	w.considerAutoReply(context.Background(), campaign, email, domain.IntentOutOfOffice, 0.97)

	if d.calls != 1 {
		t.Fatal("dispatcher was not consulted for an intent with no AI draft")
	}
	if d.draft.Subject != "Re: Out of office" || d.draft.Body != "" {
		t.Fatalf("worker passed %+v", d.draft)
	}
	if m := d.target.Merge; m.FirstName != "Mike" || m.Company != "ScaleUp" || m.JobTitle != "" || m.SenderName != "Alex" {
		t.Fatalf("merge values = %+v", m)
	}

	block := aiReplyBlock(t, convRepo.conversation)
	if block["draft_content"] != "Hi Mike, talk soon." || block["generated_by"] != "org_template" {
		t.Fatalf("fixed message not recorded: %v", block)
	}
	if block["status"] != "auto_sent" {
		t.Fatalf("status = %v", block["status"])
	}
	if convRepo.conversation.Replied {
		t.Fatal("a fixed acknowledgement must not mark the thread replied")
	}
}

func TestConsiderAutoReply_NoDraftAndNoFixedMessageWritesNothing(t *testing.T) {
	d := &recordingDispatcher{outcome: autoreply.Outcome{
		Decision: autoreply.Decision{Reason: "auto-reply is off for this organisation"},
	}}
	w, convRepo, campaign, email := autoReplyFixture(t, `{}`, d)

	w.considerAutoReply(context.Background(), campaign, email, domain.IntentNurture, 0.9)

	if convRepo.updated {
		t.Fatal("conversation was written although there was nothing to record")
	}
}

func TestConsiderAutoReply_StampsAnExistingAIDraft(t *testing.T) {
	d := &recordingDispatcher{outcome: autoreply.Outcome{
		Decision: autoreply.Decision{Reason: "below the floor"},
		Draft:    autoreply.Draft{Subject: "Re: pricing", Body: "AI text"},
	}}
	w, convRepo, campaign, email := autoReplyFixture(t,
		`{"ai_reply":{"draft_content":"AI text","draft_subject":"Re: pricing","generated_by":"coze_bot"}}`, d)

	w.considerAutoReply(context.Background(), campaign, email, domain.IntentRequestForPricing, 0.5)

	block := aiReplyBlock(t, convRepo.conversation)
	if block["generated_by"] != "coze_bot" || block["status"] != "pending_review" ||
		block["auto_reply_reason"] != "below the floor" {
		t.Fatalf("AI draft not stamped correctly: %v", block)
	}
}
