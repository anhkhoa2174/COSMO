package v3

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	operationRepo "github.com/rockship/cosmo-agents-go/internal/repository/operation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v3schema "github.com/rockship/cosmo-agents-go/internal/schema/v3"
)

// OperationHandler handles V3 operation-related requests
type OperationHandler struct {
	operationRepo *operationRepo.OperationRepository
}

// NewOperationHandler creates a new V3 OperationHandler
func NewOperationHandler(operationRepo *operationRepo.OperationRepository) *OperationHandler {
	return &OperationHandler{
		operationRepo: operationRepo,
	}
}

// GetOperation retrieves an operation by ID
// GET /v3/operations/{operation_id}
// @Summary Get operation status
// @Description Retrieves an asynchronous operation by ID including its input/output payloads
// @Tags Operations
// @Accept json
// @Produce json
// @Param operation_id path string true "Operation ID (UUID)"
// @Success 200 {object} schema.APIResponse[v3schema.OperationResponse] "Successfully retrieved operation"
// @Failure 400 {object} schema.APIResponse[any] "Invalid operation ID"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 404 {object} schema.APIResponse[any] "Operation not found"
// @Failure 500 {object} schema.APIResponse[any] "Failed to fetch operation"
// @Security BearerAuth
func (h *OperationHandler) GetOperation(c fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	_, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "Unauthorized", "",
		))
	}

	// Parse operation ID from URL parameter
	operationIDStr := c.Params("operation_id")
	operationID, err := uuid.Parse(operationIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid operation ID", err.Error(),
		))
	}

	// Fetch operation from database
	operation, err := h.operationRepo.FindByID(c.Context(), operationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Operation not found", "",
		))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to fetch operation", err.Error(),
		))
	}
	if operation == nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "Operation not found", "",
		))
	}

	// Build response
	response := v3schema.OperationResponse{
		ID:        operation.ID,
		Name:      operation.Name,
		Status:    string(operation.Status),
		CreatedAt: operation.CreatedAt.UTC(),
		UpdatedAt: operation.UpdatedAt.UTC(),
	}

	// Include input if present
	if len(operation.Input) > 0 {
		var inputData map[string]interface{}
		if err := operation.Input.Unmarshal(&inputData); err == nil {
			response.Input = inputData
		}
	}

	// Include output if present
	if len(operation.Output) > 0 {
		var outputData map[string]interface{}
		if err := operation.Output.Unmarshal(&outputData); err == nil {
			response.Output = outputData
		}
	}

	return c.Status(fiber.StatusOK).JSON(schema.SuccessResponse(response))
}
