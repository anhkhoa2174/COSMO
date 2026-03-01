package template

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	template "github.com/rockship/cosmo-agents-go/internal/repository/template"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// TemplateHandler handles template-related HTTP requests
type TemplateHandler struct {
	templateRepo *template.TemplateRepository
}

// NewTemplateHandler creates a new TemplateHandler
func NewTemplateHandler(templateRepo *template.TemplateRepository) *TemplateHandler {
	return &TemplateHandler{
		templateRepo: templateRepo,
	}
}

// Create handles POST /v1/template
// @Summary Create a new template
// @Description Creates a new email template for use in campaigns
// @Tags Templates
// @Accept json
// @Produce json
// @Param template body v1schema.CreateTemplateRequest true "Template creation data"
// @Success 201 {object} schema.APIResponse[v1schema.TemplateResponse] "Successfully created template"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/template [post]
func (h *TemplateHandler) Create(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	var req v1schema.CreateTemplateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	// Create domain template
	template := &domain.Template{
		UserID:     userID,
		CampaignID: req.CampaignID,
		Type:       req.Type,
		Category:   domain.TemplateCategory(req.Category),
		Subject:    req.Subject,
		Content:    req.Content,
	}

	// Set position
	if req.Position != nil {
		template.Position = *req.Position
	} else {
		// Auto-assign next position (0 returns default position)
		template.Position = domain.CalculateNextPosition(0)
	}

	// Set send_after
	if req.SendAfter != nil {
		template.SendAfter = *req.SendAfter
	}

	// Set metadata
	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest, "Invalid metadata format", err.Error(),
			))
		}
		template.CMetadata = metadataJSON
	}

	if _, err := h.templateRepo.Create(c.Context(), template); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to create template", err.Error(),
		))
	}

	response := toTemplateResponse(template)
	return c.Status(fiber.StatusCreated).JSON(schema.SuccessResponse(response))
}

// GetByID handles GET /v1/template/:id
// @Summary Get template by ID
// @Description Retrieves a single email template by its unique ID
// @Tags Templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID (UUID)"
// @Success 200 {object} schema.APIResponse[v1schema.TemplateResponse] "Successfully retrieved template"
// @Failure 400 {object} schema.APIResponse[any] "Invalid template ID"
// @Failure 404 {object} schema.APIResponse[any] "Template not found"
// @Security BearerAuth
// @Router /v1/template/{id} [get]
func (h *TemplateHandler) GetByID(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid template ID format", err.Error(),
		))
	}

	template, err := h.templateRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Template not found", err.Error(),
		))
	}

	response := toTemplateResponse(template)
	return c.JSON(schema.SuccessResponse(response))
}

// List handles GET /v1/template
// @Summary List templates
// @Description Retrieves a paginated list of email templates for the authenticated user
// @Tags Templates
// @Accept json
// @Produce json
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit (max 100)" default(50)
// @Param campaign_id query string false "Filter by campaign ID (UUID)"
// @Success 200 {object} schema.APIResponse[v1schema.TemplateListResponse] "Successfully retrieved templates"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/template [get]
func (h *TemplateHandler) List(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	offset := 0
	if offsetParam := c.Query("offset"); offsetParam != "" {
		if parsed, err := strconv.Atoi(offsetParam); err == nil {
			offset = parsed
		}
	}

	limit := 50
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Filter by campaign if provided
	var templates []*domain.Template
	var total int64
	var err error

	if campaignID := c.Query("campaign_id"); campaignID != "" {
		if parsedCampaignID, parseErr := uuid.Parse(campaignID); parseErr == nil {
			templates, err = h.templateRepo.FindByCampaignID(c.Context(), parsedCampaignID)
			total = int64(len(templates))
			// Apply pagination manually
			if offset < len(templates) {
				end := offset + limit
				if end > len(templates) {
					end = len(templates)
				}
				templates = templates[offset:end]
			} else {
				templates = []*domain.Template{}
			}
		}
	} else {
		templates, total, err = h.templateRepo.FindByUserID(c.Context(), userID, offset, limit)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to fetch templates", err.Error(),
		))
	}

	items := make([]v1schema.TemplateResponse, len(templates))
	for i, template := range templates {
		items[i] = toTemplateResponse(template)
	}

	page := (offset / limit) + 1
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	response := v1schema.TemplateListResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   limit,
		TotalPages: totalPages,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// Update handles PUT /v1/template/:id
// @Summary Update a template
// @Description Updates an existing email template
// @Tags Templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID (UUID)"
// @Param template body v1schema.UpdateTemplateRequest true "Template update data"
// @Success 200 {object} schema.APIResponse[v1schema.TemplateResponse] "Successfully updated template"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Permission denied"
// @Failure 404 {object} schema.APIResponse[any] "Template not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/template/{id} [put]
func (h *TemplateHandler) Update(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid template ID format", err.Error(),
		))
	}

	var req v1schema.UpdateTemplateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	template, err := h.templateRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Template not found", err.Error(),
		))
	}

	if template.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "You don't have permission to update this template", "",
		))
	}

	// Apply updates
	if req.Type != nil {
		template.Type = *req.Type
	}
	if req.Category != nil {
		template.Category = domain.TemplateCategory(*req.Category)
	}
	if req.Subject != nil {
		template.Subject = *req.Subject
	}
	if req.Content != nil {
		template.Content = *req.Content
	}
	if req.Position != nil {
		template.Position = *req.Position
	}
	if req.SendAfter != nil {
		template.SendAfter = *req.SendAfter
	}
	if req.Metadata != nil {
		metadataJSON, err := json.Marshal(req.Metadata)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest, "Invalid metadata format", err.Error(),
			))
		}
		template.CMetadata = metadataJSON
	}

	if err := h.templateRepo.Update(c.Context(), template.ID, template); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to update template", err.Error(),
		))
	}

	response := toTemplateResponse(template)
	return c.JSON(schema.SuccessResponse(response))
}

// Delete handles DELETE /v1/template/:id
// @Summary Delete a template
// @Description Deletes an email template
// @Tags Templates
// @Accept json
// @Produce json
// @Param id path string true "Template ID (UUID)"
// @Success 204 "Successfully deleted template"
// @Failure 400 {object} schema.APIResponse[any] "Invalid template ID"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Permission denied"
// @Failure 404 {object} schema.APIResponse[any] "Template not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/template/{id} [delete]
func (h *TemplateHandler) Delete(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid template ID format", err.Error(),
		))
	}

	template, err := h.templateRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Template not found", err.Error(),
		))
	}

	if template.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "You don't have permission to delete this template", "",
		))
	}

	if err := h.templateRepo.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to delete template", err.Error(),
		))
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// Reorder handles POST /v1/template/reorder
// @Summary Reorder templates
// @Description Updates the position/order of multiple templates within a campaign
// @Tags Templates
// @Accept json
// @Produce json
// @Param reorder body v1schema.ReorderTemplatesRequest true "Template reordering data"
// @Success 200 {object} schema.APIResponse[map[string]string] "Successfully reordered templates"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Permission denied"
// @Failure 404 {object} schema.APIResponse[any] "Template not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/template/reorder [post]
func (h *TemplateHandler) Reorder(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	var req v1schema.ReorderTemplatesRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	// Verify ownership of all templates
	positions := make(map[uuid.UUID]float64)
	for _, tp := range req.TemplatePositions {
		template, err := h.templateRepo.FindByID(c.Context(), tp.TemplateID)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
				fiber.StatusNotFound, "Template not found: "+tp.TemplateID.String(), err.Error(),
			))
		}
		if template.UserID != userID {
			return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
				fiber.StatusForbidden, "You don't have permission to reorder these templates", "",
			))
		}
		positions[tp.TemplateID] = tp.Position
	}

	// Reorder
	if err := h.templateRepo.ReorderTemplates(c.Context(), positions); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to reorder templates", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(map[string]string{"message": "Templates reordered successfully"}))
}

// Helper function
func toTemplateResponse(template *domain.Template) v1schema.TemplateResponse {
	metadata := make(map[string]any)
	if len(template.CMetadata) > 0 {
		json.Unmarshal(template.CMetadata, &metadata)
	}

	return v1schema.TemplateResponse{
		ID:         template.ID,
		CreatedAt:  template.CreatedAt,
		UpdatedAt:  template.UpdatedAt,
		UserID:     template.UserID,
		CampaignID: template.CampaignID,
		Type:       template.Type,
		Category:   string(template.Category),
		Subject:    template.Subject,
		Content:    template.Content,
		Position:   template.Position,
		SendAfter:  template.SendAfter,
		Metadata:   metadata,
	}
}
