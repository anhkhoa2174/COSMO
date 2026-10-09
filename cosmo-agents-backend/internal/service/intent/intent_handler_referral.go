package intent

import (
	"context"
	"regexp"
	"strings"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"

	"github.com/rs/zerolog"
)

// referralEmailRe matches email addresses mentioned in a referral reply body.
var referralEmailRe = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)

// maxReferralLeads bounds how many contacts one referral reply can create.
const maxReferralLeads = 3

// ReferralHandler creates a new lead for each contact the prospect refers us
// to, then marks the conversation replied so a human can follow up.
type ReferralHandler struct {
	emailRepo        *emailRepo.Repository
	contactRepo      *contactRepo.ContactRepository
	conversationRepo *conversationRepo.ConversationRepository
	logger           *zerolog.Logger
}

// NewReferralHandler creates a new referral handler
func NewReferralHandler(
	emailRepo *emailRepo.Repository,
	contactRepo *contactRepo.ContactRepository,
	conversationRepo *conversationRepo.ConversationRepository,
	log *zerolog.Logger,
) *ReferralHandler {
	return &ReferralHandler{
		emailRepo:        emailRepo,
		contactRepo:      contactRepo,
		conversationRepo: conversationRepo,
		logger:           log,
	}
}

// Execute extracts referred email addresses from the reply body and creates a
// contact (source "referral") for each address not already known.
func (h *ReferralHandler) Execute(
	ctx context.Context,
	campaign *domain.Campaign,
	intent domain.IntentType,
	email *domain.Email,
) (bool, error) {
	created := 0
	for _, addr := range referralEmailRe.FindAllString(email.Content, -1) {
		addr = strings.ToLower(strings.TrimRight(addr, "."))
		// The sender referring us is not the lead being referred.
		if strings.EqualFold(addr, email.FromEmail) || strings.EqualFold(addr, email.ToEmail) {
			continue
		}
		existing, err := h.contactRepo.FindByEmail(ctx, campaign.UserID, addr)
		if err != nil {
			h.logger.Error().Err(err).Str("email", addr).
				Msg("Referral: contact lookup failed")
			continue
		}
		if existing != nil {
			continue
		}
		lead := &domain.Contact{
			UserID:             campaign.UserID,
			OrganizationID:     campaign.OrganizationID,
			SourceID:           email.ID.String(),
			Source:             "referral",
			Name:               referralNameFromEmail(addr),
			ContactChannel:     "Email",
			ContactInformation: addr,
			Status:             "pending",
		}
		if err := h.contactRepo.Create(ctx, lead); err != nil {
			h.logger.Error().Err(err).Str("email", addr).
				Msg("Referral: failed to create lead contact")
			continue
		}
		created++
		h.logger.Info().
			Str("lead_email", addr).
			Str("referred_by", email.FromEmail).
			Str("campaign_id", campaign.ID.String()).
			Msg("Referral: created new lead contact")
		if created >= maxReferralLeads {
			break
		}
	}
	if created == 0 {
		h.logger.Warn().
			Str("email_id", email.ID.String()).
			Msg("Referral: no new lead extracted from reply; assigning for manual follow-up")
	}

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
	return true, nil
}

// referralNameFromEmail derives a readable placeholder name from the local
// part of an address ("maria.lopez" -> "Maria Lopez") until enrichment runs.
func referralNameFromEmail(addr string) string {
	local := strings.SplitN(addr, "@", 2)[0]
	parts := regexp.MustCompile(`[._\-]+`).Split(local, -1)
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	name := strings.TrimSpace(strings.Join(parts, " "))
	if name == "" {
		return "N/A"
	}
	return name
}
