package knowledge

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// Upload handles POST /v1/knowledge/upload
// @Summary Upload knowledge files
// @Description Uploads one or more files to the knowledge base (PDF, TXT, CSV, etc.)
// @Tags Knowledge
// @Accept multipart/form-data
// @Produce json
// @Param files formData file true "Knowledge files to upload"
// @Success 200 {object} schema.APIResponse[any]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Failure 503 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/knowledge/upload [post]
func (h *Handler) Upload(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	if h.knowledgeSvc == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			fiber.StatusServiceUnavailable, "Knowledge upload not configured", "",
		))
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Failed to parse multipart form", err.Error(),
		))
	}
	// Release temporary files promptly
	defer form.RemoveAll()

	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "No files provided", "",
		))
	}

	responses, err := h.knowledgeSvc.Upload(c.Context(), userID, files)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to upload knowledge", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(responses))
}
