package playbook

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/handler"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	playbookService "github.com/rockship/cosmo-agents-go/internal/service/playbook"
)

// AutomationHandler handles automation rule HTTP requests
type AutomationHandler struct {
	service        *playbookService.AutomationService
	responseHelper *handler.ResponseHelper
}

// NewAutomationHandler creates a new automation handler
func NewAutomationHandler(service *playbookService.AutomationService) *AutomationHandler {
	return &AutomationHandler{
		service:        service,
		responseHelper: handler.NewResponseHelper(),
	}
}

// Create handles POST /v1/automation-rules
func (h *AutomationHandler) Create(c fiber.Ctx) error {
	ctx := c.Context()

	var req v1schema.AutomationRuleRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", nil)
	}

	rule, err := h.service.CreateAutomationRule(ctx, &req)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to create automation rule", err)
	}

	return h.responseHelper.Created(c, rule)
}

// List handles GET /v1/automation-rules
func (h *AutomationHandler) List(c fiber.Ctx) error {
	ctx := c.Context()

	rules, err := h.service.ListAutomationRules(ctx)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to list automation rules", err)
	}

	return h.responseHelper.Success(c, rules)
}

// Get handles GET /v1/automation-rules/:id
func (h *AutomationHandler) Get(c fiber.Ctx) error {
	ctx := c.Context()

	ruleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid automation rule ID", nil)
	}

	rule, err := h.service.GetAutomationRule(ctx, ruleID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to get automation rule", err)
	}

	return h.responseHelper.Success(c, rule)
}

// Toggle handles PATCH /v1/automation-rules/:id/toggle
func (h *AutomationHandler) Toggle(c fiber.Ctx) error {
	ctx := c.Context()

	ruleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid automation rule ID", nil)
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", nil)
	}

	if err := h.service.ToggleAutomationRule(ctx, ruleID, req.IsActive); err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to toggle automation rule", err)
	}

	return h.responseHelper.Success(c, fiber.Map{"message": "Automation rule toggled successfully"})
}

// Delete handles DELETE /v1/automation-rules/:id
func (h *AutomationHandler) Delete(c fiber.Ctx) error {
	ctx := c.Context()

	ruleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid automation rule ID", nil)
	}

	if err := h.service.DeleteAutomationRule(ctx, ruleID); err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to delete automation rule", err)
	}

	return h.responseHelper.Success(c, fiber.Map{"message": "Automation rule deleted successfully"})
}
