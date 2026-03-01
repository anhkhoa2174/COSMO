package contact

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// GetFieldValues handles GET /v1/contacts/values
// @Summary Get distinct field values
// @Description Retrieves distinct values for specified contact fields
// @Tags Contacts
// @Accept json
// @Produce json
// @Param fields query []string true "Fields to get distinct values for"
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit" default(100)
// @Success 200 {object} schema.APIResponse[schema.PaginatedResponse[v1schema.ContactFieldValuesResponse]] "Successfully retrieved field values"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/contacts/values [get]
func (h *Handler) GetFieldValues(c fiber.Ctx) error {
	// Use auth helper
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	// Get roles for organization IDs
	roles, err := h.roleRepo.FindByUserID(c.Context(), userID)
	if err != nil || len(roles) == 0 {
		return h.responseHelper.Unauthorized(c, "You are not authorized to access this resource")
	}

	organizationIDs := make([]uuid.UUID, len(roles))
	for i, role := range roles {
		organizationIDs[i] = role.OrganizationID
	}

	// Parse fields from query
	fields := parseFieldsFromQuery(c)
	if len(fields) == 0 {
		return h.responseHelper.BadRequest(c, "Fields parameter is required", nil)
	}

	offsetInt, _ := strconv.Atoi(c.Query("offset", "0"))
	limitInt, _ := strconv.Atoi(c.Query("limit", "100"))
	if limitInt > 100 {
		limitInt = 100
	}

	// Get field values for each field
	responseList := make([]map[string]v1schema.FieldValueItem, 0, len(fields))

	for _, field := range fields {
		values, total, err := h.repo.GetDistinctFieldValues(c.Context(), field, userID, organizationIDs, offsetInt, limitInt)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return h.responseHelper.NotFound(c, err.Error(), nil)
			}
			return h.responseHelper.InternalServerError(c, "Failed to fetch field values", err)
		}

		fieldObj := map[string]v1schema.FieldValueItem{
			field: {
				List:  values,
				Total: total,
			},
		}
		responseList = append(responseList, fieldObj)
	}

	paginatedResponse := schema.PaginatedResponse[map[string]v1schema.FieldValueItem]{
		List:   responseList,
		Total:  int64(len(responseList)),
		Offset: offsetInt,
		Limit:  limitInt,
	}

	return h.responseHelper.Success(c, paginatedResponse)
}

// parseFieldsFromQuery extracts field names from query parameters
func parseFieldsFromQuery(c fiber.Ctx) []string {
	fields := make([]string, 0)
	fieldsSeen := make(map[string]struct{})

	queryArgs := c.Request().URI().QueryArgs()
	queryArgs.VisitAll(func(key, value []byte) {
		if string(key) != "fields" {
			return
		}
		field := strings.TrimSpace(string(value))
		if field == "" {
			return
		}
		if _, exists := fieldsSeen[field]; exists {
			return
		}
		fieldsSeen[field] = struct{}{}
		fields = append(fields, field)
	})

	if len(fields) == 0 {
		if raw := strings.TrimSpace(c.Query("fields", "")); raw != "" {
			for _, part := range strings.Split(raw, ",") {
				field := strings.TrimSpace(part)
				if field == "" {
					continue
				}
				if _, exists := fieldsSeen[field]; exists {
					continue
				}
				fieldsSeen[field] = struct{}{}
				fields = append(fields, field)
			}
		}
	}

	return fields
}
