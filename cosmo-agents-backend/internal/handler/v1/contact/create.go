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
	"gorm.io/gorm"
)

// Create handles POST /v1/contacts
// @Summary Create contact
// @Description Creates a new contact
// @Tags Contacts
// @Accept json
// @Produce json
// @Param body body v1schema.CreateContactRequest true "Contact data"
// @Success 200 {object} schema.APIResponse[v1schema.ContactResponse] "Successfully created contact"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 422 {object} schema.APIResponse[any] "Validation error"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/contacts [post]
func (h *Handler) Create(c fiber.Ctx) error {
	// Use auth helper to get user and organization
	user, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return h.responseHelper.HandleAuthError(c, errors.New("You are not authorized to access this resource"))
		}
		return h.responseHelper.HandleAuthError(c, err)
	}

	// Parse request
	var req v1schema.CreateContactRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	// Fill missing required fields to avoid validation errors from extension/ingest
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "N/A"
	}
	// Email is now optional - no placeholder generated

	// Validate required fields
	if err := v1validation.ValidateStruct(req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(
			schema.ErrorResponse(fiber.StatusUnprocessableEntity, err.Error(), ""),
		)
	}

	// Skip validation for ExtraFields - they become custom_fields and can have any name/format

	// Determine contact_information for duplicate check
	// For LinkedIn source: use linkedin_url
	// For other sources: use email
	contactSource := req.Source
	if contactSource == "" {
		contactSource = string(domain.ContactSourceCosmoAgents)
	}

	contactInfoForCheck := ""
	if contactSource == string(domain.ContactSourceLinkedIn) {
		// LinkedIn source: use linkedin_url
		contactInfoForCheck = req.LinkedInURL
		if contactInfoForCheck == "" && req.Profile != nil {
			if url, ok := req.Profile["linkedin_url"].(string); ok {
				contactInfoForCheck = url
			}
		}
	} else {
		// Other sources: use email (skip placeholder emails)
		if req.Email != "" && !strings.HasPrefix(req.Email, "unknown-") && req.Email != "N/A" {
			contactInfoForCheck = req.Email
		}
	}

	// Check for duplicate by contact_information
	var existingContact *domain.Contact
	if contactInfoForCheck != "" {
		existingContact, err = h.repo.FindByContactInformation(c.Context(), contactInfoForCheck, organizationID)
		if err != nil {
			return h.responseHelper.InternalServerError(c, "Failed to check duplicate contact", err)
		}
	}

	// If duplicate found, handle based on who added it
	if existingContact != nil {
		// Same user adding again → auto merge
		if existingContact.UserID == user.ID {
			return h.mergeAndUpdateContact(c, existingContact, req, user.ID, organizationID)
		}

		// Different user → return conflict
		addedByName := "another team member"
		if existingContact.UserID != uuid.Nil {
			addedByUser, err := h.userRepo.FindByID(c.Context(), existingContact.UserID)
			if err == nil && addedByUser != nil && addedByUser.Name != "" {
				addedByName = addedByUser.Name
			}
		}

		return c.Status(fiber.StatusConflict).JSON(schema.ErrorResponse(
			fiber.StatusConflict,
			fmt.Sprintf("Contact with this email or LinkedIn already exists. Added by %s.", addedByName),
			map[string]interface{}{
				"existing_contact_id": existingContact.ID,
				"added_by_name":       addedByName,
				"added_by_id":         existingContact.UserID,
			},
		))
	}

	// Create new contact (default behavior or keep_both)
	profile, err := convertRequestToProfile(req)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to process contact data", err)
	}

	// Store email in profile if provided (email field removed from Contact model)
	if req.Email != "" && req.Email != "N/A" {
		var profileMap map[string]interface{}
		if err := profile.Unmarshal(&profileMap); err != nil || profileMap == nil {
			profileMap = make(map[string]interface{})
		}
		profileMap["email"] = req.Email
		if newProfile, err := marshalJSONB(profileMap); err == nil {
			profile = newProfile
		}
	}

	// Store phone in profile if provided (phone field removed from Contact model)
	if req.Phone != "" && req.Phone != "N/A" {
		var profileMap map[string]interface{}
		if err := profile.Unmarshal(&profileMap); err != nil || profileMap == nil {
			profileMap = make(map[string]interface{})
		}
		profileMap["phone"] = req.Phone
		if newProfile, err := marshalJSONB(profileMap); err == nil {
			profile = newProfile
		}
	}

	// Store linkedin_url in profile if provided
	if req.LinkedInURL != "" {
		var profileMap map[string]interface{}
		if err := profile.Unmarshal(&profileMap); err != nil || profileMap == nil {
			profileMap = make(map[string]interface{})
		}
		profileMap["linkedin_url"] = req.LinkedInURL
		if newProfile, err := marshalJSONB(profileMap); err == nil {
			profile = newProfile
		}
	}

	// Determine contact_information based on source (reuse contactSource from duplicate check)
	// For LinkedIn: use linkedin_url, for others: use email
	contactInformation := ""
	if contactSource == string(domain.ContactSourceLinkedIn) {
		contactInformation = req.LinkedInURL
	} else if req.Email != "" && req.Email != "N/A" && !strings.HasPrefix(req.Email, "unknown-") {
		contactInformation = req.Email
	}

	contact := &domain.Contact{
		UserID:             user.ID,
		OrganizationID:     &organizationID,
		Name:               req.Name,
		Company:            req.Company,
		JobTitle:           req.JobTitle,
		Address:            req.Address,
		City:               req.City,
		Country:            req.Country,
		State:              req.State,
		Zip:                req.Zip,
		Profile:            profile,
		Source:             contactSource,
		SourceID:           uuid.New().String(),
		ContactInformation: contactInformation,
		// Outreach context fields
		Industry:         req.Industry,
		ContactChannel:   req.ContactChannel,
		LifecycleStage:   req.LifecycleStage,
		ContextLevel:     req.ContextLevel,
		OutreachDecision: req.OutreachDecision,
		Scenario:         req.Scenario,
		MessageDraft:     req.MessageDraft,
		LastOutcome:      req.LastOutcome,
		NextStep:         req.NextStep,
		Meeting:          req.Meeting,
		BusinessStage:    req.BusinessStage,
	}

	// Optional new fields
	if len(req.ConfirmedFacts) > 0 {
		if contact.ConfirmedFacts, err = marshalJSONB(req.ConfirmedFacts); err != nil {
			return h.responseHelper.InternalServerError(c, "Failed to process confirmed_facts", err)
		}
	}
	if len(req.AIInsights) > 0 {
		if contact.AIInsights, err = marshalJSONB(req.AIInsights); err != nil {
			return h.responseHelper.InternalServerError(c, "Failed to process ai_insights", err)
		}
	}
	if len(req.InsightValidation) > 0 {
		if contact.InsightValidation, err = marshalJSONB(req.InsightValidation); err != nil {
			return h.responseHelper.InternalServerError(c, "Failed to process insight_validation", err)
		}
	}
	if len(req.Scores) > 0 {
		if contact.Scores, err = marshalJSONB(req.Scores); err != nil {
			return h.responseHelper.InternalServerError(c, "Failed to process scores", err)
		}
	}

	err = h.repo.Create(c.Context(), contact)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to create contact", err)
	}

	response := v1schema.ToContactResponse(contact)

	// Best-effort auto-embedding after creation (runs async)
	if h.intelSvc != nil {
		go func() {
			if err := h.intelSvc.EmbedContact(context.Background(), user.ID, contact.ID); err != nil {
				// log silently; do not block request
				_ = err
			}
		}()

		// Auto-calculate segment fit scores after contact creation
		// This ensures new contacts are automatically assigned to appropriate segments
		go func() {
			ctx := context.Background()
			// Wait a bit for embedding to complete first
			time.Sleep(2 * time.Second)

			// Calculate scores for all active segments
			_, err := h.intelSvc.CalculateSegmentScores(ctx, user.ID, organizationID, contact.ID, nil)
			if err != nil {
				// log silently; do not block request
				_ = err
			}
		}()
	}

	if h.workerClient != nil {
		payload := map[string]interface{}{
			"contact_id": contact.ID,
			"user_id":    user.ID,
			"org_id":     organizationID,
			"event":      "contact_created",
		}
		_, _ = h.workerClient.EnqueueTask(context.Background(), worker.TypeOrchestrateContact, payload)
	}

	return h.responseHelper.Success(c, response)
}

// convertRequestToProfile converts the request to a domain.JSONB type
func convertRequestToProfile(req v1schema.CreateContactRequest) (domain.JSONB, error) {
	// Create a map to hold the profile data
	profile := make(map[string]interface{})

	// Convert the request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Unmarshal into a map
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	// Include extra fields in custom_fields with AI-like format
	if len(req.ExtraFields) > 0 {
		customFields := make(map[string]interface{})
		for k, v := range req.ExtraFields {
			// Skip system fields
			if k == "id" {
				continue
			}
			customFields[k] = map[string]interface{}{
				"value":      v,
				"updated_at": time.Now().UTC().Format(time.RFC3339),
			}
		}
		profile["custom_fields"] = customFields
	}

	// Remove fields that are not in req body
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

// marshalJSONB marshals a map into domain.JSONB
func marshalJSONB(data map[string]interface{}) (domain.JSONB, error) {
	var jsonb domain.JSONB
	if len(data) == 0 {
		return jsonb, nil
	}
	if err := jsonb.Marshal(data); err != nil {
		return nil, err
	}
	return jsonb, nil
}

// mergeAndUpdateContact merges new data into existing contact and updates it
func (h *Handler) mergeAndUpdateContact(c fiber.Ctx, existing *domain.Contact, req v1schema.CreateContactRequest, userID, organizationID uuid.UUID) error {
	// Build update attributes - only update non-empty fields
	updateAttrs := make(map[string]interface{})

	if req.Name != "" && req.Name != "N/A" {
		updateAttrs["name"] = req.Name
	}
	if req.Phone != "" {
		updateAttrs["phone"] = req.Phone
	}
	if req.Company != "" {
		updateAttrs["company"] = req.Company
	}
	if req.JobTitle != "" {
		updateAttrs["job_title"] = req.JobTitle
	}
	if req.Address != "" {
		updateAttrs["address"] = req.Address
	}
	if req.City != "" {
		updateAttrs["city"] = req.City
	}
	if req.Country != "" {
		updateAttrs["country"] = req.Country
	}
	if req.State != "" {
		updateAttrs["state"] = req.State
	}
	if req.Zip != "" {
		updateAttrs["zip"] = req.Zip
	}

	// Merge profile data
	existingProfile := make(map[string]interface{})
	if len(existing.Profile) > 0 {
		if err := json.Unmarshal(existing.Profile, &existingProfile); err == nil {
			// Merge new profile fields
			if req.Profile != nil {
				for k, v := range req.Profile {
					existingProfile[k] = v
				}
			}
			// Merge extra fields into custom_fields
			if len(req.ExtraFields) > 0 {
				customFields, ok := existingProfile["custom_fields"].(map[string]interface{})
				if !ok {
					customFields = make(map[string]interface{})
				}
				for k, v := range req.ExtraFields {
					if k == "id" {
						continue
					}
					customFields[k] = map[string]interface{}{
						"value":      v,
						"updated_at": time.Now().UTC().Format(time.RFC3339),
					}
				}
				existingProfile["custom_fields"] = customFields
			}

			var profileJSONB domain.JSONB
			if err := profileJSONB.Marshal(existingProfile); err == nil {
				updateAttrs["profile"] = profileJSONB
			}
		}
	}

	// Apply merged values to existing contact for status calculation
	if v, ok := updateAttrs["name"].(string); ok {
		existing.Name = v
	}
	if v, ok := updateAttrs["company"].(string); ok {
		existing.Company = v
	}
	if v, ok := updateAttrs["job_title"].(string); ok {
		existing.JobTitle = v
	}

	// Recalculate status after merge
	existing.CalculateStatus()
	updateAttrs["status"] = existing.Status
	updateAttrs["missing_fields"] = existing.MissingFields

	// Only update if there are changes
	if len(updateAttrs) > 0 {
		updated, err := h.repo.UpdateFields(c.Context(), existing.ID, updateAttrs, userID, organizationID)
		if err != nil {
			return h.responseHelper.InternalServerError(c, "Failed to merge contact", err)
		}
		existing = updated
	}

	response := v1schema.ToContactResponse(existing)
	return h.responseHelper.Success(c, response)
}
