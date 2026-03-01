package template

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	knowledgeRepo "github.com/rockship/cosmo-agents-go/internal/repository/knowledge"
	template "github.com/rockship/cosmo-agents-go/internal/repository/template"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// TemplateHandler handles V2 template requests
type TemplateHandler struct {
	templateRepo  *template.TemplateRepository
	knowledgeRepo *knowledgeRepo.KnowledgeRepository
}

// NewTemplateHandler creates a new V2 TemplateHandler
func NewTemplateHandler(
	templateRepo *template.TemplateRepository,
	knowledgeRepo *knowledgeRepo.KnowledgeRepository,
) *TemplateHandler {
	return &TemplateHandler{
		templateRepo:  templateRepo,
		knowledgeRepo: knowledgeRepo,
	}
}

// Get retrieves a template by ID
// @Summary Get a template
// @Description Retrieves a specific template by ID
// @Tags Templates
// @Param template_id path string true "Template ID"
// @Success 200 {object} schema.APIResponse[v2schema.TemplateGetResponse] "Template retrieved successfully"
// @Failure 404 {object} schema.APIResponse[any] "Template not found"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Router /v2/templates/{id} [get]
func (h *TemplateHandler) Get(c fiber.Ctx) error {
	// Get template ID from params
	templateID, err := uuid.Parse(c.Params("template_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid template ID",
			err.Error(),
		))
	}

	// Get user ID from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"User ID not found in context",
		))
	}

	// Find template
	template, err := h.templateRepo.FindByIDAndUserID(c.Context(), templateID, userID)
	if err != nil || template == nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound,
			"Template not found",
			"Template does not exist or access denied",
		))
	}

	// Get associated knowledge if any
	var relatedKnowledge []*domain.Knowledge
	if err := h.loadRelatedKnowledge(c.Context(), template, &relatedKnowledge); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to load related knowledge")
		// Continue without related knowledge
	}

	// Convert to response format
	response := v2schema.TemplateGetResponse{
		ID:         template.ID,
		Type:       "email", // Default type since it's not in domain model
		Subject:    template.Subject,
		Content:    template.Content,
		SendAfter:  &template.SendAfter,
		Knowledges: []v2schema.TemplateKnowledgeEntity{}, // Convert relatedKnowledge if needed
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(response))
}

// Update updates a template
// @Summary Update a template
// @Description Updates template fields (subject, content, send_after). Only provided fields will be updated.
// @Tags Templates
// @Param template_id path string true "Template ID"
// @Param body body v2schema.TemplateUpdateRequest true "Fields to update"
// @Success 200 {object} schema.APIResponse[v2schema.TemplateUpdateResponse] "Template updated successfully"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 404 {object} schema.APIResponse[any] "Template not found"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Router /v2/templates/{id} [patch]
func (h *TemplateHandler) Update(c fiber.Ctx) error {
	// Get template ID from params
	templateID, err := uuid.Parse(c.Params("template_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid template ID",
			err.Error(),
		))
	}

	// Get user ID from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"User ID not found in context",
		))
	}

	// Parse request body
	var req v2schema.TemplateUpdateRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Failed to parse request",
			err.Error(),
		))
	}

	// Find existing template
	template, err := h.templateRepo.FindByIDAndUserID(c.Context(), templateID, userID)
	if err != nil || template == nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound,
			"Template not found",
			"Template does not exist or access denied",
		))
	}

	// Update only provided fields
	if req.Subject != nil {
		template.Subject = *req.Subject
	}
	if req.Content != nil {
		template.Content = *req.Content
	}
	if req.SendAfter != nil {
		template.SendAfter = *req.SendAfter
	}

	// Save updates
	err = h.templateRepo.Update(c.Context(), templateID, template)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to update template")
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to update template",
			err.Error(),
		))
	}

	response := v2schema.TemplateUpdateResponse{
		ID:      template.ID,
		Subject: &template.Subject,
		Content: &template.Content,
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(response))
}

// Delete deletes a template
// @Summary Delete a template
// @Description Permanently deletes a template and its associations
// @Tags Templates
// @Param template_id path string true "Template ID"
// @Success 200 {object} schema.APIResponse[any] "Template deleted successfully"
// @Failure 404 {object} schema.APIResponse[any] "Template not found"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Router /v2/templates/{id} [delete]
func (h *TemplateHandler) Delete(c fiber.Ctx) error {
	// Get template ID from params
	templateID, err := uuid.Parse(c.Params("template_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid template ID",
			err.Error(),
		))
	}

	// Get user ID from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"User ID not found in context",
		))
	}

	// Find template to ensure user has access
	template, err := h.templateRepo.FindByIDAndUserID(c.Context(), templateID, userID)
	if err != nil || template == nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound,
			"Template not found",
			"Template does not exist or access denied",
		))
	}

	// Delete template
	err = h.templateRepo.Delete(c.Context(), templateID)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to delete template")
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to delete template",
			err.Error(),
		))
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(fiber.Map{
		"message": "Template deleted successfully",
	}))
}

// AddKnowledge adds knowledge items to a template
// @Summary Add knowledge to template
// @Description Associates knowledge items with a template
// @Tags Templates
// @Param template_id path string true "Template ID"
// @Param body body v2schema.AddKnowledgeRequest true "Knowledge IDs to add"
// @Success 200 {object} schema.APIResponse[v2schema.AddKnowledgeResponse] "Knowledge added successfully"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 404 {object} schema.APIResponse[any] "Template not found"
// @Router /v2/templates/{template_id}/knowledges [post]
func (h *TemplateHandler) AddKnowledge(c fiber.Ctx) error {
	// Get template ID from params
	templateID, err := uuid.Parse(c.Params("template_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid template ID",
			err.Error(),
		))
	}

	// Get user ID from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"User ID not found in context",
		))
	}

	// Parse request body
	var req v2schema.AddKnowledgeRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Failed to parse request",
			err.Error(),
		))
	}

	if len(req.KnowledgeIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"No knowledge IDs provided",
			"At least one knowledge ID is required",
		))
	}

	// Convert string IDs to UUIDs
	knowledgeIDs := make([]uuid.UUID, len(req.KnowledgeIDs))
	for i, idStr := range req.KnowledgeIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest,
				"Invalid knowledge ID",
				fmt.Sprintf("Invalid UUID: %s", idStr),
			))
		}
		knowledgeIDs[i] = id
	}

	// Find template to ensure user has access
	template, err := h.templateRepo.FindByIDAndUserID(c.Context(), templateID, userID)
	if err != nil || template == nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound,
			"Template not found",
			"Template does not exist or access denied",
		))
	}

	// Find knowledge items
	knowledges, err := h.knowledgeRepo.FindByIDs(c.Context(), knowledgeIDs)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to find knowledge items")
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to find knowledge items",
			err.Error(),
		))
	}

	if len(knowledges) != len(knowledgeIDs) {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Some knowledge items not found",
			"One or more knowledge IDs are invalid",
		))
	}

	// Add knowledge to template
	for _, knowledge := range knowledges {
		err = h.templateRepo.AddKnowledge(c.Context(), templateID, knowledge)
		if err != nil {
			logger.Logger.Error().Err(err).Msg("Failed to add knowledge to template")
			return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
				fiber.StatusInternalServerError,
				"Failed to add knowledge to template",
				err.Error(),
			))
		}
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(v2schema.AddKnowledgeResponse{
		Message: fmt.Sprintf("Added %d knowledge items to template", len(knowledges)),
	}))
}

// Helper function to load related knowledge for a template
func (h *TemplateHandler) loadRelatedKnowledge(ctx context.Context, template *domain.Template, relatedKnowledge *[]*domain.Knowledge) error {
	// This would typically involve querying TemplateKnowledge relationships
	// For now, return nil as the relationship handling is outside scope
	return nil
}

// Helper function to handle multipart file uploads
func (h *TemplateHandler) handleFileUpload(c fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return err
	}

	// Process uploaded files
	for key := range form.File {
		file := form.File[key]
		for _, fileHeader := range file {
			// Process each file
			logger.Logger.Info().Msgf("Processing file: %s", fileHeader.Filename)

			// Here you would typically:
			// 1. Validate file
			// 2. Save to storage
			// 3. Update database
			// 4. Return file URL or ID
		}
	}

	return nil
}
