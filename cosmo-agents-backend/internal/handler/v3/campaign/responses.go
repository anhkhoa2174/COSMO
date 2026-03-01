package campaign

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	v3schema "github.com/rockship/cosmo-agents-go/internal/schema/v3"
	mailService "github.com/rockship/cosmo-agents-go/internal/service/mail"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// GenerateSampleResponse handles POST /v3/campaigns/{campaign_id}/generate-sample-response
// @Summary Generate sample email response (V3)
// @Description Generates a sample response based on intent type (V3 version)
// @Tags Campaigns V3
// @Accept json
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Param body body v3schema.CampaignGenerateSampleResponseRequest true "Sample response request"
// @Success 200 {object} schema.APIResponse[any] "Sample response"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Security BearerAuth
// @Router /v3/campaigns/{campaign_id}/generate-sample-response [post]
func (h *Handler) GenerateSampleResponse(c fiber.Ctx) error {
	campaignIDStr := c.Params("campaign_id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err.Error())
	}

	var req v3schema.CampaignGenerateSampleResponseRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err.Error())
	}

	// Verify campaign exists
	_, err = h.campaignRepo.FindByID(c.Context(), campaignID)
	if err != nil {
		return notFound(c, "Campaign not found", err.Error())
	}

	// Convert intent to service.IntentType
	intentType := mailService.IntentType(req.Intent)

	// Generate sample response
	sampleResponse, err := mailService.GenerateSampleResponse(
		c.Context(),
		h.openAIClient,
		"gpt-4o-mini",
		req.OutreachTemplate,
		intentType,
		req.ContactData,
		&logger.Logger,
	)
	if err != nil {
		return internalError(c, "Failed to generate sample response", err.Error())
	}

	result := map[string]interface{}{
		"intent":   req.Intent,
		"response": sampleResponse,
	}

	logger.Logger.Info().
		Str("campaign_id", campaignID.String()).
		Str("intent", string(req.Intent)).
		Msg("Generated sample response (V3)")

	return c.JSON(schema.SuccessResponse(result))
}

// GenerateReply handles POST /v3/campaigns/{campaign_id}/generate-reply
// @Summary Generate email reply (V3)
// @Description Generates an AI reply for a conversation based on intent (V3 version)
// @Tags Campaigns V3
// @Accept json
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Param body body v2schema.CampaignGenerateReplyRequest true "Reply request"
// @Success 200 {object} schema.APIResponse[any] "Generated reply"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Security BearerAuth
// @Router /v3/campaigns/{campaign_id}/generate-reply [post]
func (h *Handler) GenerateReply(c fiber.Ctx) error {
	campaignIDStr := c.Params("campaign_id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err.Error())
	}

	var req v2schema.CampaignGenerateReplyRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err.Error())
	}

	// Get current user
	userUUID, ok := userIDFromContext(c)
	if !ok {
		return unauthorized(c, "Unauthorized", "User ID not found")
	}

	// Fetch campaign
	campaign, err := h.campaignRepo.FindByID(c.Context(), campaignID)
	if err != nil {
		return notFound(c, "Campaign not found", err.Error())
	}

	// Fetch user
	user, err := h.userRepo.FindByID(c.Context(), userUUID)
	if err != nil {
		return notFound(c, "User not found", err.Error())
	}

	// Get user's organization
	var org *domain.Organization
	if campaign.OrganizationID != nil {
		org, _ = h.orgRepo.FindByID(c.Context(), *campaign.OrganizationID)
	}

	// Build sender info
	senderInfo := mailService.SenderInfo{
		Name:  user.Name,
		Email: user.Email,
	}

	if org != nil {
		senderInfo.CompanyName = org.Name
		senderInfo.CompanyDescription = org.CompanyDescription
		if len(org.CompanyTargetingPersona) > 0 {
			senderInfo.CompanyTargetingPersona = org.CompanyTargetingPersona[0]
		}
		senderInfo.ValueOffering = org.ValueOffering
	}

	// Convert conversation to service format
	conversation := make([]mailService.EmailTemplate, len(req.Conversation))
	for i, mail := range req.Conversation {
		conversation[i] = mailService.EmailTemplate{
			FromEmail: mail.FromEmail,
			ToEmail:   mail.ToEmail,
			Subject:   mail.Subject,
			Content:   mail.Content,
		}
	}

	// Build email parameters
	emailParams := mailService.EmailParameters{
		Sender:       senderInfo,
		CampaignType: mailService.CampaignType(campaign.Playbook),
		Tone:         "Zero marketing jargon, write the way you speak (conversational) Not overly formal",
	}

	// Create mail writer
	mailWriter := mailService.NewMailWriter(h.openAIClient, "gpt-4o-mini", emailParams, &logger.Logger)

	// Determine intent
	intent := domain.IntentInterested
	if req.Intent != nil {
		intent = *req.Intent
	}

	reply, err := mailWriter.GenerateReply(c.Context(), conversation, mailService.IntentType(intent))
	if err != nil {
		return internalError(c, "Failed to generate reply", err.Error())
	}

	result := map[string]interface{}{
		"subject": reply.Subject,
		"content": reply.Content,
		"type":    reply.Type,
	}

	logger.Logger.Info().
		Str("campaign_id", campaign.ID.String()).
		Str("intent", string(intent)).
		Msg("Generated reply (V3)")

	return c.JSON(schema.SuccessResponse(result))
}
