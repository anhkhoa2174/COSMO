package contact

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

var _ = schema.APIResponse[any]{}

// Search handles POST /v2/contacts/search
// @Summary Search contacts
// @Description Searches contacts using legacy filter payloads and returns results matching the original Python API.
// @Tags V2 Contacts
// @Accept json
// @Produce json
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit" default(25)
// @Param body body v2schema.ContactSearchRequest false "Search filters"
// @Success 200 {object} schema.APIResponse[schema.PaginatedResponse[ContactEntityResponseItem]] "Successfully retrieved contacts"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v2/contacts/search [post]
func (h *Handler) Search(c fiber.Ctx) error {
	// Use auth helper
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	// Parse request
	var req v2schema.ContactSearchRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	// Parse pagination
	offsetInt, limitInt := parsePagination(c, 0, 25)

	// Get user's organizations
	roles, err := h.roleRepo.FindByUserID(c.Context(), userID)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to get user roles")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to get user roles",
		})
	}

	var orgIDs []uuid.UUID
	for _, role := range roles {
		orgIDs = append(orgIDs, role.OrganizationID)
	}

	if len(orgIDs) == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "You are not authorized to access this resource",
		})
	}

	// Build filter
	filter := map[string]interface{}{"is_deleted": false}
	if req.Filter != nil {
		filterCopy := deepCopyFilter(req.Filter)
		sanitizeFilter(filterCopy)
		for k, v := range filterCopy {
			filter[k] = v
		}
	}

	// Query contacts
	contacts, total, err := h.contactRepo.SearchWithFilter(c.Context(), userID, orgIDs, filter, &baseRepo.PaginationParams{
		Offset: offsetInt,
		Limit:  limitInt,
	})
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to search contacts")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to get contacts",
		})
	}

	// Collect unique user IDs for "Added By" info
	userIDsMap := make(map[uuid.UUID]bool)
	for _, contact := range contacts {
		userIDsMap[contact.UserID] = true
	}
	var userIDs []uuid.UUID
	for uid := range userIDsMap {
		userIDs = append(userIDs, uid)
	}

	// Batch fetch users for "Added By" info
	userMap := make(map[uuid.UUID]*struct{ Name, Email string })
	if len(userIDs) > 0 {
		users, err := h.userRepo.FindByIDs(c.Context(), userIDs)
		if err != nil {
			logger.Logger.Warn().Err(err).Msg("Failed to fetch users for added_by info, continuing without it")
		} else {
			for _, u := range users {
				userMap[u.ID] = &struct{ Name, Email string }{Name: u.Name, Email: u.Email}
			}
		}
	}

	// Convert to response entities
	entities := make([]ContactEntityResponseItem, len(contacts))
	for i, contact := range contacts {
		entity, err := h.toContactEntityInList(contact)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Failed to convert contact to entity",
			})
		}

		// Populate "Added By" info
		if userInfo, ok := userMap[contact.UserID]; ok {
			if userInfo.Name != "" {
				entity.AddedByName = &userInfo.Name
			}
			if userInfo.Email != "" {
				entity.AddedByEmail = &userInfo.Email
			}
		}

		entities[i] = ContactEntityResponseItem{Entity: entity}
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"list":   entities,
			"total":  total,
			"offset": offsetInt,
			"limit":  limitInt,
		},
	})
}
