package ai

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	aiService "github.com/rockship/cosmo-agents-go/internal/service/ai"
	aiUsecase "github.com/rockship/cosmo-agents-go/internal/usecase/ai"
)

// AIEmailHandler exposes AI email utilities over HTTP.
type AIEmailHandler struct {
	emailUsecase aiUsecase.AIEmailUsecase
}

// NewAIEmailHandler constructs a new AIEmailHandler.
func NewAIEmailHandler(emailUsecase aiUsecase.AIEmailUsecase) *AIEmailHandler {
	return &AIEmailHandler{
		emailUsecase: emailUsecase,
	}
}

// ClassifyIntent handles POST /v1/ai/emails/classify-intent
// @Summary Classify email intent
// @Description Classify the primary intent of an email's content using AI
// @Tags AI
// @Accept json
// @Produce json
// @Param request body v1schema.EmailIntentClassifyRequest true "Email content to classify"
// @Success 200 {object} schema.APIResponse[v1schema.EmailIntentClassifyResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/ai/emails/classify-intent [post]
func (h *AIEmailHandler) ClassifyIntent(c fiber.Ctx) error {
	var req v1schema.EmailIntentClassifyRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	result, err := h.emailUsecase.ClassifyIntent(c.Context(), req.Content)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError, "Failed to classify intent", err.Error(),
		))
	}

	resp := v1schema.EmailIntentClassifyResponse{
		ID:         result.ID,
		Intent:     result.Intent,
		Confidence: result.Confidence,
		Reasoning:  result.Reasoning,
	}

	return c.JSON(schema.SuccessResponse(resp))
}

// GenerateReply handles POST /v1/ai/emails/reply
// @Summary Generate AI reply for an email thread
// @Description Generate a concise AI-written reply based on the provided conversation or conversation_id
// @Tags AI
// @Accept json
// @Produce json
// @Param request body v1schema.AIReplyEmailRequest true "Conversation or conversation_id"
// @Success 200 {object} schema.APIResponse[v1schema.AIReplyEmailResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 404 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/ai/emails/reply [post]
// @Summary Generate AI reply for an email thread
// @Description Generate a concise AI-written reply based on the provided conversation or conversation_id
// @Tags AI
// @Accept json
// @Produce json
// @Param request body v1schema.AIReplyEmailRequest true "Conversation or conversation_id"
// @Success 200 {object} schema.APIResponse[v1schema.AIReplyEmailResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 404 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/ai/emails/reply [post]
func (h *AIEmailHandler) GenerateReply(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		if v, okAlt := c.Locals("userID").(uuid.UUID); okAlt {
			userID = v
		} else {
			return c.Status(http.StatusUnauthorized).JSON(schema.ErrorResponse(
				http.StatusUnauthorized, "User not authenticated", "",
			))
		}
	}

	userEmail, _ := c.Locals("user_email").(string)
	if userEmail == "" {
		userEmail, _ = c.Locals("email").(string)
	}
	userEmail = strings.TrimSpace(userEmail)
	if userEmail == "" {
		return c.Status(http.StatusUnauthorized).JSON(schema.ErrorResponse(
			http.StatusUnauthorized, "User email not found in session", "",
		))
	}

	var req v1schema.AIReplyEmailRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	conversation := make([]aiService.ConversationMessage, len(req.Conversation))
	for i, msg := range req.Conversation {
		conversation[i] = aiService.ConversationMessage{
			FromEmail: msg.FromEmail,
			ToEmail:   msg.ToEmail,
			Subject:   msg.Subject,
			Content:   msg.Content,
		}
	}

	if len(conversation) > 1 {
		lastFrom := strings.TrimSpace(conversation[len(conversation)-1].FromEmail)

		if strings.EqualFold(lastFrom, userEmail) {
			for i, j := 0, len(conversation)-1; i < j; i, j = i+1, j-1 {
				conversation[i], conversation[j] = conversation[j], conversation[i]
			}
		}
	}

	reply, err := h.emailUsecase.GenerateReply(c.Context(), userID, userEmail, req.ConversationID, conversation)
	if err != nil {
		switch {
		case errors.Is(err, aiService.ErrInvalidConversation):
			return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
				http.StatusBadRequest, "Either conversation_id or conversation must be provided", "",
			))
		case errors.Is(err, aiService.ErrConversationNotFound):
			return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(
				http.StatusNotFound, "Conversation not found", "",
			))
		case errors.Is(err, aiService.ErrLastEmailNotAddressedToUser):
			return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
				http.StatusBadRequest, "The last email must be addressed to current user", "",
			))
		case errors.Is(err, aiService.ErrAIEmailClientNotConfigured):
			return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
				http.StatusInternalServerError, "AI provider not configured", "",
			))
		default:
			return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
				http.StatusInternalServerError, "Failed to generate AI reply", err.Error(),
			))
		}
	}

	resp := v1schema.AIReplyEmailResponse{
		FromEmail: reply.FromEmail,
		ToEmail:   reply.ToEmail,
		Subject:   reply.Subject,
		Content:   reply.Content,
		Type:      reply.Type,
	}

	return c.JSON(schema.SuccessResponse(resp))
}

// GenerateEmailTemplates handles POST /v1/ai/emails/generate (deprecated).
// @Summary Generate outreach email templates (deprecated)
// @Description (Deprecated) Generate lightweight outreach templates for a campaign. Prefer newer endpoints.
// @Tags AI
// @Accept json
// @Produce json
// @Param request body v1schema.AIGenerateEmailRequest true "Campaign and client data"
// @Success 200 {object} schema.APIResponse[[]v1schema.AIGeneratedTemplate]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/ai/emails/generate [post]
func (h *AIEmailHandler) GenerateEmailTemplates(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		if v, okAlt := c.Locals("userID").(uuid.UUID); okAlt {
			userID = v
		} else {
			return c.Status(http.StatusUnauthorized).JSON(schema.ErrorResponse(
				http.StatusUnauthorized, "User not authenticated", "",
			))
		}
	}

	var req v1schema.AIGenerateEmailRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	templates, err := h.emailUsecase.GenerateDeprecatedOutreach(c.Context(), userID, req.Campaign, req.ClientData)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError, "Failed to generate templates", err.Error(),
		))
	}

	resp := make([]v1schema.AIGeneratedTemplate, len(templates))
	for i, tmpl := range templates {
		resp[i] = v1schema.AIGeneratedTemplate{
			FromEmail: tmpl.FromEmail,
			ToEmail:   tmpl.ToEmail,
			Subject:   tmpl.Subject,
			Content:   tmpl.Content,
			Type:      tmpl.Type,
		}
	}

	return c.JSON(schema.SuccessResponse(resp))
}
