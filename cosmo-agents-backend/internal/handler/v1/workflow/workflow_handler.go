package workflow

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	workflowRepo "github.com/rockship/cosmo-agents-go/internal/repository/workflow"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// WorkflowHandler handles workflow-related HTTP requests
type WorkflowHandler struct {
	workflowRepo *workflowRepo.WorkflowRepository
	userRepo     *user.UserRepository
}

// NewWorkflowHandler creates a new WorkflowHandler
func NewWorkflowHandler(workflowRepo *workflowRepo.WorkflowRepository, userRepo *user.UserRepository) *WorkflowHandler {
	return &WorkflowHandler{
		workflowRepo: workflowRepo,
		userRepo:     userRepo,
	}
}

// Create handles POST /v1/workflows
func (h *WorkflowHandler) Create(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	var req v1schema.CreateWorkflowRequest
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

	// Get user to extract organization_id
	user, err := h.userRepo.FindByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "User not found", err.Error(),
		))
	}

	// Create domain workflow
	// Note: The domain.Workflow model uses Nodes, Edges, State, CMetadata
	// but the schema uses Name, Description, IsActive, Config
	// We'll map the schema fields to the domain CMetadata field
	workflow := &domain.Workflow{
		UserID: userID,
	}

	// Store schema fields in CMetadata for now
	metadata := make(map[string]interface{})
	metadata["name"] = req.Name
	if req.Description != nil {
		metadata["description"] = *req.Description
	}
	metadata["is_active"] = req.IsActive
	if req.Config != nil {
		metadata["config"] = req.Config
	}

	// Convert metadata to JSONB
	workflow.CMetadata = domain.JSONB(mustMarshalJSON(metadata))

	if _, err := h.workflowRepo.Create(c.Context(), workflow); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to create workflow", err.Error(),
		))
	}

	// Extract metadata for response
	var orgID uuid.UUID
	if user.ID != uuid.Nil {
		// Since User doesn't have OrganizationID in the model, we use a placeholder
		// This should be updated when the domain models are aligned
		orgID = uuid.Nil
	}

	response := v1schema.WorkflowResponse{
		ID:             workflow.ID,
		UserID:         workflow.UserID,
		OrganizationID: orgID,
		Name:           req.Name,
		Description:    req.Description,
		IsActive:       req.IsActive,
		Config:         req.Config,
		CreatedAt:      workflow.CreatedAt,
		UpdatedAt:      workflow.UpdatedAt,
	}

	return c.Status(fiber.StatusCreated).JSON(schema.SuccessResponse(response))
}

// GetByID handles GET /v1/workflows/:id
func (h *WorkflowHandler) GetByID(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	id := c.Params("id")
	workflowID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid workflow ID", err.Error(),
		))
	}

	workflow, err := h.workflowRepo.FindByID(c.Context(), workflowID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Workflow not found", err.Error(),
		))
	}

	// Check ownership
	if workflow.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "Access denied", "",
		))
	}

	// Extract metadata
	metadata := extractMetadata(workflow.CMetadata)
	name, _ := metadata["name"].(string)
	description := getStringPtr(metadata, "description")
	isActive, _ := metadata["is_active"].(bool)
	config, _ := metadata["config"].(map[string]interface{})

	response := v1schema.WorkflowResponse{
		ID:             workflow.ID,
		UserID:         workflow.UserID,
		OrganizationID: uuid.Nil, // Placeholder
		Name:           name,
		Description:    description,
		IsActive:       isActive,
		Config:         config,
		CreatedAt:      workflow.CreatedAt,
		UpdatedAt:      workflow.UpdatedAt,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// List handles GET /v1/workflows
func (h *WorkflowHandler) List(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	if limit > 100 {
		limit = 100
	}

	filters := map[string]interface{}{
		"user_id": userID.String(),
	}

	pagination := baseRepo.PaginationParams{
		Offset: offset,
		Limit:  limit,
	}

	result, err := h.workflowRepo.FindAll(c.Context(), filters, &pagination)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to fetch workflows", err.Error(),
		))
	}

	list := make([]v1schema.WorkflowResponse, len(result.List))
	for i, wf := range result.List {
		metadata := extractMetadata(wf.CMetadata)
		name, _ := metadata["name"].(string)
		description := getStringPtr(metadata, "description")
		isActive, _ := metadata["is_active"].(bool)
		config, _ := metadata["config"].(map[string]interface{})

		list[i] = v1schema.WorkflowResponse{
			ID:             wf.ID,
			UserID:         wf.UserID,
			OrganizationID: uuid.Nil, // Placeholder
			Name:           name,
			Description:    description,
			IsActive:       isActive,
			Config:         config,
			CreatedAt:      wf.CreatedAt,
			UpdatedAt:      wf.UpdatedAt,
		}
	}

	response := v1schema.WorkflowListResponse{
		List:   list,
		Offset: offset,
		Limit:  limit,
		Total:  result.Total,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// Update handles PATCH /v1/workflows/:id
func (h *WorkflowHandler) Update(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	id := c.Params("id")
	workflowID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid workflow ID", err.Error(),
		))
	}

	var req v1schema.UpdateWorkflowRequest
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

	// Get existing workflow
	workflow, err := h.workflowRepo.FindByID(c.Context(), workflowID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Workflow not found", err.Error(),
		))
	}

	// Check ownership
	if workflow.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "Access denied", "",
		))
	}

	// Extract existing metadata
	metadata := extractMetadata(workflow.CMetadata)

	// Update fields
	if req.Name != nil {
		metadata["name"] = *req.Name
	}
	if req.Description != nil {
		metadata["description"] = *req.Description
	}
	if req.IsActive != nil {
		metadata["is_active"] = *req.IsActive
	}
	if req.Config != nil {
		metadata["config"] = req.Config
	}

	// Update CMetadata
	workflow.CMetadata = domain.JSONB(mustMarshalJSON(metadata))

	if err := h.workflowRepo.Update(c.Context(), workflowID, workflow); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to update workflow", err.Error(),
		))
	}

	// Build response
	name, _ := metadata["name"].(string)
	description := getStringPtr(metadata, "description")
	isActive, _ := metadata["is_active"].(bool)
	config, _ := metadata["config"].(map[string]interface{})

	response := v1schema.WorkflowResponse{
		ID:             workflow.ID,
		UserID:         workflow.UserID,
		OrganizationID: uuid.Nil, // Placeholder
		Name:           name,
		Description:    description,
		IsActive:       isActive,
		Config:         config,
		CreatedAt:      workflow.CreatedAt,
		UpdatedAt:      workflow.UpdatedAt,
	}

	return c.JSON(schema.SuccessResponse(response))
}

// Delete handles DELETE /v1/workflows/:id
func (h *WorkflowHandler) Delete(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "User not authenticated", "",
		))
	}

	id := c.Params("id")
	workflowID, err := uuid.Parse(id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid workflow ID", err.Error(),
		))
	}

	// Get existing workflow
	workflow, err := h.workflowRepo.FindByID(c.Context(), workflowID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Workflow not found", err.Error(),
		))
	}

	// Check ownership
	if workflow.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "Access denied", "",
		))
	}

	if err := h.workflowRepo.Delete(c.Context(), workflowID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to delete workflow", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse("Workflow deleted successfully"))
}

// Helper functions

// extractMetadata extracts map from JSONB
func extractMetadata(jsonb domain.JSONB) map[string]interface{} {
	if len(jsonb) == 0 {
		return make(map[string]interface{})
	}
	var metadata map[string]interface{}
	if err := mustUnmarshalJSON(jsonb, &metadata); err == nil {
		return metadata
	}
	return make(map[string]interface{})
}

// getStringPtr gets a string pointer from metadata
func getStringPtr(metadata map[string]interface{}, key string) *string {
	if val, ok := metadata[key]; ok {
		if str, ok := val.(string); ok {
			return &str
		}
	}
	return nil
}

// mustMarshalJSON marshals data to JSON bytes
func mustMarshalJSON(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return data
}

// mustUnmarshalJSON unmarshals JSON bytes to target
func mustUnmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
