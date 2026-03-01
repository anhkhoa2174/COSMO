package contact

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// ExtractFromExtension handles POST /v1/contacts/{id}/extract-from-extension
// @Summary Extract contact data from browser extension JSON
// @Description Accepts structured data from a browser extension and maps to system/custom fields
// @Tags Contacts
// @Accept json
// @Produce json
// @Param id path string true "Contact ID (UUID)"
// @Param body body v1schema.ExtractFromExtensionRequest true "Extracted data from extension"
// @Success 200 {object} schema.APIResponse[v1schema.ExtractFromExtensionResponse] "Successfully extracted data"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 404 {object} schema.APIResponse[any] "Contact not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/contacts/{id}/extract-from-extension [post]
func (h *Handler) ExtractFromExtension(c fiber.Ctx) error {
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

	var req v1schema.ExtractFromExtensionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}
	if len(req.Data) == 0 {
		return h.responseHelper.BadRequest(c, "Data is required", nil)
	}

	contact, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return h.responseHelper.NotFound(c, "Contact not found", err)
	}
	if contact.UserID != user.ID {
		return h.responseHelper.HandleAuthError(c, errors.New("You are not authorized to access this resource"))
	}

	if h.scraperSvc == nil {
		return h.responseHelper.InternalServerError(c, "Extension extraction service not available", nil)
	}

	sourceURL := req.SourceURL
	if sourceURL == "" {
		sourceURL = "extension"
	}

	log.Info().
		Str("contact_id", id.String()).
		Str("source_url", sourceURL).
		Msg("Starting extension extraction for contact")

	var profile map[string]interface{}
	if err := json.Unmarshal(contact.Profile, &profile); err != nil {
		profile = make(map[string]interface{})
	}

	extractedData := req.Data
	if req.UseAI && strings.TrimSpace(req.RawText) != "" {
		aiExtracted, err := h.scraperSvc.ExtractFromText(c.Context(), req.RawText)
		if err == nil && len(aiExtracted.Fields) > 0 {
			for key, value := range aiExtracted.Fields {
				if value == nil {
					continue
				}
				if str, ok := value.(string); ok && strings.TrimSpace(str) == "" {
					continue
				}
				if _, exists := extractedData[key]; !exists {
					extractedData[key] = value
				}
			}
		}
	}

	updatedProfile, contactUpdates, fieldsAdded := h.scraperSvc.MergeIntoContactProfile(
		profile,
		extractedData,
		sourceURL,
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

	response := &v1schema.ExtractFromExtensionResponse{
		SourceURL:      sourceURL,
		ExtractedData:  extractedData,
		FieldsAdded:    fieldsAdded,
		ContactUpdated: true,
		Message:        "Successfully extracted and saved data from extension",
	}

	return c.Status(fiber.StatusOK).JSON(schema.APIResponse[*v1schema.ExtractFromExtensionResponse]{
		Status: "success",
		Data:   response,
	})
}
