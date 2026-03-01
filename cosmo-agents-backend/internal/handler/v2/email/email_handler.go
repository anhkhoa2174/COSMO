package email

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	integrationRepo "github.com/rockship/cosmo-agents-go/internal/repository/integration"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	"github.com/rockship/cosmo-agents-go/pkg/gmail"
)

// EmailHandler handles V2 Email requests
type EmailHandler struct {
	emailRepo       *emailRepo.Repository
	integrationRepo *integrationRepo.IntegrationRepository
	agentRepo       *agentRepo.AgentRepository
}

// NewEmailHandler creates a new V2 EmailHandler
func NewEmailHandler(
	emailRepo *emailRepo.Repository,
	integrationRepo *integrationRepo.IntegrationRepository,
	agentRepo *agentRepo.AgentRepository,
) *EmailHandler {
	return &EmailHandler{
		emailRepo:       emailRepo,
		integrationRepo: integrationRepo,
		agentRepo:       agentRepo,
	}
}

// Search searches emails with filters
// POST /v2/emails/search?offset=0&limit=25
func (h *EmailHandler) Search(c fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	// Parse query params
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	if limit > 100 {
		limit = 100
	}

	// Parse request body
	var req v2schema.EmailSearchRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	// Build filter
	filter := req.Filter
	if filter == nil {
		filter = make(map[string]interface{})
	}

	// Always filter by user_id and is_deleted
	filter["user_id"] = userID
	filter["is_deleted"] = false

	// Query emails using FindAll with filter
	pagination := &baseRepo.PaginationParams{
		Offset: offset * limit,
		Limit:  limit,
	}

	result, err := h.emailRepo.FindAll(c.Context(), filter, pagination)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to search emails: %v", err),
		})
	}

	// Convert to response format
	items := make([]v2schema.EmailListItem, len(result.List))
	for i, email := range result.List {
		items[i] = v2schema.EmailListItem{
			Entity: v2schema.EmailEntity{
				ID:             email.ID,
				Subject:        ptrString(email.Subject),
				Content:        ptrString(email.Content),
				FromEmail:      ptrString(email.FromEmail),
				ToEmail:        ptrString(email.ToEmail),
				Attachments:    stringArrayToSlice(email.Attachments),
				Intents:        stringArrayToSlice(email.Intents),
				GmailMessageID: ptrString(email.GmailMessageID),
				ConversationID: email.ConversationID,
				CreatedAt:      &email.CreatedAt,
				UpdatedAt:      &email.UpdatedAt,
			},
		}
	}

	response := v2schema.EmailSearchResponse{
		List:   items,
		Offset: offset,
		Limit:  limit,
		Total:  int64(result.Total),
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   response,
	})
}

// Reply replies to an email via Gmail
// POST /v2/emails/{email_id}/reply
func (h *EmailHandler) Reply(c fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	// Parse email ID
	emailIDStr := c.Params("email_id")
	emailID, err := uuid.Parse(emailIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid email ID",
		})
	}

	// Parse request body
	var req v2schema.EmailReplyRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	// Find the original email
	email, err := h.emailRepo.FindByID(c.Context(), emailID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to find email",
		})
	}

	if email == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Email not found",
		})
	}

	// Verify email belongs to user
	if email.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "You don't have permission to reply to this email",
		})
	}

	// Prefer agent credentials (matches current auth flow)
	var accessToken string

	agent, err := h.agentRepo.FindByUserAndEmail(c.Context(), userID, email.ToEmail)
	if err == nil && agent != nil {
		if store, storeErr := agent.GetGoogleTokenStore(); storeErr == nil {
			accessToken = store.AccessToken
		}
	}

	// Fallback to integration credential if agent not found
	if accessToken == "" {
		integration, err := h.integrationRepo.FindByUserIDAndSource(c.Context(), userID, domain.SourceIntegration("gmail"))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Failed to get Gmail integration",
			})
		}

		if integration == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Gmail not connected. Please connect your Gmail account first.",
			})
		}

		var credential map[string]interface{}
		if err := json.Unmarshal(integration.Credential, &credential); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Failed to parse credentials",
			})
		}

		if token, ok := credential["access_token"].(string); ok {
			accessToken = token
		}
	}

	if accessToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Gmail access token not found",
		})
	}

	// Create Gmail client
	gmailClient, err := gmail.NewClient(c.Context(), accessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to create Gmail client: %v", err),
		})
	}

	// Build reply message
	subject := email.Subject
	if req.Subject != nil {
		subject = *req.Subject
	} else if !strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = "Re: " + subject
	}

	// Build CC list
	cc := ""
	if len(req.CC) > 0 {
		cc = strings.Join(req.CC, ",")
	}

	// Send reply
	sendReq := &gmail.SendMessageRequest{
		From:      email.ToEmail, // Our email address (where the original email was sent to)
		To:        email.FromEmail,
		Cc:        cc,
		Subject:   subject,
		Body:      req.Content,
		IsHTML:    false,
		InReplyTo: email.GmailMessageID, // For threading
	}

	_, err = gmailClient.SendMessage(c.Context(), sendReq)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Failed to send email: %v", err),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   "Email replied successfully",
	})
}

// Helper functions

func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringArrayToSlice(arr interface{}) []string {
	if arr == nil {
		return []string{}
	}

	// Handle different types
	switch v := arr.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			if str, ok := item.(string); ok {
				result[i] = str
			}
		}
		return result
	default:
		return []string{}
	}
}
