package campaign

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
	queueworker "github.com/rockship/cosmo-agents-go/pkg/worker"
)

func TestHandleExecuteCampaignEnqueuesSchedule(t *testing.T) {
	campaignID := uuid.New()
	agentID := uuid.New()
	templateID := uuid.New()
	payload := workerpayloads.ExecuteCampaignPayload{
		CampaignID: campaignID,
		UserID:     uuid.New(),
		AgentID:    agentID,
		ContactIDs: []uuid.UUID{uuid.New()},
	}

	fakeCampaignRepo := &stubCampaignRepo{campaign: &domain.Campaign{Base: domain.Base{ID: campaignID}, AgentID: &agentID, Status: domain.CampaignStatusActive}}
	fakeTemplateRepo := &stubTemplateRepo{templates: []*domain.Template{{Base: domain.Base{ID: templateID}, Subject: "S", Content: "C", SendAfter: 0}}}
	fakeTaskRepo := &stubTaskRepo{}
	fakeClient := &stubQueueClient{}

	taskData, _ := json.Marshal(payload)
	task := asynq.NewTask("campaign:execute", taskData)

	worker := New(nil, fakeClient, fakeCampaignRepo, &stubContactRepo{}, fakeTemplateRepo, fakeTaskRepo, nil)
	if err := worker.HandleExecuteCampaign(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fakeClient.lastType != queueworker.TypeScheduleTasks {
		t.Fatalf("expected schedule task enqueue, got %s", fakeClient.lastType)
	}
	if !fakeTaskRepo.createCalled {
		t.Fatalf("expected CreateTasks to be called")
	}
}

func TestHandleExecuteCampaignInactiveReturnsError(t *testing.T) {
	campaignID := uuid.New()
	payload := workerpayloads.ExecuteCampaignPayload{
		CampaignID: campaignID,
		UserID:     uuid.New(),
		AgentID:    uuid.New(),
		ContactIDs: []uuid.UUID{uuid.New()},
	}

	fakeCampaignRepo := &stubCampaignRepo{campaign: &domain.Campaign{Base: domain.Base{ID: campaignID}, Status: domain.CampaignStatusDraft}}
	taskData, _ := json.Marshal(payload)
	task := asynq.NewTask("campaign:execute", taskData)

	worker := New(nil, &stubQueueClient{}, fakeCampaignRepo, &stubContactRepo{}, &stubTemplateRepo{}, &stubTaskRepo{}, nil)
	if err := worker.HandleExecuteCampaign(context.Background(), task); err == nil {
		t.Fatalf("expected error for inactive campaign")
	}
}

func TestHandleExecuteCampaignNoTemplates(t *testing.T) {
	campaignID := uuid.New()
	payload := workerpayloads.ExecuteCampaignPayload{
		CampaignID: campaignID,
		UserID:     uuid.New(),
		AgentID:    uuid.New(),
		ContactIDs: []uuid.UUID{uuid.New()},
	}

	fakeCampaignRepo := &stubCampaignRepo{campaign: &domain.Campaign{Base: domain.Base{ID: campaignID}, Status: domain.CampaignStatusActive}}
	taskData, _ := json.Marshal(payload)
	task := asynq.NewTask("campaign:execute", taskData)

	worker := New(nil, &stubQueueClient{}, fakeCampaignRepo, &stubContactRepo{}, &stubTemplateRepo{}, &stubTaskRepo{}, nil)
	if err := worker.HandleExecuteCampaign(context.Background(), task); err == nil {
		t.Fatalf("expected error when templates not found")
	}
}

func TestHandleScheduleTasksEnqueuesSendEmail(t *testing.T) {
	campaignID := uuid.New()
	contactID := uuid.New()
	templateID := uuid.New()
	taskID := uuid.New()

	task := domain.Task{
		Base:       domain.Base{ID: taskID},
		ContactID:  contactID,
		CampaignID: campaignID,
		TemplateID: templateID,
		Status:     domain.TaskStatusPending,
	}

	fakeTaskRepo := &stubTaskRepo{
		findResult: &baseRepo.PaginatedResult[domain.Task]{List: []domain.Task{task}, Total: 1},
	}
	fakeContactRepo := &stubContactRepo{
		contacts: map[uuid.UUID]*domain.Contact{
			contactID: {Base: domain.Base{ID: contactID}, Profile: base.JSONB(`{"email":"user@example.com"}`)},
		},
	}
	fakeTemplateRepo := &stubTemplateRepo{
		templateMap: map[uuid.UUID]*domain.Template{
			templateID: {Base: domain.Base{ID: templateID}, Subject: "Subj", Content: "Body"},
		},
	}
	fakeClient := &stubQueueClient{}

	payload := workerpayloads.ScheduleTasksPayload{
		CampaignID: campaignID,
		ContactIDs: []uuid.UUID{contactID},
		AgentID:    uuid.New(),
	}
	data, _ := json.Marshal(payload)
	asynqTask := asynq.NewTask("campaign:schedule", data)

	worker := New(nil, fakeClient, nil, fakeContactRepo, fakeTemplateRepo, fakeTaskRepo, nil)
	if err := worker.HandleScheduleTasks(context.Background(), asynqTask); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fakeClient.lastType != queueworker.TypeSendEmail {
		t.Fatalf("expected send email enqueue, got %s", fakeClient.lastType)
	}
	if fakeTaskRepo.updatedStatus != domain.TaskStatusRunning {
		t.Fatalf("expected task status to be updated to running")
	}
}

func TestHandleScheduleTasksMissingContactFailsTask(t *testing.T) {
	campaignID := uuid.New()
	contactID := uuid.New()
	templateID := uuid.New()
	taskID := uuid.New()

	task := domain.Task{
		Base:       domain.Base{ID: taskID},
		ContactID:  contactID,
		CampaignID: campaignID,
		TemplateID: templateID,
		Status:     domain.TaskStatusPending,
	}

	fakeTaskRepo := &stubTaskRepo{
		findResult: &baseRepo.PaginatedResult[domain.Task]{List: []domain.Task{task}, Total: 1},
	}
	fakeTemplateRepo := &stubTemplateRepo{
		templateMap: map[uuid.UUID]*domain.Template{
			templateID: {Base: domain.Base{ID: templateID}, Subject: "Subj", Content: "Body"},
		},
	}
	fakeClient := &stubQueueClient{}

	payload := workerpayloads.ScheduleTasksPayload{
		CampaignID: campaignID,
		ContactIDs: []uuid.UUID{contactID},
		AgentID:    uuid.New(),
	}
	data, _ := json.Marshal(payload)
	asynqTask := asynq.NewTask("campaign:schedule", data)

	worker := New(nil, fakeClient, nil, &stubContactRepo{contacts: map[uuid.UUID]*domain.Contact{}}, fakeTemplateRepo, fakeTaskRepo, nil)
	if err := worker.HandleScheduleTasks(context.Background(), asynqTask); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fakeTaskRepo.updatedStatus != domain.TaskStatusFailed {
		t.Fatalf("expected task marked failed when contact missing")
	}
}

func TestHandleScheduleTasksSkipsFuture(t *testing.T) {
	campaignID := uuid.New()
	contactID := uuid.New()
	templateID := uuid.New()
	taskID := uuid.New()
	future := time.Now().Add(time.Hour)

	task := domain.Task{
		Base:       domain.Base{ID: taskID},
		ContactID:  contactID,
		CampaignID: campaignID,
		TemplateID: templateID,
		Status:     domain.TaskStatusPending,
		ScheduleAt: &future,
	}

	fakeTaskRepo := &stubTaskRepo{
		findResult: &baseRepo.PaginatedResult[domain.Task]{List: []domain.Task{task}, Total: 1},
	}
	fakeContactRepo := &stubContactRepo{
		contacts: map[uuid.UUID]*domain.Contact{
			contactID: {Base: domain.Base{ID: contactID}, Profile: base.JSONB(`{"email":"user@example.com"}`)},
		},
	}
	fakeTemplateRepo := &stubTemplateRepo{
		templateMap: map[uuid.UUID]*domain.Template{
			templateID: {Base: domain.Base{ID: templateID}, Subject: "Subj", Content: "Body"},
		},
	}
	fakeClient := &stubQueueClient{}

	payload := workerpayloads.ScheduleTasksPayload{
		CampaignID: campaignID,
		ContactIDs: []uuid.UUID{contactID},
		AgentID:    uuid.New(),
	}
	data, _ := json.Marshal(payload)
	asynqTask := asynq.NewTask("campaign:schedule", data)

	worker := New(nil, fakeClient, nil, fakeContactRepo, fakeTemplateRepo, fakeTaskRepo, nil)
	if err := worker.HandleScheduleTasks(context.Background(), asynqTask); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fakeClient.lastType != queueworker.TypeScheduleTasks {
		t.Fatalf("expected scheduler re-enqueue, got %s", fakeClient.lastType)
	}
}

func TestHandleGenerateEmailNotImplemented(t *testing.T) {
	payload := workerpayloads.GenerateEmailPayload{
		CampaignID: uuid.New(),
		ContactID:  uuid.New(),
		TemplateID: uuid.New(),
	}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask("campaign:generate_email", data)

	worker := New(nil, &stubQueueClient{}, nil, &stubContactRepo{}, &stubTemplateRepo{}, &stubTaskRepo{}, nil)
	if err := worker.HandleGenerateEmail(context.Background(), task); err == nil {
		t.Fatalf("expected not implemented error")
	}
}

// ---- stubs ----

type stubCampaignRepo struct {
	campaign *domain.Campaign
}

func (s *stubCampaignRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error) {
	return s.campaign, nil
}

type stubContactRepo struct {
	contacts map[uuid.UUID]*domain.Contact
}

func (s *stubContactRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Contact, error) {
	return s.contacts, nil
}

func (s *stubContactRepo) FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.Contact], error) {
	return &baseRepo.PaginatedResult[domain.Contact]{List: []domain.Contact{}, Total: 0}, nil
}

func (s *stubContactRepo) FindIDsByListContact(ctx context.Context, listID uuid.UUID, userID uuid.UUID, orgID *uuid.UUID, offset int, limit int) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}

type stubTemplateRepo struct {
	templates   []*domain.Template
	templateMap map[uuid.UUID]*domain.Template
}

func (s *stubTemplateRepo) FindByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*domain.Template, error) {
	return s.templates, nil
}

func (s *stubTemplateRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Template, error) {
	if s.templateMap == nil {
		s.templateMap = make(map[uuid.UUID]*domain.Template)
	}
	return s.templateMap, nil
}

type stubTaskRepo struct {
	createCalled  bool
	findResult    *baseRepo.PaginatedResult[domain.Task]
	updatedStatus domain.TaskStatus
}

func (s *stubTaskRepo) CreateTasks(ctx context.Context, attrs []domain.TaskAttributes) ([]uuid.UUID, error) {
	s.createCalled = true
	return []uuid.UUID{uuid.New()}, nil
}

func (s *stubTaskRepo) FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.Task], error) {
	return s.findResult, nil
}

func (s *stubTaskRepo) Update(ctx context.Context, id uuid.UUID, entity *domain.Task) error {
	s.updatedStatus = entity.Status
	return nil
}

type stubQueueClient struct {
	lastType string
}

func (s *stubQueueClient) EnqueueTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	s.lastType = taskType
	return nil, nil
}

func (s *stubQueueClient) EnqueueTaskAt(ctx context.Context, taskType string, payload interface{}, processAt time.Time, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	s.lastType = taskType
	return nil, nil
}
