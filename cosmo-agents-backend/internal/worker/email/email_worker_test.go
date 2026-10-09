package email

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

func TestEmailWorker_Basic(t *testing.T) {
	t.Skip("Worker tests require database setup and domain knowledge - placeholder test")
	assert.True(t, true, "EmailWorker tests would require full setup")
}

func TestGeneralEmailWorker_Basic(t *testing.T) {
	t.Skip("GeneralEmailWorker tests require database setup and domain knowledge - placeholder test")
	assert.True(t, true, "GeneralEmailWorker tests would require full setup")
}

func TestTaskTypeConstants(t *testing.T) {
	assert.Equal(t, "agent:send_email", TypeAgentSendEmail)
	assert.Equal(t, "agent:daily_reset", TypeAgentDailyReset)

	assert.Equal(t, "email:send_invite_member_email", TypeSendInviteMemberEmail)
	assert.Equal(t, "email:send", TypeSendEmail)
	assert.Equal(t, "email:process_incoming", TypeProcessIncoming)
	assert.Equal(t, "email:sync_history", TypeSyncGmailHistory)

	assert.NotEmpty(t, TypeAgentSendEmail)
	assert.NotEmpty(t, TypeAgentDailyReset)
	assert.NotEmpty(t, TypeSendInviteMemberEmail)
	assert.NotEmpty(t, TypeSendEmail)
	assert.NotEmpty(t, TypeProcessIncoming)
	assert.NotEmpty(t, TypeSyncGmailHistory)
}
func TestEmailConstants(t *testing.T) {
	assert.Equal(t, "Welcome to Cosmo Agents Your Account Awaits", SUBJECT_INVITATION)
	assert.NotEmpty(t, SUBJECT_INVITATION)
}

// Test payload structures
func TestAgentSendEmailPayload(t *testing.T) {
	taskIDs := []uuid.UUID{uuid.New(), uuid.New()}
	campaignID := uuid.New()
	agentID := uuid.New()

	payload := AgentSendEmailPayload{
		TaskIDs:    taskIDs,
		CampaignID: campaignID,
		AgentID:    agentID,
	}

	assert.Equal(t, taskIDs, payload.TaskIDs)
	assert.Equal(t, campaignID, payload.CampaignID)
	assert.Equal(t, agentID, payload.AgentID)
	assert.Len(t, payload.TaskIDs, 2)
}

func TestSendInviteMemberEmailPayload(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	payload := SendInviteMemberEmailPayload{
		UserID:         userID,
		OrganizationID: orgID,
		MemberEmail:    "test@example.com",
		MemberName:     "Test Member",
		InviterName:    "Test Inviter",
		RedirectURI:    "https://example.com/redirect",
		Url:            "https://example.com/invite",
		Email:          "test@example.com",
	}

	assert.Equal(t, userID, payload.UserID)
	assert.Equal(t, orgID, payload.OrganizationID)
	assert.Equal(t, "test@example.com", payload.MemberEmail)
	assert.Equal(t, "Test Member", payload.MemberName)
	assert.Equal(t, "Test Inviter", payload.InviterName)
	assert.Equal(t, "https://example.com/redirect", payload.RedirectURI)
	assert.Equal(t, "https://example.com/invite", payload.Url)
	assert.Equal(t, "test@example.com", payload.Email)
}

func TestSendEmailPayload(t *testing.T) {
	taskID := uuid.New()
	campaignID := uuid.New()
	templateID := uuid.New()
	agentID := uuid.New()
	contactID := uuid.New()

	payload := SendEmailPayload{
		TaskID:     &taskID,
		CampaignID: &campaignID,
		TemplateID: &templateID,
		AgentID:    agentID,
		ContactID:  contactID,
		To:         "recipient@example.com",
		Cc:         []string{"cc1@example.com", "cc2@example.com"},
		Bcc:        []string{"bcc@example.com"},
		Subject:    "Test Subject",
		Body:       "Test Body",
		IsHTML:     true,
		InReplyTo:  "original-message-id",
	}

	assert.Equal(t, &taskID, payload.TaskID)
	assert.Equal(t, &campaignID, payload.CampaignID)
	assert.Equal(t, &templateID, payload.TemplateID)
	assert.Equal(t, agentID, payload.AgentID)
	assert.Equal(t, contactID, payload.ContactID)
	assert.Equal(t, "recipient@example.com", payload.To)
	assert.Equal(t, []string{"cc1@example.com", "cc2@example.com"}, payload.Cc)
	assert.Equal(t, []string{"bcc@example.com"}, payload.Bcc)
	assert.Equal(t, "Test Subject", payload.Subject)
	assert.Equal(t, "Test Body", payload.Body)
	assert.True(t, payload.IsHTML)
	assert.Equal(t, "original-message-id", payload.InReplyTo)
}

func TestProcessIncomingEmailPayload(t *testing.T) {
	agentID := uuid.New()

	payload := ProcessIncomingEmailPayload{
		AgentID:        agentID,
		GmailMessageID: "message-123",
		GmailThreadID:  "thread-456",
	}

	assert.Equal(t, agentID, payload.AgentID)
	assert.Equal(t, "message-123", payload.GmailMessageID)
	assert.Equal(t, "thread-456", payload.GmailThreadID)
}

func TestSyncGmailHistoryPayload(t *testing.T) {
	agentID := uuid.New()

	payload := SyncGmailHistoryPayload{
		AgentID:        agentID,
		StartHistoryID: "12345",
	}

	assert.Equal(t, agentID, payload.AgentID)
	assert.Equal(t, "12345", payload.StartHistoryID)

	// Test with nil StartHistoryID
	payload2 := SyncGmailHistoryPayload{
		AgentID: agentID,
	}
	assert.Equal(t, agentID, payload2.AgentID)
	assert.Empty(t, payload2.StartHistoryID)
}

func TestSafeUUID(t *testing.T) {
	id := uuid.New()
	assert.Equal(t, id.String(), safeUUID(&id))
	assert.Equal(t, "", safeUUID(nil))
}

func TestNormalizeTemplateValue(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal string",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: "",
		},
		{
			name:     "NOT_AVAILABLE",
			input:    domain.NOT_AVAILABLE,
			expected: "",
		},
		{
			name:     "case insensitive NOT_AVAILABLE",
			input:    "n/a",
			expected: "",
		},
		{
			name:     "normal value with whitespace",
			input:    "  Test Value  ",
			expected: "Test Value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, normalizeTemplateValue(tc.input))
		})
	}
}

func TestLooksLikeMarkdown(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "bold markdown",
			input:    "This is **bold**",
			expected: true,
		},
		{
			name:     "underline markdown",
			input:    "This is __underlined__",
			expected: true,
		},
		{
			name:     "italic markdown star",
			input:    "This is *italic*",
			expected: true,
		},
		{
			name:     "italic markdown underscore",
			input:    "This is _italic_",
			expected: true,
		},
		{
			name:     "strikethrough markdown",
			input:    "This is ~~strikethrough~~",
			expected: true,
		},
		{
			name:     "HTML tag",
			input:    "This is <b>bold</b>",
			expected: true,
		},
		{
			name:     "plain text",
			input:    "This is plain text",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, looksLikeMarkdown(tc.input))
		})
	}
}

func TestPrepareEmailBody(t *testing.T) {
	testCases := []struct {
		name         string
		body         string
		isHTML       bool
		expectedBody string
		expectedHTML bool
	}{
		{
			name:         "plain text as HTML",
			body:         "Plain text content",
			isHTML:       true,
			expectedBody: "Plain text content",
			expectedHTML: true,
		},
		{
			name:         "markdown content",
			body:         "This is **bold** text",
			isHTML:       false,
			expectedBody: "This is <strong>bold</strong> text",
			expectedHTML: true,
		},
		{
			name:         "plain text not markdown",
			body:         "Just plain text",
			isHTML:       false,
			expectedBody: "Just plain text",
			expectedHTML: false,
		},
		{
			name:         "HTML content",
			body:         "<b>Bold</b> text",
			isHTML:       false,
			expectedBody: "<b>Bold</b> text",
			expectedHTML: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, isHTML := prepareEmailBody(tc.body, tc.isHTML)
			assert.Equal(t, tc.expectedBody, body)
			assert.Equal(t, tc.expectedHTML, isHTML)
		})
	}
}

func TestApplyTemplateData(t *testing.T) {
	testCases := []struct {
		name     string
		template string
		data     map[string]string
		expected string
	}{
		{
			name:     "single braces",
			template: "Hello {name}",
			data:     map[string]string{"name": "World"},
			expected: "Hello World",
		},
		{
			name:     "double braces",
			template: "Hello {{name}}",
			data:     map[string]string{"name": "World"},
			expected: "Hello World",
		},
		{
			name:     "mixed braces",
			template: "Hello {name} and {{other}}",
			data:     map[string]string{"name": "World", "other": "Universe"},
			expected: "Hello World and Universe",
		},
		{
			name:     "multiple placeholders",
			template: "{first} {last} - {{email}}",
			data:     map[string]string{"first": "John", "last": "Doe", "email": "john@example.com"},
			expected: "John Doe - john@example.com",
		},
		{
			name:     "missing data",
			template: "Hello {missing}",
			data:     map[string]string{"name": "World"},
			expected: "Hello {missing}",
		},
		{
			name:     "empty template",
			template: "",
			data:     map[string]string{"name": "World"},
			expected: "",
		},
		{
			name:     "empty data",
			template: "Hello {name}",
			data:     map[string]string{},
			expected: "Hello {name}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, applyTemplateData(tc.template, tc.data))
		})
	}
}

func TestBuildTemplateData(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	agent := &domain.Agent{
		Base: domain.Base{
			ID: uuid.New(),
		},
		UserID:         userID,
		OrganizationID: &orgID,
		Name:           "Test Agent",
		Email:          "agent@example.com",
		Signature:      "Best regards,\nTest Agent",
		Status:         domain.AgentStatusActive,
		EmailProvider:  domain.AgentEmailProviderGmail,
	}

	contact := &domain.Contact{
		Base: domain.Base{
			ID: uuid.New(),
		},
		Name:     "John Doe",
		Profile:  base.JSONB(`{"email":"john@example.com"}`),
		Company:  "Test Company",
		JobTitle: "Developer",
		Country:  "USA",
	}

	templateData := buildTemplateData(agent, contact, "")

	// Test contact fields
	assert.Equal(t, "John", templateData["contact_first_name"])
	assert.Equal(t, "Doe", templateData["contact_last_name"])
	assert.Equal(t, "john@example.com", templateData["contact_email"])
	assert.Equal(t, "Test Company", templateData["contact_company"])
	assert.Equal(t, "Developer", templateData["contact_job_title"])
	assert.Equal(t, "USA", templateData["contact_country"])

	// Test agent fields
	assert.Contains(t, templateData["agent_signature"], "Best regards")
	assert.Contains(t, templateData["agent_signature"], "Test Agent")
	assert.Equal(t, "Test Agent", templateData["sender_name"])
	assert.Equal(t, "", templateData["organization_name"]) // No organization set
}

// Benchmark tests
func BenchmarkSafeUUID(b *testing.B) {
	id := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = safeUUID(&id)
	}
}

func BenchmarkSafeUUIDNil(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = safeUUID(nil)
	}
}

func BenchmarkNormalizeTemplateValue(b *testing.B) {
	values := []string{
		"Hello World",
		"  trimmed value  ",
		domain.NOT_AVAILABLE,
		"",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = normalizeTemplateValue(values[i%len(values)])
	}
}

func BenchmarkApplyTemplateData(b *testing.B) {
	template := "Hello {first} {last} - {{email}}"
	data := map[string]string{
		"first": "John",
		"last":  "Doe",
		"email": "john@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = applyTemplateData(template, data)
	}
}

func TestEmailWorker_Integration(t *testing.T) {
	t.Skip("Integration tests require database setup")
}

func TestGeneralEmailWorker_Integration(t *testing.T) {
	t.Skip("Integration tests require database setup")
}

type EmailWorkerTestSuite struct {
	suite.Suite
}

func (suite *EmailWorkerTestSuite) SetupSuite() {
	suite.T().Skip("Test suite requires database setup")
}

func TestEmailWorkerSuite(t *testing.T) {
	suite.Run(t, new(EmailWorkerTestSuite))
}

type GeneralEmailWorkerTestSuite struct {
	suite.Suite
}

func (suite *GeneralEmailWorkerTestSuite) SetupSuite() {
	suite.T().Skip("Test suite requires database setup")
}

func TestGeneralEmailWorkerSuite(t *testing.T) {
	suite.Run(t, new(GeneralEmailWorkerTestSuite))
}
