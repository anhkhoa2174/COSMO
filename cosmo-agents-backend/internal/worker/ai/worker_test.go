package ai

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	workerpayloads "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	queueworker "github.com/rockship/cosmo-agents-go/pkg/worker"
)

func TestNewWorker(t *testing.T) {
	worker := New(nil, nil, nil, nil, nil, nil, nil, nil)
	if worker == nil {
		t.Fatalf("expected worker instance, got nil")
	}
}

func TestHandleGenerateEmail(t *testing.T) {
	campaignID := uuid.New()
	agentID := uuid.New()
	contactID := uuid.New()
	templateID := uuid.New()

	fakeCampaignRepo := &stubCampaignRepo{campaign: &domain.Campaign{Base: domain.Base{ID: campaignID}, AgentID: &agentID, Status: domain.CampaignStatusActive, UserID: uuid.New()}}
	fakeContactRepo := &stubContactRepo{contact: &domain.Contact{Base: domain.Base{ID: contactID}, Name: "John Doe", Profile: base.JSONB(`{"email":"john@example.com"}`), Company: "ACME", JobTitle: "CTO"}}
	fakeTemplateRepo := &stubTemplateRepo{template: &domain.Template{Base: domain.Base{ID: templateID}, Subject: "Subj", Content: "Body"}}
	fakeKnowledgeRepo := &stubKnowledgeRepo{}
	fakeClient := &stubWorkerClient{}
	fakeAI := &stubAIClient{emailContent: ai.EmailContent{Subject: "Hello", Body: "World"}}

	worker := New(nil, fakeClient, fakeAI, fakeCampaignRepo, fakeContactRepo, fakeTemplateRepo, nil, fakeKnowledgeRepo)

	payload := workerpayloads.GenerateEmailPayload{
		CampaignID: campaignID,
		ContactID:  contactID,
		TemplateID: templateID,
	}

	taskData, _ := json.Marshal(payload)
	task := asynq.NewTask("test", taskData)

	if err := worker.HandleGenerateEmail(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if fakeClient.lastType != queueworker.TypeSendEmail {
		t.Fatalf("expected enqueue type %s got %s", queueworker.TypeSendEmail, fakeClient.lastType)
	}
	if fakeAI.embedCalled {
		t.Fatalf("embedding should not be called in email generation")
	}
}

func TestHandleGenerateEmbedding(t *testing.T) {
	fakeAI := &stubAIClient{embedding: []float32{1, 2, 3}}
	worker := New(nil, &stubWorkerClient{}, fakeAI, nil, nil, nil, nil, &stubKnowledgeRepo{})

	payload := workerpayloads.GenerateEmbeddingPayload{
		KnowledgeID: uuid.New(),
		Text:        "hello",
	}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask("embed", data)

	if err := worker.HandleGenerateEmbedding(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !fakeAI.embedCalled {
		t.Fatalf("expected embedding to be called")
	}
}

// ---- fakes ----

type stubCampaignRepo struct {
	campaign *domain.Campaign
}

func (s *stubCampaignRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error) {
	return s.campaign, nil
}

type stubContactRepo struct {
	contact *domain.Contact
}

func (s *stubContactRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Contact, error) {
	return s.contact, nil
}

type stubTemplateRepo struct {
	template *domain.Template
}

func (s *stubTemplateRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	return s.template, nil
}

type stubKnowledgeRepo struct{}

func (s *stubKnowledgeRepo) FindAll(ctx context.Context, filter baseRepo.Filter, params *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.Knowledge], error) {
	return &baseRepo.PaginatedResult[domain.Knowledge]{List: []domain.Knowledge{}}, nil
}

type stubAIClient struct {
	emailContent ai.EmailContent
	embedding    []float32
	embedCalled  bool
}

func (s *stubAIClient) GenerateEmailContent(ctx context.Context, params ai.EmailGenerationParams) (*ai.EmailContent, error) {
	return &s.emailContent, nil
}

func (s *stubAIClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	s.embedCalled = true
	return s.embedding, nil
}

type stubWorkerClient struct {
	lastType string
}

func (s *stubWorkerClient) EnqueueTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	s.lastType = taskType
	return nil, nil
}

func (s *stubWorkerClient) EnqueueTaskAt(ctx context.Context, taskType string, payload interface{}, processAt time.Time, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	s.lastType = taskType
	return nil, nil
}
