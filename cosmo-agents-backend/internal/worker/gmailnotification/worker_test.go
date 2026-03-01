package gmailnotification

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

func TestGmailNotificationPayloadUnmarshalNumericPrecision(t *testing.T) {
	input := []byte(`{"email_address":"agent@example.com","history_id":9223372036854775807}`)

	var payload GmailNotificationPayload
	if err := json.Unmarshal(input, &payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "9223372036854775807"
	if payload.HistoryID != want {
		t.Fatalf("history_id mismatch, want %s got %s", want, payload.HistoryID)
	}
}

func TestGmailNotificationPayloadUnmarshalString(t *testing.T) {
	input := []byte(`{"email_address":"agent@example.com","history_id":"1234567890"}`)

	var payload GmailNotificationPayload
	if err := json.Unmarshal(input, &payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if payload.HistoryID != "1234567890" {
		t.Fatalf("history_id mismatch, want %s got %s", "1234567890", payload.HistoryID)
	}
}

func TestProcessTaskUpdatesAgentHistory(t *testing.T) {
	agentID := uuid.New()
	repo := &stubAgentRepo{agents: []domain.Agent{{Base: domain.Base{ID: agentID}, Email: "agent@example.com", LastHistoryID: "10"}}}
	client := &stubWorkerClient{}
	w := New(repo, nil, nil, nil, nil, nil, client)

	payload := GmailNotificationPayload{EmailAddress: "agent@example.com", HistoryID: "11"}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask("gmail:notification", data)

	if err := w.ProcessTask(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if repo.updated[agentID] != "11" {
		t.Fatalf("expected history to be updated to 11")
	}
	if client.lastType != worker.TypeSyncGmailHistory {
		t.Fatalf("expected sync task enqueue, got %s", client.lastType)
	}
}

func TestProcessTaskSkipsWhenHistoryOlder(t *testing.T) {
	agentID := uuid.New()
	repo := &stubAgentRepo{agents: []domain.Agent{{Base: domain.Base{ID: agentID}, Email: "agent@example.com", LastHistoryID: "20"}}}
	w := New(repo, nil, nil, nil, nil, nil, &stubWorkerClient{})

	payload := GmailNotificationPayload{EmailAddress: "agent@example.com", HistoryID: "15"}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask("gmail:notification", data)

	if err := w.ProcessTask(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(repo.updated) != 0 {
		t.Fatalf("expected no updates when incoming history is older")
	}
}

// --- stubs ---

type stubAgentRepo struct {
	agents  []domain.Agent
	updated map[uuid.UUID]string
}

func (s *stubAgentRepo) FindByEmail(ctx context.Context, email string) ([]domain.Agent, error) {
	return s.agents, nil
}

func (s *stubAgentRepo) UpdateLastHistoryID(ctx context.Context, agentID uuid.UUID, historyID string) error {
	if s.updated == nil {
		s.updated = make(map[uuid.UUID]string)
	}
	s.updated[agentID] = historyID
	return nil
}

type stubWorkerClient struct {
	lastType string
}

func (s *stubWorkerClient) EnqueueLowPriorityTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	s.lastType = taskType
	return nil, nil
}
