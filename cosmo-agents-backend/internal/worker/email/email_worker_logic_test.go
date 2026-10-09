package email

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	repositoryHelper "github.com/rockship/cosmo-agents-go/internal/repository/helper"
	"github.com/rockship/cosmo-agents-go/pkg/gmail"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
)

func TestHandleAgentSendEmail_ParseError(t *testing.T) {
	worker := newEmailWorkerForTest()
	task := asynq.NewTask(TypeAgentSendEmail, []byte("bad json"))

	if err := worker.HandleAgentSendEmail(context.Background(), task); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestHandleAgentSendEmail_AgentErrors(t *testing.T) {
	t.Parallel()
	payload := AgentSendEmailPayload{AgentID: uuid.New(), CampaignID: uuid.New()}
	data, _ := jsonMarshal(payload)
	task := asynq.NewTask(TypeAgentSendEmail, data)

	workerErr := newEmailWorkerForTest()
	workerErr.agentRepository.(*stubAgentRepo).getErr = errors.New("boom")
	if err := workerErr.HandleAgentSendEmail(context.Background(), task); err == nil {
		t.Fatalf("expected error from agent repository")
	}

	workerMissing := newEmailWorkerForTest()
	if err := workerMissing.HandleAgentSendEmail(context.Background(), task); err == nil {
		t.Fatalf("expected not found error")
	}
}

func TestHandleAgentSendEmail_CampaignErrors(t *testing.T) {
	payload := AgentSendEmailPayload{AgentID: uuid.New(), CampaignID: uuid.New()}
	data, _ := jsonMarshal(payload)
	task := asynq.NewTask(TypeAgentSendEmail, data)

	agent := sampleAgent()
	worker := newEmailWorkerForTest()
	worker.agentRepository.(*stubAgentRepo).agent = agent

	workerErr := *worker
	workerErr.campaignRepository = &stubCampaignRepo{err: errors.New("fail campaign")}
	if err := workerErr.HandleAgentSendEmail(context.Background(), task); err == nil {
		t.Fatalf("expected campaign fetch error")
	}

	workerMissing := *worker
	workerMissing.campaignRepository = &stubCampaignRepo{}
	if err := workerMissing.HandleAgentSendEmail(context.Background(), task); err == nil {
		t.Fatalf("expected campaign not found error")
	}
}

func TestHandleAgentSendEmail_CredentialUnmarshalError(t *testing.T) {
	agent := sampleAgent()
	agent.Credentials = base.JSON([]byte("{invalid"))

	payload := AgentSendEmailPayload{AgentID: agent.ID, CampaignID: uuid.New()}
	data, _ := jsonMarshal(payload)
	task := asynq.NewTask(TypeAgentSendEmail, data)

	worker := newEmailWorkerForTest()
	worker.agentRepository.(*stubAgentRepo).agent = agent
	worker.campaignRepository.(*stubCampaignRepo).campaign = &domain.Campaign{Base: domain.Base{ID: payload.CampaignID}}

	if err := worker.HandleAgentSendEmail(context.Background(), task); err == nil {
		t.Fatalf("expected credentials unmarshal error")
	}
}

func TestHandleAgentSendEmail_RefreshInvalidGrant(t *testing.T) {
	agent := sampleAgent()
	payload := AgentSendEmailPayload{AgentID: agent.ID, CampaignID: uuid.New()}
	data, _ := jsonMarshal(payload)
	task := asynq.NewTask(TypeAgentSendEmail, data)

	worker := newEmailWorkerForTest()
	worker.agentRepository.(*stubAgentRepo).agent = agent
	worker.campaignRepository.(*stubCampaignRepo).campaign = &domain.Campaign{Base: domain.Base{ID: payload.CampaignID}}
	worker.googleAuthService.(*stubGoogleAuthService).err = errors.New("invalid_grant")

	if err := worker.HandleAgentSendEmail(context.Background(), task); err == nil {
		t.Fatalf("expected refresh error")
	}
	if !worker.agentRepository.(*stubAgentRepo).updated {
		t.Fatalf("expected agent update after invalid grant")
	}
}

func TestHandleAgentSendEmail_DailyLimitReached(t *testing.T) {
	daily := 1
	sent := 1
	agent := sampleAgent()
	agent.DailyLimit = &daily
	agent.EmailsSentToday = &sent

	payload := AgentSendEmailPayload{AgentID: agent.ID, CampaignID: uuid.New()}
	data, _ := jsonMarshal(payload)
	task := asynq.NewTask(TypeAgentSendEmail, data)

	worker := newEmailWorkerForTest()
	worker.agentRepository.(*stubAgentRepo).agent = agent
	worker.campaignRepository.(*stubCampaignRepo).campaign = &domain.Campaign{Base: domain.Base{ID: payload.CampaignID}}
	worker.googleAuthService.(*stubGoogleAuthService).refreshed = map[string]interface{}{}

	if err := worker.HandleAgentSendEmail(context.Background(), task); err != nil {
		t.Fatalf("expected no error when daily limit reached, got %v", err)
	}
	if worker.taskRepository.(*stubTaskRepo).pendingCalled {
		t.Fatalf("tasks should not be fetched when daily limit reached")
	}
}

func TestHandleAgentSendEmail_NoTasks(t *testing.T) {
	agent := sampleAgent()
	payload := AgentSendEmailPayload{AgentID: agent.ID, CampaignID: uuid.New()}
	data, _ := jsonMarshal(payload)
	task := asynq.NewTask(TypeAgentSendEmail, data)

	worker := newEmailWorkerForTest()
	worker.agentRepository.(*stubAgentRepo).agent = agent
	worker.campaignRepository.(*stubCampaignRepo).campaign = &domain.Campaign{Base: domain.Base{ID: payload.CampaignID}}
	worker.googleAuthService.(*stubGoogleAuthService).refreshed = map[string]interface{}{}

	if err := worker.HandleAgentSendEmail(context.Background(), task); err != nil {
		t.Fatalf("expected no error with empty tasks, got %v", err)
	}
	if !worker.taskRepository.(*stubTaskRepo).pendingCalled {
		t.Fatalf("expected pending tasks fetch to be called")
	}
}

func TestHandleAgentSendEmail_TransactionFailureFlow(t *testing.T) {
	db := setupEmailDatabase(t)

	agent := sampleAgent()
	campaign := &domain.Campaign{Base: domain.Base{ID: uuid.New()}}
	taskID := uuid.New()

	if err := db.Table("tasks").Create(map[string]interface{}{
		"id":         taskID.String(),
		"status":     string(domain.TaskStatusPending),
		"is_deleted": false,
	}).Error; err != nil {
		t.Fatalf("failed to seed task: %v", err)
	}

	worker := &EmailWorker{
		db:                 db,
		agentRepository:    &stubAgentRepo{agent: agent},
		campaignRepository: &stubCampaignRepo{campaign: campaign},
		taskRepository:     &stubTaskRepo{tasks: []*domain.Task{{Base: domain.Base{ID: taskID}}}},
		emailRepository:    &stubEmailRepo{},
		conversationRepo:   &stubConversationRepo{},
		relationsHelper:    repositoryHelper.NewRelationsHelper(db),
		googleAuthService:  &stubGoogleAuthService{refreshed: map[string]interface{}{}},
		gmailService:       &stubGmailService{},
		taskMonitor:        &stubTaskMonitor{},
	}

	payload := AgentSendEmailPayload{AgentID: agent.ID, CampaignID: campaign.ID, TaskIDs: []uuid.UUID{taskID}}
	data, _ := jsonMarshal(payload)
	task := asynq.NewTask(TypeAgentSendEmail, data)

	if err := worker.HandleAgentSendEmail(context.Background(), task); err != nil {
		t.Fatalf("expected transaction to complete with handled failures, got %v", err)
	}

	if len(worker.taskRepository.(*stubTaskRepo).updatedTasks) == 0 {
		t.Fatalf("expected failed task to be updated")
	}
	if !worker.agentRepository.(*stubAgentRepo).incrementCalled {
		t.Fatalf("expected IncrementEmailCount to be called")
	}
}

func TestHandleAgentSendEmail_SuccessFlow(t *testing.T) {
	db := setupEmailDatabase(t)

	contactID := uuid.New()
	templateID := uuid.New()
	campaignID := uuid.New()
	taskID := uuid.New()
	userID := uuid.New()

	contact := domain.Contact{
		Base:    domain.Base{ID: contactID},
		UserID:  userID,
		Profile: base.JSONB(`{"email":"contact@example.com"}`),
	}
	if err := contact.Profile.Marshal(map[string]any{"email": "contact@example.com", "contact_first_name": "Bob"}); err != nil {
		t.Fatalf("failed to marshal contact profile: %v", err)
	}
	template := domain.Template{
		Base:     domain.Base{ID: templateID},
		UserID:   userID,
		Subject:  "Hi {{contact_first_name}}",
		Content:  "Hello {{contact_first_name}}",
		Position: 1,
	}
	campaign := domain.Campaign{
		Base: domain.Base{ID: campaignID},
	}
	taskModel := domain.Task{
		Base:       domain.Base{ID: taskID},
		ContactID:  contactID,
		TemplateID: templateID,
		CampaignID: campaignID,
		Status:     domain.TaskStatusPending,
	}

	if err := db.Table("contacts").Create(map[string]interface{}{
		"id":         contact.ID.String(),
		"user_id":    contact.UserID.String(),
		"email":      "contact@example.com",
		"profile":    string(contact.Profile),
		"is_deleted": false,
	}).Error; err != nil {
		t.Fatalf("failed to seed contact: %v", err)
	}
	if err := db.Table("templates").Create(map[string]interface{}{
		"id":          template.ID.String(),
		"user_id":     template.UserID.String(),
		"subject":     template.Subject,
		"content":     template.Content,
		"position":    template.Position,
		"campaign_id": campaign.ID.String(),
		"type":        template.Type,
		"category":    template.Category,
		"cmetadata":   string(template.CMetadata),
		"send_after":  template.SendAfter,
		"is_deleted":  false,
	}).Error; err != nil {
		t.Fatalf("failed to seed template: %v", err)
	}
	if err := db.Table("campaigns").Create(map[string]interface{}{
		"id":         campaign.ID.String(),
		"agent_id":   "",
		"is_deleted": false,
	}).Error; err != nil {
		t.Fatalf("failed to seed campaign: %v", err)
	}
	if err := db.Table("tasks").Create(map[string]interface{}{
		"id":          taskModel.ID.String(),
		"contact_id":  contact.ID.String(),
		"template_id": template.ID.String(),
		"campaign_id": campaign.ID.String(),
		"status":      string(taskModel.Status),
		"is_deleted":  false,
	}).Error; err != nil {
		t.Fatalf("failed to seed task: %v", err)
	}

	agent := sampleAgent()
	agent.UserID = userID

	gmailMsg := &GmailMessage{ID: "msg-1", ThreadID: "thread-1"}
	conversationID := uuid.New()

	worker := &EmailWorker{
		db:                 db,
		agentRepository:    &stubAgentRepo{agent: agent},
		campaignRepository: &stubCampaignRepo{campaign: &campaign},
		taskRepository:     &stubTaskRepo{tasks: []*domain.Task{{Base: domain.Base{ID: taskID}}}},
		emailRepository:    &stubEmailRepo{},
		conversationRepo:   &stubConversationRepo{conversationID: conversationID},
		relationsHelper:    repositoryHelper.NewRelationsHelper(db),
		googleAuthService:  &stubGoogleAuthService{refreshed: map[string]interface{}{}},
		gmailService:       &stubGmailService{msg: gmailMsg},
		taskMonitor:        &stubTaskMonitor{},
	}

	payload := AgentSendEmailPayload{AgentID: agent.ID, CampaignID: campaign.ID, TaskIDs: []uuid.UUID{taskID}}
	data, _ := jsonMarshal(payload)
	task := asynq.NewTask(TypeAgentSendEmail, data)

	if err := worker.HandleAgentSendEmail(context.Background(), task); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if !worker.agentRepository.(*stubAgentRepo).updated {
		t.Fatalf("expected agent credentials update")
	}
	if len(worker.taskMonitor.(*stubTaskMonitor).responded) != 1 {
		t.Fatalf("expected RespondToTask to be called")
	}
	if len(worker.taskMonitor.(*stubTaskMonitor).resolved) != 1 {
		t.Fatalf("expected ResolveTask to be called")
	}
	if worker.agentRepository.(*stubAgentRepo).incremented == 0 {
		t.Fatalf("expected IncrementEmailCount to update count")
	}
	if worker.emailRepository.(*stubEmailRepo).created == nil {
		t.Fatalf("expected email record to be created")
	}
	if worker.conversationRepo.(*stubConversationRepo).created == nil || worker.conversationRepo.(*stubConversationRepo).created.ID != conversationID {
		t.Fatalf("expected conversation to be created with provided ID")
	}
}

func TestHandleAgentDailyReset(t *testing.T) {
	repo := &stubAgentRepo{}
	worker := &EmailWorker{agentRepository: repo}
	task := asynq.NewTask(TypeAgentDailyReset, nil)
	if err := worker.HandleAgentDailyReset(context.Background(), task); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !repo.resetCalled {
		t.Fatalf("expected reset to be called")
	}

	repoErr := &stubAgentRepo{resetErr: errors.New("boom")}
	workerErr := &EmailWorker{agentRepository: repoErr}
	if err := workerErr.HandleAgentDailyReset(context.Background(), task); err == nil {
		t.Fatalf("expected reset error")
	}
}

func TestGeneralEmailWorker_HandleSendEmail_Success(t *testing.T) {
	agent := sampleAgent()
	cred := map[string]interface{}{
		"token":         "tok",
		"refresh_token": "ref",
		"expiry":        time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}
	if err := agent.SetCredentials(cred); err != nil {
		t.Fatalf("failed to set credentials: %v", err)
	}

	taskID := uuid.New()
	payload := SendEmailPayload{
		AgentID: agent.ID,
		To:      "to@example.com",
		Subject: "Hi",
		Body:    "Hello",
		TaskID:  &taskID,
	}
	data, _ := jsonMarshal(payload)
	task := asynq.NewTask(TypeSendEmail, data)

	emailRepo := &stubEmailRepo{}
	taskRepo := &stubTaskRepo{findTask: &domain.Task{}}
	gmClient := &stubGmailClient{msg: &gmail.Message{ID: "mid", ThreadID: "thread"}}

	worker := &GeneralEmailWorker{
		oauth2Client:     &stubOAuth2Client{},
		agentRepo:        &stubAgentRepo{agent: agent},
		emailRepo:        emailRepo,
		taskRepo:         taskRepo,
		contactRepo:      &stubContactRepo{},
		conversationRepo: &stubConversationRepo{},
		gmailFactory: func(ctx context.Context, accessToken string) (gmailClient, error) {
			return gmClient, nil
		},
	}

	if err := worker.HandleSendEmail(context.Background(), task); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if emailRepo.created == nil {
		t.Fatalf("expected email to be recorded")
	}
	if len(taskRepo.updatedTasks) == 0 {
		t.Fatalf("expected task update for completion")
	}
}

func TestGeneralEmailWorker_HandleSendEmail_RefreshToken(t *testing.T) {
	agent := sampleAgent()
	cred := map[string]interface{}{
		"token":         "tok",
		"refresh_token": "ref",
		"expiry":        time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	}
	if err := agent.SetCredentials(cred); err != nil {
		t.Fatalf("failed to set credentials: %v", err)
	}

	payload := SendEmailPayload{
		AgentID: agent.ID,
		To:      "to@example.com",
		Subject: "Hi",
		Body:    "Hello",
	}
	data, _ := jsonMarshal(payload)
	task := asynq.NewTask(TypeSendEmail, data)

	emailRepo := &stubEmailRepo{}
	agentRepo := &stubAgentRepo{agent: agent}
	oauthStub := &stubOAuth2Client{token: &googleoauth.Token{
		AccessToken:  "new",
		RefreshToken: "newref",
		Expiry:       time.Now().Add(2 * time.Hour),
	}}
	gmClient := &stubGmailClient{msg: &gmail.Message{ID: "mid"}}

	worker := &GeneralEmailWorker{
		oauth2Client:     oauthStub,
		agentRepo:        agentRepo,
		emailRepo:        emailRepo,
		taskRepo:         &stubTaskRepo{},
		contactRepo:      &stubContactRepo{},
		conversationRepo: &stubConversationRepo{},
		gmailFactory: func(ctx context.Context, accessToken string) (gmailClient, error) {
			return gmClient, nil
		},
	}

	if err := worker.HandleSendEmail(context.Background(), task); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !agentRepo.updated {
		t.Fatalf("expected agent update after refresh")
	}
}

func TestGeneralEmailWorker_HandleSyncGmailHistory_SkipWhenMissing(t *testing.T) {
	worker := &GeneralEmailWorker{
		oauth2Client: &stubOAuth2Client{},
		agentRepo:    &stubAgentRepo{},
		gmailFactory: func(ctx context.Context, accessToken string) (gmailClient, error) {
			return &stubGmailClient{}, nil
		},
	}
	payload := SyncGmailHistoryPayload{}
	data, _ := jsonMarshal(payload)
	task := asynq.NewTask(TypeSyncGmailHistory, data)

	if err := worker.HandleSyncGmailHistory(context.Background(), task); err != nil {
		t.Fatalf("expected skip, got %v", err)
	}
}

func TestGeneralEmailWorker_HandleSyncGmailHistory_Process(t *testing.T) {
	agent := sampleAgent()
	agent.LastHistoryID = "h1"
	cred := map[string]interface{}{
		"token":         "tok",
		"refresh_token": "ref",
		"expiry":        time.Now().Add(time.Hour).Format(time.RFC3339),
	}
	_ = agent.SetCredentials(cred)

	emailRepo := &stubEmailRepo{}
	existingConversation := &domain.Conversation{Base: domain.Base{ID: uuid.New()}, GmailThreadID: "thread-1"}
	convoRepo := &stubConversationRepo{conversationID: existingConversation.ID, existing: existingConversation}
	gmClient := &stubGmailClient{
		msg: &gmail.Message{
			ID:       "mid-1",
			ThreadID: "thread-1",
			Headers: map[string]string{
				"From":    "a@example.com",
				"To":      "b@example.com",
				"Subject": "hello",
			},
			Body: "body",
		},
		history: &gmail.HistoryResult{
			Messages:      []gmail.HistoryMessage{{ID: "mid-1", ThreadID: "thread-1"}},
			NewestHistory: "h2",
			NextPageToken: "",
		},
	}

	worker := &GeneralEmailWorker{
		oauth2Client:     &stubOAuth2Client{},
		agentRepo:        &stubAgentRepo{agent: agent},
		emailRepo:        emailRepo,
		taskRepo:         &stubTaskRepo{},
		contactRepo:      &stubContactRepo{},
		conversationRepo: convoRepo,
		gmailFactory: func(ctx context.Context, accessToken string) (gmailClient, error) {
			return gmClient, nil
		},
	}

	payload := SyncGmailHistoryPayload{AgentID: agent.ID, StartHistoryID: "h1"}
	data, _ := jsonMarshal(payload)
	task := asynq.NewTask(TypeSyncGmailHistory, data)

	if err := worker.HandleSyncGmailHistory(context.Background(), task); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if emailRepo.created == nil {
		t.Fatalf("expected inbound email to be saved")
	}
	// Replied để reply_worker set sau khi phân loại intent (xem NOTE trong
	// handleInboundMessage) — bước sync không được set sớm.
	if len(convoRepo.setRepliedIDs) != 0 {
		t.Fatalf("sync must not set replied flag; reply_worker owns it")
	}
	if worker.agentRepo.(*stubAgentRepo).lastHistoryID != "h2" {
		t.Fatalf("expected last history id to be updated")
	}
}

func TestPersistInboundMessage_SkipCases(t *testing.T) {
	conversation := &domain.Conversation{Base: domain.Base{ID: uuid.New()}, GmailThreadID: "thread-1"}
	worker := &GeneralEmailWorker{
		emailRepo:        &stubEmailRepo{created: &domain.Email{}},
		conversationRepo: &stubConversationRepo{existing: conversation},
	}

	sentMsg := &gmail.Message{LabelIDs: []string{"SENT"}}
	if err := worker.persistInboundMessage(context.Background(), sampleAgent(), sentMsg); err != nil {
		t.Fatalf("expected sent message to be skipped, got %v", err)
	}

	existingMsg := &gmail.Message{ID: "mid-existing", ThreadID: conversation.GmailThreadID, LabelIDs: []string{}}
	if err := worker.persistInboundMessage(context.Background(), sampleAgent(), existingMsg); err != nil {
		t.Fatalf("expected existing message to be skipped, got %v", err)
	}
}

func TestFormatTemplateAndSanitize(t *testing.T) {
	worker := &EmailWorker{}
	result := worker.formatTemplate("Hello {{name}}", map[string]interface{}{"name": "World"})
	if result != "Hello World" {
		t.Fatalf("unexpected template result: %s", result)
	}

	cred := map[string]interface{}{
		"access_token":  "secret",
		"refresh_token": "secret",
		"password":      "hidden",
		"keep":          "ok",
	}
	sanitizeCredentialsForLogging(cred)
	if _, ok := cred["access_token"]; ok || len(cred) != 1 {
		t.Fatalf("sanitizeCredentialsForLogging did not remove sensitive fields")
	}
}

// --- helpers and stubs ---

func newEmailWorkerForTest() *EmailWorker {
	return &EmailWorker{
		agentRepository:    &stubAgentRepo{},
		campaignRepository: &stubCampaignRepo{},
		taskRepository:     &stubTaskRepo{},
		emailRepository:    &stubEmailRepo{},
		conversationRepo:   &stubConversationRepo{},
		googleAuthService:  &stubGoogleAuthService{},
		gmailService:       &stubGmailService{},
		taskMonitor:        &stubTaskMonitor{},
	}
}

func sampleAgent() *domain.Agent {
	return &domain.Agent{
		Base:            domain.Base{ID: uuid.New()},
		UserID:          uuid.New(),
		Email:           "agent@example.com",
		Credentials:     base.JSON([]byte(`{}`)),
		DailyLimit:      intPtr(10),
		EmailsSentToday: intPtr(0),
	}
}

func setupEmailDatabase(t *testing.T) *gorm.DB {
	db, err := pgtest.Open(t, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	stmts := []string{
		`CREATE TABLE contacts (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			name TEXT,
			email TEXT,
			profile TEXT,
			is_deleted BOOLEAN DEFAULT false,
		status TEXT,
		missing_fields TEXT,
		contact_information TEXT,
		industry TEXT,
			contact_channel TEXT,
			context_level TEXT,
			outreach_decision TEXT,
			scenario TEXT,
			message_draft TEXT,
			last_outcome TEXT,
			next_step TEXT,
			outreach_stage TEXT,
			followup_count INTEGER,
			meeting TEXT,
			business_stage TEXT
		);`,
		`CREATE TABLE templates (id TEXT PRIMARY KEY, user_id TEXT, subject TEXT, content TEXT, position REAL, campaign_id TEXT, type TEXT, category TEXT, cmetadata TEXT, send_after INTEGER, is_deleted BOOLEAN DEFAULT false);`,
		`CREATE TABLE campaigns (id TEXT PRIMARY KEY, agent_id TEXT, is_deleted BOOLEAN DEFAULT false);`,
		`CREATE TABLE tasks (id TEXT PRIMARY KEY, contact_id TEXT, campaign_id TEXT, template_id TEXT, status TEXT, error TEXT, is_deleted BOOLEAN DEFAULT false);`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to create schema: %v", err)
		}
	}
	return db
}

type stubAgentRepo struct {
	agent           *domain.Agent
	getErr          error
	updateErr       error
	incrementErr    error
	updated         bool
	incremented     int
	incrementCalled bool
	resetCalled     bool
	resetErr        error
	lastHistoryID   string
}

func (s *stubAgentRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.agent, nil
}

func (s *stubAgentRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error) {
	return s.GetByID(ctx, id)
}

func (s *stubAgentRepo) Update(ctx context.Context, id uuid.UUID, agent *domain.Agent) error {
	s.updated = true
	return s.updateErr
}

func (s *stubAgentRepo) IncrementEmailCount(ctx context.Context, agentID string, count int) error {
	s.incrementCalled = true
	s.incremented += count
	return s.incrementErr
}

func (s *stubAgentRepo) ResetDailyEmailCounts(ctx context.Context) error {
	s.resetCalled = true
	return s.resetErr
}

func (s *stubAgentRepo) UpdateLastHistoryID(ctx context.Context, agentID uuid.UUID, historyID string) error {
	s.lastHistoryID = historyID
	return nil
}

type stubCampaignRepo struct {
	campaign *domain.Campaign
	err      error
}

func (s *stubCampaignRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error) {
	return s.campaign, s.err
}

type stubTaskRepo struct {
	tasks         []*domain.Task
	err           error
	pendingCalled bool
	updateErr     error
	updatedTasks  []uuid.UUID
	findTask      *domain.Task
	findErr       error
}

func (s *stubTaskRepo) GetPendingTasksByIDs(ctx context.Context, taskIDs []uuid.UUID) ([]*domain.Task, error) {
	s.pendingCalled = true
	if s.err != nil {
		return nil, s.err
	}
	return s.tasks, nil
}

func (s *stubTaskRepo) Update(ctx context.Context, id uuid.UUID, task *domain.Task) error {
	s.updatedTasks = append(s.updatedTasks, id)
	return s.updateErr
}

func (s *stubTaskRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	return s.findTask, nil
}

type stubEmailRepo struct {
	created *domain.Email
	err     error
}

func (s *stubEmailRepo) Create(ctx context.Context, email *domain.Email) (*domain.Email, error) {
	s.created = email
	return email, s.err
}

func (s *stubEmailRepo) FindByGmailMessageID(ctx context.Context, gmailMessageID string) (*domain.Email, error) {
	return s.created, s.err
}

type stubConversationRepo struct {
	err            error
	conversationID uuid.UUID
	created        *domain.Conversation
	existing       *domain.Conversation
	setRepliedIDs  []uuid.UUID
}

func (s *stubConversationRepo) Create(ctx context.Context, conversation *domain.Conversation) (*domain.Conversation, error) {
	if s.conversationID != uuid.Nil {
		conversation.Base = domain.Base{ID: s.conversationID}
	}
	s.created = conversation
	return conversation, s.err
}

func (s *stubConversationRepo) FindByGmailThreadID(ctx context.Context, threadID string) (*domain.Conversation, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.existing, nil
}

func (s *stubConversationRepo) SetReplied(ctx context.Context, conversationID uuid.UUID, replied bool) error {
	s.setRepliedIDs = append(s.setRepliedIDs, conversationID)
	return s.err
}

func (s *stubConversationRepo) Update(ctx context.Context, id uuid.UUID, conversation *domain.Conversation) error {
	return s.err
}

type stubContactRepo struct {
	contact *domain.Contact
	err     error
}

func (s *stubContactRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Contact, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.contact, nil
}

func (s *stubContactRepo) FindByEmail(ctx context.Context, userID uuid.UUID, email string) (*domain.Contact, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.contact, nil
}

type stubOAuth2Client struct {
	token *googleoauth.Token
	err   error
}

func (s *stubOAuth2Client) RefreshToken(ctx context.Context, refreshToken string) (*googleoauth.Token, error) {
	return s.token, s.err
}

type stubGmailClient struct {
	msg     *gmail.Message
	history *gmail.HistoryResult
	err     error
}

func (s *stubGmailClient) SendMessage(ctx context.Context, req *gmail.SendMessageRequest) (*gmail.Message, error) {
	return s.msg, s.err
}

func (s *stubGmailClient) GetMessage(ctx context.Context, messageID string) (*gmail.Message, error) {
	return s.msg, s.err
}

func (s *stubGmailClient) ListHistory(ctx context.Context, startHistoryID string, pageToken string) (*gmail.HistoryResult, error) {
	if s.history != nil {
		return s.history, s.err
	}
	return &gmail.HistoryResult{}, s.err
}

func (s *stubGmailClient) Close() {}

type stubGoogleAuthService struct {
	refreshed map[string]interface{}
	err       error
}

func (s *stubGoogleAuthService) Refresh(ctx context.Context, credentials map[string]interface{}, scopes []string) (map[string]interface{}, error) {
	return s.refreshed, s.err
}

type stubGmailService struct {
	msg *GmailMessage
	err error
}

func (s *stubGmailService) SendEmail(ctx context.Context, sender string, to []string, subject string, body map[string]string) (*GmailMessage, error) {
	return s.msg, s.err
}

type stubTaskMonitor struct {
	responded []string
	resolved  []string
	errResp   error
	errRes    error
}

func (s *stubTaskMonitor) RespondToTask(ctx context.Context, taskIDs []string) error {
	s.responded = append(s.responded, taskIDs...)
	return s.errResp
}

func (s *stubTaskMonitor) ResolveTask(ctx context.Context, taskIDs []string) error {
	s.resolved = append(s.resolved, taskIDs...)
	return s.errRes
}

func intPtr(v int) *int {
	return &v
}

// jsonMarshal is split out to avoid importing encoding/json in tests repeatedly
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
