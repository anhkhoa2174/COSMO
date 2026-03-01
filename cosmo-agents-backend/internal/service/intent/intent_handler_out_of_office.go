package intent

import (
	"context"
	"fmt"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	"github.com/rs/zerolog"
)

// OutOfOfficeHandler handles out-of-office auto-replies
// It marks the conversation as replied to avoid repeated processing
type OutOfOfficeHandler struct {
	emailRepo        *emailRepo.Repository
	contactRepo      *contactRepo.ContactRepository
	conversationRepo *conversationRepo.ConversationRepository
	logger           *zerolog.Logger
}

// NewOutOfOfficeHandler creates a new out-of-office handler
func NewOutOfOfficeHandler(
	emailRepo *emailRepo.Repository,
	contactRepo *contactRepo.ContactRepository,
	conversationRepo *conversationRepo.ConversationRepository,
	log *zerolog.Logger,
) *OutOfOfficeHandler {
	return &OutOfOfficeHandler{
		emailRepo:        emailRepo,
		contactRepo:      contactRepo,
		conversationRepo: conversationRepo,
		logger:           log,
	}
}

// Execute marks the conversation as replied (no further action needed for OOO)
func (h *OutOfOfficeHandler) Execute(
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

	// Mark as replied so we don't process this conversation again

	conversation.Replied = true

	if err := h.conversationRepo.Update(ctx, conversation.ID, conversation); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update conversation replied status")
		return false, err
	}

	h.logger.Info().
		Str("conversation_id", conversation.ID.String()).
		Str("email_id", email.ID.String()).
		Msg("Successfully handled out-of-office reply")

	return true, nil
}
