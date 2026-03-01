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
	v3schema "github.com/rockship/cosmo-agents-go/internal/schema/v3"
	mailService "github.com/rockship/cosmo-agents-go/internal/service/mail"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// GenerateTemplate handles POST /v3/campaigns/{campaign_id}/templates
// @Summary Generate new email template (V3)
// @Description Uses AI to generate a new outreach email template (V3 version)
// @Tags Campaigns V3
// @Accept json
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Param body body v3schema.CampaignGenerateTemplateRequest true "Generation request"
// @Success 200 {object} schema.APIResponse[any] "Generated template"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Security BearerAuth
// @Router /v3/campaigns/{campaign_id}/templates [post]
func (h *Handler) GenerateTemplate(c fiber.Ctx) error {
	campaignIDStr := c.Params("campaign_id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err.Error())
	}

	var req v3schema.CampaignGenerateTemplateRequest
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

	// Generate template
	response, err := h.generateCampaignTemplate(c.Context(), campaign, userUUID, req.Prompt, req.Tone, req.DocumentGids, nil)
	if err != nil {
		return internalError(c, "Failed to generate template", err.Error())
	}

	logger.Logger.Info().
		Str("campaign_id", campaign.ID.String()).
		Str("template_id", response.ID.String()).
		Msg("Successfully generated campaign template (V3)")

	return c.JSON(schema.SuccessResponse(response))
}

// SaveExternalTemplate handles POST /v3/campaigns/{campaign_id}/templates/external
// @Summary Save externally generated template
// @Description Saves a template that was generated outside the system
// @Tags Campaigns V3
// @Accept json
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Param body body v3schema.CampaignSaveExternalTemplateRequest true "External template"
// @Success 200 {object} schema.APIResponse[v3schema.CampaignSaveExternalTemplateResponse] "Saved template"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Security BearerAuth
// @Router /v3/campaigns/{campaign_id}/templates/external [post]
func (h *Handler) SaveExternalTemplate(c fiber.Ctx) error {
	campaignIDStr := c.Params("campaign_id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err.Error())
	}

	var req v3schema.CampaignSaveExternalTemplateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err.Error())
	}

	// Get current user
	userUUID, ok := userIDFromContext(c)
	if !ok {
		return unauthorized(c, "Unauthorized", "User ID not found")
	}

	// Verify campaign exists
	campaign, err := h.campaignRepo.FindByID(c.Context(), campaignID)
	if err != nil {
		return notFound(c, "Campaign not found", err.Error())
	}

	// Get existing templates ordered by position
	allTemplates, err := h.templateRepo.FindByCampaignID(c.Context(), campaign.ID)
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

	// Determine send_after value
	sendAfter := len(outreachTemplates)
	if req.SendAfter != nil {
		sendAfter = *req.SendAfter
	}

	// Determine template type
	var templateType string
	if req.Type != nil && *req.Type != "" {
		// Use type from request if provided
		templateType = *req.Type
		logger.Logger.Info().
			Str("template_type", templateType).
			Msg("Using template type from request")
	} else {
		// Auto-calculate type based on position (legacy behavior)
		templateType = "First Email"
		if len(outreachTemplates) == 1 {
			templateType = "Follow-up Email 1"
		} else if len(outreachTemplates) == 2 {
			templateType = "Follow-up Email 2"
		} else if len(outreachTemplates) >= 3 {
			templateType = fmt.Sprintf("Follow-up Email %d", len(outreachTemplates))
		}
		logger.Logger.Info().
			Str("template_type", templateType).
			Int("position", len(outreachTemplates)).
			Msg("Auto-calculated template type based on position")
	}

	// Check if template with this type already exists (for UPSERT behavior)
	existingFilter := baseRepo.Filter{
		"user_id":     userUUID,
		"campaign_id": campaign.ID,
		"type":        templateType,
	}
	existingTemplates, _ := h.templateRepo.FindAll(c.Context(), existingFilter, nil)

	var created *domain.Template

	if len(existingTemplates.List) > 0 {
		// Update existing template
		existing := &existingTemplates.List[0]
		existing.Subject = req.Subject
		existing.Content = req.Content
		existing.SendAfter = sendAfter

		if updateErr := h.templateRepo.Update(c.Context(), existing.ID, existing); updateErr != nil {
			return internalError(c, "Failed to update template", updateErr.Error())
		}

		created = existing

		logger.Logger.Info().
			Str("campaign_id", campaign.ID.String()).
			Str("template_id", created.ID.String()).
			Str("type", templateType).
			Msg("Successfully updated external template (V3)")
	} else {
		// Create new template with retry logic for race conditions
		const maxRetries = 3
		var lastErr error

		for attempt := 0; attempt < maxRetries; attempt++ {
			// Re-fetch templates on retry to get latest count
			if attempt > 0 {
				allTemplates, err = h.templateRepo.FindByCampaignID(c.Context(), campaign.ID)
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

				// Recalculate type if it was auto-generated (not provided by request)
				if req.Type == nil || *req.Type == "" {
					if len(outreachTemplates) == 0 {
						templateType = "First Email"
					} else if len(outreachTemplates) == 1 {
						templateType = "Follow-up Email 1"
					} else if len(outreachTemplates) == 2 {
						templateType = "Follow-up Email 2"
					} else {
						templateType = fmt.Sprintf("Follow-up Email %d", len(outreachTemplates))
					}
				}
				// If type was provided by request, keep using it

				logger.Logger.Info().
					Int("attempt", attempt+1).
					Int("templates_count", len(outreachTemplates)).
					Str("template_type", templateType).
					Msg("Retrying external template creation with updated count")
			}

			newTemplate := &domain.Template{
				UserID:     userUUID,
				CampaignID: &campaign.ID,
				Type:       templateType,
				Category:   "outreach",
				Subject:    req.Subject,
				Content:    req.Content,
				Position:   float64(len(outreachTemplates) + 1),
				SendAfter:  sendAfter,
			}
			newTemplate.ID = uuid.New()

			var createErr error
			created, createErr = h.templateRepo.Create(c.Context(), newTemplate)
			if createErr != nil {
				// Check if it's a unique constraint violation
				if baseRepo.IsUniqueViolation(createErr) && attempt < maxRetries-1 {
					logger.Logger.Warn().
						Err(createErr).
						Str("template_type", templateType).
						Int("attempt", attempt+1).
						Msg("Unique constraint violation, retrying external template creation")
					lastErr = createErr
					continue // Retry
				}

				// Not a unique violation or max retries reached
				logger.Logger.Error().
					Err(createErr).
					Str("template_type", templateType).
					Int("attempt", attempt+1).
					Msg("Failed to save external template")
				return internalError(c, "Failed to save template", createErr.Error())
			}

			// Success!
			logger.Logger.Info().
				Str("campaign_id", campaign.ID.String()).
				Str("template_id", created.ID.String()).
				Str("type", templateType).
				Int("attempt", attempt+1).
				Msg("Successfully saved external template (V3)")
			break
		}

		if created == nil {
			return internalError(c, "Failed to create template after retries", lastErr.Error())
		}
	}

	response := v3schema.CampaignSaveExternalTemplateResponse{
		ID:        created.ID,
		Type:      created.Type,
		Subject:   created.Subject,
		Content:   created.Content,
		SendAfter: created.SendAfter,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// RegenerateTemplate handles POST /v3/campaigns/{campaign_id}/templates/{template_id}
// @Summary Regenerate template with feedback (V3)
// @Description Regenerate a specific template based on user feedback (V3 version)
// @Tags Campaigns V3
// @Accept json
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Param template_id path string true "Template ID"
// @Param body body v3schema.CampaignRegenerateTemplateRequest true "Regeneration request"
// @Success 200 {object} schema.APIResponse[any] "Regenerated template"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Security BearerAuth
// @Router /v3/campaigns/{campaign_id}/templates/{template_id} [post]
func (h *Handler) RegenerateTemplate(c fiber.Ctx) error {
	campaignIDStr := c.Params("campaign_id")
	templateIDStr := c.Params("template_id")

	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err.Error())
	}

	templateID, err := uuid.Parse(templateIDStr)
	if err != nil {
		return badRequest(c, "Invalid template ID", err.Error())
	}

	var req v3schema.CampaignRegenerateTemplateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err.Error())
	}

	// Get current user
	userUUID, ok := userIDFromContext(c)
	if !ok {
		return unauthorized(c, "Unauthorized", "User ID not found")
	}

	// Fetch campaign and template
	campaign, err := h.campaignRepo.FindByID(c.Context(), campaignID)
	if err != nil {
		return notFound(c, "Campaign not found", err.Error())
	}

	template, err := h.templateRepo.FindByID(c.Context(), templateID)
	if err != nil {
		return notFound(c, "Template not found", err.Error())
	}

	// Regenerate template
	response, err := h.generateCampaignTemplate(c.Context(), campaign, userUUID, &req.Prompt, req.Tone, req.DocumentGids, template)
	if err != nil {
		return internalError(c, "Failed to regenerate template", err.Error())
	}

	logger.Logger.Info().
		Str("campaign_id", campaign.ID.String()).
		Str("template_id", response.ID.String()).
		Msg("Successfully regenerated campaign template (V3)")

	return c.JSON(schema.SuccessResponse(response))
}

// generateCampaignTemplate is the shared helper for generating templates
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
		ClientFields:           []string{},
	}

	// Create mail writer
	mailWriter := mailService.NewMailWriter(h.openAIClient, "gpt-4o-mini", emailParams, &logger.Logger)

	// Get previous templates ordered by position
	// Use FindByCampaignID for proper ordering instead of FindAll
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

	logger.Logger.Info().
		Str("campaign_id", campaign.ID.String()).
		Int("previous_templates_count", len(outreachTemplates)).
		Msg("Fetched previous templates for campaign")

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
		// Update existing template (regenerate case)
		existingTemplate.Subject = generatedEmail.Subject
		existingTemplate.Content = generatedEmail.Content
		existingTemplate.Type = string(generatedEmail.Type)

		if err := h.templateRepo.Update(ctx, existingTemplate.ID, existingTemplate); err != nil {
			return nil, fmt.Errorf("failed to update template: %w", err)
		}
		savedTemplate = existingTemplate
	} else {
		// Always create new template (no UPSERT behavior to avoid overwriting)
		// Retry logic to handle race conditions when concurrent requests generate same type
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

			templateType := string(generatedEmail.Type)

			logger.Logger.Info().
				Str("template_type", templateType).
				Int("position", len(outreachTemplates)+1).
				Int("send_after", len(outreachTemplates)).
				Int("attempt", attempt+1).
				Msg("Creating new template")

			newTemplate := &domain.Template{
				UserID:     userID,
				CampaignID: &campaign.ID,
				Type:       templateType,
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
						Str("template_type", templateType).
						Int("attempt", attempt+1).
						Msg("Unique constraint violation, retrying with updated template count")
					lastErr = err
					continue // Retry
				}

				// Not a unique violation or max retries reached
				logger.Logger.Error().
					Err(err).
					Str("template_type", templateType).
					Str("campaign_id", campaign.ID.String()).
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
