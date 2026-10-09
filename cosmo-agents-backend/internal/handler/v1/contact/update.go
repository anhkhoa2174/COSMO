package contact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Update handles PATCH /v1/contacts/{id}
// @Summary Update contact
// @Description Updates a contact by ID (partial update with custom fields support)
// @Tags Contacts
// @Accept json
// @Produce json
// @Param id path string true "Contact ID (UUID)"
// @Param body body v1schema.UpdateContactRequest true "Updated contact data"
// @Success 200 {object} schema.APIResponse[v1schema.ContactResponse] "Successfully updated contact"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 404 {object} schema.APIResponse[any] "Contact not found"
// @Failure 422 {object} schema.APIResponse[any] "Invalid field"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/contacts/{id} [patch]
func (h *Handler) Update(c fiber.Ctx) error {
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
	var req v1schema.UpdateContactRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}
	// An empty email means "clear it". The validator's omitempty does not skip
	// an empty string behind a non-nil pointer, so it failed as an invalid
	// address and the contact could not be saved without one.
	validated := req
	if validated.Email != nil && strings.TrimSpace(*validated.Email) == "" {
		validated.Email = nil
	}
	// Same contract as Create: tag validation failures are 422.
	if err := v1validation.ValidateStruct(validated); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(
			schema.ErrorResponse(fiber.StatusUnprocessableEntity, err.Error(), ""),
		)
	}

	// Build attributes map for validation (exclude extra fields - they go to custom_fields)
	attributesForValidation := buildUpdateAttributes(&req, false)

	// Validate only system fields, not extra fields (custom fields can have any name)
	if len(attributesForValidation) > 0 {
		if err := h.fieldValidator.ValidateContactFields(c.Context(), attributesForValidation, organizationID); err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(
				schema.ErrorResponse(fiber.StatusUnprocessableEntity, err.Error(), ""),
			)
		}
	}

	updateAttributes := buildUpdateAttributes(&req, false)
	convertMapJSONToJSONB(updateAttributes)
	// Email and phone have no column since migrations 000041/000043; they are
	// written to the profile below. Left in, GORM emits SET email=..., which
	// Postgres rejects, so any edit carrying either field returned 500.
	delete(updateAttributes, "email")
	delete(updateAttributes, "phone")

	// Fetch the existing contact
	existingContact, err := h.repo.GetByID(c.Context(), id)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to fetch contact", err)
	}
	if existingContact == nil {
		return h.responseHelper.NotFound(c, "Contact not found", nil)
	}

	profile, err := convertUpdateRequestToProfile(req, existingContact)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to process contact data", err)
	}
	updateAttributes["profile"] = profile

	// Sync: if contact_information was set via custom fields, also update the column
	if ciVal, ok := req.ExtraFields["contact_information"].(string); ok && ciVal != "" && req.ContactInformation == nil {
		updateAttributes["contact_information"] = ciVal
		existingContact.ContactInformation = ciVal
	}

	// contact_information is the dedupe key and the send worker's recipient.
	// When it was derived from the email or LinkedIn URL (Create sets it so),
	// a new value must move it too, or duplicates are checked and messages are
	// sent against the old address.
	if _, set := updateAttributes["contact_information"]; !set {
		var oldProfile map[string]interface{}
		_ = existingContact.Profile.Unmarshal(&oldProfile)
		for _, f := range []struct {
			key    string
			newVal *string
		}{{"email", req.Email}, {"linkedin_url", req.LinkedInURL}} {
			key, newVal := f.key, f.newVal
			if newVal == nil || strings.TrimSpace(*newVal) == "" {
				continue
			}
			oldVal, _ := oldProfile[key].(string)
			ci := existingContact.ContactInformation
			if ci == "" || (oldVal != "" && strings.EqualFold(ci, oldVal)) {
				updateAttributes["contact_information"] = strings.TrimSpace(*newVal)
				existingContact.ContactInformation = strings.TrimSpace(*newVal)
				break
			}
		}
	}

	// Apply updates to existing contact for status calculation
	if req.Name != nil {
		existingContact.Name = *req.Name
	}
	// Email is now stored in profile - update existingContact.Profile for status calculation
	if req.Email != nil {
		existingContact.Profile = profile
	}
	if req.Company != nil {
		existingContact.Company = *req.Company
	}
	if req.JobTitle != nil {
		existingContact.JobTitle = *req.JobTitle
	}
	if req.Source != nil {
		existingContact.Source = *req.Source
	}
	if req.ContactInformation != nil {
		existingContact.ContactInformation = *req.ContactInformation
	}

	// Recalculate status after update
	existingContact.CalculateStatus()
	updateAttributes["status"] = existingContact.Status
	updateAttributes["missing_fields"] = existingContact.MissingFields

	// Update contact
	contact, err := h.repo.UpdateFields(c.Context(), id, updateAttributes, user.ID, organizationID)
	if err != nil {
		if err.Error() == "Contact not found" {
			return h.responseHelper.NotFound(c, "Contact not found", nil)
		}
		return h.responseHelper.InternalServerError(c, "Failed to update contact", err)
	}

	// Trigger re-enrichment if important fields were updated
	// This includes: custom fields, company, job_title, confirmed_facts, or any field that affects AI insights
	shouldReEnrich := len(req.ExtraFields) > 0 ||
		req.Company != nil ||
		req.JobTitle != nil ||
		req.ConfirmedFacts != nil ||
		req.Name != nil

	if shouldReEnrich && h.intelSvc != nil {
		log.Info().
			Str("contact_id", id.String()).
			Int("extra_fields_count", len(req.ExtraFields)).
			Bool("company_changed", req.Company != nil).
			Bool("job_title_changed", req.JobTitle != nil).
			Bool("confirmed_facts_changed", req.ConfirmedFacts != nil).
			Bool("name_changed", req.Name != nil).
			Msg("Triggering background re-enrichment after important fields updated")
		go func() {
			// Run in background to avoid blocking the response
			// Use background context since request context will be cancelled
			ctx := context.Background()
			// Force refresh to regenerate AI insights with new data
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

	// Auto-calculate segment fit scores after contact update
	// This ensures contacts are automatically assigned to appropriate segments
	if h.intelSvc != nil {
		go func() {
			ctx := context.Background()
			log.Info().
				Str("contact_id", id.String()).
				Msg("Auto-calculating segment fit scores after contact update")

			// Calculate scores for all active segments
			_, err := h.intelSvc.CalculateSegmentScores(ctx, user.ID, organizationID, id, nil)
			if err != nil {
				log.Error().
					Err(err).
					Str("contact_id", id.String()).
					Msg("Auto-calculate segment scores failed")
			} else {
				log.Info().
					Str("contact_id", id.String()).
					Msg("Auto-calculate segment scores completed")
			}
		}()
	}

	if h.workerClient != nil {
		payload := map[string]interface{}{
			"contact_id": id,
			"user_id":    user.ID,
			"org_id":     organizationID,
			"event":      "contact_updated",
		}
		_, _ = h.workerClient.EnqueueTask(context.Background(), worker.TypeOrchestrateContact, payload)
	}

	response := v1schema.ToContactResponse(contact)
	return h.responseHelper.Success(c, response)
}

// buildUpdateAttributes constructs the attributes map from UpdateContactRequest
func buildUpdateAttributes(req *v1schema.UpdateContactRequest, isIncludeExtraFields bool) map[string]interface{} {
	attributes := make(map[string]interface{})

	if req.Name != nil {
		attributes["name"] = *req.Name
	}
	if req.Email != nil {
		attributes["email"] = *req.Email
	}
	if req.Phone != nil {
		attributes["phone"] = *req.Phone
	}
	if req.Company != nil {
		attributes["company"] = *req.Company
	}
	if req.JobTitle != nil {
		attributes["job_title"] = *req.JobTitle
	}
	if req.Address != nil {
		attributes["address"] = *req.Address
	}
	if req.City != nil {
		attributes["city"] = *req.City
	}
	if req.Country != nil {
		attributes["country"] = *req.Country
	}
	if req.State != nil {
		attributes["state"] = *req.State
	}
	if req.Zip != nil {
		attributes["zip"] = *req.Zip
	}
	if req.DoNotContact != nil {
		attributes["do_not_contact"] = *req.DoNotContact
	}
	if req.ConfirmedFacts != nil {
		attributes["confirmed_facts"] = req.ConfirmedFacts
	}
	if req.AIInsights != nil {
		attributes["ai_insights"] = req.AIInsights
	}
	if req.InsightValidation != nil {
		attributes["insight_validation"] = req.InsightValidation
	}
	if req.Scores != nil {
		attributes["scores"] = req.Scores
	}

	// New system fields for status calculation
	if req.Source != nil {
		attributes["source"] = *req.Source
	}
	if req.ContactInformation != nil {
		attributes["contact_information"] = *req.ContactInformation
	}

	// Outreach context fields
	if req.Industry != nil {
		attributes["industry"] = *req.Industry
	}
	if req.ContactChannel != nil {
		attributes["contact_channel"] = *req.ContactChannel
	}
	if req.ContextLevel != nil {
		attributes["context_level"] = *req.ContextLevel
	}
	if req.OutreachDecision != nil {
		attributes["outreach_decision"] = *req.OutreachDecision
	}
	if req.Scenario != nil {
		attributes["scenario"] = *req.Scenario
	}
	if req.MessageDraft != nil {
		attributes["message_draft"] = *req.MessageDraft
	}
	if req.LastOutcome != nil {
		attributes["last_outcome"] = *req.LastOutcome
	}
	if req.NextStep != nil {
		attributes["next_step"] = *req.NextStep
	}
	if req.Meeting != nil {
		attributes["meeting"] = *req.Meeting
	}
	if req.BusinessStage != nil {
		attributes["business_stage"] = *req.BusinessStage
	}

	// Add extra fields
	if isIncludeExtraFields {
		for key, value := range req.ExtraFields {
			if key != "id" {
				attributes[key] = value
			}
		}
	}

	return attributes
}

// convertMapJSONToJSONB marshals map[string]interface{} values into domain.JSONB for JSONB columns.
func convertMapJSONToJSONB(attrs map[string]interface{}) {
	for _, key := range []string{"confirmed_facts", "ai_insights", "insight_validation", "scores"} {
		if raw, ok := attrs[key]; ok {
			if m, okCast := raw.(map[string]interface{}); okCast {
				if jb, err := marshalJSONB(m); err == nil {
					attrs[key] = jb
				}
			}
		}
	}
}

// convertUpdateRequestToProfile merges the existing contact profile with the update request
func convertUpdateRequestToProfile(req v1schema.UpdateContactRequest, existingContact *domain.Contact) (domain.JSONB, error) {
	profile := make(map[string]interface{})
	newProfile := make(map[string]interface{})

	// Unmarshal existing profile if it exists
	if len(existingContact.Profile) > 0 {
		if err := json.Unmarshal(existingContact.Profile, &profile); err != nil {
			return nil, fmt.Errorf("failed to unmarshal existing profile: %w", err)
		}
	}

	// Convert the request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Unmarshal into a map
	if err := json.Unmarshal(data, &newProfile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	for key, value := range newProfile {
		profile[key] = value
	}

	// Include extra fields in custom_fields with AI-like format
	if len(req.ExtraFields) > 0 {
		// Get existing custom_fields or create new
		customFields := make(map[string]interface{})
		if existingCF, ok := profile["custom_fields"].(map[string]interface{}); ok {
			customFields = existingCF
		}

		// Add/update/delete extra fields
		for k, v := range req.ExtraFields {
			// Skip system fields
			if k == "id" {
				continue
			}

			// If value is empty string, delete the field
			if v == "" {
				delete(customFields, k)
				continue
			}

			// Otherwise update/add the field
			customFields[k] = map[string]interface{}{
				"value":      v,
				"updated_at": time.Now().UTC().Format(time.RFC3339),
			}
		}
		profile["custom_fields"] = customFields
	}

	// Keep what the caller sent under profile before the key itself is dropped.
	mergeProfileRequest(profile, req.Profile)

	// Remove fields that are not in req body
	delete(profile, "id")
	delete(profile, "tags")
	delete(profile, "profile")
	delete(profile, "organization_id")
	delete(profile, "do_not_contact")
	delete(profile, "confirmed_facts")
	delete(profile, "ai_insights")
	delete(profile, "insight_validation")
	delete(profile, "scores")

	// Convert all empty strings to null in the profile
	for key, value := range profile {
		if str, ok := value.(string); ok && str == "" {
			profile[key] = nil
		}
	}

	// Convert the map to JSONB
	var jsonb domain.JSONB
	if err := jsonb.Marshal(profile); err != nil {
		return nil, fmt.Errorf("failed to marshal profile to JSONB: %w", err)
	}

	return jsonb, nil
}
