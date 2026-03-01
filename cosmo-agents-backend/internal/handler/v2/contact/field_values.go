package contact

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// GetFieldValues handles GET /v2/contacts/values
func (h *Handler) GetFieldValues(c fiber.Ctx) error {
	// Use auth helper
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	// Parse fields
	fieldsParam := c.Query("fields", "")
	if fieldsParam == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "fields parameter is required",
		})
	}

	fields := strings.Split(fieldsParam, ",")
	for i, field := range fields {
		fields[i] = strings.TrimSpace(field)
	}

	// Parse pagination
	offsetInt, _ := strconv.Atoi(c.Query("offset", "0"))
	limitInt, _ := strconv.Atoi(c.Query("limit", "100"))
	if limitInt > 100 {
		limitInt = 100
	}

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

	// Get field values
	result := make(map[string]interface{})
	for _, field := range fields {
		values, total, err := h.contactRepo.GetDistinctFieldValues(c.Context(), field, userID, orgIDs, offsetInt, limitInt)
		if err != nil {
			logger.Logger.Error().Err(err).Str("field", field).Msg("Failed to get field values")
			continue
		}

		result[field] = fiber.Map{
			"list":  values,
			"total": total,
		}
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"list":   []interface{}{result},
			"total":  int64(len(result)),
			"offset": offsetInt,
			"limit":  limitInt,
		},
	})
}
