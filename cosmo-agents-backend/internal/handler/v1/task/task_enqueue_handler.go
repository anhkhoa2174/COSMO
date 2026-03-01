package task

import (
	"github.com/gofiber/fiber/v3"

	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	agent "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	campaignRepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	internalworker "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// TaskEnqueueHandler handles task enqueueing operations.
type TaskEnqueueHandler struct {
	workerClient *worker.Client
	campaignRepo *campaignRepo.CampaignRepository
	agentRepo    *agent.AgentRepository
}

// NewTaskEnqueueHandler creates a new task enqueue handler.
func NewTaskEnqueueHandler(
	workerClient *worker.Client,
	campaignRepo *campaignRepo.CampaignRepository,
	agentRepo *agent.AgentRepository,
) *TaskEnqueueHandler {
	return &TaskEnqueueHandler{
		workerClient: workerClient,
		campaignRepo: campaignRepo,
		agentRepo:    agentRepo,
	}
}

// EnqueueSendEmail enqueues an email sending task.
// @Summary Enqueue send email task
// @Description Enqueues a background job to send an email
// @Tags Background Jobs
// @Accept json
// @Produce json
// @Param task body v1schema.EnqueueSendEmailRequest true "Email sending task data"
// @Success 200 {object} schema.APIResponse[any] "Successfully enqueued email task"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/tasks/enqueue/send-email [post]
func (h *TaskEnqueueHandler) EnqueueSendEmail(c fiber.Ctx) error {
	var req v1schema.EnqueueSendEmailRequest

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if err := v1validation.ValidateStruct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	// Create payload
	payload := internalworker.SendEmailPayload{
		AgentID:     req.AgentID,
		ContactID:   req.ContactID,
		CampaignID:  req.CampaignID,
		TemplateID:  req.TemplateID,
		TaskID:      req.TaskID,
		To:          req.To,
		Cc:          req.Cc,
		Bcc:         req.Bcc,
		Subject:     req.Subject,
		Body:        req.Body,
		IsHTML:      req.IsHTML,
		InReplyTo:   req.InReplyTo,
		ScheduledAt: req.ScheduledAt,
	}

	// Enqueue task
	var taskInfo interface{}
	var err error

	if req.ScheduledAt != nil {
		// Schedule for specific time
		taskInfo, err = h.workerClient.EnqueueTaskAt(c.Context(), worker.TypeSendEmail, payload, *req.ScheduledAt)
	} else {
		// Enqueue immediately
		taskInfo, err = h.workerClient.EnqueueTask(c.Context(), worker.TypeSendEmail, payload)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to enqueue task", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"message":   "Email task enqueued successfully",
		"task_info": taskInfo,
	}))
}

// EnqueueExecuteCampaign enqueues a campaign execution task.
// @Summary Enqueue campaign execution task
// @Description Enqueues a background job to execute a campaign
// @Tags Background Jobs
// @Accept json
// @Produce json
// @Param task body v1schema.EnqueueExecuteCampaignRequest true "Campaign execution task data"
// @Success 200 {object} schema.APIResponse[any] "Successfully enqueued campaign execution"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 404 {object} schema.APIResponse[any] "Campaign or agent not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/tasks/enqueue/execute-campaign [post]
func (h *TaskEnqueueHandler) EnqueueExecuteCampaign(c fiber.Ctx) error {
	var req v1schema.EnqueueExecuteCampaignRequest

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if err := v1validation.ValidateStruct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	// Verify campaign exists
	campaign, err := h.campaignRepo.FindByID(c.Context(), req.CampaignID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Campaign not found", err.Error(),
		))
	}

	// Verify agent exists
	agent, err := h.agentRepo.FindByID(c.Context(), req.AgentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Agent not found", err.Error(),
		))
	}

	// Create payload
	payload := internalworker.ExecuteCampaignPayload{
		CampaignID: req.CampaignID,
		UserID:     campaign.UserID,
		AgentID:    req.AgentID,
		ContactIDs: req.ContactIDs,
	}

	// Enqueue as critical task
	taskInfo, err := h.workerClient.EnqueueCriticalTask(c.Context(), worker.TypeExecuteCampaign, payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to enqueue campaign", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"message":     "Campaign execution enqueued successfully",
		"campaign_id": campaign.ID,
		"agent_email": agent.Email,
		"task_info":   taskInfo,
	}))
}

// EnqueueScheduleTasks enqueues a task scheduling job.
// @Summary Enqueue task scheduling job
// @Description Enqueues a background job to schedule tasks for a campaign
// @Tags Background Jobs
// @Accept json
// @Produce json
// @Param task body v1schema.EnqueueScheduleTasksRequest true "Task scheduling job data"
// @Success 200 {object} schema.APIResponse[any] "Successfully enqueued schedule tasks job"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/tasks/enqueue/schedule-tasks [post]
func (h *TaskEnqueueHandler) EnqueueScheduleTasks(c fiber.Ctx) error {
	var req v1schema.EnqueueScheduleTasksRequest

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if err := v1validation.ValidateStruct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	// Create payload
	payload := internalworker.ScheduleTasksPayload{
		CampaignID: req.CampaignID,
		ContactIDs: req.ContactIDs,
		AgentID:    req.AgentID,
	}

	// Enqueue task
	taskInfo, err := h.workerClient.EnqueueTask(c.Context(), worker.TypeScheduleTasks, payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to enqueue schedule tasks", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"message":   "Schedule tasks job enqueued successfully",
		"task_info": taskInfo,
	}))
}

// EnqueueSyncAgent enqueues an agent sync task.
// @Summary Enqueue agent sync task
// @Description Enqueues a background job to sync an agent's emails from Gmail
// @Tags Background Jobs
// @Accept json
// @Produce json
// @Param task body v1schema.EnqueueSyncAgentRequest true "Agent sync task data"
// @Success 200 {object} schema.APIResponse[any] "Successfully enqueued agent sync"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 404 {object} schema.APIResponse[any] "Agent not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/tasks/enqueue/sync-agent [post]
func (h *TaskEnqueueHandler) EnqueueSyncAgent(c fiber.Ctx) error {
	var req v1schema.EnqueueSyncAgentRequest

	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid request body", err.Error(),
		))
	}

	if err := v1validation.ValidateStruct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Validation failed", err.Error(),
		))
	}

	// Verify agent exists
	agent, err := h.agentRepo.FindByID(c.Context(), req.AgentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Agent not found", err.Error(),
		))
	}

	// Create payload
	payload := internalworker.SyncAgentPayload{
		AgentID: req.AgentID,
		UserID:  agent.UserID,
	}

	// Enqueue as low priority task (background)
	taskInfo, err := h.workerClient.EnqueueLowPriorityTask(c.Context(), worker.TypeSyncAgent, payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to enqueue agent sync", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{
		"message":   "Agent sync enqueued successfully",
		"task_info": taskInfo,
	}))
}
