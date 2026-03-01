package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1conversation "github.com/rockship/cosmo-agents-go/internal/handler/v1/conversation"
	v1pagination "github.com/rockship/cosmo-agents-go/internal/handler/v1/pagination"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	internalworker "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// Constants for pagination and limits
const (
	DefaultPageSize = 25
	MaxPageSize     = 100
	DefaultOffset   = 0

	// Filter limits to prevent oversized requests that could impact performance
	MaxFilterArraySize   = 50
	MaxFilterValueLength = 1000
)

// Email label constants for consistent usage
const (
	EmailLabelOpened  = "opened"
	EmailLabelBounced = "bounced"
)

// Allowed filter keys for search endpoints to prevent SQL injection
var AllowedAgentFilterKeys = map[string]bool{
	"status":          true,
	"organization_id": true,
	"type":            true,
	"active":          true,
	"name":            true,
}

// Standardized error messages for consistent API responses
const (
	ErrUserNotAuthenticated  = "User not authenticated"
	ErrInvalidAgentID        = "Invalid agent ID format"
	ErrAgentNotFound         = "Agent not found"
	ErrAccessDenied          = "Access denied"
	ErrInvalidRequestBody    = "Invalid request body"
	ErrValidationFailed      = "Validation failed"
	ErrInternalServer        = "Internal server error"
	ErrTooManyFilterItems    = "Too many items in filter"
	ErrFailedToLoadUserRoles = "Failed to load user roles"
	ErrInvalidRequestFormat  = "Invalid request format"
)

// AgentHandler handles agent-related HTTP requests
type AgentHandler struct {
	agentRepo        *agentRepo.AgentRepository
	conversationRepo *conversationRepo.ConversationRepository
	userRepo         *user.UserRepository
	emailRepo        *emailRepo.Repository
	roleRepo         *roleRepo.RoleRepository
	workerClient     workerClient
	mapper           *AgentMapper
	statsCalculator  *AgentStatsCalculator
}

type workerClient interface {
	EnqueueLowPriorityTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// NewAgentHandler creates a new AgentHandler
func NewAgentHandler(
	agentRepo *agentRepo.AgentRepository,
	conversationRepo *conversationRepo.ConversationRepository,
	userRepo *user.UserRepository,
	emailRepo *emailRepo.Repository,
	roleRepo *roleRepo.RoleRepository,
	workerClient workerClient,
) (*AgentHandler, error) {

	db := agentRepo.GetDB()

	return &AgentHandler{
		agentRepo:        agentRepo,
		conversationRepo: conversationRepo,
		userRepo:         userRepo,
		emailRepo:        emailRepo,
		roleRepo:         roleRepo,
		workerClient:     workerClient,
		mapper:           NewAgentMapper(),
		statsCalculator:  NewAgentStatsCalculator(db),
	}, nil
}

// Create handles POST /v1/agents
// @Summary Create a new agent
// @Description Creates a new AI agent with the provided configuration
// @Tags Agents
// @Accept json
// @Produce json
// @Param agent body v1schema.CreateAgentRequest true "Agent creation data"
// @Success 201 {object} schema.APIResponse[v1schema.AgentResponse] "Successfully created agent"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/agents [post]
func (h *AgentHandler) Create(c fiber.Ctx) error {
	// Get authenticated user
	userID, err := GetAuthenticatedUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, ErrUserNotAuthenticated, "",
		))
	}

	var req v1schema.CreateAgentRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, ErrInvalidRequestBody, err.Error(),
		))
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, ErrValidationFailed, err.Error(),
		))
	}

	// Validate organization access if OrganizationID is provided
	if req.OrganizationID != nil {
		if err := h.ValidateOrganizationAccess(c, userID, *req.OrganizationID); err != nil {
			return err
		}
	}

	agent := &domain.Agent{
		UserID:         userID,
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Email:          req.Email,
		Persona:        req.Persona,
		EmailProvider:  domain.AgentEmailProvider(req.EmailProvider),
		Signature:      req.Signature,
		Picture:        req.Picture,
		DailyLimit:     req.DailyLimit,
		MaxDailyLimit:  req.MaxDailyLimit,
	}

	if req.CMetadata != nil {
		agent.SetMetadata(req.CMetadata)
	}

	if req.Credentials != nil {
		credentialsJSON, err := json.Marshal(req.Credentials)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest, "Invalid credentials format", err.Error(),
			))
		}

		agent.Credentials = credentialsJSON

		logger.Logger.Info().
			Str("user_id", userID.String()).
			Str("agent_email", agent.Email).
			Msg("Agent credentials provided on create request")
	}

	if _, err := h.agentRepo.Create(c.Context(), agent); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to create agent", err.Error(),
		))
	}

	// Convert to response
	response := h.mapper.ToAgentResponse(agent)
	return c.Status(fiber.StatusCreated).JSON(schema.SuccessResponse(response))
}

// GetByID handles GET /v1/agents/:id
// @Summary Get agent by ID
// @Description Retrieves a single agent by its unique ID
// @Tags Agents
// @Accept json
// @Produce json
// @Param id path string true "Agent ID (UUID)"
// @Success 200 {object} schema.APIResponse[v1schema.AgentGetResponse] "Successfully retrieved agent"
// @Failure 400 {object} schema.APIResponse[any] "Invalid agent ID"
// @Failure 404 {object} schema.APIResponse[any] "Agent not found"
// @Security BearerAuth
// @Router /v1/agents/{id} [get]
func (h *AgentHandler) GetByID(c fiber.Ctx) error {
	// Get authenticated user
	userID, err := GetAuthenticatedUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, ErrUserNotAuthenticated, "",
		))
	}

	// Parse agent ID
	agentID, err := ParseAgentID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, ErrInvalidAgentID, err.Error(),
		))
	}

	// Check agent access permissions
	agent, err := h.CheckAgentAccessPermission(c, userID, agentID)
	if err != nil {
		return err
	}

	// Re-evaluate token status so FE sees expired/revoked tokens as invalid_grant.
	if newStatus, checkErr := agent.CheckTokenStatus(); checkErr == nil && newStatus != agent.Status {
		agent.Status = newStatus
		if newStatus == domain.AgentStatusInvalidGrant {
			invalid := false
			agent.ValidCred = &invalid
		}
		if updateErr := h.agentRepo.Update(c.Context(), agent.ID, agent); updateErr != nil {
			logger.Logger.Warn().
				Err(updateErr).
				Str("agent_id", agent.ID.String()).
				Msg("Failed to persist updated agent status after token check")
		}
	}

	// Calculate email statistics
	emailStats, err := h.statsCalculator.CalculateEmailStatistics(c.Context(), agentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to calculate email statistics", err.Error(),
		))
	}

	// Get inbox detail information
	inboxDetail, err := h.statsCalculator.GetInboxDetail(c.Context(), agent)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to get inbox details", err.Error(),
		))
	}

	// Build response
	response := v1schema.AgentGetResponse{
		Entity:          h.mapper.ToAgentResponse(agent),
		InboxDetail:     inboxDetail,
		EmailStatistics: emailStats,
	}

	return c.JSON(schema.SuccessResponse(response))
}

func (h *AgentHandler) List(c fiber.Ctx) error {
	userID, err := GetAuthenticatedUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, ErrUserNotAuthenticated, "",
		))
	}

	// Parse pagination parameters
	pagination := v1pagination.ParsePaginationParams(c)
	offset, limit := pagination.Offset, pagination.Limit
	paginationParams := struct {
		Offset int
		Limit  int
	}{
		Offset: offset,
		Limit:  limit,
	}

	// Convert to repository pagination params
	repoPagination := baseRepo.PaginationParams{
		Offset: paginationParams.Offset,
		Limit:  paginationParams.Limit,
	}

	// Build filters
	filter := baseRepo.Filter{
		"user_id":    userID,
		"is_deleted": false,
	}

	// Filter by status if provided
	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}

	// Filter by organization if provided
	if orgID := c.Query("organization_id"); orgID != "" {
		if parsedOrgID, err := uuid.Parse(orgID); err == nil {
			filter["organization_id"] = parsedOrgID
		}
	}

	// Get agents
	result, err := h.agentRepo.FindAll(c.Context(), filter, &repoPagination)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to fetch agents", err.Error(),
		))
	}

	// Convert to response using mapper
	items := make([]v1schema.AgentResponse, len(result.List))
	for i, agent := range result.List {
		if newStatus, checkErr := agent.CheckTokenStatus(); checkErr == nil && newStatus != agent.Status {
			agent.Status = newStatus
			if newStatus == domain.AgentStatusInvalidGrant {
				invalid := false
				agent.ValidCred = &invalid
			}
			if updateErr := h.agentRepo.Update(c.Context(), agent.ID, &agent); updateErr != nil {
				logger.Logger.Warn().
					Err(updateErr).
					Str("agent_id", agent.ID.String()).
					Msg("Failed to persist agent status after token check (list)")
			}
		}
		items[i] = h.mapper.ToAgentResponse(&agent)
	}

	page := (repoPagination.Offset / repoPagination.Limit) + 1
	totalPages := int((result.Total + int64(repoPagination.Limit) - 1) / int64(repoPagination.Limit))

	response := v1schema.AgentListResponse{
		Items:      items,
		Total:      result.Total,
		Page:       page,
		PageSize:   repoPagination.Limit,
		TotalPages: totalPages,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// Search handles POST /v1/agents/search
// @Summary Search agents
// @Description Returns a paginated list of agents matching provided filters
// @Tags Agents
// @Accept json
// @Produce json
// @Param offset query int false "Pagination page index (starting at 0)" default(0)
// @Param limit query int false "Page size (max 100)" default(25)
// @Param body body v1schema.AgentSearchRequest false "Search filters (optional)"
// @Success 200 {object} schema.APIResponse[v1schema.AgentSearchResponse] "Search results"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/agents/search [post]
func (h *AgentHandler) Search(c fiber.Ctx) error {
	// Get authenticated user
	userID, err := GetAuthenticatedUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, ErrUserNotAuthenticated, "",
		))
	}

	var req v1schema.AgentSearchRequest
	if body := c.Body(); len(body) > 0 {
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest, ErrInvalidRequestBody, err.Error(),
			))
		}
	}

	pagination := v1pagination.ParsePaginationParams(c)
	offset, limit := pagination.Offset, pagination.Limit
	paginationParams := struct {
		Offset int
		Limit  int
	}{
		Offset: offset,
		Limit:  limit,
	}
	skip := paginationParams.Offset * paginationParams.Limit

	// Validate and parse filters
	var filter baseRepo.Filter
	if req.Filter != nil {
		filter, err = ValidateAndParseFilter(req.Filter)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest, "Invalid filter", err.Error(),
			))
		}
	} else {
		filter = make(baseRepo.Filter)
	}

	filter["user_id"] = userID
	filter["is_deleted"] = false

	repoPagination := baseRepo.PaginationParams{
		Offset: skip,
		Limit:  paginationParams.Limit,
	}

	result, err := h.agentRepo.FindAll(c.Context(), filter, &repoPagination)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to search agents", err.Error(),
		))
	}

	// Convert to response using mapper
	list := make([]v1schema.AgentListItem, len(result.List))
	for i := range result.List {
		if newStatus, checkErr := result.List[i].CheckTokenStatus(); checkErr == nil && newStatus != result.List[i].Status {
			result.List[i].Status = newStatus
			if newStatus == domain.AgentStatusInvalidGrant {
				invalid := false
				result.List[i].ValidCred = &invalid
			}
			if updateErr := h.agentRepo.Update(c.Context(), result.List[i].ID, &result.List[i]); updateErr != nil {
				logger.Logger.Warn().
					Err(updateErr).
					Str("agent_id", result.List[i].ID.String()).
					Msg("Failed to persist agent status after token check (search)")
			}
		}
		list[i] = h.mapper.ToAgentListItem(&result.List[i])
	}

	response := v1schema.AgentSearchResponse{
		List:   list,
		Offset: paginationParams.Offset,
		Limit:  paginationParams.Limit,
		Total:  result.Total,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// Update handles PUT /v1/agents/:id
// @Summary Update an agent
// @Description Updates an existing agent's configuration
// @Tags Agents
// @Accept json
// @Produce json
// @Param id path string true "Agent ID (UUID)"
// @Param agent body v1schema.UpdateAgentRequest true "Agent update data"
// @Success 200 {object} schema.APIResponse[v1schema.AgentResponse] "Successfully updated agent"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Permission denied"
// @Failure 404 {object} schema.APIResponse[any] "Agent not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/agents/{id} [put]
func (h *AgentHandler) Update(c fiber.Ctx) error {
	// Get authenticated user
	userID, err := GetAuthenticatedUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, ErrUserNotAuthenticated, "",
		))
	}

	// Parse agent ID
	agentID, err := ParseAgentID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, ErrInvalidAgentID, err.Error(),
		))
	}

	var req v1schema.UpdateAgentRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, ErrInvalidRequestBody, err.Error(),
		))
	}

	// Validate request
	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, ErrValidationFailed, err.Error(),
		))
	}

	// Get existing agent
	agent, err := h.agentRepo.FindByID(c.Context(), agentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, ErrAgentNotFound, err.Error(),
		))
	}

	// Check ownership - only agent owners can update
	if agent.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, ErrAccessDenied, "",
		))
	}

	// Apply updates
	if req.Name != nil {
		agent.Name = *req.Name
	}
	if req.Email != nil {
		agent.Email = *req.Email
	}
	if req.Persona != nil {
		agent.Persona = *req.Persona
	}
	if req.EmailProvider != nil {
		agent.EmailProvider = domain.AgentEmailProvider(*req.EmailProvider)
	}
	if req.Signature != nil {
		agent.Signature = *req.Signature
	}
	if req.Picture != nil {
		agent.Picture = *req.Picture
	}
	if req.DailyLimit != nil {
		agent.DailyLimit = req.DailyLimit
	}
	if req.Status != nil {
		agent.Status = domain.AgentStatus(*req.Status)
	}

	// Optional: persist working_hours into cmetadata for parity
	if req.WorkingHours != nil {
		// Load existing metadata
		meta := map[string]any{}
		if len(agent.CMetadata) > 0 {
			_ = json.Unmarshal(agent.CMetadata, &meta)
		}
		meta["working_hours"] = *req.WorkingHours
		_ = agent.SetMetadata(meta)
	}

	// Save updates
	if err := h.agentRepo.Update(c.Context(), agent.ID, agent); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to update agent", err.Error(),
		))
	}

	response := h.mapper.ToAgentResponse(agent)
	return c.JSON(schema.SuccessResponse(response))
}

// Delete handles DELETE /v1/agents/:id (soft delete)
// @Summary Delete an agent
// @Description Soft deletes an agent by marking it as deleted
// @Tags Agents
// @Accept json
// @Produce json
// @Param id path string true "Agent ID (UUID)"
// @Success 204 "Successfully deleted agent"
// @Failure 400 {object} schema.APIResponse[any] "Invalid agent ID"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Permission denied"
// @Failure 404 {object} schema.APIResponse[any] "Agent not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/agents/{id} [delete]
func (h *AgentHandler) Delete(c fiber.Ctx) error {
	// Get authenticated user
	userID, err := GetAuthenticatedUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, ErrUserNotAuthenticated, "",
		))
	}

	// Parse agent ID
	agentID, err := ParseAgentID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, ErrInvalidAgentID, err.Error(),
		))
	}

	// Get existing agent
	agent, err := h.agentRepo.FindByID(c.Context(), agentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, ErrAgentNotFound, err.Error(),
		))
	}

	// Check ownership - only agent owners can delete
	if agent.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, ErrAccessDenied, "",
		))
	}

	// Soft delete
	if err := h.agentRepo.Delete(c.Context(), agentID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to delete agent", err.Error(),
		))
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

func (h *AgentHandler) Sync(c fiber.Ctx) error {
	// Get authenticated user
	userID, err := GetAuthenticatedUser(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, ErrUserNotAuthenticated, "",
		))
	}

	// Parse agent ID
	agentID, err := ParseAgentID(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, ErrInvalidAgentID, err.Error(),
		))
	}

	var req v1schema.SyncAgentRequest
	if err := c.Bind().JSON(&req); err != nil {
		// Default to incremental sync if no body provided
		req.FullSync = false
	}

	// Get existing agent
	agent, err := h.agentRepo.FindByID(c.Context(), agentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, ErrAgentNotFound, err.Error(),
		))
	}

	// Check ownership - only agent owners can sync
	if agent.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, ErrAccessDenied, "",
		))
	}

	if h.workerClient == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Worker client not configured", "",
		))
	}

	startHistoryID := agent.LastHistoryID
	if req.FullSync {
		startHistoryID = ""
	}

	payload := internalworker.SyncGmailHistoryPayload{
		AgentID:        agent.ID,
		StartHistoryID: startHistoryID,
	}

	if _, err := h.workerClient.EnqueueLowPriorityTask(c.Context(), worker.TypeSyncGmailHistory, payload); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to enqueue Gmail sync", err.Error(),
		))
	}

	response := v1schema.SyncAgentResponse{
		Message:       "Gmail sync task enqueued successfully",
		EmailsSynced:  0,
		TasksEnqueued: 1,
		SyncedAt:      time.Now(),
	}

	return c.JSON(schema.SuccessResponse(response))
}

// GetConversations handles POST /v1/agents/:id/conversations/search
// @Summary Get agent conversations
// @Description Retrieves conversations for an agent with filtering, sorting, and pagination
// @Tags Agents
// @Accept json
// @Produce json
// @Param id path string true "Agent ID" format(uuid)
// @Param body body v1schema.AgentGetConversationRequest false "Search filters (optional)"
// @Param conversation_type query string false "Conversation Type filter" Enums(sent, assign_to_ai, assign_to_human)
// @Param newest query bool false "Sort by updated_at descending if true, ascending if false" default(true)
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit (max 100)" default(25) maximum(100)
// @Success 200 {object} schema.APIResponse[schema.PaginatedResponse[v1schema.AgentGetConversationsResponse]] "Successfully retrieved conversations"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Access denied"
// @Failure 404 {object} schema.APIResponse[any] "Agent not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/agents/{id}/conversations/search [post]
func (h *AgentHandler) GetConversations(c fiber.Ctx) error {
	// Get user ID from context
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	// Parse agent ID from URL
	agentIDStr := c.Params("id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid agent ID", err.Error(),
		))
	}

	// Parse query parameters, including optional conversation type filter
	conversationType := c.Query("conversation_type")

	if conversationType != "" && !v1conversation.ValidConversationTypesExported[conversationType] { // Shared whitelist with conversation handler
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Invalid conversation type",
			"Allowed values are: sent, assign_to_ai, assign_to_human, or empty",
		))
	}

	if len(conversationType) > v1conversation.MaxConversationTypeLengthExported {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest,
			"Conversation type too long",
			"Maximum length is 50 characters",
		))
	}

	newest := c.Query("newest", "true") == "true"

	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil || offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(c.Query("limit", strconv.Itoa(DefaultPageSize)))
	if err != nil || limit <= 0 || limit > MaxPageSize {
		limit = DefaultPageSize
	}

	// Parse request body for filters
	var req v1schema.AgentGetConversationRequest
	if c.Body() != nil && len(c.Body()) > 0 {
		if err := c.Bind().JSON(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest, "Invalid request body", err.Error(),
			))
		}

		// Validate request
		if err := v1validation.ValidateStruct(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
				fiber.StatusBadRequest, "Validation failed", err.Error(),
			))
		}

		// Validate filter array sizes to prevent oversized requests
		if req.Filter != nil {
			if labels, ok := req.Filter["labels"].([]interface{}); ok && len(labels) > MaxFilterArraySize {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Too many labels in filter",
					fmt.Sprintf("Maximum %d labels allowed, got %d", MaxFilterArraySize, len(labels)),
				))
			}

			if intents, ok := req.Filter["intents"].([]interface{}); ok && len(intents) > MaxFilterArraySize {
				return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
					fiber.StatusBadRequest, "Too many intents in filter",
					fmt.Sprintf("Maximum %d intents allowed, got %d", MaxFilterArraySize, len(intents)),
				))
			}
		}
	}

	// Get agent and verify access
	agent, err := h.agentRepo.FindByID(c.Context(), agentID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Agent not found", err.Error(),
		))
	}

	hasAccess := false

	if agent.UserID == userID {
		hasAccess = true
	} else if agent.OrganizationID != nil {
		// Non-owners need organization membership - optimized query to avoid N+1
		role, err := h.roleRepo.FindByUserAndOrganization(c.Context(), userID, *agent.OrganizationID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
				fiber.StatusInternalServerError, ErrFailedToLoadUserRoles, err.Error(),
			))
		}

		hasAccess = role != nil
	}

	if !hasAccess {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "You don't have permission to access this agent's conversations", "",
		))
	}

	// Convert filter to repository format
	var filter *conversationRepo.ConversationSearchFilter
	if req.Filter != nil {
		filter = &conversationRepo.ConversationSearchFilter{}

		// Extract labels from filter map with validation
		if labels, ok := req.Filter["labels"].([]interface{}); ok {
			labelStrings := make([]string, 0, len(labels))
			for _, label := range labels {
				if s, ok := label.(string); ok && s != "" {
					labelStrings = append(labelStrings, s)
				}
			}
			if len(labelStrings) > 0 {
				filter.Labels = labelStrings
			}
		}

		// Extract intents from filter map with validation
		if intents, ok := req.Filter["intents"].([]interface{}); ok {
			intentStrings := make([]string, 0, len(intents))
			for _, intent := range intents {
				if s, ok := intent.(string); ok && s != "" {
					intentStrings = append(intentStrings, s)
				}
			}
			if len(intentStrings) > 0 {
				filter.Intents = intentStrings
			}
		}

		// Extract status from filter map
		if status, ok := req.Filter["status"].(string); ok {
			filter.Status = &status
		}

		// Extract replied from filter map
		if replied, ok := req.Filter["replied"].(bool); ok {
			filter.Replied = &replied
		}

		// Extract is_deleted flag to allow viewing archived conversations
		if isDeleted, ok := req.Filter["is_deleted"].(bool); ok {
			filter.IsDeleted = &isDeleted
		}
	}

	// Convert conversation type
	var convType *domain.ConversationType
	if conversationType != "" {
		ct := domain.ConversationType(conversationType)
		convType = &ct
	}

	repoCtx := c.Context()
	if repoCtx == nil {
		logger.Logger.Warn().Str("agent_id", agentID.String()).Msg("fiber context missing; using background context for repository call")
		repoCtx = context.Background()
	}
	if agent.UserID != uuid.Nil {
		repoCtx = conversationRepo.WithAgentUserID(repoCtx, agent.UserID)
	}

	// Get conversations with filters
	conversations, total, err := h.conversationRepo.FindByAgentIDWithFilters(
		repoCtx, agentID, filter, convType, newest, offset, limit,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to retrieve conversations", err.Error(),
		))
	}

	// Get conversation IDs for batch email lookup (prevents N+1 query)
	conversationIDs := make([]uuid.UUID, len(conversations))
	for i, conv := range conversations {
		conversationIDs[i] = conv.ID
	}

	// Batch fetch latest emails for all conversations
	latestEmails, err := h.emailRepo.FindLatestByConversationIDs(c.Context(), conversationIDs)
	if err != nil {
		logger.Logger.Warn().Err(err).Msg("Failed to retrieve latest emails for conversations")
		latestEmails = make(map[uuid.UUID]*domain.Email)
	}

	// Convert to response format matching Python structure with entity and latest_email
	responseItems := make([]v1schema.AgentGetConversationsResponse, len(conversations))
	for i, conv := range conversations {
		latestEmail := latestEmails[conv.ID]

		// Build email entity using shared helper which guarantees non-nil slices
		emailEntity := v1conversation.ToEmailEntityFromDomain(latestEmail)

		var labels []string
		if conv.Labels != nil {
			labels = conv.Labels
		} else {
			labels = []string{}
		}

		var intents []string
		if conv.Intents != nil {
			intents = conv.Intents
		} else {
			intents = []string{}
		}

		responseItems[i] = v1schema.AgentGetConversationsResponse{
			Entity: v1schema.ConversationEntity{
				ID:         conv.ID,
				UserID:     conv.UserID,
				Labels:     labels,
				Replied:    conv.Replied,
				CampaignID: conv.CampaignID,
				AssigneeID: conv.AssigneeID,
				Intents:    intents,
				IsDeleted:  conv.IsDeleted,
				CreatedAt:  conv.CreatedAt,
				UpdatedAt:  conv.UpdatedAt,
			},
			LatestEmail: emailEntity,
		}
	}

	response := schema.PaginatedResponse[v1schema.AgentGetConversationsResponse]{
		List:   responseItems,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}

	return c.JSON(schema.SuccessResponse(response))
}
