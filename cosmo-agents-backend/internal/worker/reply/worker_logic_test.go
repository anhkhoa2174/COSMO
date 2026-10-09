package reply

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	intentService "github.com/rockship/cosmo-agents-go/internal/service/intent"
)

func TestProcessTask_Success(t *testing.T) {
	emailID := uuid.New()
	convID := uuid.New()
	campaignID := uuid.New()

	emailRepo := &stubEmailRepo{
		email: &domain.Email{Base: domain.Base{ID: emailID}, ConversationID: &convID, CampaignID: &campaignID, Content: "hello"},
	}
	convRepo := &stubConversationRepo{
		conversation: &domain.Conversation{Base: domain.Base{ID: convID}},
	}
	campaignRepo := &stubCampaignRepo{campaign: &domain.Campaign{Base: domain.Base{ID: campaignID}, Status: domain.CampaignStatusActive}}
	classifier := &stubClassifier{intent: domain.IntentInterested}
	handler := &stubHandler{ok: true}
	factory := &stubHandlerFactory{handler: handler}
	log := zerolog.New(nil)

	worker := New(emailRepo, convRepo, campaignRepo, nil, classifier, factory, nil, &log)

	payload := HandleReplyPayload{EmailID: emailID}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeHandleReply, data)

	if err := worker.ProcessTask(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !handler.executed || !emailRepo.updated || !convRepo.updated {
		t.Fatalf("expected handler execute and repos updated")
	}
}

func TestProcessTask_SkipsHandledConversation(t *testing.T) {
	emailID := uuid.New()
	convID := uuid.New()
	campaignID := uuid.New()

	emailRepo := &stubEmailRepo{
		email: &domain.Email{Base: domain.Base{ID: emailID}, ConversationID: &convID, CampaignID: &campaignID, Content: "hello"},
	}
	convRepo := &stubConversationRepo{
		conversation: &domain.Conversation{
			Base:    domain.Base{ID: convID},
			Replied: true,
			// Skip chỉ khi đã có draft AI thật sự (draft rỗng thì phải làm lại)
			CMetadata: base.JSONB(`{"ai_reply":{"draft_content":"Hi, thanks for reaching out!"}}`),
		},
	}
	campaignRepo := &stubCampaignRepo{campaign: &domain.Campaign{Base: domain.Base{ID: campaignID}, Status: domain.CampaignStatusActive}}
	classifier := &stubClassifier{intent: domain.IntentInterested}
	factory := &stubHandlerFactory{handler: &stubHandler{ok: true}}
	log := zerolog.New(nil)

	worker := New(emailRepo, convRepo, campaignRepo, nil, classifier, factory, nil, &log)

	payload := HandleReplyPayload{EmailID: emailID}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeHandleReply, data)

	if err := worker.ProcessTask(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if convRepo.updated {
		t.Fatalf("expected no updates when conversation already has an AI draft")
	}
}

func TestProcessTask_HandlerMissing(t *testing.T) {
	emailID := uuid.New()
	convID := uuid.New()
	campaignID := uuid.New()

	emailRepo := &stubEmailRepo{
		email: &domain.Email{Base: domain.Base{ID: emailID}, ConversationID: &convID, CampaignID: &campaignID, Content: "hello"},
	}
	convRepo := &stubConversationRepo{
		conversation: &domain.Conversation{Base: domain.Base{ID: convID}},
	}
	campaignRepo := &stubCampaignRepo{campaign: &domain.Campaign{Base: domain.Base{ID: campaignID}, Status: domain.CampaignStatusActive}}
	classifier := &stubClassifier{intent: domain.IntentInterested}
	factory := &stubHandlerFactory{handler: nil}
	log := zerolog.New(nil)

	worker := New(emailRepo, convRepo, campaignRepo, nil, classifier, factory, nil, &log)

	payload := HandleReplyPayload{EmailID: emailID}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeHandleReply, data)

	if err := worker.ProcessTask(context.Background(), task); err == nil {
		t.Fatalf("expected error when handler missing")
	}
}

func TestProcessTask_HandlerUnsuccessful(t *testing.T) {
	emailID := uuid.New()
	convID := uuid.New()
	campaignID := uuid.New()

	emailRepo := &stubEmailRepo{
		email: &domain.Email{Base: domain.Base{ID: emailID}, ConversationID: &convID, CampaignID: &campaignID, Content: "hello"},
	}
	convRepo := &stubConversationRepo{
		conversation: &domain.Conversation{Base: domain.Base{ID: convID}},
	}
	campaignRepo := &stubCampaignRepo{campaign: &domain.Campaign{Base: domain.Base{ID: campaignID}, Status: domain.CampaignStatusActive}}
	classifier := &stubClassifier{intent: domain.IntentInterested}
	handler := &stubHandler{ok: false}
	factory := &stubHandlerFactory{handler: handler}
	log := zerolog.New(nil)

	worker := New(emailRepo, convRepo, campaignRepo, nil, classifier, factory, nil, &log)

	payload := HandleReplyPayload{EmailID: emailID}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeHandleReply, data)

	if err := worker.ProcessTask(context.Background(), task); err == nil {
		t.Fatalf("expected error when handler returns unsuccessful")
	}
}

// ---- stubs ----

type stubEmailRepo struct {
	email   *domain.Email
	updated bool
}

func (s *stubEmailRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Email, error) {
	return s.email, nil
}

func (s *stubEmailRepo) Update(ctx context.Context, id uuid.UUID, email *domain.Email) error {
	s.updated = true
	return nil
}

type stubConversationRepo struct {
	conversation *domain.Conversation
	updated      bool
}

func (s *stubConversationRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Conversation, error) {
	return s.conversation, nil
}

func (s *stubConversationRepo) Update(ctx context.Context, id uuid.UUID, conversation *domain.Conversation) error {
	s.updated = true
	return nil
}

type stubCampaignRepo struct {
	campaign *domain.Campaign
}

func (s *stubCampaignRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Campaign, error) {
	return s.campaign, nil
}

type stubClassifier struct {
	intent domain.IntentType
}

func (s *stubClassifier) Classify(ctx context.Context, content string) (domain.IntentType, error) {
	return s.intent, nil
}

func (s *stubClassifier) ClassifyDetailed(ctx context.Context, content string) (intentService.DetailedIntent, error) {
	return intentService.DetailedIntent{Intent: s.intent, Confidence: 0.9, Reasoning: "stub"}, nil
}

type stubHandlerFactory struct {
	handler intentService.IntentHandler
}

func (s *stubHandlerFactory) Build(campaign *domain.Campaign, intent domain.IntentType) (intentService.IntentHandler, error) {
	return s.handler, nil
}

type stubHandler struct {
	ok       bool
	executed bool
}

func (s *stubHandler) Execute(ctx context.Context, campaign *domain.Campaign, intent domain.IntentType, email *domain.Email) (bool, error) {
	s.executed = true
	return s.ok, nil
}

// Ensure stubs satisfy interfaces
var _ emailRepository = (*stubEmailRepo)(nil)
var _ conversationRepository = (*stubConversationRepo)(nil)
var _ campaignRepository = (*stubCampaignRepo)(nil)
var _ intentClassifier = (*stubClassifier)(nil)
var _ intentHandlerFactory = (*stubHandlerFactory)(nil)
