package knowledge

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/schema"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
)

// Delete deletes a knowledge item
// DELETE /v2/knowledge/{knowledge_id}
// @Summary Delete knowledge entry
// @Description Deletes a knowledge entry (V2 API - simplified)
// @Tags Knowledge V2
// @Produce json
// @Param knowledge_id path string true "Knowledge ID (UUID)"
// @Success 200 {object} schema.APIResponse[[]v2schema.KnowledgeRead]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 403 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v2/knowledge/{knowledge_id} [delete]
func (h *Handler) Delete(c fiber.Ctx) error {
	// Get current user from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"",
		))
	}

	knowledgeID, err := uuid.Parse(c.Params("knowledge_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid knowledge ID",
			err.Error(),
		))
	}

	// Find the knowledge item
	knowledge, err := h.knowledgeRepo.FindByID(c.Context(), knowledgeID)
	if err != nil || knowledge == nil {
		return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(fiber.Map{
			"message": "Knowledge not found",
		}))
	}

	// Verify ownership
	if knowledge.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden,
			"Permission denied",
			"",
		))
	}

	// Delete the knowledge
	err = h.knowledgeRepo.Delete(c.Context(), knowledgeID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to delete knowledge",
			err.Error(),
		))
	}

	// Return the deleted knowledge
	response := mapDomainKnowledgeToV2(knowledge)

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse([]v2schema.KnowledgeRead{response}))
}
