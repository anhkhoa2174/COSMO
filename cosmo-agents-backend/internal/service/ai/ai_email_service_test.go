package ai

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/repository/email"
	"github.com/rockship/cosmo-agents-go/internal/repository/knowledge"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
)

func TestMapIntentToResponse(t *testing.T) {
	tests := []struct {
		name     string
		intent   domain.IntentType
		expected EmailIntentResult
	}{
		{
			name:     "interested intent",
			intent:   domain.IntentInterested,
			expected: EmailIntentResult{ID: 1, Intent: "Interested"},
		},
		{
			name:     "not interested intent",
			intent:   domain.IntentNotInterested,
			expected: EmailIntentResult{ID: 2, Intent: "Not interested"},
		},
		{
			name:     "request for pricing intent",
			intent:   domain.IntentRequestForPricing,
			expected: EmailIntentResult{ID: 3, Intent: "Request for pricing"},
		},
		{
			name:     "request for info intent",
			intent:   domain.IntentRequestForInfo,
			expected: EmailIntentResult{ID: 4, Intent: "Request for information"},
		},
		{
			name:     "do not contact intent",
			intent:   domain.IntentDoNotContact,
			expected: EmailIntentResult{ID: 5, Intent: "Do not contact"},
		},
		{
			name:     "out of office intent",
			intent:   domain.IntentOutOfOffice,
			expected: EmailIntentResult{ID: 6, Intent: "Out of office"},
		},
		{
			name:     "unknown intent",
			intent:   domain.IntentType("unknown"),
			expected: EmailIntentResult{ID: 7, Intent: "Unknown intent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapIntentToResponse(tt.intent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildReplySubject(t *testing.T) {
	tests := []struct {
		name     string
		original string
		expected string
	}{
		{
			name:     "plain subject",
			original: "Hello World",
			expected: "Re: Hello World",
		},
		{
			name:     "already has Re:",
			original: "Re: Hello World",
			expected: "Re: Hello World",
		},
		{
			name:     "empty subject",
			original: "",
			expected: "Re: your message",
		},
		{
			name:     "whitespace only",
			original: "   ",
			expected: "Re: your message",
		},
		{
			name:     "already has re: (case insensitive)",
			original: "RE: Hello World",
			expected: "RE: Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildReplySubject(tt.original)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractReplyJSON(t *testing.T) {
	tests := []struct {
		name            string
		response        string
		expectedSubject string
		expectedBody    string
	}{
		{
			name: "valid JSON response",
			response: `{
				"subject": "Test Subject",
				"body": "Test Body"
			}`,
			expectedSubject: "Test Subject",
			expectedBody:    "Test Body",
		},
		{
			name:            "JSON embedded in text",
			response:        "Here's the response: {\"subject\": \"Test\", \"body\": \"Content\"}",
			expectedSubject: "Test",
			expectedBody:    "Content",
		},
		{
			name:            "invalid JSON",
			response:        "Invalid JSON response",
			expectedSubject: "",
			expectedBody:    "",
		},
		{
			name:            "empty response",
			response:        "",
			expectedSubject: "",
			expectedBody:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject, body := extractReplyJSON(tt.response)
			assert.Equal(t, tt.expectedSubject, subject)
			assert.Equal(t, tt.expectedBody, body)
		})
	}
}

func TestBuildThreadContext(t *testing.T) {
	thread := []ConversationMessage{
		{
			FromEmail: "sender@example.com",
			ToEmail:   "user@example.com",
			Subject:   "Test Subject 1",
			Content:   "Test content 1",
		},
		{
			FromEmail: "user@example.com",
			ToEmail:   "sender@example.com",
			Subject:   "Re: Test Subject 1",
			Content:   "Test content 2",
		},
	}

	result := buildThreadContext(thread)

	assert.Contains(t, result, "sender@example.com")
	assert.Contains(t, result, "user@example.com")
	assert.Contains(t, result, "Test Subject 1")
	assert.Contains(t, result, "Test content 1")
	assert.Contains(t, result, "Test content 2")
	assert.Contains(t, result, "[Email #1]")
	assert.Contains(t, result, "[Email #2]")
}

// Test helper functions and error scenarios
func TestAIEmailService_ConstantValidation(t *testing.T) {
	// Test that error constants are defined correctly
	assert.Equal(t, "ai email service: openai client not configured", ErrAIEmailClientNotConfigured.Error())
	assert.Equal(t, "conversation not found", ErrConversationNotFound.Error())
	assert.Equal(t, "either conversation_id or conversation must be provided", ErrInvalidConversation.Error())
	assert.Equal(t, "the last email in conversation was not addressed to the current user", ErrLastEmailNotAddressedToUser.Error())
}

func TestAIEmailService_StructureValidation(t *testing.T) {
	// Test that domain structures can be created and used correctly
	conversationMessage := ConversationMessage{
		FromEmail: "test@example.com",
		ToEmail:   "user@example.com",
		Subject:   "Test Subject",
		Content:   "Test content",
	}

	assert.Equal(t, "test@example.com", conversationMessage.FromEmail)
	assert.Equal(t, "user@example.com", conversationMessage.ToEmail)
	assert.Equal(t, "Test Subject", conversationMessage.Subject)
	assert.Equal(t, "Test content", conversationMessage.Content)

	aiReplyEmail := AIReplyEmail{
		FromEmail: "user@example.com",
		ToEmail:   "test@example.com",
		Subject:   "Re: Test Subject",
		Content:   "Test reply content",
		Type:      "Reply",
	}

	assert.Equal(t, "user@example.com", aiReplyEmail.FromEmail)
	assert.Equal(t, "test@example.com", aiReplyEmail.ToEmail)
	assert.Equal(t, "Re: Test Subject", aiReplyEmail.Subject)
	assert.Equal(t, "Test reply content", aiReplyEmail.Content)
	assert.Equal(t, "Reply", aiReplyEmail.Type)

	intentResult := EmailIntentResult{
		ID:     1,
		Intent: "Interested",
	}

	assert.Equal(t, 1, intentResult.ID)
	assert.Equal(t, "Interested", intentResult.Intent)

	aiTemplate := AIGeneratedTemplate{
		FromEmail: "user@example.com",
		ToEmail:   "test@example.com",
		Subject:   "Test Template Subject",
		Content:   "Test template content",
		Type:      "First Email",
	}

	assert.Equal(t, "user@example.com", aiTemplate.FromEmail)
	assert.Equal(t, "test@example.com", aiTemplate.ToEmail)
	assert.Equal(t, "Test Template Subject", aiTemplate.Subject)
	assert.Equal(t, "Test template content", aiTemplate.Content)
	assert.Equal(t, "First Email", aiTemplate.Type)
}

func TestAIEmailService_ResolveConversationErrors(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name               string
		userID             uuid.UUID
		conversationID     *uuid.UUID
		manualConversation []ConversationMessage
		expectedError      error
	}{
		{
			name:               "invalid conversation parameters - both nil",
			userID:             userID,
			conversationID:     nil,
			manualConversation: []ConversationMessage{},
			expectedError:      ErrInvalidConversation,
		},
		{
			name:               "invalid conversation parameters - empty manual",
			userID:             userID,
			conversationID:     nil,
			manualConversation: []ConversationMessage{},
			expectedError:      ErrInvalidConversation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &AIEmailService{} // Create with nil dependencies for error testing

			result, err := service.resolveConversation(context.Background(), tt.userID, tt.conversationID, tt.manualConversation)

			assert.Error(t, err)
			assert.ErrorIs(t, err, tt.expectedError)
			assert.Nil(t, result)
		})
	}
}

func TestAIEmailService_ClassifyIntentError(t *testing.T) {
	// Test the case where intent classifier is not configured
	service := &AIEmailService{} // Create with nil dependencies

	result, err := service.ClassifyIntent(context.Background(), "test content")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIEmailClientNotConfigured)
	assert.Nil(t, result)
}

func TestAIEmailService_GenerateReplyError(t *testing.T) {
	// Test the case where OpenAI client or intent classifier is not configured
	userID := uuid.New()
	conversationID := uuid.New()
	userEmail := "user@example.com"

	service := &AIEmailService{} // Create with nil dependencies

	result, err := service.GenerateReply(context.Background(), userID, userEmail, &conversationID, nil)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAIEmailClientNotConfigured)
	assert.Nil(t, result)
}

func TestAIEmailService_GenerateDeprecatedOutreach(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name       string
		userID     uuid.UUID
		campaign   string
		clientData map[string]interface{}
		expected   []AIGeneratedTemplate
	}{
		{
			name:     "with client name",
			userID:   userID,
			campaign: "test campaign",
			clientData: map[string]interface{}{
				"name": "John Doe",
			},
			expected: []AIGeneratedTemplate{
				{
					FromEmail: "",
					ToEmail:   "",
					Type:      "First Email",
					Subject:   "Quick intro for John Doe",
					Content:   "Hi John Doe,\n\nHope you're doing well. Wanted to share a short overview of how we can help. Let me know if you have a few minutes this week to chat.\n\nBest,\n",
				},
				{
					FromEmail: "",
					ToEmail:   "",
					Type:      "Follow-up Email 1",
					Subject:   "Just checking in",
					Content:   "Hi John Doe,\n\nWanted to bump this to the top of your inbox in case it got lost. Happy to send more context or schedule time.\n\nBest,\n",
				},
				{
					FromEmail: "",
					ToEmail:   "",
					Type:      "Follow-up Email 2",
					Subject:   "Should I keep this on the radar?",
					Content:   "Hi John Doe,\n\nTotally understand if now isn't the right time. Let me know if you'd prefer I reach back out later in the year. Either way, thanks for taking a look!\n\nBest,\n",
				},
			},
		},
		{
			name:     "without client name",
			userID:   userID,
			campaign: "test campaign",
			clientData: map[string]interface{}{
				"name": "",
			},
			expected: []AIGeneratedTemplate{
				{
					FromEmail: "",
					ToEmail:   "",
					Type:      "First Email",
					Subject:   "Quick intro for there",
					Content:   "Hi there,\n\nHope you're doing well. Wanted to share a short overview of how we can help. Let me know if you have a few minutes this week to chat.\n\nBest,\n",
				},
				{
					FromEmail: "",
					ToEmail:   "",
					Type:      "Follow-up Email 1",
					Subject:   "Just checking in",
					Content:   "Hi there,\n\nWanted to bump this to the top of your inbox in case it got lost. Happy to send more context or schedule time.\n\nBest,\n",
				},
				{
					FromEmail: "",
					ToEmail:   "",
					Type:      "Follow-up Email 2",
					Subject:   "Should I keep this on the radar?",
					Content:   "Hi there,\n\nTotally understand if now isn't the right time. Let me know if you'd prefer I reach back out later in the year. Either way, thanks for taking a look!\n\nBest,\n",
				},
			},
		},
		{
			name:     "nil client name",
			userID:   userID,
			campaign: "test campaign",
			clientData: map[string]interface{}{
				"name": nil,
			},
			expected: []AIGeneratedTemplate{
				{
					FromEmail: "",
					ToEmail:   "",
					Type:      "First Email",
					Subject:   "Quick intro for there",
					Content:   "Hi there,\n\nHope you're doing well. Wanted to share a short overview of how we can help. Let me know if you have a few minutes this week to chat.\n\nBest,\n",
				},
				{
					FromEmail: "",
					ToEmail:   "",
					Type:      "Follow-up Email 1",
					Subject:   "Just checking in",
					Content:   "Hi there,\n\nWanted to bump this to the top of your inbox in case it got lost. Happy to send more context or schedule time.\n\nBest,\n",
				},
				{
					FromEmail: "",
					ToEmail:   "",
					Type:      "Follow-up Email 2",
					Subject:   "Should I keep this on the radar?",
					Content:   "Hi there,\n\nTotally understand if now isn't the right time. Let me know if you'd prefer I reach back out later in the year. Either way, thanks for taking a look!\n\nBest,\n",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &AIEmailService{}

			result, err := service.GenerateDeprecatedOutreach(context.Background(), tt.userID, tt.campaign, tt.clientData)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func newTestEmailDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	db = db.Session(&gorm.Session{AllowGlobalUpdate: true})
	require.NoError(t, db.AutoMigrate(&domain.Email{}, &domain.User{}, &domain.Knowledge{}))
	return db
}

func TestAIEmailService_ResolveConversationAndKnowledge(t *testing.T) {
	db := newTestEmailDB(t)
	emailRepo := email.NewEmailRepository(db)
	kRepo := knowledge.NewKnowledgeRepository(db)
	userRepo := userRepo.NewUserRepository(db)

	ctx := context.Background()
	userID := uuid.New()

	createdUser, err := userRepo.Create(ctx, &domain.User{Base: domain.Base{ID: userID}, Email: "user@example.com", Name: "User", Provider: "google"})
	require.NoError(t, err)
	require.NotNil(t, createdUser)

	conversationID := uuid.New()
	// newest first (created_at desc)
	emailNew := &domain.Email{
		ConversationID: &conversationID,
		UserID:         userID,
		FromEmail:      "sender@example.com",
		ToEmail:        "user@example.com",
		Subject:        "New",
		Content:        "Latest content",
	}
	emailOld := &domain.Email{
		ConversationID: &conversationID,
		UserID:         userID,
		FromEmail:      "user@example.com",
		ToEmail:        "sender@example.com",
		Subject:        "Old",
		Content:        "Old content",
	}
	_, err = emailRepo.Create(ctx, emailOld)
	require.NoError(t, err)
	require.NoError(t, db.Model(&domain.Email{}).Where("subject = ?", "Old").Update("created_at", time.Now().Add(-time.Hour)).Error)
	_, err = emailRepo.Create(ctx, emailNew)
	require.NoError(t, err)

	k1 := &domain.Knowledge{
		UserID:      userID,
		SummaryPair: []string{"short", "long summary"},
	}
	require.NoError(t, db.Create(k1).Error)

	service := &AIEmailService{
		emailRepo:     emailRepo,
		knowledgeRepo: kRepo,
		userRepo:      userRepo,
	}

	thread, err := service.resolveConversation(ctx, userID, &conversationID, nil)
	require.NoError(t, err)
	require.Len(t, thread, 2)
	assert.Equal(t, "New", thread[0].Subject)

	knowledgeCtx, err := service.buildKnowledgeContext(ctx, userID, "Body", "Interested")
	require.NoError(t, err)
	assert.Contains(t, knowledgeCtx, "long summary")
}

func TestAIEmailService_GenerateReplyWithOpenAI(t *testing.T) {
	client := newPatchedAIClient(t, `{"choices":[{"message":{"content":"{\"subject\":\"Hello\",\"body\":\"World\"}"}}]}`)
	service := &AIEmailService{openAIClient: client}
	user := &domain.User{Email: "user@example.com", Name: "User Name"}
	thread := []ConversationMessage{
		{FromEmail: "from@example.com", ToEmail: "user@example.com", Subject: "Hi", Content: "Body"},
	}
	body, subject, err := service.generateReplyWithOpenAI(context.Background(), user, thread, EmailIntentResult{Intent: "Interested"}, "knowledge", "Hi", "from@example.com")
	require.NoError(t, err)
	assert.Equal(t, "Hello", subject)
	assert.Equal(t, "World", body)
}

type staticHTTPClient struct {
	body string
}

func (c staticHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(c.body)),
		Request:    req,
	}, nil
}

// newPatchedAIClient returns an OpenAI client that always responds with the provided JSON payload.
// It avoids real network calls by injecting a static HTTP client.
func newPatchedAIClient(t *testing.T, response string) *ai.OpenAIClient {
	t.Helper()
	client := ai.NewOpenAIClient(ai.Config{APIKey: "test-key", Model: "test-model"})

	// Patch underlying OpenAI client to avoid network calls.
	clientField := reflect.ValueOf(client).Elem().FieldByName("client")
	clientPtr := reflect.NewAt(clientField.Type(), unsafe.Pointer(clientField.UnsafeAddr())).Elem()
	oaClient := clientPtr.Interface().(*openai.Client)

	cfgField := reflect.ValueOf(oaClient).Elem().FieldByName("config")
	cfgPtr := reflect.NewAt(cfgField.Type(), unsafe.Pointer(cfgField.UnsafeAddr())).Elem()
	cfg := cfgPtr.Interface().(openai.ClientConfig)
	cfg.BaseURL = "http://example.com"
	cfg.HTTPClient = staticHTTPClient{body: response}
	cfgPtr.Set(reflect.ValueOf(cfg))

	return client
}
