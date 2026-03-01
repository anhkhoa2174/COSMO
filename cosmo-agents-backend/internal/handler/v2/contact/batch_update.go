package contact

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// BatchUpdate handles POST /v2/contacts/batch
func (h *Handler) BatchUpdate(c fiber.Ctx) error {
	// Use auth helper
	user, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	// Parse request
	var req BatchUpdateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid request body",
		})
	}

	if len(req.Emails) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "emails array cannot be empty",
		})
	}

	if len(req.Fields) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "fields map cannot be empty",
		})
	}

	// Validate fields using field validator
	if err := h.fieldValidator.ValidateContactFields(c.Context(), req.Fields, organizationID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Invalid field: %v", err),
		})
	}

	// Get contacts by email
	var contacts []domain.Contact
	if err := h.contactRepo.GetDB().WithContext(c.Context()).
		Where("email IN ? AND user_id = ? AND is_deleted = ?", req.Emails, user.ID, false).
		Find(&contacts).Error; err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to get contacts")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to get contacts",
		})
	}

	// Update each contact
	updatedCount := 0
	for i := range contacts {
		for fieldName, value := range req.Fields {
			// Use UpdateFields method to update each contact
			_, err := h.contactRepo.UpdateFields(c.Context(), contacts[i].ID, map[string]interface{}{fieldName: value}, user.ID, organizationID)
			if err != nil {
				logger.Logger.Error().Err(err).Str("contact_id", contacts[i].ID.String()).Msg("Failed to update contact")
				continue
			}
		}
		updatedCount++
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"updated_count": updatedCount,
			"message":       fmt.Sprintf("Successfully updated %d contacts", updatedCount),
		},
	})
}
