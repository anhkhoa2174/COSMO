package lab

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	"github.com/rockship/cosmo-agents-go/internal/repository/organization"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

const NOT_AVAILABLE = "N/A"

// Intent IDs mapping (from Python intent_classifier.py)
var INTENTS = map[int]string{
	1: "Interested",
	2: "Not Interested",
	3: "Referral",
	4: "Request for Pricing",
	5: "Request for Information",
	6: "Nurture",
	7: "Do Not Contact",
	8: "Out of Office",
	9: "Unknown",
}

// LabHandler provides experimental/testing endpoints
type LabHandler struct {
	emailRepo *emailRepo.Repository
	userRepo  *user.UserRepository
	orgRepo   *organization.OrganizationRepository
	openaiKey string
}

// NewLabHandler creates a new Lab handler
func NewLabHandler(
	emailRepo *emailRepo.Repository,
	userRepo *user.UserRepository,
	orgRepo *organization.OrganizationRepository,
	openaiKey string,
) *LabHandler {
	return &LabHandler{
		emailRepo: emailRepo,
		userRepo:  userRepo,
		orgRepo:   orgRepo,
		openaiKey: openaiKey,
	}
}

// userIDFromContext extracts user ID from fiber context
func userIDFromContext(c fiber.Ctx) (uuid.UUID, bool) {
	userID := c.Locals("user_id")
	if userID == nil {
		return uuid.Nil, false
	}
	uid, ok := userID.(uuid.UUID)
	return uid, ok
}

// GenerateSampleResponse generates sample customer responses for different intents
// @Summary Generate sample responses
// @Description Simulates customer responses based on given content and intent
// @Tags lab
// @Accept json
// @Produce json
// @Param request body v1schema.GenerateSampleResponseRequest true "Request body"
// @Success 200 {object} map[string]interface{} "Sample responses"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/lab/generate-sample-response [post]
func (h *LabHandler) GenerateSampleResponse(c fiber.Ctx) error {
	userID, ok := userIDFromContext(c)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(schema.ErrorResponse(
			http.StatusUnauthorized,
			"User not authenticated",
			"",
		))
	}

	var req v1schema.GenerateSampleResponseRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest,
			"Invalid request body",
			err.Error(),
		))
	}

	// Get current user
	user, err := h.userRepo.FindByID(c.Context(), userID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(
			http.StatusNotFound,
			"User not found",
			err.Error(),
		))
	}

	// Get user's main organization
	orgs, err := h.orgRepo.FindByUserID(c.Context(), userID)
	if err != nil || len(orgs) == 0 {
		return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(
			http.StatusNotFound,
			"User organization not found",
			"",
		))
	}
	org := &orgs[0] // Use first organization

	// Determine which intents to try
	intentsToTry := []int{}
	if req.IntentID != nil {
		intentsToTry = append(intentsToTry, *req.IntentID)
	} else {
		// Try all intents
		for id := range INTENTS {
			intentsToTry = append(intentsToTry, id)
		}
	}

	// Generate responses for each intent
	responses := make([]v1schema.SampleResponseItem, 0, len(intentsToTry))
	for _, intentID := range intentsToTry {
		response, err := h.generateResponseForIntent(
			c.Context(),
			req.Content,
			user,
			org,
			req.Contact,
			intentID,
		)
		if err != nil {
			log := logger.FromContext(c.Context())
			log.Error().Err(err).Int("intent_id", intentID).Msg("Failed to generate response")
			continue
		}

		responses = append(responses, v1schema.SampleResponseItem{
			Intent:   INTENTS[intentID],
			Response: response,
		})
	}

	return c.JSON(schema.SuccessResponse(responses))
}

// GetEmailsWithinConversation retrieves all emails within a conversation
// @Summary Get emails in conversation
// @Description Retrieves all emails within a specific conversation
// @Tags lab
// @Accept json
// @Produce json
// @Param conversation_id path string true "Conversation ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/lab/conversations/{conversation_id}/emails [get]
func (h *LabHandler) GetEmailsWithinConversation(c fiber.Ctx) error {
	conversationIDStr := c.Params("conversation_id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(
			http.StatusBadRequest,
			"Invalid conversation ID",
			err.Error(),
		))
	}

	// Build filter for emails in this conversation
	filters := map[string]interface{}{
		"conversation_id": conversationID,
	}

	emails, err := h.emailRepo.FindAll(c.Context(), filters, &baseRepo.PaginationParams{
		Limit:  1000, // Get all emails (reasonable limit)
		Offset: 0,
	})
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(
			http.StatusInternalServerError,
			"Failed to retrieve emails",
			err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(emails))
}

// generateResponseForIntent generates a customer response for a specific intent
func (h *LabHandler) generateResponseForIntent(
	ctx context.Context,
	content string,
	user *domain.User,
	org *domain.Organization,
	contact map[string]interface{},
	intentID int,
) (string, error) {
	// Build contact description
	includedFields := []string{"first_name", "last_name", "email", "phone", "company", "job_title", "address", "city", "country", "state", "zip"}
	contactDesc := ""
	for _, field := range includedFields {
		if val, ok := contact[field]; ok && val != nil && val != NOT_AVAILABLE {
			contactDesc += fmt.Sprintf("- %s: %v\n", field, val)
		}
	}

	// Build user description
	userDesc := ""
	if user.Email != "" {
		userDesc += fmt.Sprintf("- Email: %s\n", user.Email)
	}
	if user.Name != "" {
		userDesc += fmt.Sprintf("- Name: %s\n", user.Name)
	}

	// Build org description
	orgDesc := ""
	if org.Name != "" {
		orgDesc += fmt.Sprintf("- Name: %s\n", org.Name)
	}

	// Create OpenAI client
	client := openai.NewClient(option.WithAPIKey(h.openaiKey))

	// Build messages
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(fmt.Sprintf(
			"You will act like a customer receiving a mail and you have to reply back matching the given intent: %s.",
			INTENTS[intentID],
		)),
		openai.UserMessage(fmt.Sprintf(
			"Here is some information about the customer you are playing:\n%s",
			contactDesc,
		)),
		openai.UserMessage(fmt.Sprintf(
			"Here is information about the sender (whom you are replying to):\n1. Personal Information:\n%s\n2. Organization information:\n%s",
			userDesc,
			orgDesc,
		)),
		openai.UserMessage(fmt.Sprintf(
			"Email content:\n%s\n\nResponse:",
			content,
		)),
	}

	// Call OpenAI
	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    openai.ChatModelGPT4o,
	}
	completion, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", err
	}

	if len(completion.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return completion.Choices[0].Message.Content, nil
}
