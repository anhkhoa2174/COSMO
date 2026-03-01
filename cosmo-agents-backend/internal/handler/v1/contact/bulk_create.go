package contact

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
	"gorm.io/gorm"
)

// BulkCreateRequest represents the bulk contact creation request
type BulkCreateRequest struct {
	Contacts []BulkContactItem `json:"contacts" validate:"required,min=1,max=500"`
}

// BulkContactItem represents a single contact in bulk create
type BulkContactItem struct {
	Name        string `json:"name"`
	FullName    string `json:"full_name"` // Alias for name
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Company     string `json:"company"`
	JobTitle    string `json:"job_title"`
	LinkedInURL string `json:"linkedin_url"`
	Source      string `json:"source"`
}

// BulkCreateResponse represents the bulk creation result
type BulkCreateResponse struct {
	Created    int      `json:"created"`
	Skipped    int      `json:"skipped"`
	Errors     []string `json:"errors,omitempty"`
	ContactIDs []string `json:"contact_ids,omitempty"`
}

// BulkCreate handles POST /v1/contacts/bulk
// @Summary Bulk create contacts
// @Description Creates multiple contacts at once (max 500)
// @Tags Contacts
// @Accept json
// @Produce json
// @Param body body BulkCreateRequest true "Contacts data"
// @Success 200 {object} schema.APIResponse[BulkCreateResponse] "Bulk creation result"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/contacts/bulk [post]
func (h *Handler) BulkCreate(c fiber.Ctx) error {
	// Get user and organization
	user, organizationID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return h.responseHelper.HandleAuthError(c, errors.New("You are not authorized to access this resource"))
		}
		return h.responseHelper.HandleAuthError(c, err)
	}

	// Parse request
	var req BulkCreateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	if len(req.Contacts) == 0 {
		return h.responseHelper.BadRequest(c, "No contacts provided", nil)
	}

	if len(req.Contacts) > 500 {
		return h.responseHelper.BadRequest(c, "Maximum 500 contacts per request", nil)
	}

	response := BulkCreateResponse{
		Created:    0,
		Skipped:    0,
		Errors:     []string{},
		ContactIDs: []string{},
	}

	for i, item := range req.Contacts {
		// Use name field, fallback to full_name for backwards compatibility
		name := item.Name
		if name == "" && item.FullName != "" {
			name = item.FullName
		}

		// Skip if no identifying info
		if name == "" && item.Email == "" && item.LinkedInURL == "" {
			response.Skipped++
			continue
		}

		// Check for duplicate by email or LinkedIn URL
		if item.Email != "" {
			existing, _ := h.repo.GetByEmail(c.Context(), item.Email, user.ID)
			if existing != nil {
				response.Skipped++
				continue
			}
		}

		if item.LinkedInURL != "" {
			existing, _ := h.repo.GetByLinkedInURL(c.Context(), item.LinkedInURL, user.ID)
			if existing != nil {
				response.Skipped++
				continue
			}
		}

		// Build profile with linkedin_url, email, and phone
		profile := make(map[string]interface{})
		if item.LinkedInURL != "" {
			profile["linkedin_url"] = item.LinkedInURL
		}
		// Store email and phone in profile
		if item.Email != "" {
			profile["email"] = item.Email
		}
		if item.Phone != "" {
			profile["phone"] = item.Phone
		}
		profile["imported_at"] = time.Now().UTC().Format(time.RFC3339)

		var profileJSONB domain.JSONB
		if len(profile) > 0 {
			_ = profileJSONB.Marshal(profile)
		}

		// Determine source - use request source or default to "cosmo-agents"
		contactSource := item.Source
		if contactSource == "" {
			contactSource = string(domain.ContactSourceCosmoAgents)
		}

		// Generate SourceID - use LinkedIn URL if available, otherwise generate UUID
		sourceID := item.LinkedInURL
		if sourceID == "" {
			sourceID = uuid.New().String()
		}

		// Determine contact_information based on source
		// For LinkedIn: use linkedin_url, for others: use email
		contactInformation := ""
		if contactSource == string(domain.ContactSourceLinkedIn) {
			contactInformation = item.LinkedInURL
		} else if item.Email != "" && !strings.HasPrefix(item.Email, "unknown-") {
			contactInformation = item.Email
		}

		// Create contact
		contact := &domain.Contact{
			UserID:             user.ID,
			OrganizationID:     &organizationID,
			Name:               name,
			Company:            item.Company,
			JobTitle:           item.JobTitle,
			Profile:            profileJSONB,
			Source:             contactSource,
			SourceID:           sourceID,
			ContactInformation: contactInformation,
		}

		err := h.repo.Create(c.Context(), contact)
		if err != nil {
			response.Errors = append(response.Errors,
				"Contact "+string(rune(i+1))+": "+err.Error())
			continue
		}

		response.Created++
		response.ContactIDs = append(response.ContactIDs, contact.ID.String())

		// Queue background tasks
		if h.workerClient != nil {
			payload := map[string]interface{}{
				"contact_id": contact.ID,
				"user_id":    user.ID,
				"org_id":     organizationID,
				"event":      "contact_created",
			}
			_, _ = h.workerClient.EnqueueTask(context.Background(), worker.TypeOrchestrateContact, payload)
		}
	}

	return c.JSON(schema.SuccessResponse(response))
}

// parseFullName splits a full name into first and last name
func parseFullName(fullName string) (firstName, lastName string) {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return "", ""
	}

	parts := strings.Fields(fullName)
	if len(parts) == 1 {
		return parts[0], ""
	}

	firstName = parts[0]
	lastName = strings.Join(parts[1:], " ")
	return firstName, lastName
}
