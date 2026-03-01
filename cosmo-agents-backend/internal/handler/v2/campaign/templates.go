package campaign

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	mailService "github.com/rockship/cosmo-agents-go/internal/service/mail"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// GenerateTemplate handles POST /v2/campaigns/{campaign_id}/templates
// @Summary Generate new email template in sequence
// @Description Uses AI to generate a new outreach email template
// @Tags Campaigns V2 - has V3
// @Accept json
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Param body body v2schema.CampaignGenerateTemplateRequest true "Generation request"
// @Success 200 {object} schema.APIResponse[v2schema.CampaignGenerateTemplateResponse] "Generated template"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Security BearerAuth
// @Router /v2/campaigns/{campaign_id}/templates [post]
func (h *Handler) GenerateTemplate(c fiber.Ctx) error {
	// Check if OpenAI client is available
	if h.openAIClient == nil {
		return serviceUnavailable(c, "AI service unavailable", fmt.Errorf("OpenAI client is not configured"))
	}

	campaignIDStr := c.Params("campaign_id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	var req v2schema.CampaignGenerateTemplateRequest
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

	// Generate template using mail writer service
	response, err := h.generateCampaignTemplate(c.Context(), campaign, userID, req.Prompt, req.Tone, req.DocumentGids, nil)
	if err != nil {
		return internalError(c, "Failed to generate template", err)
	}

	logger.Logger.Info().
		Str("campaign_id", campaign.ID.String()).
		Str("template_id", response.ID.String()).
		Msg("Successfully generated campaign template")

	return c.JSON(schema.SuccessResponse(response))
}

// RegenerateTemplate handles POST /v2/campaigns/{campaign_id}/templates/{template_id}
// @Summary Regenerate template with feedback
// @Description Regenerate a specific template based on user feedback
// @Tags Campaigns V2 - has V3
// @Accept json
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Param template_id path string true "Template ID"
// @Param body body v2schema.CampaignRegenerateTemplateRequest true "Regeneration request"
// @Success 200 {object} schema.APIResponse[v2schema.CampaignRegenerateTemplateResponse] "Regenerated template"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Security BearerAuth
// @Router /v2/campaigns/{campaign_id}/templates/{template_id} [post]
func (h *Handler) RegenerateTemplate(c fiber.Ctx) error {
	// Check if OpenAI client is available
	if h.openAIClient == nil {
		return serviceUnavailable(c, "AI service unavailable", fmt.Errorf("OpenAI client is not configured"))
	}

	campaignIDStr := c.Params("campaign_id")
	templateIDStr := c.Params("template_id")

	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	templateID, err := uuid.Parse(templateIDStr)
	if err != nil {
		return badRequest(c, "Invalid template ID", err)
	}

	var req v2schema.CampaignRegenerateTemplateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	// Get current user
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	// Fetch campaign and template
	campaign, err := h.campaignRepo.FindByID(c.Context(), campaignID)
	if err != nil {
		return notFound(c, "Campaign not found")
	}

	template, err := h.templateRepo.FindByID(c.Context(), templateID)
	if err != nil {
		return notFound(c, "Template not found")
	}

	// Regenerate template
	response, err := h.generateCampaignTemplate(c.Context(), campaign, userID, &req.Prompt, req.Tone, req.DocumentGids, template)
	if err != nil {
		return internalError(c, "Failed to regenerate template", err)
	}

	logger.Logger.Info().
		Str("campaign_id", campaign.ID.String()).
		Str("template_id", response.ID.String()).
		Msg("Successfully regenerated campaign template")

	return c.JSON(schema.SuccessResponse(response))
}

// generateCampaignTemplate is a helper method that generates or regenerates a campaign template
func (h *Handler) generateCampaignTemplate(
	ctx context.Context,
	campaign *domain.Campaign,
	userID uuid.UUID,
	prompt *string,
	tone *string,
	documentGids []string,
	existingTemplate *domain.Template,
) (*v2schema.CampaignGenerateTemplateResponse, error) {
	// Fetch user
	user, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Get user's organization
	var org *domain.Organization
	if campaign.OrganizationID != nil {
		org, _ = h.orgRepo.FindByID(ctx, *campaign.OrganizationID)
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

	// Fetch documents if document GIDs provided
	// TODO: Implement document fetching from knowledge base

	// Build email parameters
	toneStr := "Zero marketing jargon, write the way you speak (conversational) Not overly formal"
	if tone != nil {
		toneStr = *tone
	}

	promptStr := ""
	if prompt != nil {
		promptStr = *prompt
	}

	emailParams := mailService.EmailParameters{
		Sender:                 senderInfo,
		CampaignType:           mailService.CampaignType(campaign.Playbook),
		Tone:                   toneStr,
		AdditionalInstructions: promptStr,
		ClientFields:           []string{}, // TODO: Extract from campaign metadata
	}

	// Create mail writer
	mailWriter := mailService.NewMailWriter(h.openAIClient, "gpt-4o-mini", emailParams, &logger.Logger)

	// Get previous templates ordered by position
	allTemplates, err := h.templateRepo.FindByCampaignID(ctx, campaign.ID)
	if err != nil {
		logger.Logger.Warn().Err(err).Msg("Failed to fetch templates, using empty list")
		allTemplates = []*domain.Template{}
	}

	// Filter only outreach templates
	var outreachTemplates []*domain.Template
	for _, t := range allTemplates {
		if t.Category == "outreach" {
			outreachTemplates = append(outreachTemplates, t)
		}
	}

	// Convert to EmailTemplate format
	prevEmails := make([]mailService.EmailTemplate, 0)
	for _, t := range outreachTemplates {
		prevEmails = append(prevEmails, mailService.EmailTemplate{
			FromEmail: user.Email,
			ToEmail:   "{to_email}",
			Type:      mailService.EmailType(t.Type),
			Subject:   t.Subject,
			Content:   t.Content,
		})
	}

	// Generate template
	generatedEmail, err := mailWriter.GenerateSingleOutreach(ctx, prevEmails)
	if err != nil {
		return nil, fmt.Errorf("failed to generate email: %w", err)
	}

	// Save or update template
	var savedTemplate *domain.Template
	if existingTemplate != nil {
		// Update existing template
		existingTemplate.Subject = generatedEmail.Subject
		existingTemplate.Content = generatedEmail.Content
		existingTemplate.Type = string(generatedEmail.Type)

		if err := h.templateRepo.Update(ctx, existingTemplate.ID, existingTemplate); err != nil {
			return nil, fmt.Errorf("failed to update template: %w", err)
		}
		savedTemplate = existingTemplate
	} else {
		// Create new template with retry logic for race conditions
		const maxRetries = 3
		var created *domain.Template
		var lastErr error

		for attempt := 0; attempt < maxRetries; attempt++ {
			// Re-fetch templates on retry to get latest count
			if attempt > 0 {
				allTemplates, err = h.templateRepo.FindByCampaignID(ctx, campaign.ID)
				if err != nil {
					logger.Logger.Warn().Err(err).Msg("Failed to re-fetch templates on retry")
					allTemplates = []*domain.Template{}
				}
				outreachTemplates = []*domain.Template{}
				for _, t := range allTemplates {
					if t.Category == "outreach" {
						outreachTemplates = append(outreachTemplates, t)
					}
				}

				// Regenerate email with updated context
				prevEmails = make([]mailService.EmailTemplate, 0)
				for _, t := range outreachTemplates {
					prevEmails = append(prevEmails, mailService.EmailTemplate{
						FromEmail: user.Email,
						ToEmail:   "{to_email}",
						Type:      mailService.EmailType(t.Type),
						Subject:   t.Subject,
						Content:   t.Content,
					})
				}
				generatedEmail, err = mailWriter.GenerateSingleOutreach(ctx, prevEmails)
				if err != nil {
					return nil, fmt.Errorf("failed to regenerate email on retry: %w", err)
				}

				logger.Logger.Info().
					Int("attempt", attempt+1).
					Int("templates_count", len(outreachTemplates)).
					Str("new_type", string(generatedEmail.Type)).
					Msg("Retrying template creation with updated count")
			}

			newTemplate := &domain.Template{
				UserID:     userID,
				CampaignID: &campaign.ID,
				Type:       string(generatedEmail.Type),
				Category:   "outreach",
				Subject:    generatedEmail.Subject,
				Content:    generatedEmail.Content,
				Position:   float64(len(outreachTemplates) + 1),
				SendAfter:  len(outreachTemplates),
			}
			newTemplate.ID = uuid.New()

			created, err = h.templateRepo.Create(ctx, newTemplate)
			if err != nil {
				// Check if it's a unique constraint violation
				if baseRepo.IsUniqueViolation(err) && attempt < maxRetries-1 {
					logger.Logger.Warn().
						Err(err).
						Str("template_type", string(generatedEmail.Type)).
						Int("attempt", attempt+1).
						Msg("Unique constraint violation, retrying with updated template count")
					lastErr = err
					continue // Retry
				}

				// Not a unique violation or max retries reached
				logger.Logger.Error().
					Err(err).
					Str("template_type", string(generatedEmail.Type)).
					Int("attempt", attempt+1).
					Msg("Failed to create template")
				return nil, fmt.Errorf("failed to save template: %w", err)
			}

			// Success!
			logger.Logger.Info().
				Str("template_id", created.ID.String()).
				Str("template_type", created.Type).
				Int("attempt", attempt+1).
				Msg("Successfully created new template")
			savedTemplate = created
			break
		}

		if created == nil {
			return nil, fmt.Errorf("failed to create template after %d attempts: %w", maxRetries, lastErr)
		}
	}

	response := &v2schema.CampaignGenerateTemplateResponse{
		ID: savedTemplate.ID,
		Template: v2schema.EmailTemplateStructure{
			Category: string(savedTemplate.Category),
			Type:     savedTemplate.Type,
			Subject:  savedTemplate.Subject,
			Content:  savedTemplate.Content,
		},
	}

	return response, nil
}
