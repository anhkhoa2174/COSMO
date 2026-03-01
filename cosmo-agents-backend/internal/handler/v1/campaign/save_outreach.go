package campaign

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// SaveOutreach handles PATCH /v1/campaigns/{id}/save-outreach
// @Summary Save email outreach sequence
// @Description Saves the generated email templates to the campaign
// @Tags Campaigns
// @Accept json
// @Produce json
// @Param id path string true "Campaign ID (UUID)"
// @Param body body v1schema.CampaignSaveOutreachRequest true "Outreach sequence"
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[string] "Outreach saved"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 403 {object} schema.APIResponse[any] "Forbidden"
// @Failure 404 {object} schema.APIResponse[any] "Not Found"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v1/campaigns/{id}/save-outreach [patch]
func (h *Handler) SaveOutreach(c fiber.Ctx) error {
	// Parse campaign ID from URL
	campaignID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c, "Invalid campaign ID", err)
	}

	// Extract user ID from context
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	// Fetch campaign with relations
	campaign, err := h.campaignRepo.FindWithRelations(c.Context(), campaignID)
	if err != nil {
		return internalError(c, "Failed to fetch campaign", err)
	}
	if campaign == nil {
		return notFound(c, "Campaign not found")
	}

	// Check access permissions
	// Allow access if: 1) User owns the campaign, OR 2) User has role in campaign's organization
	if campaign.UserID != userID {
		// Check if user has access to campaign's organization
		if campaign.OrganizationID == nil {
			return forbidden(c, "You don't have permission to update this campaign")
		}

		// Get all user's organizations (not just primary)
		roles, err := h.roleRepo.FindByUserID(c.Context(), userID)
		if err != nil {
			return internalError(c, "Failed to verify organization access", err)
		}

		hasAccess := false
		for _, role := range roles {
			if !role.IsDeleted && role.Status == domain.RoleStatusActive &&
				role.OrganizationID == *campaign.OrganizationID {
				hasAccess = true
				break
			}
		}

		if !hasAccess {
			return forbidden(c, "You don't have permission to update this campaign")
		}
	}

	// Parse and validate request body
	var req v1schema.CampaignSaveOutreachRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return badRequest(c, "Validation failed", err)
	}

	// Convert request sequence to domain model
	sequence := make([]domain.EmailSequenceItem, len(req.Sequence))
	for i, item := range req.Sequence {
		seq := domain.EmailSequenceItem{
			Type: item.Type,
		}
		if item.Subject != nil {
			seq.Subject = *item.Subject
		}
		if item.Content != nil {
			seq.Content = *item.Content
		}
		sequence[i] = seq
	}

	// Update campaign metadata with new sequence
	campaign.CMetadata.Sequence = sequence

	// Save to database
	if err := h.campaignRepo.UpdateAttributes(c.Context(), campaign.ID, map[string]interface{}{
		"cmetadata": campaign.CMetadata,
	}); err != nil {
		return internalError(c, "Failed to save outreach sequence", err)
	}

	return c.JSON(schema.SuccessResponse("Outreach sequence saved successfully"))
}
