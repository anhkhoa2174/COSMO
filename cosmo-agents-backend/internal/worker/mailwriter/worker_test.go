package mailwriter

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	mailService "github.com/rockship/cosmo-agents-go/internal/service/mail"
)

func TestGenerateCampaignEmailTask(t *testing.T) {
	task, err := NewGenerateCampaignEmailTask(uuid.New(), uuid.New(), "first", map[string]string{"name": "test"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TypeGenerateCampaignEmail {
		t.Fatalf("expected task type %s, got %s", TypeGenerateCampaignEmail, task.Type())
	}
}

func TestGenerateReplyEmailTask(t *testing.T) {
	task, err := NewGenerateReplyEmailTask(uuid.New(), uuid.New(), mailService.IntentType("positive"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TypeGenerateReplyEmail {
		t.Fatalf("expected task type %s, got %s", TypeGenerateReplyEmail, task.Type())
	}
}

func TestTypeConstantsPresent(t *testing.T) {
	if TypeEmailIndexing == "" || TypeGenerateCampaignEmail == "" || TypeGenerateReplyEmail == "" {
		t.Fatalf("expected type constants to be non-empty")
	}
}

func TestProcessGenerateCampaignEmail(t *testing.T) {
	t.Skip("processGenerateCampaignEmail now requires DB-backed campaign/knowledge repositories (RAG lookup); needs integration environment")
	fakeWriter := &stubMailWriter{}
	logger := zerolog.New(io.Discard)
	worker := New(nil, nil, nil, nil, nil, nil, nil, fakeWriter, nil, &logger)

	payload := GenerateCampaignEmailPayload{
		CampaignID:   uuid.New(),
		ContactID:    uuid.New(),
		TemplateType: "first",
	}
	task, _ := NewGenerateCampaignEmailTask(payload.CampaignID, payload.ContactID, payload.TemplateType, nil, nil)

	if err := worker.ProcessTask(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !fakeWriter.outreachCalled {
		t.Fatalf("expected outreach generator to be called")
	}
}

func TestProcessGenerateReplyEmail(t *testing.T) {
	t.Skip("processGenerateCampaignEmail now requires DB-backed campaign/knowledge repositories (RAG lookup); needs integration environment")
	fakeWriter := &stubMailWriter{}
	logger := zerolog.New(io.Discard)
	worker := New(nil, nil, nil, nil, nil, nil, nil, fakeWriter, nil, &logger)

	payload := GenerateReplyEmailPayload{
		CampaignID:     uuid.New(),
		ConversationID: uuid.New(),
		Intent:         mailService.IntentType("positive"),
	}
	task, _ := NewGenerateReplyEmailTask(payload.CampaignID, payload.ConversationID, payload.Intent, nil)

	if err := worker.ProcessTask(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !fakeWriter.replyCalled {
		t.Fatalf("expected reply generator to be called")
	}
}

func TestHandleEmailIndexing(t *testing.T) {
	t.Skip("processGenerateCampaignEmail now requires DB-backed campaign/knowledge repositories (RAG lookup); needs integration environment")
	fakeWriter := &stubMailWriter{}
	logger := zerolog.New(io.Discard)
	worker := New(nil, nil, nil, nil, nil, nil, nil, fakeWriter, nil, &logger)

	payload := EmailIndexingPayload{
		Collection:        "test",
		SequenceEmailsStr: "hello",
		EmbeddingGID:      "gid",
		Metadata:          map[string]interface{}{"k": "v"},
	}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeEmailIndexing, data)

	if err := worker.ProcessTask(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// --- stubs ---

type stubMailWriter struct {
	outreachCalled bool
	replyCalled    bool
}

func (s *stubMailWriter) GenerateSingleOutreach(ctx context.Context, previous []mailService.EmailTemplate, docs ...mailService.Document) (*mailService.EmailTemplate, error) {
	s.outreachCalled = true
	return &mailService.EmailTemplate{Subject: "S", Content: "C"}, nil
}

func (s *stubMailWriter) GenerateReply(ctx context.Context, conversation []mailService.EmailTemplate, intent mailService.IntentType) (*mailService.EmailTemplate, error) {
	s.replyCalled = true
	return &mailService.EmailTemplate{Subject: "R", Content: "C"}, nil
}
