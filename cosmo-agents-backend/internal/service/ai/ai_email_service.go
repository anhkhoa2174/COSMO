package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	knowledgeRepo "github.com/rockship/cosmo-agents-go/internal/repository/knowledge"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	intentService "github.com/rockship/cosmo-agents-go/internal/service/intent"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// ErrAIEmailClientNotConfigured is returned when OpenAI API credentials are missing.
var ErrAIEmailClientNotConfigured = errors.New("ai email service: openai client not configured")

// ErrConversationNotFound indicates that no conversation emails could be located.
var ErrConversationNotFound = errors.New("conversation not found")

// ErrInvalidConversation indicates malformed conversation payload.
var ErrInvalidConversation = errors.New("either conversation_id or conversation must be provided")

// ErrLastEmailNotAddressedToUser indicates the last message was not delivered to the authenticated user.
var ErrLastEmailNotAddressedToUser = errors.New("the last email in conversation was not addressed to the current user")

// AIEmailService orchestrates AI-powered email utilities (classify intent, generate replies, outreach templates).
type AIEmailService struct {
	emailRepo        *emailRepo.Repository
	userRepo         *user.UserRepository
	knowledgeRepo    *knowledgeRepo.KnowledgeRepository
	intentClassifier *intentService.IntentClassifier
	openAIClient     *ai.OpenAIClient
	logger           *zerolog.Logger
}

// ConversationMessage represents a simplified email in a conversation thread.
type ConversationMessage struct {
	FromEmail string
	ToEmail   string
	Subject   string
	Content   string
}

// AIReplyEmail encapsulates the AI-generated reply template.
type AIReplyEmail struct {
	FromEmail string
	ToEmail   string
	Subject   string
	Content   string
	Type      string
}

// EmailIntentResult mirrors the Python response for intent classification.
type EmailIntentResult struct {
	ID     int
	Intent string
}

// AIGeneratedTemplate represents an outreach template (deprecated endpoint).
type AIGeneratedTemplate struct {
	FromEmail string
	ToEmail   string
	Subject   string
	Content   string
	Type      string
}

// NewAIEmailService constructs a new AIEmailService.
func NewAIEmailService(
	emailRepo *emailRepo.Repository,
	userRepo *user.UserRepository,
	knowledgeRepo *knowledgeRepo.KnowledgeRepository,
	intentClassifier *intentService.IntentClassifier,
	openAIClient *ai.OpenAIClient,
) *AIEmailService {
	l := logger.Logger
	return &AIEmailService{
		emailRepo:        emailRepo,
		userRepo:         userRepo,
		knowledgeRepo:    knowledgeRepo,
		intentClassifier: intentClassifier,
		openAIClient:     openAIClient,
		logger:           &l,
	}
}

// ClassifyIntent delegates to the OpenAI-powered classifier.
func (s *AIEmailService) ClassifyIntent(ctx context.Context, content string) (*EmailIntentResult, error) {
	if s.intentClassifier == nil {
		return nil, ErrAIEmailClientNotConfigured
	}
	intent, err := s.intentClassifier.Classify(ctx, content)
	if err != nil {
		return nil, err
	}

	result := mapIntentToResponse(intent)
	return &result, nil
}

// GenerateReply crafts a reply email leveraging the conversation history, knowledge base, and OpenAI.
func (s *AIEmailService) GenerateReply(
	ctx context.Context,
	userID uuid.UUID,
	userEmail string,
	reqConversationID *uuid.UUID,
	reqManualConversation []ConversationMessage,
) (*AIReplyEmail, error) {
	if s.openAIClient == nil || s.intentClassifier == nil {
		return nil, ErrAIEmailClientNotConfigured
	}

	thread, err := s.resolveConversation(ctx, userID, reqConversationID, reqManualConversation)
	if err != nil {
		return nil, err
	}

	lastMessage := thread[0] // newest first
	if !strings.EqualFold(strings.TrimSpace(lastMessage.ToEmail), strings.TrimSpace(userEmail)) {
		return nil, ErrLastEmailNotAddressedToUser
	}

	intent, err := s.intentClassifier.Classify(ctx, lastMessage.Content)
	if err != nil {
		return nil, fmt.Errorf("classify intent failed: %w", err)
	}

	intentResponse := mapIntentToResponse(intent)

	user, err := s.userRepo.FindWithOrganizations(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	knowledgeCtx, err := s.buildKnowledgeContext(ctx, userID)
	if err != nil {
		return nil, err
	}

	replyBody, replySubject, err := s.generateReplyWithOpenAI(ctx, user, thread, intentResponse, knowledgeCtx, lastMessage.Subject, lastMessage.FromEmail)
	if err != nil {
		return nil, err
	}

	if replySubject == "" {
		replySubject = buildReplySubject(lastMessage.Subject)
	}

	return &AIReplyEmail{
		FromEmail: userEmail,
		ToEmail:   strings.TrimSpace(lastMessage.FromEmail),
		Subject:   replySubject,
		Content:   replyBody,
		Type:      "Reply",
	}, nil
}

// GenerateDeprecatedOutreach returns lightweight placeholder templates for the deprecated endpoint.
func (s *AIEmailService) GenerateDeprecatedOutreach(ctx context.Context, userID uuid.UUID, campaign string, clientData map[string]interface{}) ([]AIGeneratedTemplate, error) {
	clientName := fmt.Sprintf("%v", clientData["name"])
	if clientName == "<nil>" || clientName == "" {
		clientName = "there"
	}

	first := AIGeneratedTemplate{
		FromEmail: "",
		ToEmail:   "",
		Type:      "First Email",
		Subject:   fmt.Sprintf("Quick intro for %s", clientName),
		Content:   fmt.Sprintf("Hi %s,\n\nHope you're doing well. Wanted to share a short overview of how we can help. Let me know if you have a few minutes this week to chat.\n\nBest,\n", clientName),
	}
	followUpOne := AIGeneratedTemplate{
		FromEmail: "",
		ToEmail:   "",
		Type:      "Follow-up Email 1",
		Subject:   "Just checking in",
		Content:   fmt.Sprintf("Hi %s,\n\nWanted to bump this to the top of your inbox in case it got lost. Happy to send more context or schedule time.\n\nBest,\n", clientName),
	}
	followUpTwo := AIGeneratedTemplate{
		FromEmail: "",
		ToEmail:   "",
		Type:      "Follow-up Email 2",
		Subject:   "Should I keep this on the radar?",
		Content:   fmt.Sprintf("Hi %s,\n\nTotally understand if now isn't the right time. Let me know if you'd prefer I reach back out later in the year. Either way, thanks for taking a look!\n\nBest,\n", clientName),
	}

	return []AIGeneratedTemplate{first, followUpOne, followUpTwo}, nil
}

func (s *AIEmailService) resolveConversation(ctx context.Context, userID uuid.UUID, conversationID *uuid.UUID, manual []ConversationMessage) ([]ConversationMessage, error) {
	switch {
	case conversationID != nil:
		emails, err := s.emailRepo.FindByConversationAndUser(ctx, *conversationID, userID, 10)
		if err != nil {
			return nil, fmt.Errorf("failed to load conversation: %w", err)
		}
		if len(emails) == 0 {
			return nil, ErrConversationNotFound
		}

		thread := make([]ConversationMessage, len(emails))
		for i, email := range emails {
			thread[i] = ConversationMessage{
				FromEmail: email.FromEmail,
				ToEmail:   email.ToEmail,
				Subject:   email.Subject,
				Content:   email.Content,
			}
		}
		return thread, nil

	case len(manual) > 0:
		thread := make([]ConversationMessage, len(manual))
		copy(thread, manual)
		return thread, nil

	default:
		return nil, ErrInvalidConversation
	}
}

func (s *AIEmailService) buildKnowledgeContext(ctx context.Context, userID uuid.UUID) (string, error) {
	knowledges, _, err := s.knowledgeRepo.GetByUserID(ctx, userID.String(), 3, 0)
	if err != nil {
		return "", fmt.Errorf("failed to fetch knowledge context: %w", err)
	}

	if len(knowledges) == 0 {
		return "No internal knowledge available.", nil
	}

	var parts []string
	for _, k := range knowledges {
		if len(k.SummaryPair) == 0 {
			continue
		}
		if len(k.SummaryPair) == 1 {
			parts = append(parts, strings.TrimSpace(k.SummaryPair[0]))
			continue
		}
		parts = append(parts, strings.TrimSpace(k.SummaryPair[1]))
	}

	if len(parts) == 0 {
		return "No internal knowledge available.", nil
	}
	return strings.Join(parts, "\n---\n"), nil
}

func (s *AIEmailService) generateReplyWithOpenAI(
	ctx context.Context,
	user *domain.User,
	thread []ConversationMessage,
	intent EmailIntentResult,
	knowledgeContext string,
	lastSubject string,
	recipientEmail string,
) (string, string, error) {
	if s.openAIClient == nil {
		return "", "", ErrAIEmailClientNotConfigured
	}

	threadContext := buildThreadContext(thread)

	systemPrompt := `You are an expert sales assistant who crafts short email replies.
Write naturally, stay concise (2-4 sentences), and avoid any sign-off or signature.
Your reply must address the sender's intent and be grounded in the provided knowledge.
If information is missing, acknowledge it honestly rather than inventing details.`

	userPrompt := fmt.Sprintf(`Sender: %s <%s>
Detected intent: %s
Knowledge references:
%s

Recent conversation (most recent first):
%s

Generate a friendly reply from %s to %s.
Respond in JSON with the following shape:
{
  "subject": "string",
  "body": "string"
}
Keep the body under 150 words and do not include a closing signature.`,
		user.Name, user.Email, intent.Intent, knowledgeContext, threadContext, user.Name, recipientEmail)

	response, err := s.openAIClient.ChatCompletion(ctx, ai.ChatCompletionRequest{
		SystemPrompt: systemPrompt,
		Messages: []ai.Message{
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.4,
		MaxTokens:   600,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to generate reply: %w", err)
	}

	parsedSubject, parsedBody := extractReplyJSON(response)
	if parsedBody == "" {
		parsedBody = strings.TrimSpace(response)
	}
	if parsedSubject == "" {
		parsedSubject = buildReplySubject(lastSubject)
	}

	return parsedBody, parsedSubject, nil
}

func buildThreadContext(thread []ConversationMessage) string {
	var sb strings.Builder
	for idx, msg := range thread {
		fmt.Fprintf(&sb, "[Email #%d]\nFrom: %s\nTo: %s\nSubject: %s\nBody:\n%s\n\n",
			idx+1,
			strings.TrimSpace(msg.FromEmail),
			strings.TrimSpace(msg.ToEmail),
			strings.TrimSpace(msg.Subject),
			strings.TrimSpace(msg.Content),
		)
		if idx >= 4 {
			break
		}
	}
	return sb.String()
}

func buildReplySubject(original string) string {
	trimmed := strings.TrimSpace(original)
	if trimmed == "" {
		return "Re: your message"
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "re:") {
		return trimmed
	}
	return "Re: " + trimmed
}

func extractReplyJSON(response string) (string, string) {
	type reply struct {
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	var r reply
	if err := json.Unmarshal([]byte(response), &r); err == nil {
		return strings.TrimSpace(r.Subject), strings.TrimSpace(r.Body)
	}

	// Attempt to locate JSON block within the response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(response[start:end+1]), &r); err == nil {
			return strings.TrimSpace(r.Subject), strings.TrimSpace(r.Body)
		}
	}

	return "", ""
}

func mapIntentToResponse(intent domain.IntentType) EmailIntentResult {
	switch intent {
	case domain.IntentInterested:
		return EmailIntentResult{ID: 1, Intent: "Interested"}
	case domain.IntentNotInterested:
		return EmailIntentResult{ID: 2, Intent: "Not interested"}
	case domain.IntentRequestForPricing:
		return EmailIntentResult{ID: 3, Intent: "Request for pricing"}
	case domain.IntentRequestForInfo:
		return EmailIntentResult{ID: 4, Intent: "Request for information"}
	case domain.IntentDoNotContact:
		return EmailIntentResult{ID: 5, Intent: "Do not contact"}
	case domain.IntentOutOfOffice:
		return EmailIntentResult{ID: 6, Intent: "Out of office"}
	default:
		return EmailIntentResult{ID: 7, Intent: "Unknown intent"}
	}
}
