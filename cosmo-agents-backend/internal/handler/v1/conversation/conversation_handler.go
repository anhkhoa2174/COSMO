package conversation

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1filter "github.com/rockship/cosmo-agents-go/internal/handler/v1/filter"
	v1pagination "github.com/rockship/cosmo-agents-go/internal/handler/v1/pagination"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// Input validation constants
const (
	MaxConversationTypeLength = 50
	MaxPaginationLimit        = 100
)

// Valid conversation types (whitelist for security)
var ValidConversationTypes = map[string]bool{
	"sent":            true,
	"assign_to_ai":    true,
	"assign_to_human": true,
	"":                true, // Empty string is valid (means no filter)
}

// Exported variables and constants for other packages
var ValidConversationTypesExported = map[string]bool{
	"sent":            true,
	"assign_to_ai":    true,
	"assign_to_human": true,
	"":                true, // Empty string is valid (means no filter)
}

const MaxConversationTypeLengthExported = 50

// ConversationHandler handles conversation-related HTTP requests
type ConversationHandler struct {
	conversationRepo *conversationRepo.ConversationRepository
	emailRepo        *emailRepo.Repository
	userRepo         *user.UserRepository
}

// NewConversationHandler creates a new ConversationHandler
func NewConversationHandler(conversationRepo *conversationRepo.ConversationRepository, emailRepo *emailRepo.Repository, userRepo *user.UserRepository) *ConversationHandler {
	return &ConversationHandler{
		conversationRepo: conversationRepo,
		emailRepo:        emailRepo,
		userRepo:         userRepo,
	}
}

// toEmailEntity converts domain.Email to v1schema.EmailEntity (DRY principle)
func (h *ConversationHandler) toEmailEntity(email *domain.Email) v1schema.EmailEntity {
	return toEmailEntityFromDomain(email)
}

// toEmailEntityFromDomain is a package-level helper that converts domain.Email
// to API EmailEntity and ensures slices are non-nil so JSON encodes empty arrays.
func toEmailEntityFromDomain(email *domain.Email) v1schema.EmailEntity {
	if email == nil {
		empty := ""
		return v1schema.EmailEntity{
			Subject:     &empty,
			Content:     &empty,
			FromEmail:   &empty,
			ToEmail:     &empty,
			Attachments: []string{},
			Intents:     []string{},
		}
	}

	emailID := email.ID

	// Convert pq.StringArray (alias of []string) to plain []string and ensure non-nil
	var attachments []string
	if email.Attachments == nil {
		attachments = []string{}
	} else {
		attachments = []string(email.Attachments)
	}

	var intents []string
	if email.Intents == nil {
		intents = []string{}
	} else {
		intents = []string(email.Intents)
	}

	return v1schema.EmailEntity{
		ID:          &emailID,
		Subject:     &email.Subject,
		Content:     &email.Content,
		FromEmail:   &email.FromEmail,
		ToEmail:     &email.ToEmail,
		Attachments: attachments,
		Intents:     intents,
		CreatedAt:   &email.CreatedAt,
		UpdatedAt:   &email.UpdatedAt,
	}
}

// ToEmailEntityFromDomain is an exported version for other packages to use
func ToEmailEntityFromDomain(email *domain.Email) v1schema.EmailEntity {
	return toEmailEntityFromDomain(email)
}

// handleError logs internal errors and returns sanitized responses (security best practice)
func (h *ConversationHandler) handleError(c fiber.Ctx, err error, message string) error {
	// TODO: Add proper logging here (import logger package)
	// logger.Logger.Error().Err(err).Msg(message)

	// Don't expose internal error details to prevent information leakage
	return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
		fiber.StatusInternalServerError, message, "",
	))
}

// Search handles POST /v1/conversations/search
// @Summary Search conversations
// @Description Search conversations owned by the authenticated user with optional filters
// @Tags Conversations
// @Accept json
// @Produce json
// @Param conversation_type query string false "Conversation Type (sent, assign_to_ai, assign_to_human)"
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit (max 100)" default(25)
// @Param body body v1schema.ConversationSearchRequest false "Search filters"
// @Success 200 {object} schema.APIResponse[v1schema.ConversationSearchResponse] "Successfully retrieved conversations"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/conversations/search [post]
func (h *ConversationHandler) Search(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	// Parse and validate pagination parameters with proper error reporting
	pagination, paginationErr := v1pagination.ValidatePaginationOrError(c)
	if paginationErr != nil {
		return paginationErr // Error response already sent by ValidatePaginationOrError
	}
	offset := pagination.Offset
	limit := pagination.Limit

	conversationType := c.Query("conversation_type")

	// Validate conversation type (whitelist approach for security)
	if !ValidConversationTypes[conversationType] {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid conversation type",
			"Allowed values are: sent, assign_to_ai, assign_to_human, or empty",
		))
	}

	// Additional length validation as safety net
	if len(conversationType) > MaxConversationTypeLength {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Conversation type too long",
			"Maximum length is 50 characters",
		))
	}

	// Parse request body
	var req v1schema.ConversationSearchRequest
	if err := c.Bind().JSON(&req); err != nil {
		// Empty body is acceptable
		req.Filter = nil
	}

	// Validate filter to prevent injection attacks
	if err := v1filter.ValidateFilter(req.Filter); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid filter", err.Error(),
		))
	}

	// Search conversations
	conversations, total, err := h.conversationRepo.SearchConversations(
		c.Context(), userID, conversationType, offset, limit, req.Filter,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to search conversations", err.Error(),
		))
	}

	// Fetch latest emails for all conversations in batch (prevents N+1 queries)
	var emails []*domain.Email
	if len(conversations) > 0 {
		conversationIDs := make([]uuid.UUID, len(conversations))
		for i, conv := range conversations {
			conversationIDs[i] = conv.ID
		}

		// Use batch method to get latest emails efficiently
		emailMap, err := h.emailRepo.FindLatestByConversationIDs(c.Context(), conversationIDs)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
				fiber.StatusInternalServerError, "Failed to fetch emails", err.Error(),
			))
		}
		if emailMap == nil {
			emailMap = make(map[uuid.UUID]*domain.Email)
		}

		// Pre-allocate slice with correct size to ensure index alignment
		emails = make([]*domain.Email, len(conversations))

		// Map emails to correct positions based on conversation order
		for i, conv := range conversations {
			if email, exists := emailMap[conv.ID]; exists {
				emails[i] = email
			} else {
				emails[i] = nil // Explicit nil for conversations without emails
			}
		}
	}

	// Build response with conversation entities and latest emails
	items := make([]v1schema.ConversationListItem, len(conversations))

	for i, conversation := range conversations {
		items[i] = v1schema.ConversationListItem{
			Entity: v1schema.ConversationEntity{
				ID:         conversation.ID,
				UserID:     conversation.UserID,
				Labels:     []string(conversation.Labels),
				Replied:    conversation.Replied,
				CampaignID: conversation.CampaignID,
				AssigneeID: conversation.AssigneeID,
				Intents:    []string(conversation.Intents),
				Status:     string(conversation.Status),
				IsDeleted:  conversation.IsDeleted,
				CreatedAt:  conversation.CreatedAt,
				UpdatedAt:  conversation.UpdatedAt,
			},
		}

		// Always populate LatestEmail (helper returns empty entity with empty arrays when email is nil)
		if i < len(emails) {
			items[i].LatestEmail = h.toEmailEntity(emails[i])
		} else {
			items[i].LatestEmail = h.toEmailEntity(nil)
		}
	}

	response := v1schema.ConversationSearchResponse{
		List:   items,
		Offset: offset,
		Limit:  limit,
		Total:  total,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// GetAssignees handles GET /v1/conversations/assignee
// @Summary Get assignees
// @Description Get distinct assignees from conversations owned by the authenticated user
// @Tags Conversations
// @Accept json
// @Produce json
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit (max 100)" default(25)
// @Success 200 {object} schema.APIResponse[v1schema.AssigneeListResponse] "Successfully retrieved assignees"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/conversations/assignee [get]
func (h *ConversationHandler) GetAssignees(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	// Parse and validate pagination parameters with proper error reporting
	pagination, paginationErr := v1pagination.ValidatePaginationOrError(c)
	if paginationErr != nil {
		return paginationErr // Error response already sent by ValidatePaginationOrError
	}
	offset := pagination.Offset
	limit := pagination.Limit

	// Get distinct assignee IDs
	assigneeIDs, total, err := h.conversationRepo.GetDistinctAssignees(c.Context(), userID, offset, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to get assignees", err.Error(),
		))
	}

	// Fetch user details for all assignees in one query (avoids N+1)
	users, err := h.userRepo.FindByIDs(c.Context(), assigneeIDs)
	if err != nil {
		return h.handleError(c, err, "Failed to fetch user details")
	}
	if users == nil {
		users = []*domain.User{} // Prevent nil slice iteration
	}

	// Convert to assignee entities
	assignees := make([]v1schema.AssigneeEntity, 0, len(users))
	for _, user := range users {
		assignees = append(assignees, v1schema.AssigneeEntity{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		})
	}

	response := v1schema.AssigneeListResponse{
		List:   assignees,
		Offset: offset,
		Limit:  limit,
		Total:  total,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// GetConversationDetail handles GET /v1/conversations/:id
// @Summary Get conversation detail
// @Description Retrieves a conversation with all its emails and marks it as read
// @Tags Conversations
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID (UUID)"
// @Success 200 {object} schema.APIResponse[v1schema.ConversationReadResponse] "Successfully retrieved conversation"
// @Failure 400 {object} schema.APIResponse[any] "Invalid conversation ID"
// @Failure 403 {object} schema.APIResponse[any] "Permission denied"
// @Failure 404 {object} schema.APIResponse[any] "Conversation not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/conversations/{id} [get]
func (h *ConversationHandler) GetConversationDetail(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid conversation ID format", err.Error(),
		))
	}

	// Find conversation (including deleted ones as per Python spec)
	conversation, err := h.conversationRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Conversation not found", err.Error(),
		))
	}

	// Check ownership
	if conversation.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "You don't have permission to view this conversation", "",
		))
	}

	// Get all emails in the conversation
	emails, err := h.emailRepo.FindByConversationID(c.Context(), conversation.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to fetch emails", err.Error(),
		))
	}

	// Mark as read (similar to Python implementation)
	// Make a copy to avoid inconsistent state if update fails
	conversationCopy := *conversation
	conversationCopy.Read()

	if err := h.conversationRepo.Update(c.Context(), conversationCopy.ID, &conversationCopy); err != nil {
		log.Error().
			Err(err).
			Str("conversation_id", conversation.ID.String()).
			Msg("CRITICAL: failed to mark conversation as read - returning error to prevent inconsistent state")

		// Return error instead of continuing with inconsistent state
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to mark conversation as read",
			"Please try again",
		))
	}

	// Update was successful, re-fetch the updated conversation from DB to ensure latest state
	updatedConversation, err := h.conversationRepo.FindByID(c.Context(), conversationCopy.ID)
	if err != nil {
		log.Error().
			Err(err).
			Str("conversation_id", conversationCopy.ID.String()).
			Msg("CRITICAL: failed to fetch updated conversation after mark as read")
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to fetch updated conversation after mark as read",
			"Please try again",
		))
	}
	conversation = updatedConversation

	// Build email entities
	emailEntities := make([]v1schema.EmailEntity, len(emails))
	for i, email := range emails {
		emailEntities[i] = h.toEmailEntity(email)
	}

	response := v1schema.ConversationReadResponse{
		ID:         conversation.ID,
		UserID:     conversation.UserID,
		Labels:     conversation.Labels,
		Replied:    conversation.Replied,
		CampaignID: conversation.CampaignID,
		AssigneeID: conversation.AssigneeID,
		Status:     string(conversation.Status),
		CreatedAt:  conversation.CreatedAt,
		UpdatedAt:  conversation.UpdatedAt,
		Emails:     emailEntities,
		IsDeleted:  conversation.IsDeleted,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// DeleteConversation handles DELETE /v1/conversations/:id
// @Summary Delete conversation
// @Description Soft-delete a conversation owned by the requester
// @Tags Conversations
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID (UUID)"
// @Success 200 {object} schema.APIResponse[string] "Successfully deleted"
// @Failure 400 {object} schema.APIResponse[any] "Invalid conversation ID"
// @Failure 403 {object} schema.APIResponse[any] "Permission denied"
// @Failure 404 {object} schema.APIResponse[any] "Conversation not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/conversations/{id} [delete]
func (h *ConversationHandler) DeleteConversation(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid conversation ID format", err.Error(),
		))
	}

	// Find conversation
	conversation, err := h.conversationRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Conversation not found", err.Error(),
		))
	}

	// Check ownership
	if conversation.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "You don't have permission to delete this conversation", "",
		))
	}

	// Soft delete
	if err := h.conversationRepo.SoftDelete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to delete conversation", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse("Successfully deleted"))
}

// AssignConversation handles POST /v1/conversations/:id/assign
// @Summary Assign conversation to user
// @Description Assigns a conversation to a specific user (modified to match Python spec)
// @Tags Conversations
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID (UUID)"
// @Param assignee_id query string false "Assignee ID (UUID)"
// @Param conversation_type query string false "Conversation Type"
// @Success 200 {object} schema.APIResponse[v1schema.ConversationResponse] "Successfully assigned conversation"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Permission denied"
// @Failure 404 {object} schema.APIResponse[any] "Conversation not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/conversations/{id}/assign [post]
func (h *ConversationHandler) AssignConversation(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid conversation ID format", err.Error(),
		))
	}

	// Get assignee_id from query parameter (optional)
	var assigneeID *uuid.UUID
	if assigneeIDParam := c.Query("assignee_id"); assigneeIDParam != "" {
		parsed, err := uuid.Parse(assigneeIDParam)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest, "Invalid assignee ID format", err.Error(),
			))
		}
		assigneeID = &parsed
	}

	conversationType := c.Query("conversation_type")

	conversation, err := h.conversationRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Conversation not found", err.Error(),
		))
	}

	if conversation.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "You don't have permission to assign this conversation", "",
		))
	}

	// Apply assignment based on conversation type or direct assignee_id
	if assigneeID != nil {
		conversation.AssigneeID = assigneeID
	} else if conversationType != "" {
		// Handle assignment logic based on conversation type
		// This is simplified - in production you might have more complex logic
		if conversationType == string(domain.ConversationTypeAI) {
			conversation.AssigneeID = nil
		}
		// For ConversationTypeHuman, you might want to assign to a specific user
		// For now, we'll leave it as is if no assignee_id is provided
	}

	if err := h.conversationRepo.Update(c.Context(), conversation.ID, conversation); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to assign conversation", err.Error(),
		))
	}

	response := toConversationResponse(conversation)
	return c.JSON(schema.SuccessResponse(response))
}

// Helper function
func toConversationResponse(conversation *domain.Conversation) v1schema.ConversationResponse {
	metadata := make(map[string]any)
	if len(conversation.CMetadata) > 0 {
		if err := json.Unmarshal(conversation.CMetadata, &metadata); err != nil {
			log.Warn().
				Err(err).
				Str("conversation_id", conversation.ID.String()).
				Msg("failed to unmarshal conversation metadata")
			metadata = make(map[string]any)
		}
	}

	return v1schema.ConversationResponse{
		ID:            conversation.ID,
		CreatedAt:     conversation.CreatedAt,
		UpdatedAt:     conversation.UpdatedAt,
		UserID:        conversation.UserID,
		GmailThreadID: conversation.GmailThreadID,
		CampaignID:    conversation.CampaignID,
		AgentID:       conversation.AgentID,
		AssigneeID:    conversation.AssigneeID,
		Labels:        conversation.Labels,
		Intents:       conversation.Intents,
		Replied:       conversation.Replied,
		Status:        string(conversation.Status),
		Metadata:      metadata,
		IsDeleted:     conversation.IsDeleted,
	}
}
