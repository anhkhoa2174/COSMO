package email

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	taskRepo "github.com/rockship/cosmo-agents-go/internal/repository/task"
	"github.com/rockship/cosmo-agents-go/internal/schema"
)

// SimpleEmailHandler provides basic email functionality without complex relations
// This is a temporary fix while the full relations package is being integrated
type SimpleEmailHandler struct {
	taskRepo *taskRepo.TaskRepository
	roleRepo *roleRepo.RoleRepository
}

// NewSimpleEmailHandler creates a new simple email handler
func NewSimpleEmailHandler(taskRepo *taskRepo.TaskRepository, roleRepo *roleRepo.RoleRepository) *SimpleEmailHandler {
	return &SimpleEmailHandler{
		taskRepo: taskRepo,
		roleRepo: roleRepo,
	}
}

// GetEmail handles GET /v1/emails/{id}
// @Summary Get email by ID
// @Description Get information about a specific email
// @Tags Emails
// @Accept json
// @Produce json
// @Param id path string true "Email ID"
// @Success 200 {object} schema.APIResponse[any] "Email information"
// @Failure 404 {object} schema.APIResponse[any] "Email not found"
// @Router /v1/emails/{id} [get]
func (h *SimpleEmailHandler) GetEmail(c fiber.Ctx) error {
	emailID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid email ID format",
			"ID must be a valid UUID",
		))
	}

	task, err := h.taskRepo.FindByID(c.Context(), emailID)
	if err != nil || task == nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound,
			"Email not found",
			"",
		))
	}

	// Simple response without complex relations
	response := map[string]interface{}{
		"id":         task.ID,
		"status":     task.Status,
		"created_at": task.CreatedAt,
		"updated_at": task.UpdatedAt,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// ListEmails handles GET /v1/emails
// @Summary List emails
// @Description Get a list of emails with pagination
// @Tags Emails
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} schema.APIResponse[any] "List of emails"
// @Router /v1/emails [get]
func (h *SimpleEmailHandler) ListEmails(c fiber.Ctx) error {
	// Simple implementation without complex filtering
	return c.JSON(schema.SuccessResponse(map[string]interface{}{
		"emails": []interface{}{},
		"total":  0,
		"page":   1,
		"limit":  20,
	}))
}
