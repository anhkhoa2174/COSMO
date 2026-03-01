package task

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	taskRepo "github.com/rockship/cosmo-agents-go/internal/repository/task"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// TaskHandler handles task-related HTTP requests
type TaskHandler struct {
	taskRepo *taskRepo.TaskRepository
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(taskRepo *taskRepo.TaskRepository) *TaskHandler {
	return &TaskHandler{
		taskRepo: taskRepo,
	}
}

// Create handles POST /v1/task
// @Summary Create a new task
// @Description Creates a new task for scheduling email sending or other campaign actions
// @Tags Tasks
// @Accept json
// @Produce json
// @Param task body v1schema.CreateTaskRequest true "Task creation data"
// @Success 201 {object} schema.APIResponse[v1schema.TaskResponse] "Successfully created task"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/task [post]
func (h *TaskHandler) Create(c fiber.Ctx) error {
	var req v1schema.CreateTaskRequest
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

	// Create task
	task := &domain.Task{
		ContactID:  req.ContactID,
		CampaignID: req.CampaignID,
		TemplateID: req.TemplateID,
		ScheduleAt: req.ScheduleAt,
		Status:     domain.TaskStatusPending,
	}

	// Set priority
	if req.Priority != "" {
		task.Priority = domain.TaskPriority(req.Priority)
	} else {
		task.Priority = domain.TaskPriorityMedium
	}

	// Set payload
	if req.Payload != nil {
		if err := task.SetPayload(req.Payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest, "Invalid payload format", err.Error(),
			))
		}
	}

	if _, err := h.taskRepo.Create(c.Context(), task); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to create task", err.Error(),
		))
	}

	response := toTaskResponse(task)
	return c.Status(fiber.StatusCreated).JSON(schema.SuccessResponse(response))
}

// GetByID handles GET /v1/task/:id
// @Summary Get task by ID
// @Description Retrieves a single task by its unique ID
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID (UUID)"
// @Success 200 {object} schema.APIResponse[v1schema.TaskResponse] "Successfully retrieved task"
// @Failure 400 {object} schema.APIResponse[any] "Invalid task ID"
// @Failure 404 {object} schema.APIResponse[any] "Task not found"
// @Security BearerAuth
// @Router /v1/task/{id} [get]
func (h *TaskHandler) GetByID(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid task ID format", err.Error(),
		))
	}

	task, err := h.taskRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Task not found", err.Error(),
		))
	}

	response := toTaskResponse(task)
	return c.JSON(schema.SuccessResponse(response))
}

// List handles GET /v1/task
// @Summary List tasks
// @Description Retrieves a paginated list of tasks with optional filtering
// @Tags Tasks
// @Accept json
// @Produce json
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit (max 100)" default(50)
// @Param campaign_id query string false "Filter by campaign ID (UUID)"
// @Param contact_id query string false "Filter by contact ID (UUID)"
// @Param status query string false "Filter by task status"
// @Success 200 {object} schema.APIResponse[v1schema.TaskListResponse] "Successfully retrieved tasks"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/task [get]
func (h *TaskHandler) List(c fiber.Ctx) error {
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

	// Build filters
	filter := baseRepo.Filter{}

	// Filter by campaign if provided
	if campaignID := c.Query("campaign_id"); campaignID != "" {
		if parsedCampID, parseErr := uuid.Parse(campaignID); parseErr == nil {
			filter["campaign_id"] = parsedCampID
		}
	}

	// Filter by contact if provided
	if contactID := c.Query("contact_id"); contactID != "" {
		if parsedContactID, parseErr := uuid.Parse(contactID); parseErr == nil {
			filter["contact_id"] = parsedContactID
		}
	}

	// Filter by status if provided
	if status := c.Query("status"); status != "" {
		filter["status"] = domain.TaskStatus(status)
	}

	// Get tasks
	pagination := &baseRepo.PaginationParams{
		Offset: offset,
		Limit:  limit,
	}

	result, err := h.taskRepo.FindAll(c.Context(), filter, pagination)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to fetch tasks", err.Error(),
		))
	}

	tasks := make([]*domain.Task, len(result.List))
	for i := range result.List {
		tasks[i] = &result.List[i]
	}

	items := make([]v1schema.TaskResponse, len(tasks))
	for i, task := range tasks {
		items[i] = toTaskResponse(task)
	}

	page := (offset / limit) + 1
	totalPages := int((result.Total + int64(limit) - 1) / int64(limit))

	response := v1schema.TaskListResponse{
		Items:      items,
		Total:      result.Total,
		Page:       page,
		PageSize:   limit,
		TotalPages: totalPages,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// Update handles PUT /v1/task/:id
// @Summary Update a task
// @Description Updates an existing task's status, priority, or other properties
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID (UUID)"
// @Param task body v1schema.UpdateTaskRequest true "Task update data"
// @Success 200 {object} schema.APIResponse[v1schema.TaskResponse] "Successfully updated task"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 404 {object} schema.APIResponse[any] "Task not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/task/{id} [put]
func (h *TaskHandler) Update(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid task ID format", err.Error(),
		))
	}

	var req v1schema.UpdateTaskRequest
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

	task, err := h.taskRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Task not found", err.Error(),
		))
	}

	// Apply updates
	if req.ScheduleAt != nil {
		task.ScheduleAt = req.ScheduleAt
	}
	if req.Priority != nil {
		task.Priority = domain.TaskPriority(*req.Priority)
	}
	if req.Status != nil {
		newStatus := domain.TaskStatus(*req.Status)
		if err := task.ValidateStatusTransition(newStatus); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest, "Invalid status transition", err.Error(),
			))
		}
		task.Status = newStatus

		// Set timestamps based on status
		now := time.Now()
		switch newStatus {
		case domain.TaskStatusRunning:
			task.TriggeredAt = &now
		case domain.TaskStatusDone:
			task.DoneAt = &now
		}
	}
	if req.Payload != nil {
		if err := task.SetPayload(req.Payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest, "Invalid payload format", err.Error(),
			))
		}
	}
	if req.Error != nil {
		task.Error = req.Error
	}

	if err := h.taskRepo.Update(c.Context(), task.ID, task); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to update task", err.Error(),
		))
	}

	response := toTaskResponse(task)
	return c.JSON(schema.SuccessResponse(response))
}

// Respond handles POST /v1/task/:id/respond
// @Summary Mark task as responded
// @Description Marks a task as responded with optional response data
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID (UUID)"
// @Param response body v1schema.TaskRespondRequest false "Response data (optional)"
// @Success 200 {object} schema.APIResponse[v1schema.TaskResponse] "Successfully marked task as responded"
// @Failure 400 {object} schema.APIResponse[any] "Invalid task ID"
// @Failure 404 {object} schema.APIResponse[any] "Task not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/task/{id}/respond [post]
func (h *TaskHandler) Respond(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid task ID format", err.Error(),
		))
	}

	var req v1schema.TaskRespondRequest
	if err := c.Bind().JSON(&req); err != nil {
		// Allow empty body
		req.ResponseData = make(map[string]any)
	}

	task, err := h.taskRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Task not found", err.Error(),
		))
	}

	// Mark as responded
	now := time.Now()
	task.RespondedAt = &now
	task.Status = domain.TaskStatusDone

	// Update payload with response data
	if len(req.ResponseData) > 0 {
		currentPayload, _ := task.GetPayload()
		if currentPayload == nil {
			currentPayload = make(map[string]any)
		}
		currentPayload["response"] = req.ResponseData
		task.SetPayload(currentPayload)
	}

	if err := h.taskRepo.Update(c.Context(), task.ID, task); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to mark task as responded", err.Error(),
		))
	}

	response := toTaskResponse(task)
	return c.JSON(schema.SuccessResponse(response))
}

// Resolve handles POST /v1/task/:id/resolve
// @Summary Resolve a task
// @Description Marks a task as done or failed with optional error information
// @Tags Tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID (UUID)"
// @Param resolution body v1schema.TaskResolveRequest true "Resolution data"
// @Success 200 {object} schema.APIResponse[v1schema.TaskResponse] "Successfully resolved task"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 404 {object} schema.APIResponse[any] "Task not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/task/{id}/resolve [post]
func (h *TaskHandler) Resolve(c fiber.Ctx) error {
	idParam := c.Params("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid task ID format", err.Error(),
		))
	}

	var req v1schema.TaskResolveRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	task, err := h.taskRepo.FindByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Task not found", err.Error(),
		))
	}

	// Resolve task
	now := time.Now()
	task.DoneAt = &now

	if req.Success {
		task.Status = domain.TaskStatusDone
	} else {
		task.Status = domain.TaskStatusFailed
		if req.Error != "" {
			errorStr := req.Error
			task.Error = &errorStr
		}
	}

	if err := h.taskRepo.Update(c.Context(), task.ID, task); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to resolve task", err.Error(),
		))
	}

	response := toTaskResponse(task)
	return c.JSON(schema.SuccessResponse(response))
}

// Helper function
func toTaskResponse(task *domain.Task) v1schema.TaskResponse {
	payload, _ := task.GetPayload()

	return v1schema.TaskResponse{
		ID:          task.ID,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
		ContactID:   task.ContactID,
		CampaignID:  task.CampaignID,
		TemplateID:  task.TemplateID,
		ScheduleAt:  task.ScheduleAt,
		RespondedAt: task.RespondedAt,
		TriggeredAt: task.TriggeredAt,
		DoneAt:      task.DoneAt,
		Priority:    string(task.Priority),
		Status:      string(task.Status),
		Payload:     payload,
		Error:       task.Error,
	}
}
