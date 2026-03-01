package contact

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// List handles GET /v1/contact and POST /v1/contact/search
// @Summary List contacts
// @Description Retrieves a paginated list of contacts with optional filtering
// @Tags Contacts
// @Accept json
// @Produce json
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit" default(50)
// @Param filter body v1schema.ListContactsRequest false "Filter criteria (for POST)"
// @Success 200 {object} schema.APIResponse[schema.PaginatedResponse[v1schema.ContactResponse]] "Successfully retrieved contacts"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/contact [get]
func (h *Handler) List(c fiber.Ctx) error {
	user, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	// Parse pagination params
	offsetInt, _ := strconv.Atoi(c.Query("offset", "0"))
	limitInt, _ := strconv.Atoi(c.Query("limit", "50"))

	pagination := &baseRepo.PaginationParams{
		Offset: offsetInt,
		Limit:  limitInt,
	}
	pagination.Validate()

	// Parse filter from request body (for POST) or query (for GET)
	var req v1schema.ListContactsRequest
	if c.Method() == "POST" {
		if err := c.Bind().JSON(&req); err != nil {
			return h.responseHelper.BadRequest(c, "Invalid request body", err)
		}
	}

	// Build filter based on role
	// Admin sees all contacts in organization, member sees only their own
	filter := baseRepo.Filter{
		"is_deleted":      false,
		"organization_id": organizationID,
	}

	// Check user's role in the organization to determine visibility
	isAdmin := h.authHelper.IsAdminInOrganization(c.Context(), user.ID, organizationID)

	if !isAdmin {
		// Member: filter by user_id to see only their own contacts
		filter["user_id"] = user.ID
	}
	if req.Filter != nil {
		for k, v := range req.Filter {
			filter[k] = v
		}
	}

	// Fetch contacts
	result, err := h.repo.FindAll(c.Context(), filter, pagination)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to fetch contacts", err)
	}

	// Convert to response DTOs
	responses := make([]*v1schema.ContactResponse, len(result.List))
	for i, contact := range result.List {
		responses[i] = v1schema.ToContactResponse(&contact)
	}

	paginatedResponse := schema.PaginatedResponse[*v1schema.ContactResponse]{
		List:   responses,
		Total:  result.Total,
		Offset: result.Offset,
		Limit:  result.Limit,
	}

	return h.responseHelper.Success(c, paginatedResponse)
}
