package customfield

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// List handles GET /v1/custom-fields
// @Summary List custom fields
// @Description Get all custom fields for the authenticated user with optional filtering by entity_type
// @Tags Custom Fields
// @Accept json
// @Produce json
// @Param entity_type query string false "Filter by entity type (contact, list_contact, campaign, etc.)"
// @Param offset query int false "Pagination offset (default: 0)"
// @Param limit query int false "Pagination limit (default: 25, max: 100)"
// @Success 200 {object} schema.APIResponse[v1schema.CustomFieldListResponse] "Custom fields retrieved successfully"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 500 {object} schema.APIResponse[any] "Failed to fetch custom fields"
// @Router /v1/custom-fields [get]
// @Security BearerAuth
func (h *Handler) List(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return unauthorized(c, "User not authenticated")
	}

	offset, limit := parsePagination(c, 0, 25)
	if limit > 100 {
		limit = 100
	}

	entityType := c.Query("entity_type", "")

	filters := map[string]interface{}{
		"user_id": userID.String(),
	}

	if entityType != "" {
		filters["entity_type"] = entityType
	}

	pagination := baseRepo.PaginationParams{
		Offset: offset,
		Limit:  limit,
	}

	result, err := h.customFieldRepo.FindAll(c.Context(), filters, &pagination)
	if err != nil {
		return internalError(c, "Failed to fetch custom fields", err)
	}

	list := make([]v1schema.CustomFieldResponse, len(result.List))
	for i, cf := range result.List {
		list[i] = v1schema.CustomFieldResponse{
			ID:             cf.ID,
			UserID:         cf.UserID,
			OrganizationID: cf.OrganizationID,
			Name:           cf.Name,
			NormalizedName: cf.NormalizedName,
			DataType:       cf.DataType,
			EntityType:     cf.EntityType,
			IsRequired:     cf.IsRequired,
			Options:        []string(cf.Options),
			SampleData:     cf.SampleData,
			FallbackValue:  cf.FallbackValue,
			CreatedAt:      cf.CreatedAt,
			UpdatedAt:      cf.UpdatedAt,
		}
	}

	response := v1schema.CustomFieldListResponse{
		List:   list,
		Offset: offset,
		Limit:  limit,
		Total:  result.Total,
	}

	return c.JSON(schema.SuccessResponse(response))
}
