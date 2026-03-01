package campaign

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	mailService "github.com/rockship/cosmo-agents-go/internal/service/mail"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// GenerateSampleResponse handles POST /v2/campaigns/{campaign_id}/generate-sample-response
// @Summary Generate sample email response
// @Description Generates a sample response based on intent type
// @Tags Campaigns V2 - has V3
// @Accept json
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Param body body v2schema.CampaignGenerateSampleResponseRequest true "Sample response request"
// @Success 200 {object} schema.APIResponse[v2schema.CampaignGenerateSampleResponseResponse] "Sample response"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Security BearerAuth
// @Router /v2/campaigns/{campaign_id}/generate-sample-response [post]
func (h *Handler) GenerateSampleResponse(c fiber.Ctx) error {
	// Check if OpenAI client is available
	if h.openAIClient == nil {
		return serviceUnavailable(c, "AI service unavailable", fmt.Errorf("OpenAI client is not configured"))
	}

	campaignIDStr := c.Params("campaign_id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	var req v2schema.CampaignGenerateSampleResponseRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	// Verify campaign exists
	_, err = h.campaignRepo.FindByID(c.Context(), campaignID)
	if err != nil {
		return notFound(c, "Campaign not found")
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
		return internalError(c, "Failed to generate sample response", err)
	}

	response := v2schema.CampaignGenerateSampleResponseResponse{
		Intent:   req.Intent,
		Response: sampleResponse,
	}

	logger.Logger.Info().
		Str("campaign_id", campaignID.String()).
		Str("intent", string(req.Intent)).
		Msg("Generated sample response")

	return c.JSON(schema.SuccessResponse(response))
}

// ClassifySampleResponse handles POST /v2/campaigns/{campaign_id}/classify-sample-response
// @Summary Classify sample response intent
// @Description Classifies the intent of a sample email response
// @Tags Campaigns V2 - has V3
// @Accept json
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Param body body v2schema.CampaignClassifySampleResponseRequest true "Classification request"
// @Success 200 {object} schema.APIResponse[string] "Classified intent"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Security BearerAuth
// @Router /v2/campaigns/{campaign_id}/classify-sample-response [post]
func (h *Handler) ClassifySampleResponse(c fiber.Ctx) error {
	var req v2schema.CampaignClassifySampleResponseRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	if h.intentClassifier == nil {
		return internalError(c, "Intent classifier not available", nil)
	}

	// Classify intent
	intent, err := h.intentClassifier.Classify(c.Context(), req.Response)
	if err != nil {
		return internalError(c, "Failed to classify intent", err)
	}

	logger.Logger.Info().
		Str("classified_intent", string(intent)).
		Msg("Classified sample response")

	return c.JSON(schema.SuccessResponse(string(intent)))
}

// GenerateReply handles POST /v2/campaigns/{campaign_id}/generate-reply
// @Summary Generate email reply
// @Description Generates an AI reply for a conversation based on intent
// @Tags Campaigns V2 - has V3
// @Accept json
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Param body body v2schema.CampaignGenerateReplyRequest true "Reply request"
// @Success 200 {object} schema.APIResponse[any] "Generated reply"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Security BearerAuth
// @Router /v2/campaigns/{campaign_id}/generate-reply [post]
func (h *Handler) GenerateReply(c fiber.Ctx) error {
	// Check if OpenAI client is available
	if h.openAIClient == nil {
		return serviceUnavailable(c, "AI service unavailable", fmt.Errorf("OpenAI client is not configured"))
	}

	campaignIDStr := c.Params("campaign_id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	var req v2schema.CampaignGenerateReplyRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	// Get current user
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	// Fetch campaign
	campaign, err := h.campaignRepo.FindByID(c.Context(), campaignID)
	if err != nil {
		return notFound(c, "Campaign not found")
	}

	// Fetch user
	user, err := h.userRepo.FindByID(c.Context(), userID)
	if err != nil {
		return notFound(c, "User not found")
	}

	// Get user's organization
	var org *domain.Organization
	if campaign.OrganizationID != nil {
		org, _ = h.orgRepo.FindByID(c.Context(), *campaign.OrganizationID)
	}

	// Convert conversation to email templates
	conversation := make([]mailService.EmailTemplate, len(req.Conversation))
	for i, mail := range req.Conversation {
		conversation[i] = mailService.EmailTemplate{
			FromEmail: mail.FromEmail,
			ToEmail:   mail.ToEmail,
			Subject:   mail.Subject,
			Content:   mail.Content,
		}
	}

	// Determine intent
	intent := domain.IntentInterested
	if req.Intent != nil {
		intent = *req.Intent
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

	// Build email parameters
	emailParams := mailService.EmailParameters{
		Sender:       senderInfo,
		CampaignType: mailService.CampaignType(campaign.Playbook),
		Tone:         "Zero marketing jargon, write the way you speak (conversational) Not overly formal",
	}

	// Create mail writer
	mailWriter := mailService.NewMailWriter(h.openAIClient, "gpt-4o-mini", emailParams, &logger.Logger)

	// Generate reply
	reply, err := mailWriter.GenerateReply(c.Context(), conversation, mailService.IntentType(intent))
	if err != nil {
		return internalError(c, "Failed to generate reply", err)
	}

	result := map[string]interface{}{
		"subject": reply.Subject,
		"content": reply.Content,
		"type":    reply.Type,
	}

	logger.Logger.Info().
		Str("campaign_id", campaign.ID.String()).
		Str("intent", string(intent)).
		Msg("Generated reply")

	return c.JSON(schema.SuccessResponse(result))
}
