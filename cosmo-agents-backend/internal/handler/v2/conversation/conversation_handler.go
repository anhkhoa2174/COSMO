package conversation

import (
	"encoding/json"
	"strconv"
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
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	Labels     []string   `json:"labels"`
	Replied    bool       `json:"replied"`
	CampaignID *uuid.UUID `json:"campaign_id,omitempty"`
	AssigneeID *uuid.UUID `json:"assignee_id,omitempty"`
	IsDeleted  bool       `json:"is_deleted"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
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

	// conversation_type filtering is not yet implemented - return 501 if requested
	if conversationType != "" {
		logger.Logger.Warn().
			Str("conversation_type", conversationType).
			Str("user_id", userID.String()).
			Msg("conversation_type filtering requested but not implemented")
		return c.Status(fiber.StatusNotImplemented).JSON(schema.ErrorResponse(
			fiber.StatusNotImplemented,
			"Feature not implemented",
			"conversation_type filtering is not yet supported. Please use other filters.",
		))
	}

	// Build base query
	query := h.conversationRepo.GetDB().WithContext(c.Context()).
		Model(&domain.Conversation{}).
		Where("user_id = ? AND is_deleted = ?", userID, false)

	// conversation_type filtering removed - feature returns 501 Not Implemented above

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
