package knowledge

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
)

// Upload handles file upload to knowledge base
// POST /v2/knowledge/upload
// @Summary Upload knowledge files
// @Description Uploads knowledge files to the knowledge base (V2 API)
// @Tags Knowledge V2
// @Accept multipart/form-data
// @Produce json
// @Param files formData file true "Knowledge files to upload"
// @Success 200 {object} schema.APIResponse[[]v2schema.UploadKnowledgeResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 503 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v2/knowledge/upload [post]
func (h *Handler) Upload(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"",
		))
	}

	if h.knowledgeSvc == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(schema.ErrorResponse(
			fiber.StatusServiceUnavailable,
			"Knowledge upload not configured",
			"",
		))
	}

	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Failed to parse multipart form",
			err.Error(),
		))
	}
	defer form.RemoveAll()

	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"No files provided",
			"",
		))
	}

	uploaded, err := h.knowledgeSvc.Upload(c.Context(), userID, files)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to upload knowledge",
			err.Error(),
		))
	}

	response := make([]v2schema.UploadKnowledgeResponse, 0, len(uploaded))
	for _, item := range uploaded {
		response = append(response, toV2UploadResponse(item))
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(response))
}

func toV2UploadResponse(resp v1schema.UploadKnowledgeResponse) v2schema.UploadKnowledgeResponse {
	return v2schema.UploadKnowledgeResponse{
		Result: v2schema.UploadKnowledgeResult{
			Msg:       resp.Result.Msg,
			Knowledge: convertKnowledgeRead(resp.Result.Knowledge),
		},
	}
}

func convertKnowledgeRead(src *v1schema.KnowledgeRead) *v2schema.KnowledgeRead {
	if src == nil {
		return nil
	}

	metadata := make(map[string]interface{})
	if src.CMetadata != nil {
		metadata = src.CMetadata
	}

	name := src.ID.String()
	if origin, ok := metadata["origin"].(map[string]interface{}); ok {
		if filename, ok := origin["filename"].(string); ok && filename != "" {
			name = filename
		}
	}

	embeddingGID := ""
	if src.EmbeddingGID != nil {
		embeddingGID = *src.EmbeddingGID
	}

	summary := joinSummaryPair(src.SummaryPair)

	dest := &v2schema.KnowledgeRead{
		ID:           src.ID,
		UserID:       src.UserID,
		Name:         name,
		SourceType:   src.SourceType,
		SummaryPair:  summary,
		EmbeddingGID: embeddingGID,
		Collection:   "",
		CMetadata:    metadata,
		IsDeleted:    src.IsDeleted,
		CreatedAt:    src.CreatedAt,
		UpdatedAt:    src.UpdatedAt,
	}
	if src.CozeDatasetID != nil {
		dest.CozeDatasetID = src.CozeDatasetID
	}
	return dest
}
