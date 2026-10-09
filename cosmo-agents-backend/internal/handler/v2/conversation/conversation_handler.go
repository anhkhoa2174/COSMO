package conversation

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// ConversationHandler handles V2 conversation API requests
type ConversationHandler struct {
	conversationRepo *conversationRepo.ConversationRepository
	emailRepo        *emailRepo.Repository
	campaignRepo     *campaignRepo.CampaignRepository
}

// NewConversationHandler creates a new V2 ConversationHandler
func NewConversationHandler(
	conversationRepo *conversationRepo.ConversationRepository,
	emailRepo *emailRepo.Repository,
	campaignRepo *campaignRepo.CampaignRepository,
) *ConversationHandler {
	return &ConversationHandler{
		conversationRepo: conversationRepo,
		emailRepo:        emailRepo,
		campaignRepo:     campaignRepo,
	}
}

// ConversationSearchRequest represents the search request body
type ConversationSearchRequest struct {
	Filter map[string]interface{} `json:"filter"`
}

// ConversationEntity represents a conversation in the response
type ConversationEntity struct {
	ID         uuid.UUID       `json:"id"`
	UserID     uuid.UUID       `json:"user_id"`
	Labels     []string        `json:"labels"`
	Replied    bool            `json:"replied"`
	CampaignID *uuid.UUID      `json:"campaign_id,omitempty"`
	AssigneeID *uuid.UUID      `json:"assignee_id,omitempty"`
	CMetadata  json.RawMessage `json:"cmetadata,omitempty"`
	IsDeleted  bool            `json:"is_deleted"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// EmailEntity represents an email in the response
type EmailEntity struct {
	ID             uuid.UUID  `json:"id"`
	Subject        *string    `json:"subject,omitempty"`
	Content        *string    `json:"content,omitempty"`
	FromEmail      *string    `json:"from_email,omitempty"`
	ToEmail        *string    `json:"to_email,omitempty"`
	Attachments    []string   `json:"attachments,omitempty"`
	Intents        []string   `json:"intents,omitempty"`
	GmailMessageID *string    `json:"gmail_message_id,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

// SearchConversations handles POST /v2/conversations/search
func (h *ConversationHandler) SearchConversations(c fiber.Ctx) error {
	var req ConversationSearchRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	offset := c.Query("offset", "0")
	limit := c.Query("limit", "25")
	offsetInt := 0
	limitInt := 25
	if val, err := strconv.Atoi(offset); err == nil {
		offsetInt = val
	}
	if val, err := strconv.Atoi(limit); err == nil {
		limitInt = val
		if limitInt > 100 {
			limitInt = 100
		}
	}

	conversationType := c.Query("conversation_type")

	// Build base query
	query := h.conversationRepo.GetDB().WithContext(c.Context()).
		Model(&domain.Conversation{}).
		Where("user_id = ? AND is_deleted = ?", userID, false)

	// The inbox tabs send this on every request. It used to return 501, which
	// meant three of the five tabs showed an error instead of their contents.
	switch conversationType {
	case "sent":
		// Threads this account has actually answered.
		query = query.Where("replied = ?", true)
	case "assign_to_human":
		// Handed to a teammate, so it is off the automated path.
		query = query.Where("assignee_id IS NOT NULL")
	case "assign_to_ai":
		// Nobody has claimed it and the pipeline left a draft to review.
		query = query.Where(
			"assignee_id IS NULL AND cmetadata -> 'ai_reply' ->> 'draft_content' <> ''")
	case "":
		// No tab filter — all mail.
	default:
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Unknown conversation_type",
			"conversation_type must be one of: sent, assign_to_ai, assign_to_human",
		))
	}

	// Apply custom filter if provided
	if req.Filter != nil {
		for k, v := range req.Filter {
			switch k {
			case "labels":
				if labels, ok := v.([]interface{}); ok {
					var labelStrs []string
					for _, label := range labels {
						if str, ok := label.(string); ok {
							labelStrs = append(labelStrs, str)
						}
					}
					query = query.Where("labels @> ?", labelStrs)
				}
			case "replied":
				if replied, ok := v.(bool); ok {
					query = query.Where("replied = ?", replied)
				}
			case "campaign_id":
				if campaignID, ok := v.(string); ok {
					if id, err := uuid.Parse(campaignID); err == nil {
						query = query.Where("campaign_id = ?", id)
					}
				}

			case "q":
				// Free-text search over the messages, not the thread row: a
				// conversation carries no subject or body of its own, so the
				// terms a user actually remembers only exist on its emails.
				term, ok := v.(string)
				if !ok {
					break
				}
				term = strings.TrimSpace(term)
				if term == "" {
					break
				}
				like := "%" + strings.ReplaceAll(
					strings.ReplaceAll(term, "%", `\%`), "_", `\_`) + "%"
				query = query.Where(`id IN (
						SELECT conversation_id FROM emails
						WHERE conversation_id IS NOT NULL
						  AND (subject ILIKE ? OR content ILIKE ?
						       OR from_email ILIKE ? OR to_email ILIKE ?)
					)`, like, like, like, like)

			case "intent":
				// Both the thread and its emails carry a classified intent;
				// match either so a thread is found by the label the user saw
				// on it in the list.
				if intent, ok := v.(string); ok && intent != "" {
					query = query.Where(`(intents::text ILIKE ? OR id IN (
							SELECT conversation_id FROM emails
							WHERE conversation_id IS NOT NULL
							  AND intents::text ILIKE ?
						))`, "%"+intent+"%", "%"+intent+"%")
				}

			case "has_draft":
				// Threads where the pipeline left a reply waiting for review.
				if want, ok := v.(bool); ok {
					cond := "cmetadata -> 'ai_reply' ->> 'draft_content' <> ''"
					if want {
						query = query.Where(cond)
					} else {
						query = query.Where("NOT (" + cond + ") OR cmetadata -> 'ai_reply' IS NULL")
					}
				}

			case "since_days":
				// Relative rather than absolute dates: the client asks for
				// "last 7 days" without either side agreeing on a timezone.
				if days, ok := v.(float64); ok && days > 0 {
					query = query.Where("updated_at >= ?",
						time.Now().AddDate(0, 0, -int(days)))
				}
			}
		}
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to count conversations")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to count conversations",
		})
	}

	// Get conversations
	var conversations []domain.Conversation
	if err := query.
		Order("updated_at DESC").
		Offset(offsetInt * limitInt).
		Limit(limitInt).
		Find(&conversations).Error; err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to get conversations")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to get conversations",
		})
	}

	// Get latest email for each conversation
	var conversationIDs []uuid.UUID
	for _, conv := range conversations {
		conversationIDs = append(conversationIDs, conv.ID)
	}

	// Get latest emails - one per conversation
	latestEmails := make(map[uuid.UUID]*domain.Email)
	if len(conversationIDs) > 0 {
		var emails []domain.Email
		if err := h.emailRepo.GetDB().WithContext(c.Context()).
			Raw(`
				SELECT DISTINCT ON (conversation_id) *
				FROM emails
				WHERE conversation_id = ANY($1)
				ORDER BY conversation_id, created_at DESC
			`, conversationIDs).
			Scan(&emails).Error; err != nil {
			logger.Logger.Error().Err(err).Msg("Failed to get latest emails")
		} else {
			for i := range emails {
				if emails[i].ConversationID != nil {
					latestEmails[*emails[i].ConversationID] = &emails[i]
				}
			}
		}
	}

	// Build response
	var list []map[string]interface{}
	for _, conv := range conversations {
		// Parse conversation metadata
		var cmetadata map[string]interface{}
		if len(conv.CMetadata) > 0 {
			json.Unmarshal(conv.CMetadata, &cmetadata)
		}

		convEntity := ConversationEntity{
			ID:         conv.ID,
			UserID:     conv.UserID,
			Labels:     conv.Labels,
			Replied:    conv.Replied,
			CampaignID: conv.CampaignID,
			AssigneeID: conv.AssigneeID,
			CMetadata:  json.RawMessage(conv.CMetadata),
			IsDeleted:  conv.IsDeleted,
			CreatedAt:  conv.CreatedAt,
			UpdatedAt:  conv.UpdatedAt,
		}

		item := map[string]interface{}{
			"entity": convEntity,
		}

		// Add latest email if available
		if latestEmail, exists := latestEmails[conv.ID]; exists {
			emailEntity := EmailEntity{
				ID:        latestEmail.ID,
				Subject:   &latestEmail.Subject,
				Content:   &latestEmail.Content,
				FromEmail: &latestEmail.FromEmail,
				ToEmail:   &latestEmail.ToEmail,
				CreatedAt: &latestEmail.CreatedAt,
				UpdatedAt: &latestEmail.UpdatedAt,
			}

			if latestEmail.GmailMessageID != "" {
				emailEntity.GmailMessageID = &latestEmail.GmailMessageID
			}

			// Attachments and Intents are already pq.StringArray, just copy them
			if len(latestEmail.Attachments) > 0 {
				emailEntity.Attachments = latestEmail.Attachments
			}

			if len(latestEmail.Intents) > 0 {
				emailEntity.Intents = latestEmail.Intents
			}

			item["latest_email"] = emailEntity
		}

		list = append(list, item)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"list":   list,
			"offset": offsetInt,
			"limit":  limitInt,
			"total":  total,
		},
	})
}
