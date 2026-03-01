package campaign

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// CreateDraftTemplate handles GET /v2/campaigns/{campaign_id}/draft-templates
// @Summary Create or get draft template
// @Description Gets or creates a draft template for a specific intent type. Returns existing draft template if already created for the given intent.
// @Tags Campaigns V2 - Done
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Param intent query string true "Intent type (e.g., Interested, Not interested, Referral, Request for pricing, Request for information, Nurture, Do not contact, Out of office, Unknown intent)"
// @Success 200 {object} schema.APIResponse[v2schema.CampaignDraftTemplateCreateResponse] "Draft template"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v2/campaigns/{campaign_id}/draft-templates [get]
func (h *Handler) CreateDraftTemplate(c fiber.Ctx) error {
	campaignIDStr := c.Params("campaign_id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	intentStr := c.Query("intent")
	if intentStr == "" {
		return badRequest(c, "Missing intent parameter", fmt.Errorf("intent query parameter is required"))
	}

	intent := domain.IntentType(intentStr)
	if !intent.IsValid() {
		return badRequest(c, "Invalid intent type", fmt.Errorf("Intent '%s' is not valid", intentStr))
	}

	// Get current user
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	// Check if draft template already exists
	filter := baseRepo.Filter{
		"campaign_id": campaignID,
		"intent":      intent,
	}

	draftTemplates, err := h.draftTemplateRepo.FindAll(c.Context(), filter, nil)
	if err == nil && len(draftTemplates.List) > 0 {
		// Return existing draft template
		dt := &draftTemplates.List[0]
		response := v2schema.CampaignDraftTemplateCreateResponse{
			ID:         dt.ID,
			Intent:     domain.IntentType(dt.Intent),
			CampaignID: dt.CampaignID,
			TemplateID: dt.TemplateID,
			CreatedAt:  dt.CreatedAt,
			UpdatedAt:  dt.UpdatedAt,
			Template: v2schema.CampaignDraftTemplateDetail{
				Type:    "draft_" + string(intent),
				Subject: "", // Draft templates typically don't have subject/content initially
				Content: "",
			},
		}
		return c.JSON(schema.SuccessResponse(response))
	}

	// Create new template first
	template := &domain.Template{
		UserID:     userID,
		CampaignID: &campaignID,
		Type:       "draft_" + string(intent),
		Subject:    "",
		Content:    "",
	}
	template.ID = uuid.New()

	createdTemplate, err := h.templateRepo.Create(c.Context(), template)
	if err != nil {
		return internalError(c, "Failed to create template", err)
	}

	// Create new draft template
	draftTemplate := &domain.DraftTemplate{
		CampaignID: campaignID,
		Intent:     string(intent),
		TemplateID: createdTemplate.ID,
	}
	draftTemplate.ID = uuid.New()

	created, err := h.draftTemplateRepo.Create(c.Context(), draftTemplate)
	if err != nil {
		return internalError(c, "Failed to create draft template", err)
	}

	response := v2schema.CampaignDraftTemplateCreateResponse{
		ID:         created.ID,
		Intent:     domain.IntentType(created.Intent),
		CampaignID: created.CampaignID,
		TemplateID: created.TemplateID,
		CreatedAt:  created.CreatedAt,
		UpdatedAt:  created.UpdatedAt,
		Template: v2schema.CampaignDraftTemplateDetail{
			Type:    "draft_" + string(intent),
			Subject: "",
			Content: "",
		},
	}

	logger.Logger.Info().
		Str("campaign_id", campaignID.String()).
		Str("intent", string(intent)).
		Str("draft_template_id", created.ID.String()).
		Msg("Created draft template")

	return c.JSON(schema.SuccessResponse(response))
}
