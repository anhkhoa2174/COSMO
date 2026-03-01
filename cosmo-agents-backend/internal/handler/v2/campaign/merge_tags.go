package campaign

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// GetMergeTags handles GET /v2/campaigns/{campaign_id}/merge-tags
// @Summary Get available merge tags for campaign
// @Description Returns merge tags that can be used in email templates
// @Tags Campaigns V2 - Done
// @Produce json
// @Param campaign_id path string true "Campaign ID"
// @Success 200 {object} schema.APIResponse[map[string][]string] "Merge tags grouped by category"
// @Failure 400 {object} schema.APIResponse[any] "Invalid campaign ID"
// @Failure 404 {object} schema.APIResponse[any] "Campaign not found"
// @Security BearerAuth
// @Router /v2/campaigns/{campaign_id}/merge-tags [get]
func (h *Handler) GetMergeTags(c fiber.Ctx) error {
	campaignIDStr := c.Params("campaign_id")
	campaignID, err := uuid.Parse(campaignIDStr)
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	// Verify campaign exists
	campaign, err := h.campaignRepo.FindByID(c.Context(), campaignID)
	if err != nil {
		return notFound(c, "Campaign not found")
	}

	// Get allowed merge tags from campaign's client_fields (if available)
	// For now, return standard merge tags
	allowedTags := []string{
		"contact_email",
		"contact_company",
		"contact_country",
		"contact_job_title",
		"contact_last_name",
		"contact_first_name",
		"organization_name",
		"organization_company_url",
		"agent_signature",
		"sale_rep_first_name",
		"sale_rep_last_name",
		"sale_rep_email",
		"sale_rep_calendar_link",
	}

	// Group by prefix
	groups := []string{"contact", "organization", "agent", "sale_rep"}
	result := make(map[string][]string)

	for _, group := range groups {
		result[group] = []string{}
		for _, tag := range allowedTags {
			if strings.HasPrefix(tag, group+"_") {
				result[group] = append(result[group], tag)
			}
		}
	}

	logger.Logger.Info().
		Str("campaign_id", campaign.ID.String()).
		Int("total_tags", len(allowedTags)).
		Msg("Retrieved merge tags for campaign")

	return c.JSON(schema.SuccessResponse(result))
}
