package contact

import (
	"fmt"
	"strings"

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

	// Ownership and identity are not editable here. The field validator lists
	// user_id and organization_id as ordinary contact fields, so a batch could
	// move the caller's contacts to another user or organisation.
	for name := range req.Fields {
		if batchProtectedFields[strings.ToLower(strings.TrimSpace(name))] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": fmt.Sprintf("Field '%s' cannot be changed in a batch update", name),
			})
		}
	}

	// Validate fields using field validator
	if err := h.fieldValidator.ValidateContactFields(c.Context(), req.Fields, organizationID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Invalid field: %v", err),
		})
	}

	// Get contacts by email. The email column was dropped (migration 000041);
	// the address now lives in profile->>'email' and, for most sources, in
	// contact_information, so match either, case-insensitively.
	lowered := make([]string, len(req.Emails))
	for i, e := range req.Emails {
		lowered[i] = strings.ToLower(strings.TrimSpace(e))
	}
	var contacts []domain.Contact
	if err := h.contactRepo.GetDB().WithContext(c.Context()).
		Where("(LOWER(profile->>'email') IN ? OR LOWER(contact_information) IN ?) AND user_id = ? AND is_deleted = ?", lowered, lowered, user.ID, false).
		Find(&contacts).Error; err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to get contacts")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to get contacts",
		})
	}

	// Update each contact. A contact counts as updated only when every field
	// was written; it used to be counted even when all of them failed.
	updatedCount := 0
	failed := map[string]string{}
	for i := range contacts {
		ok := true
		for fieldName, value := range req.Fields {
			if _, err := h.contactRepo.UpdateFields(c.Context(), contacts[i].ID, map[string]interface{}{fieldName: value}, user.ID, organizationID); err != nil {
				logger.Logger.Error().Err(err).Str("contact_id", contacts[i].ID.String()).Str("field", fieldName).Msg("Failed to update contact")
				failed[contacts[i].ID.String()] = fmt.Sprintf("could not update '%s'", fieldName)
				ok = false
				break
			}
		}
		if ok {
			updatedCount++
		}
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"updated_count": updatedCount,
			"failed":        failed,
			"message":       fmt.Sprintf("Updated %d of %d contacts", updatedCount, len(contacts)),
		},
	})
}

// batchProtectedFields may not be set through a batch update.
var batchProtectedFields = map[string]bool{
	"id": true, "user_id": true, "organization_id": true,
	"created_at": true, "updated_at": true, "is_deleted": true, "deleted_at": true,
	"source": true, "source_id": true,
}
