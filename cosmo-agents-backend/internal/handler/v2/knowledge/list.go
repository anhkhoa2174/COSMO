package knowledge

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/schema"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
)

// List returns user's knowledge
// GET /v2/knowledge
// @Summary List user's knowledge
// @Description Retrieves paginated list of knowledge entries for the authenticated user (V2 API)
// @Tags Knowledge V2
// @Produce json
// @Param limit query int false "Maximum number of items (default: 10)"
// @Param skip query int false "Number of items to skip (default: 0)"
// @Success 200 {object} schema.APIResponse[[]v2schema.KnowledgeRead]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v2/knowledge [get]
func (h *Handler) List(c fiber.Ctx) error {
	// Get current user from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"",
		))
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	skip, _ := strconv.Atoi(c.Query("skip", "0"))

	// Get knowledge items for the user
	knowledges, _, err := h.knowledgeRepo.GetByUserID(c.Context(), userID.String(), limit, skip)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to fetch knowledge",
			err.Error(),
		))
	}

	// Convert to response format
	response := make([]v2schema.KnowledgeRead, 0, len(knowledges))
	for idx := range knowledges {
		response = append(response, mapDomainKnowledgeToV2(&knowledges[idx]))
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(response))
}
