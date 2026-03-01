package campaign

import (
	"github.com/gofiber/fiber/v3"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
)

// SearchCampaigns handles POST /v2/campaigns/search.
// @Summary Search Campaigns
// @Description Search campaigns based on provided filters with pagination support.
// @Tags Campaigns V2 - Done
// @Accept json
// @Produce json
// @Param body body v2schema.CampaignSearchRequest false "Search Filters"
// @Param offset query int false "Pagination offset"
// @Param limit query int false "Pagination limit"
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[schema.PaginatedResponse[v2schema.CampaignListItem]] "Successful Response"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v2/campaigns/search [post]
func (h *Handler) SearchCampaigns(c fiber.Ctx) error {
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	offset, limit := parsePagination(c, 0, 25)

	var req v2schema.CampaignSearchRequest
	if len(c.Body()) > 0 {
		if err := c.Bind().JSON(&req); err != nil {
			return badRequest(c, "Invalid request body", err)
		}
	}

	filter := baseRepo.Filter{}
	if req.Filter != nil {
		for key, value := range req.Filter {
			filter[key] = value
		}
	}

	rows, total, err := h.campaignRepo.FindWithStats(c.Context(), userID, nil, filter, &baseRepo.PaginationParams{Offset: offset, Limit: limit})
	if err != nil {
		return internalError(c, "Failed to search campaigns", err)
	}

	items, err := h.buildListItems(c.Context(), rows)
	if err != nil {
		return internalError(c, "Failed to build campaign response", err)
	}

	response := schema.PaginatedResponse[v2schema.CampaignListItem]{
		List:   items,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}

	return c.JSON(schema.SuccessResponse(response))
}
