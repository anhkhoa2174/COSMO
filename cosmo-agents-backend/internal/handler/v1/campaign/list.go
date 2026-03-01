package campaign

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// List handles GET /v1/campaigns.
// @Summary List Campaigns
// @Description Retrieve a paginated list of campaigns with optional filtering by status and organization.
// @Tags Campaigns
// @Accept json
// @Produce json
// @Param status query string false "Filter by campaign status"
// @Param organization_id query string false "Filter by organization ID"
// @Param offset query int false "Pagination offset"
// @Param limit query int false "Pagination limit"
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[schema.PaginatedResponse[v1schema.CampaignListItem]] "Successful Response"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v1/campaigns [get]
func (h *Handler) List(c fiber.Ctx) error {
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	offset, limit := parsePagination(c, 0, 25)

	primaryOrgID, err := h.roleRepo.FindPrimaryOrganization(c.Context(), userID)
	if err != nil {
		return internalError(c, "Failed to determine primary organization", err)
	}
	if primaryOrgID == nil {
		return unauthorizedAccessResponse(c)
	}

	filter := baseRepo.Filter{}
	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}
	if org := c.Query("organization_id"); org != "" {
		orgUUID, err := uuid.Parse(org)
		if err != nil {
			return badRequest(c, "Invalid organization_id", err)
		}
		if orgUUID != *primaryOrgID {
			return forbidden(c, "You are not authorized to access this organization")
		}
		filter["organization_id"] = orgUUID
	}

	rows, total, err := h.campaignRepo.FindWithStats(
		c.Context(),
		userID,
		primaryOrgID,
		filter,
		&baseRepo.PaginationParams{Offset: offset, Limit: limit},
	)
	if err != nil {
		return internalError(c, "Failed to list campaigns", err)
	}

	items, err := h.buildListItems(c.Context(), rows)
	if err != nil {
		return internalError(c, "Failed to build campaign response", err)
	}

	response := schema.PaginatedResponse[v1schema.CampaignListItem]{
		List:   items,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// buildListItems builds listing items and enriches agent metadata
func (h *Handler) buildListItems(ctx context.Context, rows []campaignRepo.CampaignWithStats) ([]v1schema.CampaignListItem, error) {
	items := make([]v1schema.CampaignListItem, len(rows))

	agentIDs := make([]uuid.UUID, 0)
	seen := map[uuid.UUID]struct{}{}
	for _, row := range rows {
		if row.AgentID != nil {
			if _, ok := seen[*row.AgentID]; !ok {
				seen[*row.AgentID] = struct{}{}
				agentIDs = append(agentIDs, *row.AgentID)
			}
		}
	}

	// Note: Agent preloading removed to avoid circular imports
	// Agent data should be loaded separately using relations package if needed

	for i, row := range rows {
		campaign := row.Campaign
		// Note: Campaign.Agent relationship removed to avoid circular imports
		// Agent data should be loaded separately using relations package if needed
		var creator *string
		if row.Creator.Valid {
			creator = &row.Creator.String
		}
		items[i] = v1schema.ToCampaignListItem(&campaign, creator, row.Sent, row.Reply, row.Interested, row.ReplyRate, row.InterestRate)
	}

	return items, nil
}
