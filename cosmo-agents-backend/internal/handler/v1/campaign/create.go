package campaign

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// Create handles POST /v1/campaigns.
// @Summary Create Campaign
// @Description Create a new campaign with the provided details.
// @Tags Campaigns
// @Accept json
// @Produce json
// @Param body body v1schema.CampaignCreateRequest true "Campaign Creation Request"
// @Security BearerAuth
// @Success 201 {object} schema.APIResponse[v1schema.CampaignResponse] "Successful Response"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v1/campaigns [post]
func (h *Handler) Create(c fiber.Ctx) error {
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	var err error

	var req v1schema.CampaignCreateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return badRequest(c, "Validation failed", err)
	}

	organizationID := req.OrganizationID
	if organizationID != nil {
		// An organization named in the body must be one the caller belongs to;
		// otherwise the campaign lands in a stranger's organization, where its
		// members can list, open and edit it.
		roles, err := h.roleRepo.FindByUserID(c.Context(), userID)
		if err != nil {
			return internalError(c, "Failed to verify user organizations", err)
		}
		if !userInOrganization(*organizationID, roles) {
			return forbidden(c, "You are not a member of this organization")
		}
	} else {
		organizationID, err = h.roleRepo.FindPrimaryOrganization(c.Context(), userID)
		if err != nil {
			return internalError(c, "Failed to determine primary organization", err)
		}
		if organizationID == nil {
			return badRequest(c, "User must belong to an organization", errors.New("missing organization"))
		}
	}

	if req.AgentID != nil {
		ok, err := h.agentUsableBy(c.Context(), *req.AgentID, userID)
		if err != nil {
			return internalError(c, "Failed to verify agent", err)
		}
		if !ok {
			return forbidden(c, "You cannot send from this agent")
		}
	}

	name := req.Name
	if name == nil {
		generated := deriveCampaignName(req.Playbook)
		name = &generated
	}

	campaign := &domain.Campaign{
		UserID:         userID,
		OrganizationID: organizationID,
		ListContactID:  req.ListContactID,
		Schedule:       req.Schedule,
		Playbook:       req.Playbook,
		AgentID:        req.AgentID,
	}

	if req.CampaignID != nil {
		campaign.ID = *req.CampaignID
	}

	if name != nil {
		campaign.Name = *name
	}

	if req.Status != nil {
		campaign.Status = domain.CampaignStatus(*req.Status)
	} else {
		campaign.Status = domain.CampaignStatusDraft
	}

	created, err := h.campaignRepo.Create(c.Context(), campaign)
	if err != nil {
		return internalError(c, "Failed to create campaign", err)
	}

	// Note: Campaign.Notifications relationship removed to avoid circular imports
	// Notification data should be loaded separately using relations package if needed

	return c.Status(fiber.StatusCreated).JSON(schema.SuccessResponse(v1schema.ToCampaignResponse(created)))
}
