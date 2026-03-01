package knowledge

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// Search handles POST /v1/knowledge/search
// @Summary Search knowledge base
// @Description Performs semantic search across user's knowledge base
// @Tags Knowledge
// @Accept json
// @Produce json
// @Param request body v1schema.KnowledgeSearchRequest true "Search request"
// @Success 200 {object} schema.APIResponse[[]v1schema.KnowledgeSearchResult]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/knowledge/search [post]
func (h *Handler) Search(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	var req v1schema.KnowledgeSearchRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	// Set default top_k if not provided
	topK := 10
	if req.TopK != nil && *req.TopK > 0 {
		topK = *req.TopK
		if topK > 100 {
			topK = 100
		}
	}

	filter := req.Filter
	if filter == nil {
		filter = make(map[string]interface{})
	}

	if h.knowledgeSvc == nil {
		return c.JSON(schema.SuccessResponse([]v1schema.KnowledgeSearchResult{}))
	}

	results, err := h.knowledgeSvc.Search(c.Context(), userID, req.Query, topK, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to search knowledge", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(results))
}
