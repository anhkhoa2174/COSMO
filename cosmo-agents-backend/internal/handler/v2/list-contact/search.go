package listcontact

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// Search handles POST /v2/list-contacts/search
// @Summary Search list contacts
// @Description Search and filter list contacts for the current user
// @Tags List Contacts V2
// @Accept json
// @Produce json
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit" default(25)
// @Param body body v2schema.ListContactSearchRequest true "Search filters"
// @Success 200 {object} schema.APIResponse[v2schema.ListContactSearchResponse] "Search results"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Security BearerAuth
// @Router /v2/list-contacts/search [post]
func (h *Handler) Search(c fiber.Ctx) error {
	// Get current user from context
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	// Parse pagination parameters
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	if limit > 100 {
		limit = 100
	}

	// Parse request body
	var req v2schema.ListContactSearchRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	// Get list contacts for the user
	lists, total, err := h.listContactRepo.GetByUserID(c.Context(), userID.String(), limit, offset*limit)
	if err != nil {
		return internalError(c, "Failed to fetch list contacts", err)
	}

	// Get user info for creator field
	user, _ := h.userRepo.FindByID(c.Context(), userID)
	creatorName := "Unknown"
	if user != nil {
		creatorName = user.Name
	}

	// Build response
	items := make([]v2schema.ListContactListItem, 0, len(lists))
	for _, list := range lists {
		// TODO: Get actual campaign count and size
		// For now, using placeholder values
		listContactID := list.ID
		item := v2schema.ListContactListItem{
			Entity: v2schema.ListContactEntity{
				ID:            list.ID,
				Name:          list.Name,
				Source:        &list.Source,
				SourceID:      &list.SourceID,
				HubspotID:     list.HubspotID,
				ListContactID: &listContactID,
				CreatedAt:     list.CreatedAt,
				UpdatedAt:     list.UpdatedAt,
			},
			Creator:          creatorName,
			NumberOfCampaign: 0,
			Size:             0,
		}
		items = append(items, item)
	}

	response := v2schema.ListContactSearchResponse{
		List:   items,
		Offset: offset,
		Limit:  limit,
		Total:  total,
	}

	logger.Logger.Info().
		Str("user_id", userID.String()).
		Int64("total", total).
		Int("returned", len(items)).
		Msg("List contacts search completed")

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(response))
}
