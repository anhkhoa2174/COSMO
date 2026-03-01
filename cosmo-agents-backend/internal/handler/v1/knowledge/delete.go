package knowledge

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/mapper"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// Delete handles DELETE /v1/knowledge/:id
// @Summary Delete knowledge entry
// @Description Soft deletes a knowledge entry and enqueues pruning for its embeddings
// @Tags Knowledge
// @Produce json
// @Param id path string true "Knowledge ID (UUID)"
// @Success 200 {object} schema.APIResponse[[]v1schema.KnowledgeRead]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 403 {object} schema.APIResponse[any]
// @Failure 404 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/knowledge/{id} [delete]
func (h *Handler) Delete(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	id := c.Params("id")
	knowledgeID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid knowledge ID", err.Error(),
		))
	}

	// Get existing knowledge
	knowledge, err := h.knowledgeRepo.FindByID(c.Context(), knowledgeID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Knowledge not found", err.Error(),
		))
	}

	// Check ownership
	if knowledge.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "Access denied", "",
		))
	}

	if err := h.knowledgeRepo.Delete(c.Context(), knowledgeID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to delete knowledge", err.Error(),
		))
	}

	if h.knowledgeSvc != nil && knowledge.EmbeddingGID != nil {
		if err := h.knowledgeSvc.EnqueuePruning(c.Context(), []string{*knowledge.EmbeddingGID}); err != nil {
			// Best-effort: log and continue so response matches data state
			logger.Logger.Warn().Err(err).Str("embedding_gid", *knowledge.EmbeddingGID).Msg("Failed to enqueue knowledge pruning")
		}
	}

	knowledge.IsDeleted = true
	response := []v1schema.KnowledgeRead{mapper.ToKnowledgeRead(knowledge)}

	return c.JSON(schema.SuccessResponse(response))
}
