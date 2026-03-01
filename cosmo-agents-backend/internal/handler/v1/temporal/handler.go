package temporal

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/handler"
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	_ "github.com/rockship/cosmo-agents-go/internal/schema" // imported for swagger
	"github.com/rockship/cosmo-agents-go/internal/temporal/workflows"
	"go.temporal.io/sdk/client"
)

// Handler handles Temporal workflow-related HTTP requests
type Handler struct {
	temporalClient client.Client
	authHelper     *middleware.AuthHelper
	responseHelper *handler.ResponseHelper
}

// New creates a new temporal handler
func New(
	temporalClient client.Client,
	userRepo *userRepo.UserRepository,
	roleRepo *roleRepo.RoleRepository,
) *Handler {
	return &Handler{
		temporalClient: temporalClient,
		authHelper:     middleware.NewAuthHelper(userRepo, roleRepo),
		responseHelper: handler.NewResponseHelper(),
	}
}

// ============ Request/Response types ============

type StartFullAnalysisRequest struct {
	ContactID string `json:"contact_id"`
	SessionID string `json:"session_id,omitempty"`
}

type StartBatchEnrichmentRequest struct {
	ContactIDs   []string `json:"contact_ids"`
	ForceRefresh bool     `json:"force_refresh"`
}

type StartSegmentAnalysisRequest struct {
	SegmentID string `json:"segment_id"`
}

type StartDailyAnalyticsRequest struct {
	Date string `json:"date"` // YYYY-MM-DD
}

type WorkflowStartedResponse struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
	Status     string `json:"status"`
}

type WorkflowStatusResponse struct {
	WorkflowID string      `json:"workflow_id"`
	RunID      string      `json:"run_id"`
	Status     string      `json:"status"`
	Result     interface{} `json:"result,omitempty"`
	Error      string      `json:"error,omitempty"`
}

// ============ Handlers ============

// StartFullAnalysis starts a full contact analysis workflow
// @Summary Start full contact analysis workflow
// @Description Runs a complete analysis on a contact including enrichment, scoring, and email generation
// @Tags Temporal Workflows
// @Accept json
// @Produce json
// @Param body body StartFullAnalysisRequest true "Analysis request"
// @Success 200 {object} schema.APIResponse[WorkflowStartedResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/workflows/full-analysis [post]
func (h *Handler) StartFullAnalysis(c fiber.Ctx) error {
	if h.temporalClient == nil {
		return h.responseHelper.InternalServerError(c, "Temporal not configured", nil)
	}

	user, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req StartFullAnalysisRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "invalid request body", err)
	}

	if req.ContactID == "" {
		return h.responseHelper.BadRequest(c, "contact_id is required", nil)
	}

	// Generate session ID if not provided
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	input := workflows.FullAnalysisInput{
		ContactID: req.ContactID,
		UserID:    user.ID,
		OrgID:     orgID,
		SessionID: sessionID,
	}

	options := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("full-analysis-%s-%s", req.ContactID, sessionID),
		TaskQueue: workflows.AgentTaskQueue,
	}

	run, err := h.temporalClient.ExecuteWorkflow(c.Context(), options, workflows.FullContactAnalysisWorkflow, input)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to start workflow", err)
	}

	return h.responseHelper.Success(c, WorkflowStartedResponse{
		WorkflowID: run.GetID(),
		RunID:      run.GetRunID(),
		Status:     "started",
	})
}

// StartBatchEnrichment starts a batch enrichment workflow
// @Summary Start batch enrichment workflow
// @Description Enriches multiple contacts in parallel
// @Tags Temporal Workflows
// @Accept json
// @Produce json
// @Param body body StartBatchEnrichmentRequest true "Batch enrichment request"
// @Success 200 {object} schema.APIResponse[WorkflowStartedResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/workflows/batch-enrichment [post]
func (h *Handler) StartBatchEnrichment(c fiber.Ctx) error {
	if h.temporalClient == nil {
		return h.responseHelper.InternalServerError(c, "Temporal not configured", nil)
	}

	user, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req StartBatchEnrichmentRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "invalid request body", err)
	}

	if len(req.ContactIDs) == 0 {
		return h.responseHelper.BadRequest(c, "contact_ids is required", nil)
	}

	input := workflows.BatchEnrichmentInput{
		ContactIDs:   req.ContactIDs,
		ForceRefresh: req.ForceRefresh,
		UserID:       user.ID,
		OrgID:        orgID,
	}

	options := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("batch-enrichment-%s-%d", orgID.String(), len(req.ContactIDs)),
		TaskQueue: workflows.AgentTaskQueue,
	}

	run, err := h.temporalClient.ExecuteWorkflow(c.Context(), options, workflows.BatchEnrichmentWorkflow, input)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to start workflow", err)
	}

	return h.responseHelper.Success(c, WorkflowStartedResponse{
		WorkflowID: run.GetID(),
		RunID:      run.GetRunID(),
		Status:     "started",
	})
}

// StartSegmentAnalysis starts a segment analysis workflow
// @Summary Start segment analysis workflow
// @Description Analyzes a segment's health and performance
// @Tags Temporal Workflows
// @Accept json
// @Produce json
// @Param body body StartSegmentAnalysisRequest true "Segment analysis request"
// @Success 200 {object} schema.APIResponse[WorkflowStartedResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/workflows/segment-analysis [post]
func (h *Handler) StartSegmentAnalysis(c fiber.Ctx) error {
	if h.temporalClient == nil {
		return h.responseHelper.InternalServerError(c, "Temporal not configured", nil)
	}

	user, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req StartSegmentAnalysisRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "invalid request body", err)
	}

	if req.SegmentID == "" {
		return h.responseHelper.BadRequest(c, "segment_id is required", nil)
	}

	input := workflows.SegmentAnalysisInput{
		SegmentID: req.SegmentID,
		UserID:    user.ID,
		OrgID:     orgID,
	}

	options := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("segment-analysis-%s", req.SegmentID),
		TaskQueue: workflows.AgentTaskQueue,
	}

	run, err := h.temporalClient.ExecuteWorkflow(c.Context(), options, workflows.SegmentAnalysisWorkflow, input)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to start workflow", err)
	}

	return h.responseHelper.Success(c, WorkflowStartedResponse{
		WorkflowID: run.GetID(),
		RunID:      run.GetRunID(),
		Status:     "started",
	})
}

// StartDailyAnalytics starts a daily analytics workflow
// @Summary Start daily analytics workflow
// @Description Generates daily analytics report
// @Tags Temporal Workflows
// @Accept json
// @Produce json
// @Param body body StartDailyAnalyticsRequest true "Daily analytics request"
// @Success 200 {object} schema.APIResponse[WorkflowStartedResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/workflows/daily-analytics [post]
func (h *Handler) StartDailyAnalytics(c fiber.Ctx) error {
	if h.temporalClient == nil {
		return h.responseHelper.InternalServerError(c, "Temporal not configured", nil)
	}

	_, orgID, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req StartDailyAnalyticsRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "invalid request body", err)
	}

	if req.Date == "" {
		return h.responseHelper.BadRequest(c, "date is required (YYYY-MM-DD)", nil)
	}

	input := workflows.DailyAnalyticsInput{
		Date:  req.Date,
		OrgID: orgID,
	}

	options := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("daily-analytics-%s-%s", orgID.String(), req.Date),
		TaskQueue: workflows.AgentTaskQueue,
	}

	run, err := h.temporalClient.ExecuteWorkflow(c.Context(), options, workflows.DailyAnalyticsWorkflow, input)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "failed to start workflow", err)
	}

	return h.responseHelper.Success(c, WorkflowStartedResponse{
		WorkflowID: run.GetID(),
		RunID:      run.GetRunID(),
		Status:     "started",
	})
}

// GetWorkflowStatus gets the status of a workflow
// @Summary Get workflow status
// @Description Gets the current status and result of a workflow
// @Tags Temporal Workflows
// @Produce json
// @Param workflow_id path string true "Workflow ID"
// @Param run_id query string false "Run ID (optional)"
// @Success 200 {object} schema.APIResponse[WorkflowStatusResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/workflows/{workflow_id}/status [get]
func (h *Handler) GetWorkflowStatus(c fiber.Ctx) error {
	if h.temporalClient == nil {
		return h.responseHelper.InternalServerError(c, "Temporal not configured", nil)
	}

	_, _, err := h.authHelper.GetUserAndOrganization(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	workflowID := c.Params("workflow_id")
	if workflowID == "" {
		return h.responseHelper.BadRequest(c, "workflow_id is required", nil)
	}

	runID := c.Query("run_id", "")

	// Get workflow run
	run := h.temporalClient.GetWorkflow(c.Context(), workflowID, runID)

	response := WorkflowStatusResponse{
		WorkflowID: workflowID,
		RunID:      run.GetRunID(),
	}

	// Try to get result (will return error if still running)
	var result interface{}
	err = run.Get(c.Context(), &result)
	if err != nil {
		// Check if workflow is still running
		response.Status = "running"
		response.Error = err.Error()
	} else {
		response.Status = "completed"
		response.Result = result
	}

	return h.responseHelper.Success(c, response)
}
