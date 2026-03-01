package playbook

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/handler"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	playbookService "github.com/rockship/cosmo-agents-go/internal/service/playbook"
)

// Handler handles playbook HTTP requests
type Handler struct {
	service        *playbookService.Service
	responseHelper *handler.ResponseHelper
}

// NewHandler creates a new playbook handler
func NewHandler(service *playbookService.Service) *Handler {
	return &Handler{
		service:        service,
		responseHelper: handler.NewResponseHelper(),
	}
}

// Create handles POST /v1/playbooks
func (h *Handler) Create(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse request
	var req v1schema.CreatePlaybookRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", nil)
	}

	// Call service
	playbook, err := h.service.CreatePlaybook(ctx, &req)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to create playbook", err)
	}

	return h.responseHelper.Created(c, playbook)
}

// List handles GET /v1/playbooks
func (h *Handler) List(c fiber.Ctx) error {
	ctx := c.Context()

	playbooks, err := h.service.ListPlaybooks(ctx)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to list playbooks", err)
	}

	return h.responseHelper.Success(c, playbooks)
}

// Get handles GET /v1/playbooks/:id
func (h *Handler) Get(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse playbook ID
	playbookID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid playbook ID", nil)
	}

	playbook, err := h.service.GetPlaybook(ctx, playbookID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to get playbook", err)
	}

	return h.responseHelper.Success(c, playbook)
}

// Delete handles DELETE /v1/playbooks/:id
func (h *Handler) Delete(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse playbook ID
	playbookID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid playbook ID", nil)
	}

	if err := h.service.DeletePlaybook(ctx, playbookID); err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to delete playbook", err)
	}

	return h.responseHelper.Success(c, fiber.Map{"message": "Playbook deleted successfully"})
}

// GenerateContent handles POST /v1/playbooks/:id/stages/:stage_id/generate-content
func (h *Handler) GenerateContent(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse playbook ID
	playbookID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid playbook ID", nil)
	}

	// Get stage ID from params
	stageID := c.Params("stage_id")
	if stageID == "" {
		return h.responseHelper.BadRequest(c, "Stage ID is required", nil)
	}

	// Parse request body
	var req v1schema.GenerateContentRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", nil)
	}

	// Call service
	content, err := h.service.GenerateContent(ctx, playbookID, req.ContactID, stageID, req.AIGenerationPrompt)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to generate content", err)
	}

	return h.responseHelper.Success(c, content)
}
