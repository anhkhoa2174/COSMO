package inbound_lead_form

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	inboundLeadFormService "github.com/rockship/cosmo-agents-go/internal/service/inbound_lead_form"
)

type LeadFormIntegrationHandler struct {
	service *inboundLeadFormService.LeadFormIntegrationService
}

func NewLeadFormIntegrationHandler(service *inboundLeadFormService.LeadFormIntegrationService) *LeadFormIntegrationHandler {
	return &LeadFormIntegrationHandler{service: service}
}

func (h *LeadFormIntegrationHandler) Create(c fiber.Ctx) error {
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	var req v1schema.LeadFormIntegrationCreateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error()))
	}

	resp, err := h.service.Create(c.Context(), userID, &req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, err.Error(), ""))
	}

	return c.Status(http.StatusCreated).JSON(schema.SuccessResponse(resp))
}

func (h *LeadFormIntegrationHandler) Get(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, "Invalid integration id", err.Error()))
	}

	resp, err := h.service.Get(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(http.StatusInternalServerError, "Failed to fetch integration", err.Error()))
	}
	if resp == nil {
		return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(http.StatusNotFound, "Integration not found", ""))
	}
	return c.JSON(schema.SuccessResponse(resp))
}

func (h *LeadFormIntegrationHandler) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, "Invalid integration id", err.Error()))
	}

	deleted, err := h.service.Delete(c.Context(), id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(http.StatusInternalServerError, "Failed to delete integration", err.Error()))
	}
	if !deleted {
		return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(http.StatusNotFound, "Integration not found", ""))
	}

	return c.JSON(schema.SuccessResponse(fiber.Map{"message": "Integration deleted"}))
}

func (h *LeadFormIntegrationHandler) UpdateMappings(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, "Invalid integration id", err.Error()))
	}

	var req v1schema.LeadFormIntegrationUpdateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error()))
	}

	resp, err := h.service.UpdateMappings(c.Context(), id, req.FieldMappings)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, err.Error(), ""))
	}
	if resp == nil {
		return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(http.StatusNotFound, "Integration not found", ""))
	}

	return c.JSON(schema.SuccessResponse(resp))
}
