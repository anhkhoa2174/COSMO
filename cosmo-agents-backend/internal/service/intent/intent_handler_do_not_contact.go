package intent

import (
	"context"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"

	"github.com/rs/zerolog"
)

// DoNotContactHandler marks the contact as do-not-contact
type DoNotContactHandler struct {
	emailRepo        *emailRepo.Repository
	contactRepo      *contactRepo.ContactRepository
	conversationRepo *conversationRepo.ConversationRepository
	logger           *zerolog.Logger
}

// NewDoNotContactHandler creates a new do-not-contact handler
func NewDoNotContactHandler(
	emailRepo *emailRepo.Repository,
	contactRepo *contactRepo.ContactRepository,
	conversationRepo *conversationRepo.ConversationRepository,
	log *zerolog.Logger,
) *DoNotContactHandler {
	return &DoNotContactHandler{
		emailRepo:        emailRepo,
		contactRepo:      contactRepo,
		conversationRepo: conversationRepo,
		logger:           log,
	}
}

// Execute marks the contact as do-not-contact and marks conversation as replied
func (h *DoNotContactHandler) Execute(
	ctx context.Context,
	campaign *domain.Campaign,
	intent domain.IntentType,
	email *domain.Email,
) (bool, error) {
	// TODO: Find and update contact
	h.logger.Info().
		Str("from_email", email.FromEmail).
		Str("email_id", email.ID.String()).
		Msg("Do-not-contact handler - implementation simplified")

	// Mark conversation as replied
	conversation, err := h.conversationRepo.FindByID(ctx, *email.ConversationID)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to find conversation")
		return false, err
	}

	conversation.Replied = true
	if err := h.conversationRepo.Update(ctx, conversation.ID, conversation); err != nil {
		h.logger.Error().Err(err).Msg("Failed to update conversation replied status")
		return false, err
	}

	h.logger.Info().
		Str("conversation_id", conversation.ID.String()).
		Msg("Successfully marked conversation as replied")

	return true, nil
}
