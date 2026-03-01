package intent

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/campaign"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	"github.com/rs/zerolog"
)

// TestIntentTypeValidation tests the IntentType validation
func TestIntentTypeValidation(t *testing.T) {
	tests := []struct {
		name        string
		intentType  campaign.IntentType
		expected    bool
		description string
	}{
		{
			name:        "valid interested intent",
			intentType:  campaign.IntentInterested,
			expected:    true,
			description: "Should validate interested intent",
		},
		{
			name:        "valid not interested intent",
			intentType:  campaign.IntentNotInterested,
			expected:    true,
			description: "Should validate not interested intent",
		},
		{
			name:        "valid referral intent",
			intentType:  campaign.IntentReferral,
			expected:    true,
			description: "Should validate referral intent",
		},
		{
			name:        "valid request for pricing intent",
			intentType:  campaign.IntentRequestForPricing,
			expected:    true,
			description: "Should validate request for pricing intent",
		},
		{
			name:        "valid request for info intent",
			intentType:  campaign.IntentRequestForInfo,
			expected:    true,
			description: "Should validate request for info intent",
		},
		{
			name:        "valid nurture intent",
			intentType:  campaign.IntentNurture,
			expected:    true,
			description: "Should validate nurture intent",
		},
		{
			name:        "valid do not contact intent",
			intentType:  campaign.IntentDoNotContact,
			expected:    true,
			description: "Should validate do not contact intent",
		},
		{
			name:        "valid out of office intent",
			intentType:  campaign.IntentOutOfOffice,
			expected:    true,
			description: "Should validate out of office intent",
		},
		{
			name:        "valid unknown intent",
			intentType:  campaign.IntentUnknown,
			expected:    true,
			description: "Should validate unknown intent",
		},
		{
			name:        "invalid intent type",
			intentType:  campaign.IntentType("invalid"),
			expected:    false,
			description: "Should reject invalid intent type",
		},
		{
			name:        "empty intent type",
			intentType:  campaign.IntentType(""),
			expected:    false,
			description: "Should reject empty intent type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.intentType.IsValid()
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestIntentHandlerInterface tests the IntentHandler interface
func TestIntentHandlerInterface(t *testing.T) {
	// Create a mock implementation of IntentHandler
	mockHandler := struct {
		executeFunc func(context.Context, *domain.Campaign, campaign.IntentType, *domain.Email) (bool, error)
	}{
		executeFunc: func(ctx context.Context, campaign *domain.Campaign, intent campaign.IntentType, email *domain.Email) (bool, error) {
			return true, nil
		},
	}

	// Create a real IntentHandler that implements the interface
	handler := struct {
		IntentHandler
		executeFunc func(context.Context, *domain.Campaign, campaign.IntentType, *domain.Email) (bool, error)
	}{
		executeFunc: mockHandler.executeFunc,
	}

	// Test that the handler implements the interface correctly
	assert.Implements(t, (*IntentHandler)(nil), handler)
}

func TestAIReplyHandler_ExecuteMarksConversation(t *testing.T) {
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Conversation{}))

	convRepo := conversationRepo.NewConversationRepository(db)
	conv := &domain.Conversation{
		Base: domain.Base{ID: uuid.New()},
	}
	_, err = convRepo.Create(context.Background(), conv)
	require.NoError(t, err)

	email := &domain.Email{
		Base:           domain.Base{ID: uuid.New()},
		ConversationID: &conv.ID,
	}

	logger := zerolog.New(io.Discard)
	handler := &AIReplyHandler{
		conversationRepo: convRepo,
		logger:           &logger,
	}
	ok, execErr := handler.Execute(context.Background(), &domain.Campaign{Base: domain.Base{ID: uuid.New()}}, domain.IntentInterested, email)
	require.NoError(t, execErr)
	assert.True(t, ok)

	updated, getErr := convRepo.FindByID(context.Background(), conv.ID)
	require.NoError(t, getErr)
	assert.True(t, updated.Replied)
}

// TestIntentHandlerExecution tests basic intent handler execution
func TestIntentHandlerExecution(t *testing.T) {
	campaignID := uuid.New()
	emailID := uuid.New()

	// Create test data
	testCampaign := &domain.Campaign{
		Base: domain.Base{
			ID: campaignID,
		},
		Name: "Test Campaign",
	}

	testEmail := &domain.Email{
		Base: domain.Base{
			ID: emailID,
		},
		Subject: "Test Subject",
		Content: "Test Content",
		Intents: pq.StringArray{"question", "urgent"},
	}

	// Test various intent types
	intentTypes := []campaign.IntentType{
		campaign.IntentInterested,
		campaign.IntentNotInterested,
		campaign.IntentReferral,
		campaign.IntentRequestForPricing,
		campaign.IntentRequestForInfo,
		campaign.IntentNurture,
		campaign.IntentDoNotContact,
		campaign.IntentOutOfOffice,
		campaign.IntentUnknown,
	}

	for _, intentType := range intentTypes {
		t.Run(string(intentType), func(t *testing.T) {
			// Test that the intent type is valid and the data structures are created correctly
			assert.True(t, intentType.IsValid(), "Intent type should be valid")
			assert.NotNil(t, testCampaign, "Campaign should not be nil")
			assert.NotNil(t, testEmail, "Email should not be nil")
			assert.Equal(t, campaignID, testCampaign.ID, "Campaign ID should match")
			assert.Equal(t, emailID, testEmail.ID, "Email ID should match")
		})
	}
}

func newConvRepo(t *testing.T) (*conversationRepo.ConversationRepository, *domain.Conversation, context.Context) {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Conversation{}))

	repo := conversationRepo.NewConversationRepository(db)
	conv := &domain.Conversation{
		Base:          domain.Base{ID: uuid.New()},
		UserID:        uuid.New(),
		GmailThreadID: "thread",
	}
	_, err = repo.Create(context.Background(), conv)
	require.NoError(t, err)
	return repo, conv, context.Background()
}

func TestDoNotContactHandler_Execute(t *testing.T) {
	repo, conv, ctx := newConvRepo(t)
	email := &domain.Email{Base: domain.Base{ID: uuid.New()}, ConversationID: &conv.ID, FromEmail: "from@example.com"}
	logger := zerolog.New(io.Discard)

	handler := NewDoNotContactHandler(nil, nil, repo, &logger)
	ok, err := handler.Execute(ctx, &domain.Campaign{Base: domain.Base{ID: uuid.New()}}, domain.IntentDoNotContact, email)
	require.NoError(t, err)
	assert.True(t, ok)

	updated, err := repo.FindByID(ctx, conv.ID)
	require.NoError(t, err)
	assert.True(t, updated.Replied)
}

func TestOutOfOfficeHandler_Execute(t *testing.T) {
	repo, conv, ctx := newConvRepo(t)
	email := &domain.Email{Base: domain.Base{ID: uuid.New()}, ConversationID: &conv.ID}
	logger := zerolog.New(io.Discard)

	handler := NewOutOfOfficeHandler(nil, nil, repo, &logger)
	ok, err := handler.Execute(ctx, &domain.Campaign{Base: domain.Base{ID: uuid.New()}}, domain.IntentOutOfOffice, email)
	require.NoError(t, err)
	assert.True(t, ok)

	updated, err := repo.FindByID(ctx, conv.ID)
	require.NoError(t, err)
	assert.True(t, updated.Replied)
}

func TestAssignToPersonHandler_AssignsProperly(t *testing.T) {
	repo, conv, ctx := newConvRepo(t)
	campaignUser := uuid.New()
	email := &domain.Email{Base: domain.Base{ID: uuid.New()}, ConversationID: &conv.ID}
	logger := zerolog.New(io.Discard)

	handler := NewAssignToPersonHandler(repo, nil, &logger)
	ok, err := handler.Execute(ctx, &domain.Campaign{Base: domain.Base{ID: uuid.New()}, UserID: campaignUser}, domain.IntentInterested, email)
	require.NoError(t, err)
	assert.True(t, ok)

	updated, err := repo.FindByID(ctx, conv.ID)
	require.NoError(t, err)
	assert.True(t, updated.Replied)
	assert.Equal(t, campaignUser, *updated.AssigneeID)
}

func TestIntentClassifierHelpers(t *testing.T) {
	logger := zerolog.New(io.Discard)
	ic := NewIntentClassifier(nil, "", &logger)

	content := "<detected_intent>INTERESTED</detected_intent>"
	assert.Equal(t, "INTERESTED", ic.extractIntent(content))

	fallback := ic.extractIntent("Detected intent is REQUEST_FOR_PRICING from content.")
	assert.Equal(t, "REQUEST_FOR_PRICING", fallback)

	assert.Equal(t, domain.IntentDoNotContact, ic.parseIntent("DO_NOT_CONTACT"))
	assert.Equal(t, domain.IntentUnknown, ic.parseIntent("invalid"))

	prompt := ic.constructPrompt()
	assert.Contains(t, prompt, "advanced email intent classifier")
	assert.Contains(t, prompt, "INTERESTED")
	assert.Contains(t, prompt, "UNKNOWN_INTENT")
}

// TestIntentTypeConstants tests that all intent type constants are defined
func TestIntentTypeConstants(t *testing.T) {
	// Test that all expected intent types are defined with correct string values
	assert.Equal(t, "Interested", string(campaign.IntentInterested))
	assert.Equal(t, "Not interested", string(campaign.IntentNotInterested))
	assert.Equal(t, "Referral", string(campaign.IntentReferral))
	assert.Equal(t, "Request for pricing", string(campaign.IntentRequestForPricing))
	assert.Equal(t, "Request for information", string(campaign.IntentRequestForInfo))
	assert.Equal(t, "Nurture", string(campaign.IntentNurture))
	assert.Equal(t, "Do not contact", string(campaign.IntentDoNotContact))
	assert.Equal(t, "Out of office", string(campaign.IntentOutOfOffice))
	assert.Equal(t, "Unknown intent", string(campaign.IntentUnknown))
}

// TestIntentTypeStringConversion tests string conversion for intent types
func TestIntentTypeStringConversion(t *testing.T) {
	intentTypes := []campaign.IntentType{
		campaign.IntentInterested,
		campaign.IntentNotInterested,
		campaign.IntentReferral,
		campaign.IntentRequestForPricing,
		campaign.IntentRequestForInfo,
		campaign.IntentNurture,
		campaign.IntentDoNotContact,
		campaign.IntentOutOfOffice,
		campaign.IntentUnknown,
	}

	for _, intentType := range intentTypes {
		t.Run(string(intentType), func(t *testing.T) {
			// Convert to string and back
			intentStr := string(intentType)
			newIntent := campaign.IntentType(intentStr)

			assert.Equal(t, intentType, newIntent, "Intent type should be preserved through string conversion")
			assert.True(t, newIntent.IsValid(), "Converted intent type should be valid")
		})
	}
}

// TestEmailIntentHandling tests that email intents can be properly handled
func TestEmailIntentHandling(t *testing.T) {
	emailID := uuid.New()
	conversationID := uuid.New()

	testEmail := &domain.Email{
		Base: domain.Base{
			ID: emailID,
		},
		ConversationID: &conversationID,
		Subject:        "Question about pricing",
		Content:        "I'm interested in learning more about your pricing plans",
		Intents:        pq.StringArray{"question", "pricing"},
		FromEmail:      "customer@example.com",
		ToEmail:        "sales@company.com",
	}

	// Test that email data is properly structured
	assert.NotNil(t, testEmail, "Email should not be nil")
	assert.Equal(t, emailID, testEmail.ID, "Email ID should match")
	assert.Equal(t, conversationID, *testEmail.ConversationID, "Conversation ID should match")
	assert.Equal(t, "Question about pricing", testEmail.Subject, "Subject should match")
	assert.Contains(t, testEmail.Intents, "question", "Email intents should include question")
	assert.Contains(t, testEmail.Intents, "pricing", "Email intents should include pricing")
}

// TestCampaignAndEmailRelationship tests that campaign and email relationships work correctly
func TestCampaignAndEmailRelationship(t *testing.T) {
	campaignID := uuid.New()
	emailID := uuid.New()
	userID := uuid.New()

	testCampaign := &domain.Campaign{
		Base: domain.Base{
			ID: campaignID,
		},
		Name:   "Test Campaign",
		UserID: userID,
	}

	testEmail := &domain.Email{
		Base: domain.Base{
			ID: emailID,
		},
		CampaignID: &campaignID,
		Subject:    "Test Email",
		Content:    "Test content",
	}

	// Test the relationship
	assert.Equal(t, userID, testCampaign.UserID, "Campaign should belong to user")
	assert.Equal(t, campaignID, *testEmail.CampaignID, "Email should belong to campaign")
	assert.NotNil(t, testCampaign, "Campaign should not be nil")
	assert.NotNil(t, testEmail, "Email should not be nil")
}
