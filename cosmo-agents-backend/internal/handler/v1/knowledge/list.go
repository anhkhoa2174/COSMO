package knowledge

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/mapper"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// List handles GET /v1/knowledge
// @Summary List knowledge entries
// @Description Retrieves paginated list of knowledge entries for the authenticated user
// @Tags Knowledge
// @Produce json
// @Param limit query int false "Maximum number of items (default: 10, max: 100)"
// @Param skip query int false "Number of items to skip (default: 0)"
// @Param offset query int false "Alias for skip"
// @Success 200 {object} schema.APIResponse[[]v1schema.KnowledgeRead]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/knowledge [get]
func (h *Handler) List(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	skipParam := c.Query("skip", "")
	if skipParam == "" {
		skipParam = c.Query("offset", "0")
	}
	offset, _ := strconv.Atoi(skipParam)

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	knowledges, _, err := h.knowledgeRepo.GetByUserID(c.Context(), userID.String(), limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to fetch knowledge", err.Error(),
		))
	}

	list := make([]v1schema.KnowledgeRead, len(knowledges))
	for i, k := range knowledges {
		list[i] = mapper.ToKnowledgeRead(&k)
	}

	return c.JSON(schema.SuccessResponse(list))
}
