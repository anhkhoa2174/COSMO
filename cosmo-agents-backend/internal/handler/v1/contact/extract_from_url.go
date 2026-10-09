package contact

import (
	"encoding/json"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// ExtractFromURL handles POST /v1/contacts/{id}/extract-from-url
// @Summary Extract contact data from URL
// @Description Fetches a profile URL (LinkedIn, Twitter, etc.) and uses AI to extract structured data into custom fields
// @Tags Contacts
// @Accept json
// @Produce json
// @Param id path string true "Contact ID (UUID)"
// @Param body body v1schema.ExtractFromURLRequest true "URL to extract from"
// @Success 200 {object} schema.APIResponse[v1schema.ExtractFromURLResponse] "Successfully extracted data"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 404 {object} schema.APIResponse[any] "Contact not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/contacts/{id}/extract-from-url [post]
func (h *Handler) ExtractFromURL(c fiber.Ctx) error {
	// Use auth helper
	user, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return h.responseHelper.HandleAuthError(c, errors.New("You are not authorized to access this resource"))
		}
		return h.responseHelper.HandleAuthError(c, err)
	}

	// Parse contact ID
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

	// Parse request
	var req v1schema.ExtractFromURLRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	// Validate URL
	if req.URL == "" {
		return h.responseHelper.BadRequest(c, "URL is required", nil)
	}

	// Get existing contact
	contact, err := h.repo.GetByID(c.Context(), id)
	if err != nil || contact == nil {
		return h.responseHelper.NotFound(c, "Contact not found", err)
	}

	// Check authorization
	if contact.UserID != user.ID {
		// Answered as not found, like a missing contact, so the endpoint cannot
		// be used to learn which contact IDs exist in other organisations.
		return h.responseHelper.NotFound(c, "Contact not found", nil)
	}

	// Check if scraper service is available
	if h.scraperSvc == nil {
		return h.responseHelper.InternalServerError(c, "URL extraction service not available", nil)
	}

	log.Info().
		Str("contact_id", id.String()).
		Str("url", req.URL).
		Msg("Starting URL extraction for contact")

	// Step 1: Extract data from URL using AI
	extracted, err := h.scraperSvc.ExtractFromURL(c.Context(), req.URL)
	if err != nil {
		log.Error().
			Err(err).
			Str("contact_id", id.String()).
			Str("url", req.URL).
			Msg("Failed to extract from URL")
		return h.responseHelper.InternalServerError(c, "Failed to extract data from URL", err)
	}

	// Step 2: Parse existing profile
	var profile map[string]interface{}
	if err := json.Unmarshal(contact.Profile, &profile); err != nil {
		profile = make(map[string]interface{})
	}

	// Step 3: Merge extracted data into profile and collect contact-level updates
	updatedProfile, contactUpdates, fieldsAdded := h.scraperSvc.MergeIntoContactProfile(
		profile,
		extracted.Fields,
		req.URL,
	)

	// Step 4: Marshal updated profile
	profileJSON, err := json.Marshal(updatedProfile)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to marshal profile", err)
	}

	// Step 5: Update contact (profile + standard fields if present) - best effort even if fields are empty
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

	log.Info().
		Str("contact_id", id.String()).
		Str("url", req.URL).
		Int("fields_added", len(fieldsAdded)).
		Msg("Successfully extracted and saved data from URL")

	// Return response
	response := &v1schema.ExtractFromURLResponse{
		URL:            req.URL,
		ExtractedData:  extracted.Fields,
		FieldsAdded:    fieldsAdded,
		ContactUpdated: true,
		Message:        "Successfully extracted and saved data from URL",
	}

	return c.Status(fiber.StatusOK).JSON(schema.APIResponse[*v1schema.ExtractFromURLResponse]{
		Status: "success",
		Data:   response,
	})
}
