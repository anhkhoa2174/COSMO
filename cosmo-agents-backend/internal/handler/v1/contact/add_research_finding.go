package contact

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// AddResearchFinding handles POST /v1/contacts/{id}/research-findings
// @Summary Add research finding to contact
// @Description Adds a research finding to contact's profile (custom field + activity log)
// @Tags Contacts
// @Accept json
// @Produce json
// @Param id path string true "Contact ID (UUID)"
// @Param body body v1schema.AddResearchFindingRequest true "Research finding data"
// @Success 200 {object} schema.APIResponse[v1schema.ContactResponse] "Successfully added finding"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 404 {object} schema.APIResponse[any] "Contact not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/contacts/{id}/research-findings [post]
func (h *Handler) AddResearchFinding(c fiber.Ctx) error {
	// Use auth helper
	user, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return h.responseHelper.HandleAuthError(c, errors.New("You are not authorized to access this resource"))
		}
		return h.responseHelper.HandleAuthError(c, err)
	}

	// Parse ID
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", err)
	}

	// Parse request
	var req v1schema.AddResearchFindingRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	// Get existing contact
	contact, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to fetch contact", err)
	}

	// Check authorization. A missing contact and someone else's contact get
	// the same 404, so the endpoint cannot be used to probe for contact ids.
	if contact == nil || contact.UserID != user.ID {
		return h.responseHelper.NotFound(c, "Contact not found", nil)
	}

	// Parse existing profile
	var profile map[string]interface{}
	if err := json.Unmarshal(contact.Profile, &profile); err != nil {
		profile = make(map[string]interface{})
	}

	// Ensure research_findings array exists
	if profile["research_findings"] == nil {
		profile["research_findings"] = []interface{}{}
	}

	findings, ok := profile["research_findings"].([]interface{})
	if !ok {
		findings = []interface{}{}
	}

	// Add new finding
	finding := map[string]interface{}{
		"category":      req.Category,
		"field_name":    req.FieldName,
		"value":         req.Value,
		"source":        req.Source,
		"added_at":      time.Now().UTC().Format(time.RFC3339),
		"added_by":      user.ID.String(),
		"priority":      req.Priority,
		"why_important": req.WhyImportant,
	}

	findings = append(findings, finding)
	profile["research_findings"] = findings

	// Also add to custom fields for easy access
	if profile["custom_fields"] == nil {
		profile["custom_fields"] = make(map[string]interface{})
	}

	customFields, ok := profile["custom_fields"].(map[string]interface{})
	if !ok {
		customFields = make(map[string]interface{})
	}

	// Store the value in custom fields with the field name
	customFields[req.FieldName] = map[string]interface{}{
		"value":      req.Value,
		"source":     req.Source,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}

	profile["custom_fields"] = customFields

	// Marshal updated profile
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to marshal profile", err)
	}

	// Update contact
	contact.Profile = domain.JSONB(profileJSON)
	_, err = h.repo.UpdateFields(c.Context(), id, map[string]interface{}{
		"profile": contact.Profile,
	}, user.ID, organizationID)

	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to update contact", err)
	}

	// Trigger re-enrichment when custom fields change
	// This allows AI to re-analyze holistically with new data
	if h.intelSvc != nil {
		log.Info().
			Str("contact_id", id.String()).
			Str("field_name", req.FieldName).
			Msg("Triggering background re-enrichment after research finding added")
		go func() {
			// Run in background to avoid blocking the response
			// Use background context since request context will be cancelled
			ctx := context.Background()
			// Force refresh to regenerate AI insights with new custom field data
			result, err := h.intelSvc.EnrichContact(ctx, user.ID, organizationID, id, true)
			if err != nil {
				log.Error().
					Err(err).
					Str("contact_id", id.String()).
					Msg("Background re-enrichment failed")
			} else {
				log.Info().
					Str("contact_id", id.String()).
					Bool("had_insights", result != nil).
					Msg("Background re-enrichment completed successfully")
			}
		}()
	}

	// Fetch updated contact
	updated, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to fetch updated contact", err)
	}

	// Convert to response
	response := v1schema.ToContactResponse(updated)

	return c.Status(fiber.StatusOK).JSON(schema.APIResponse[*v1schema.ContactResponse]{
		Status: "success",
		Data:   response,
	})
}
