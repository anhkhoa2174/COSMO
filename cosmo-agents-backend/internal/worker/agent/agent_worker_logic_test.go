package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

func TestHandleAgentCreate_Success(t *testing.T) {
	repo := &stubAgentRepo{}
	worker := &AgentWorker{agentRepository: repo}

	payload := AgentCreatePayload{
		UserID:        uuid.New(),
		Name:          "John",
		Email:         "john@example.com",
		EmailProvider: "gmail",
	}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentCreate, data)

	if err := worker.HandleAgentCreate(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.created == nil || repo.created.Email != payload.Email {
		t.Fatalf("expected agent to be created")
	}
}

func TestHandleAgentUpdate_NotFound(t *testing.T) {
	repo := &stubAgentRepo{}
	worker := &AgentWorker{agentRepository: repo}

	payload := AgentUpdatePayload{AgentID: uuid.New()}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentUpdate, data)

	if err := worker.HandleAgentUpdate(context.Background(), task); err == nil {
		t.Fatalf("expected error when agent not found")
	}
}

func TestHandleAgentDelete(t *testing.T) {
	id := uuid.New()
	userID := uuid.New()
	repo := &stubAgentRepo{agent: &domain.Agent{Base: domain.Base{ID: id}, UserID: userID}}
	worker := &AgentWorker{agentRepository: repo}

	payload := AgentDeletePayload{AgentID: id, UserID: userID}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentDelete, data)

	if err := worker.HandleAgentDelete(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !repo.deleted {
		t.Fatalf("expected delete to be called")
	}
}

func TestHandleAgentSyncEmails_SkipUnknownAgent(t *testing.T) {
	repo := &stubAgentRepo{}
	worker := &AgentWorker{agentRepository: repo}

	payload := AgentSyncEmailsPayload{AgentID: uuid.New(), UserID: uuid.New()}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentSyncEmails, data)

	if err := worker.HandleAgentSyncEmails(context.Background(), task); err == nil {
		t.Fatalf("expected error when agent not found")
	}
}

func TestHandleAgentCreate_Validation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		payload AgentCreatePayload
	}{
		{
			name:    "missing name",
			payload: AgentCreatePayload{UserID: uuid.New(), Email: "john@example.com", EmailProvider: "gmail"},
		},
		{
			name:    "missing email",
			payload: AgentCreatePayload{UserID: uuid.New(), Name: "John", EmailProvider: "gmail"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := &AgentWorker{agentRepository: &stubAgentRepo{}}
			data, _ := json.Marshal(tt.payload)
			task := asynq.NewTask(TypeAgentCreate, data)
			if err := worker.HandleAgentCreate(context.Background(), task); err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}

func TestHandleAgentCreate_ParseError(t *testing.T) {
	worker := &AgentWorker{agentRepository: &stubAgentRepo{}}
	task := asynq.NewTask(TypeAgentCreate, []byte("{bad json"))
	if err := worker.HandleAgentCreate(context.Background(), task); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestHandleAgentUpdate_Success(t *testing.T) {
	agentID := uuid.New()
	userID := uuid.New()
	agent := &domain.Agent{Base: domain.Base{ID: agentID}, UserID: userID, Name: "Old", Email: "old@example.com"}
	repo := &stubAgentRepo{agent: agent}
	worker := &AgentWorker{agentRepository: repo}

	newName := "New"
	newEmail := "new@example.com"
	status := string(domain.AgentStatusInactive)
	provider := string(domain.AgentEmailProviderOutlook)
	daily := 10
	maxDaily := 20
	payload := AgentUpdatePayload{
		AgentID:       agentID,
		Name:          &newName,
		Email:         &newEmail,
		Status:        &status,
		EmailProvider: &provider,
		DailyLimit:    &daily,
		MaxDailyLimit: &maxDaily,
		Credentials:   map[string]interface{}{"refresh_token": "r"},
	}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentUpdate, data)

	if err := worker.HandleAgentUpdate(context.Background(), task); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !repo.updated {
		t.Fatalf("expected update to be called")
	}
	if agent.Name != newName || agent.Email != newEmail {
		t.Fatalf("expected agent fields to be updated")
	}
}

func TestHandleAgentUpdate_GetError(t *testing.T) {
	repo := &stubAgentRepo{getErr: errors.New("boom")}
	worker := &AgentWorker{agentRepository: repo}
	payload := AgentUpdatePayload{AgentID: uuid.New()}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentUpdate, data)

	if err := worker.HandleAgentUpdate(context.Background(), task); err == nil {
		t.Fatalf("expected error on get")
	}
}

func TestHandleAgentDelete_NotFound(t *testing.T) {
	worker := &AgentWorker{agentRepository: &stubAgentRepo{}}
	payload := AgentDeletePayload{AgentID: uuid.New(), UserID: uuid.New()}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentDelete, data)

	if err := worker.HandleAgentDelete(context.Background(), task); err == nil {
		t.Fatalf("expected error when agent missing")
	}
}

func TestHandleAgentDelete_Unauthorized(t *testing.T) {
	agentID := uuid.New()
	repo := &stubAgentRepo{agent: &domain.Agent{Base: domain.Base{ID: agentID}, UserID: uuid.New()}}
	worker := &AgentWorker{agentRepository: repo}
	payload := AgentDeletePayload{AgentID: agentID, UserID: uuid.New()}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentDelete, data)

	if err := worker.HandleAgentDelete(context.Background(), task); err == nil {
		t.Fatalf("expected authorization error")
	}
}

func TestHandleAgentDelete_GetError(t *testing.T) {
	repo := &stubAgentRepo{getErr: errors.New("fetch failed")}
	worker := &AgentWorker{agentRepository: repo}
	payload := AgentDeletePayload{AgentID: uuid.New(), UserID: uuid.New()}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentDelete, data)

	if err := worker.HandleAgentDelete(context.Background(), task); err == nil {
		t.Fatalf("expected fetch error")
	}
}

func TestHandleAgentSyncEmails_ParseError(t *testing.T) {
	worker := &AgentWorker{agentRepository: &stubAgentRepo{}}
	task := asynq.NewTask(TypeAgentSyncEmails, []byte("not json"))
	if err := worker.HandleAgentSyncEmails(context.Background(), task); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestHandleAgentSyncEmails_GetError(t *testing.T) {
	repo := &stubAgentRepo{getErr: errors.New("boom")}
	worker := &AgentWorker{agentRepository: repo}
	payload := AgentSyncEmailsPayload{AgentID: uuid.New(), UserID: uuid.New()}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentSyncEmails, data)

	if err := worker.HandleAgentSyncEmails(context.Background(), task); err == nil {
		t.Fatalf("expected fetch error")
	}
}

func TestHandleAgentSyncEmails_Unauthorized(t *testing.T) {
	ownerID := uuid.New()
	repo := &stubAgentRepo{agent: &domain.Agent{Base: domain.Base{ID: uuid.New()}, UserID: ownerID}}
	worker := &AgentWorker{agentRepository: repo}
	payload := AgentSyncEmailsPayload{AgentID: repo.agent.ID, UserID: uuid.New()}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentSyncEmails, data)

	if err := worker.HandleAgentSyncEmails(context.Background(), task); err == nil {
		t.Fatalf("expected authorization error")
	}
}

func TestHandleAgentSyncEmails_UpdateHistoryError(t *testing.T) {
	ownerID := uuid.New()
	agentID := uuid.New()
	repo := &stubAgentRepo{
		agent:            &domain.Agent{Base: domain.Base{ID: agentID}, UserID: ownerID, LastHistoryID: "last"},
		updateHistoryErr: errors.New("update failed"),
	}
	worker := &AgentWorker{agentRepository: repo}
	payload := AgentSyncEmailsPayload{AgentID: agentID, UserID: ownerID}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentSyncEmails, data)

	if err := worker.HandleAgentSyncEmails(context.Background(), task); err == nil {
		t.Fatalf("expected update history error")
	}
}

func TestHandleAgentSyncEmails_Success(t *testing.T) {
	ownerID := uuid.New()
	agentID := uuid.New()
	repo := &stubAgentRepo{
		agent: &domain.Agent{Base: domain.Base{ID: agentID}, UserID: ownerID, LastHistoryID: "history-1"},
	}
	worker := &AgentWorker{agentRepository: repo}
	payload := AgentSyncEmailsPayload{AgentID: agentID, UserID: ownerID}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentSyncEmails, data)

	if err := worker.HandleAgentSyncEmails(context.Background(), task); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if repo.lastHistoryID != "history-1" {
		t.Fatalf("expected last history ID to be recorded, got %s", repo.lastHistoryID)
	}
}

func TestHandleAgentRefreshToken_ParseError(t *testing.T) {
	worker := &AgentWorker{agentRepository: &stubAgentRepo{}}
	task := asynq.NewTask(TypeAgentRefreshToken, []byte("nope"))
	if err := worker.HandleAgentRefreshToken(context.Background(), task); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestHandleAgentRefreshToken_GetError(t *testing.T) {
	repo := &stubAgentRepo{getErr: errors.New("boom")}
	worker := &AgentWorker{agentRepository: repo}
	payload := AgentRefreshTokenPayload{AgentID: uuid.New(), UserID: uuid.New()}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentRefreshToken, data)

	if err := worker.HandleAgentRefreshToken(context.Background(), task); err == nil {
		t.Fatalf("expected fetch error")
	}
}

func TestHandleAgentRefreshToken_NotFound(t *testing.T) {
	worker := &AgentWorker{agentRepository: &stubAgentRepo{}}
	payload := AgentRefreshTokenPayload{AgentID: uuid.New(), UserID: uuid.New()}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentRefreshToken, data)

	if err := worker.HandleAgentRefreshToken(context.Background(), task); err == nil {
		t.Fatalf("expected not found error")
	}
}

func TestHandleAgentRefreshToken_Unauthorized(t *testing.T) {
	ownerID := uuid.New()
	agentID := uuid.New()
	repo := &stubAgentRepo{agent: &domain.Agent{Base: domain.Base{ID: agentID}, UserID: ownerID}}
	worker := &AgentWorker{agentRepository: repo}
	payload := AgentRefreshTokenPayload{AgentID: agentID, UserID: uuid.New()}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentRefreshToken, data)

	if err := worker.HandleAgentRefreshToken(context.Background(), task); err == nil {
		t.Fatalf("expected authorization error")
	}
}

func TestHandleAgentRefreshToken_UpdateError(t *testing.T) {
	ownerID := uuid.New()
	agentID := uuid.New()
	repo := &stubAgentRepo{agent: &domain.Agent{Base: domain.Base{ID: agentID}, UserID: ownerID}, updErr: errors.New("save failed")}
	worker := &AgentWorker{agentRepository: repo}
	payload := AgentRefreshTokenPayload{AgentID: agentID, UserID: ownerID}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentRefreshToken, data)

	if err := worker.HandleAgentRefreshToken(context.Background(), task); err == nil {
		t.Fatalf("expected update error")
	}
}

func TestHandleAgentRefreshToken_Success(t *testing.T) {
	ownerID := uuid.New()
	agentID := uuid.New()
	repo := &stubAgentRepo{agent: &domain.Agent{Base: domain.Base{ID: agentID}, UserID: ownerID}}
	worker := &AgentWorker{agentRepository: repo}
	payload := AgentRefreshTokenPayload{AgentID: agentID, UserID: ownerID}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeAgentRefreshToken, data)

	if err := worker.HandleAgentRefreshToken(context.Background(), task); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !repo.updated {
		t.Fatalf("expected update to be called for refresh")
	}
}

// ---- stubs ----

type stubAgentRepo struct {
	created *domain.Agent
	agent   *domain.Agent
	updated bool
	deleted bool
	getErr  error
	updErr  error
	delErr  error

	lastHistoryID    string
	updateHistoryErr error
}

func (s *stubAgentRepo) Create(ctx context.Context, agent *domain.Agent) (*domain.Agent, error) {
	s.created = agent
	return agent, nil
}

func (s *stubAgentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.agent, nil
}

func (s *stubAgentRepo) Update(ctx context.Context, id uuid.UUID, agent *domain.Agent) error {
	s.updated = true
	return s.updErr
}

func (s *stubAgentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	s.deleted = true
	return s.delErr
}

func (s *stubAgentRepo) UpdateLastHistoryID(ctx context.Context, agentID uuid.UUID, historyID string) error {
	s.lastHistoryID = historyID
	return s.updateHistoryErr
}

var _ agentRepository = (*stubAgentRepo)(nil)
