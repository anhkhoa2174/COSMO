package template

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	draftTemplateRepo "github.com/rockship/cosmo-agents-go/internal/repository/draft_template"
)

// DraftTemplateHandler handles V2 draft template-related requests
type DraftTemplateHandler struct {
	draftTemplateRepo *draftTemplateRepo.DraftTemplateRepository
}

// NewDraftTemplateHandler creates a new V2 DraftTemplateHandler
func NewDraftTemplateHandler(draftTemplateRepo *draftTemplateRepo.DraftTemplateRepository) *DraftTemplateHandler {
	return &DraftTemplateHandler{
		draftTemplateRepo: draftTemplateRepo,
	}
}

// TemplateDetail represents the template details in the response
type TemplateDetail struct {
	Type    string `json:"type"`
	Subject string `json:"subject"`
	Content string `json:"content"`
}

// DraftTemplateResponse represents the draft template response structure
type DraftTemplateResponse struct {
	ID         uuid.UUID      `json:"id"`
	Intent     string         `json:"intent"`
	CampaignID uuid.UUID      `json:"campaign_id"`
	TemplateID uuid.UUID      `json:"template_id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	Template   TemplateDetail `json:"template"`
}

// GetDraftTemplate retrieves a draft template by ID
// GET /v2/draft-templates/{draft_template_id}
func (h *DraftTemplateHandler) GetDraftTemplate(c fiber.Ctx) error {
	// Parse draft template ID from URL parameter
	draftTemplateIDStr := c.Params("draft_template_id")
	draftTemplateID, err := uuid.Parse(draftTemplateIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid draft template ID",
		})
	}

	// Fetch draft template with associated template from database
	draftTemplate, err := h.draftTemplateRepo.FindByIDWithTemplate(c.Context(), draftTemplateID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Draft Template not found",
		})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Failed to fetch draft template",
		})
	}
	if draftTemplate == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Draft Template not found",
		})
	}

	if draftTemplate.Template == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Draft Template is missing template data",
		})
	}

	// Build response with template details
	response := DraftTemplateResponse{
		ID:         draftTemplate.ID,
		Intent:     draftTemplate.Intent,
		CampaignID: draftTemplate.CampaignID,
		TemplateID: draftTemplate.TemplateID,
		CreatedAt:  draftTemplate.CreatedAt.UTC(),
		UpdatedAt:  draftTemplate.UpdatedAt.UTC(),
		Template: TemplateDetail{
			Type:    draftTemplate.Template.Type,
			Subject: draftTemplate.Template.Subject,
			Content: draftTemplate.Template.Content,
		},
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "success",
		"data":   response,
	})
}
