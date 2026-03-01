package listcontact

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// Search handles POST /v1/list-contacts/search
// @Summary Search list contacts
// @Description Search and filter contact lists with pagination. Automatically filtered by user_id.
// @Tags List Contacts
// @Accept json
// @Produce json
// @Param request body v1schema.ListContactSearchRequest true "Search filters"
// @Param offset query int false "Pagination offset (default: 0)"
// @Param limit query int false "Pagination limit (default: 25, max: 100)"
// @Success 200 {object} schema.APIResponse[v1schema.ListContactSearchResponse] "Search results with pagination"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request body"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 500 {object} schema.APIResponse[any] "Failed to search list contacts"
// @Router /v1/list-contacts/search [post]
// @Security BearerAuth
func (h *Handler) Search(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return unauthorized(c, "User not authenticated")
	}

	var req v1schema.ListContactSearchRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	offset, limit := parsePagination(c, 0, 25)
	if limit > 100 {
		limit = 100
	}

	if req.Filter == nil {
		req.Filter = make(map[string]interface{})
	}
	req.Filter["user_id"] = userID.String()

	pagination := baseRepo.PaginationParams{
		Offset: offset,
		Limit:  limit,
	}

	result, err := h.listContactRepo.SearchOptimized(c.Context(), req.Filter, &pagination)
	if err != nil {
		return internalError(c, "Failed to search list contacts", err)
	}

	// For performance, use a simple creator name (can be enhanced later if needed)
	creatorName := "User" // Simplified for search performance

	// Collect list IDs for batch counting
	listIDs := make([]uuid.UUID, len(result.List))
	for i, lc := range result.List {
		listIDs[i] = lc.ID
	}

	// Batch count contacts and inbound forms (eliminate N+1 queries)
	contactCounts, _ := h.listContactRepo.BatchCountContacts(c.Context(), listIDs)
	inboundFormCounts, _ := h.listContactRepo.BatchCountInboundForms(c.Context(), listIDs)

	list := make([]v1schema.ListContactListItem, len(result.List))
	for i, lc := range result.List {
		contactCount := contactCounts[lc.ID]
		inboundFormCount := inboundFormCounts[lc.ID]

		list[i] = v1schema.ListContactListItem{
			Entity: v1schema.ListContactSearchResponseEntity{
				ID:        lc.ID,
				Name:      lc.Name,
				Source:    &lc.Source,
				SourceID:  &lc.SourceID,
				HubspotID: lc.HubspotID,
				CreatedAt: lc.CreatedAt,
				UpdatedAt: lc.UpdatedAt,
			},
			Creator:              creatorName,
			NumberOfInboundForms: int(inboundFormCount),
			Size:                 int(contactCount),
		}
	}

	response := v1schema.ListContactSearchResponse{
		List:   list,
		Offset: offset,
		Limit:  limit,
		Total:  result.Total,
	}

	return c.JSON(schema.SuccessResponse(response))
}
