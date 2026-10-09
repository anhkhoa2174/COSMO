package contact

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// ExtractFromImage handles POST /v1/contacts/{id}/extract-from-image
// @Summary Extract contact data from screenshot
// @Description Upload a screenshot image and extract structured data using AI vision
// @Tags Contacts
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Contact ID (UUID)"
// @Param image formData file true "Screenshot image"
// @Success 200 {object} schema.APIResponse[v1schema.ExtractFromImageResponse] "Successfully extracted data"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 404 {object} schema.APIResponse[any] "Contact not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/contacts/{id}/extract-from-image [post]
func (h *Handler) ExtractFromImage(c fiber.Ctx) error {
	user, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return h.responseHelper.HandleAuthError(c, errors.New("You are not authorized to access this resource"))
		}
		return h.responseHelper.HandleAuthError(c, err)
	}

	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

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

	contact, err := h.repo.GetByID(c.Context(), id)
	if err != nil || contact == nil {
		return h.responseHelper.NotFound(c, "Contact not found", err)
	}
	if contact.UserID != user.ID {
		// Answered as not found, like a missing contact, so the endpoint cannot
		// be used to learn which contact IDs exist in other organisations.
		return h.responseHelper.NotFound(c, "Contact not found", nil)
	}

	if h.scraperSvc == nil {
		return h.responseHelper.InternalServerError(c, "Image extraction service not available", nil)
	}

	log.Info().
		Str("contact_id", id.String()).
		Str("filename", fileHeader.Filename).
		Msg("Starting image extraction for contact")

	extracted, err := h.scraperSvc.ExtractFromImage(c.Context(), imageBytes, fileHeader.Header.Get("Content-Type"))
	if err != nil {
		log.Error().
			Err(err).
			Str("contact_id", id.String()).
			Msg("Failed to extract from image")
		return h.responseHelper.InternalServerError(c, "Failed to extract data from image", err)
	}

	var profile map[string]interface{}
	if err := json.Unmarshal(contact.Profile, &profile); err != nil {
		profile = make(map[string]interface{})
	}

	updatedProfile, contactUpdates, fieldsAdded := h.scraperSvc.MergeIntoContactProfile(
		profile,
		extracted.Fields,
		"screenshot",
	)

	profileJSON, err := json.Marshal(updatedProfile)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to marshal profile", err)
	}

	contact.Profile = domain.JSONB(profileJSON)
	updatePayload := map[string]interface{}{
		"profile": contact.Profile,
	}
	for k, v := range contactUpdates {
		updatePayload[k] = v
	}

	_, err = h.repo.UpdateFields(c.Context(), id, updatePayload, user.ID, organizationID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to update contact", err)
	}

	response := &v1schema.ExtractFromImageResponse{
		ExtractedData:  extracted.Fields,
		FieldsAdded:    fieldsAdded,
		ContactUpdated: true,
		Message:        "Successfully extracted and saved data from image",
	}

	return c.Status(fiber.StatusOK).JSON(schema.APIResponse[*v1schema.ExtractFromImageResponse]{
		Status: "success",
		Data:   response,
	})
}
