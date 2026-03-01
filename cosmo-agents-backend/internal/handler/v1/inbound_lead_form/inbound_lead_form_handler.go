package inbound_lead_form

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	inboundLeadFormService "github.com/rockship/cosmo-agents-go/internal/service/inbound_lead_form"
)

type InboundLeadFormHandler struct {
	service *inboundLeadFormService.InboundLeadFormService
}

func NewInboundLeadFormHandler(svc *inboundLeadFormService.InboundLeadFormService) *InboundLeadFormHandler {
	return &InboundLeadFormHandler{service: svc}
}

// CreateInboundLeadForm handles POST /v1/inbound-lead-forms
// @Summary Create a new inbound lead form
// @Description Create a new inbound lead form for the authenticated user
// @Tags inbound-lead-form
// @Accept json
// @Produce json
// @Param body body v1schema.InboundLeadFormCreateRequest true "Inbound Lead Form Data"
// @Success 201 {object} schema.APIResponse[v1schema.InboundLeadFormResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/inbound-lead-forms [post]
func (h *InboundLeadFormHandler) Create(c fiber.Ctx) error {
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	var req v1schema.InboundLeadFormCreateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error()))
	}

	resp, err := h.service.Create(c.Context(), userID, &req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, err.Error(), ""))
	}

	return c.Status(http.StatusCreated).JSON(schema.SuccessResponse(resp))
}

// ListInboundLeadForms handles GET /v1/inbound-lead-forms
// @Summary List inbound lead forms
// @Tags inbound-lead-form
// @Produce json
// @Success 200 {object} schema.APIResponse[[]v1schema.InboundLeadFormListItem]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/inbound-lead-forms [get]
func (h *InboundLeadFormHandler) List(c fiber.Ctx) error {
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	forms, err := h.service.List(c.Context(), userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(http.StatusInternalServerError, "Failed to list forms", err.Error()))
	}

	return c.JSON(schema.SuccessResponse(forms))
}

// GetInboundLeadForm handles GET /v1/inbound-lead-forms/:id
// @Summary Get inbound lead form by ID
// @Tags inbound-lead-form
// @Param id path string true "Inbound Lead Form ID"
// @Produce json
// @Success 200 {object} schema.APIResponse[v1schema.InboundLeadFormResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 404 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/inbound-lead-forms/{id} [get]
func (h *InboundLeadFormHandler) Get(c fiber.Ctx) error {
	identifier := c.Params("identifier")

	uid, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	resp, err := h.service.Get(c.Context(), uid, identifier)
	if err != nil {
		if errors.Is(err, schema.ErrForbidden) {
			return c.Status(http.StatusForbidden).
				JSON(schema.ErrorResponse(http.StatusForbidden, "Forbidden", "You do not own this form"))
		}
		return c.Status(http.StatusInternalServerError).
			JSON(schema.ErrorResponse(http.StatusInternalServerError, "Failed to fetch form", err.Error()))
	}

	if resp == nil {
		return c.Status(http.StatusNotFound).
			JSON(schema.ErrorResponse(http.StatusNotFound, "Form not found", nil))
	}

	return c.JSON(schema.SuccessResponse(resp))
}

// UpdateInboundLeadForm handles PUT /v1/inbound-lead-forms/:id
// @Summary Update inbound lead form
// @Tags inbound-lead-form
// @Accept json
// @Produce json
// @Param id path string true "Inbound Lead Form ID"
// @Param body body v1schema.InboundLeadFormUpdateRequest true "Updated Lead Form Data"
// @Success 200 {object} schema.APIResponse[v1schema.InboundLeadFormResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 404 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/inbound-lead-forms/{id} [put]
func (h *InboundLeadFormHandler) Update(c fiber.Ctx) error {
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	identifier := c.Params("identifier")

	var req v1schema.InboundLeadFormUpdateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error()))
	}

	resp, err := h.service.Update(c.Context(), userID, identifier, &req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, err.Error(), ""))
	}
	if resp == nil {
		return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(http.StatusNotFound, "Form not found", ""))
	}

	return c.JSON(schema.SuccessResponse(resp))
}

// DeleteInboundLeadForm handles DELETE /v1/inbound-lead-forms/:id
// @Summary Delete inbound lead form
// @Tags inbound-lead-form
// @Param id path string true "Inbound Lead Form ID"
// @Success 204 {object} nil
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 404 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/inbound-lead-forms/{id} [delete]
func (h *InboundLeadFormHandler) Delete(c fiber.Ctx) error {
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	identifier := c.Params("identifier")
	deleted, err := h.service.Delete(c.Context(), userID, identifier)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, err.Error(), ""))
	}
	if !deleted {
		return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(http.StatusNotFound, "Form not found", ""))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{"message": "Form deleted"}))
}

// SubmitInboundLeadForm handles POST /v1/inbound-lead-forms/:id/submit
// @Summary Submit inbound lead form
// @Description Submit the inbound lead form for final processing or lead creation.
// @Tags inbound-lead-form
// @Accept json
// @Produce json
// @Param id path string true "Inbound Lead Form ID"
// @Param body body map[string]any true "Submitted Lead Form Data"
// @Success 200 {object} schema.APIResponse[v1schema.LeadFormSubmitResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 404 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v1/inbound-lead-forms/{id}/submit [post]
func (h *InboundLeadFormHandler) Submit(c fiber.Ctx) error {
	identifier := c.Params("identifier")
	var payload map[string]any
	if err := c.Bind().JSON(&payload); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error()))
	}

	resp, err := h.service.Submit(c.Context(), identifier, payload)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, err.Error(), ""))
	}
	if resp == nil {
		return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(http.StatusNotFound, "Form not found", ""))
	}

	return c.JSON(schema.SuccessResponse(resp))
}

// Helper functions reused across handlers.
func userIDFromContext(c fiber.Ctx) (uuid.UUID, bool) {
	if id, ok := c.Locals("user_id").(uuid.UUID); ok {
		return id, true
	}
	if id, ok := c.Locals("userID").(uuid.UUID); ok {
		return id, true
	}
	return uuid.UUID{}, false
}

func unauthorizedResponse(c fiber.Ctx) error {
	return c.Status(http.StatusUnauthorized).JSON(schema.ErrorResponse(http.StatusUnauthorized, "User not authenticated", ""))
}
