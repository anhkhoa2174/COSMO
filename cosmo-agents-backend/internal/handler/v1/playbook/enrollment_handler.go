package playbook

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/handler"
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	playbookService "github.com/rockship/cosmo-agents-go/internal/service/playbook"
)

// EnrollmentHandler handles enrollment HTTP requests
type EnrollmentHandler struct {
	service        *playbookService.EnrollmentService
	responseHelper *handler.ResponseHelper
}

// NewEnrollmentHandler creates a new enrollment handler
func NewEnrollmentHandler(service *playbookService.EnrollmentService) *EnrollmentHandler {
	return &EnrollmentHandler{
		service:        service,
		responseHelper: handler.NewResponseHelper(),
	}
}

// EnrollContact handles POST /v1/contacts/:id/enroll
func (h *EnrollmentHandler) EnrollContact(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse contact ID
	contactID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid contact ID", nil)
	}

	// Parse request body
	var req v1schema.EnrollContactRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", nil)
	}

	// Call service
	if err := h.service.EnrollContact(ctx, contactID, req.PlaybookID); err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to enroll contact", err)
	}

	return h.responseHelper.Success(c, fiber.Map{"message": "Contact enrolled successfully"})
}

// ListPendingApprovals handles GET /v1/enrollment-approvals/pending
func (h *EnrollmentHandler) ListPendingApprovals(c fiber.Ctx) error {
	ctx := c.Context()

	approvals, err := h.service.ListPendingApprovals(ctx)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to list pending approvals", err)
	}

	return h.responseHelper.Success(c, approvals)
}

// ApproveEnrollment handles POST /v1/enrollment-approvals/:id/approve
func (h *EnrollmentHandler) ApproveEnrollment(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse request ID
	requestID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request ID", nil)
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return h.responseHelper.Unauthorized(c, "User not authenticated")
	}

	// Call service
	if err := h.service.ApproveEnrollment(ctx, requestID, userID); err != nil {
		if errors.Is(err, playbookService.ErrAlreadyDecided) {
			return c.Status(fiber.StatusConflict).JSON(schema.ErrorResponse(fiber.StatusConflict, err.Error(), nil))
		}
		return h.responseHelper.InternalServerError(c, "Failed to approve enrollment", err)
	}

	return h.responseHelper.Success(c, fiber.Map{"message": "Enrollment approved successfully"})
}

// RejectEnrollment handles POST /v1/enrollment-approvals/:id/reject
func (h *EnrollmentHandler) RejectEnrollment(c fiber.Ctx) error {
	ctx := c.Context()

	// Parse request ID
	requestID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request ID", nil)
	}

	// Get user ID from context
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return h.responseHelper.Unauthorized(c, "User not authenticated")
	}

	// Call service
	if err := h.service.RejectEnrollment(ctx, requestID, userID); err != nil {
		if errors.Is(err, playbookService.ErrAlreadyDecided) {
			return c.Status(fiber.StatusConflict).JSON(schema.ErrorResponse(fiber.StatusConflict, err.Error(), nil))
		}
		return h.responseHelper.InternalServerError(c, "Failed to reject enrollment", err)
	}

	return h.responseHelper.Success(c, fiber.Map{"message": "Enrollment rejected successfully"})
}

// GetEnrollment handles GET /v1/enrollments/:id
func (h *EnrollmentHandler) GetEnrollment(c fiber.Ctx) error {
	ctx := c.Context()

	enrollmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid enrollment ID", nil)
	}

	enrollment, err := h.service.GetEnrollmentByID(ctx, enrollmentID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to get enrollment", err)
	}

	return h.responseHelper.Success(c, enrollment)
}

// UpdateEnrollmentStatus handles PATCH /v1/enrollments/:id/status
func (h *EnrollmentHandler) UpdateEnrollmentStatus(c fiber.Ctx) error {
	ctx := c.Context()

	enrollmentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid enrollment ID", nil)
	}

	var req struct {
		Status string `json:"status"` // "active", "paused", "completed"
	}
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", nil)
	}

	if err := h.service.UpdateEnrollmentStatus(ctx, enrollmentID, req.Status); err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to update enrollment status", err)
	}

	return h.responseHelper.Success(c, fiber.Map{"message": "Enrollment status updated successfully"})
}
