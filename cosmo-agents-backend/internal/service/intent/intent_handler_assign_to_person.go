package intent

import (
	"context"
	"fmt"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	"github.com/rs/zerolog"

	"github.com/google/uuid"
)

// AssignToPersonHandler assigns the conversation to a specific user
type AssignToPersonHandler struct {
	conversationRepo *conversationRepo.ConversationRepository
	assigneeID       *uuid.UUID
	logger           *zerolog.Logger
}

// NewAssignToPersonHandler creates a new assign-to-person handler
func NewAssignToPersonHandler(
	conversationRepo *conversationRepo.ConversationRepository,
	assigneeID *uuid.UUID,
	log *zerolog.Logger,
) *AssignToPersonHandler {
	return &AssignToPersonHandler{
		conversationRepo: conversationRepo,
		assigneeID:       assigneeID,
		logger:           log,
	}
}

// Execute assigns the conversation to the specified user
func (h *AssignToPersonHandler) Execute(
	ctx context.Context,
	campaign *domain.Campaign,
	intent domain.IntentType,
	email *domain.Email,
) (bool, error) {
	// Find conversation
	conversation, err := h.conversationRepo.FindByID(ctx, *email.ConversationID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to find conversation")
		return false, fmt.Errorf("failed to find conversation: %w", err)
	}

	// Use provided assignee ID, or default to campaign user
	assigneeID := h.assigneeID
	if assigneeID == nil {
		assigneeID = &campaign.UserID
	}

	// Assign conversation to user
	conversation.AssigneeID = assigneeID

	// Mark as replied

	conversation.Replied = true

	if err := h.conversationRepo.Update(ctx, conversation.ID, conversation); err != nil {
		h.logger.Error().Err(err).Msg("Failed to assign conversation")
		return false, err
	}

	h.logger.Info().
		Str("conversation_id", conversation.ID.String()).
		Str("assignee_id", assigneeID.String()).
		Msg("Successfully assigned conversation to person")

	return true, nil
}
