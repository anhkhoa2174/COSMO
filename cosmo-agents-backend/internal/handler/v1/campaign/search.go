package campaign

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// SearchCampaigns handles POST /v1/campaigns/search.
// @Summary Search Campaigns
// @Description Search campaigns based on provided filters with pagination support.
// @Tags Campaigns
// @Accept json
// @Produce json
// @Param body body v1schema.CampaignGetRequest false "Search Filters"
// @Param offset query int false "Pagination offset"
// @Param limit query int false "Pagination limit"
// @Security BearerAuth
// @Success 200 {object} schema.APIResponse[schema.PaginatedResponse[v1schema.CampaignListItem]] "Successful Response"
// @Failure 400 {object} schema.APIResponse[any] "Bad Request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal Server Error"
// @Router /v1/campaigns/search [post]
func (h *Handler) SearchCampaigns(c fiber.Ctx) error {
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	offset, limit := parsePagination(c, 0, 25)

	var req v1schema.CampaignGetRequest
	if len(c.Body()) > 0 {
		if err := c.Bind().JSON(&req); err != nil {
			return badRequest(c, "Invalid request body", err)
		}
	}

	// "$raw" is spliced into the WHERE clause verbatim by the filter builder.
	// From a request body that is SQL injection, and "1=1) OR (1=1" escapes the
	// caller's user_id scope entirely.
	if containsRawFilter(req.Filter) {
		return badRequest(c, "Invalid filter", errors.New("raw filter expressions are not allowed"))
	}

	filter := baseRepo.Filter{}
	for key, value := range req.Filter {
		filter[key] = value
	}

	rows, total, err := h.campaignRepo.FindWithStats(c.Context(), userID, nil, filter, &baseRepo.PaginationParams{Offset: offset, Limit: limit})
	if err != nil {
		return internalError(c, "Failed to search campaigns", err)
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

// containsRawFilter reports whether a filter, at any depth of $and/$or
// nesting, carries a raw SQL expression.
func containsRawFilter(v any) bool {
	switch t := v.(type) {
	case map[string]any:
		for k, sub := range t {
			if k == string(baseRepo.OpRaw) || containsRawFilter(sub) {
				return true
			}
		}
	case []any:
		for _, sub := range t {
			if containsRawFilter(sub) {
				return true
			}
		}
	}
	return false
}
