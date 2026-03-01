package contact

import (
	"io"

	"github.com/gofiber/fiber/v3"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rs/zerolog/log"
)

// ExtractFromImagePreview handles POST /v1/extract-from-image-preview
// @Summary Extract data from screenshot (no contact required)
// @Description Upload a screenshot and extract structured data for creating a new contact
// @Tags Contacts
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Screenshot image"
// @Success 200 {object} schema.APIResponse[v1schema.ExtractFromImagePreviewResponse] "Successfully extracted data"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/extract-from-image-preview [post]
func (h *Handler) ExtractFromImagePreview(c fiber.Ctx) error {
	// No auth required for preview - just extract data
	// (Or keep auth if you want to limit usage)

	fileHeader, err := c.FormFile("image")
	if err != nil {
		return h.responseHelper.BadRequest(c, "Image file is required", err)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to open uploaded file", err)
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(file)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to read uploaded file", err)
	}

	if h.scraperSvc == nil {
		return h.responseHelper.InternalServerError(c, "Image extraction service not available", nil)
	}

	log.Info().
		Str("filename", fileHeader.Filename).
		Msg("Starting image extraction for preview (create mode)")

	extracted, err := h.scraperSvc.ExtractFromImage(c.Context(), imageBytes, fileHeader.Header.Get("Content-Type"))
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to extract from image preview")
		return h.responseHelper.InternalServerError(c, "Failed to extract data from image", err)
	}

	response := &v1schema.ExtractFromImagePreviewResponse{
		ExtractedData: extracted.Fields,
	}

	return c.Status(fiber.StatusOK).JSON(schema.APIResponse[*v1schema.ExtractFromImagePreviewResponse]{
		Status: "success",
		Data:   response,
	})
}
