package v1

import (
	"github.com/google/uuid"
)

// HardcodedMail mirrors the Python API payload structure used for ad-hoc conversations.
type HardcodedMail struct {
	ToEmail   string `json:"to_email"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	FromEmail string `json:"from_email"`
	Subject   string `json:"subject"`
}

// AIReplyEmailRequest represents POST /v1/ai/emails/reply payload.
type AIReplyEmailRequest struct {
	ConversationID *uuid.UUID      `json:"conversation_id,omitempty"`
	Conversation   []HardcodedMail `json:"conversation,omitempty"`
}

// AIGenerateEmailRequest represents POST /v1/ai/emails/generate payload.
type AIGenerateEmailRequest struct {
	ClientData map[string]interface{} `json:"client_data" validate:"required"`
	Campaign   string                 `json:"campaign,omitempty"`
}

// EmailIntentClassifyRequest represents POST /v1/ai/emails/classify-intent payload.
type EmailIntentClassifyRequest struct {
	Content string `json:"content" validate:"required,min=5"`
}

// EmailIntentClassifyResponse mirrors the Python response.
type EmailIntentClassifyResponse struct {
	ID         int     `json:"id"`
	Intent     string  `json:"intent"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning,omitempty"`
}

// AIReplyEmailResponse describes the AI-generated reply template.
type AIReplyEmailResponse struct {
	FromEmail string `json:"from_email"`
	ToEmail   string `json:"to_email"`
	Subject   string `json:"subject"`
	Content   string `json:"content"`
	Type      string `json:"type"`
}

// EmailTemplateItem mirrors python's template item payload.
type EmailTemplateItem struct {
	Type    string `json:"type"`
	Subject string `json:"subject"`
	Content string `json:"content"`
}

// AIGenerateEmailResponse matches python response for campaign template generation.
type AIGenerateEmailResponse struct {
	Template EmailTemplateItem `json:"template"`
	Preview  EmailTemplateItem `json:"preview"`
}

// AIGeneratedTemplate represents a generated outreach email template (deprecated endpoint).
type AIGeneratedTemplate struct {
	FromEmail string `json:"from_email"`
	ToEmail   string `json:"to_email"`
	Subject   string `json:"subject"`
	Content   string `json:"content"`
	Type      string `json:"type"`
}
