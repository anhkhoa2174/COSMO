package inbound_lead_form

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/core"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	v2schema "github.com/rockship/cosmo-agents-go/internal/schema/v2"
	inboundLeadFormService "github.com/rockship/cosmo-agents-go/internal/service/inbound_lead_form"
)

// InboundLeadFormHandler handles V2 inbound lead form-related requests
type InboundLeadFormHandler struct {
	session  core.Session
	service  *inboundLeadFormService.InboundLeadFormService
	userRepo *user.UserRepository
}

// NewInboundLeadFormHandler creates a new V2 InboundLeadFormHandler
func NewInboundLeadFormHandler(
	session core.Session,
	service *inboundLeadFormService.InboundLeadFormService,
	userRepo *user.UserRepository,
) *InboundLeadFormHandler {
	return &InboundLeadFormHandler{
		session:  session,
		service:  service,
		userRepo: userRepo,
	}
}

// CreateInboundLeadForm handles POST /v2/inbound-lead-forms
// @Summary Create a new inbound lead form
// @Description Create a new inbound lead form for the authenticated user
// @Tags Inbound Lead Forms V2
// @Accept json
// @Produce json
// @Param body body v2schema.InboundLeadFormCreateRequest true "Inbound Lead Form Data"
// @Success 201 {object} schema.APIResponse[v2schema.InboundLeadFormResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v2/inbound-lead-forms [post]
func (h *InboundLeadFormHandler) Create(c fiber.Ctx) error {
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	var req v2schema.InboundLeadFormCreateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error()))
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, "Validation failed", err.Error()))
	}

	// Convert v2 request to v1 for service layer compatibility
	v1Req := convertV2CreateRequestToV1(&req)

	resp, err := h.service.Create(c.Context(), userID, v1Req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, err.Error(), ""))
	}

	// Convert v1 response to v2
	v2Resp := convertV1ResponseToV2(resp)

	return c.Status(http.StatusCreated).JSON(schema.SuccessResponse(v2Resp))
}

// ListInboundLeadForms handles GET /v2/inbound-lead-forms
// @Summary List inbound lead forms
// @Tags Inbound Lead Forms V2
// @Produce json
// @Success 200 {object} schema.APIResponse[[]v2schema.InboundLeadFormListItem]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v2/inbound-lead-forms [get]
func (h *InboundLeadFormHandler) List(c fiber.Ctx) error {
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	forms, err := h.service.List(c.Context(), userID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(schema.ErrorResponse(http.StatusInternalServerError, "Failed to list forms", err.Error()))
	}

	// Convert v1 list items to v2
	var v2Forms []v2schema.InboundLeadFormListItem
	for _, form := range forms {
		v2Forms = append(v2Forms, convertV1ListItemToV2(form))
	}

	return c.JSON(schema.SuccessResponse(v2Forms))
}

// GetInboundLeadForm handles GET /v2/inbound-lead-forms/{id}
// @Summary Get inbound lead form by ID
// @Tags Inbound Lead Forms V2
// @Param id path string true "Inbound Lead Form ID"
// @Produce json
// @Success 200 {object} schema.APIResponse[v2schema.InboundLeadFormResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 404 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v2/inbound-lead-forms/{id} [get]
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

	// Convert v1 response to v2
	v2Resp := convertV1ResponseToV2(resp)

	return c.JSON(schema.SuccessResponse(v2Resp))
}

// UpdateInboundLeadForm handles PUT /v2/inbound-lead-forms/{id}
// @Summary Update inbound lead form
// @Tags Inbound Lead Forms V2
// @Accept json
// @Produce json
// @Param id path string true "Inbound Lead Form ID"
// @Param body body v2schema.InboundLeadFormUpdateRequest true "Updated Lead Form Data"
// @Success 200 {object} schema.APIResponse[v2schema.InboundLeadFormResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 404 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v2/inbound-lead-forms/{id} [put]
func (h *InboundLeadFormHandler) Update(c fiber.Ctx) error {
	userID, ok := userIDFromContext(c)
	if !ok {
		return unauthorizedResponse(c)
	}

	identifier := c.Params("identifier")

	var req v2schema.InboundLeadFormUpdateRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error()))
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, "Validation failed", err.Error()))
	}

	// Convert v2 request to v1 for service layer compatibility
	v1Req := convertV2UpdateRequestToV1(&req)

	resp, err := h.service.Update(c.Context(), userID, identifier, v1Req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(schema.ErrorResponse(http.StatusBadRequest, err.Error(), ""))
	}
	if resp == nil {
		return c.Status(http.StatusNotFound).JSON(schema.ErrorResponse(http.StatusNotFound, "Form not found", ""))
	}

	// Convert v1 response to v2
	v2Resp := convertV1ResponseToV2(resp)

	return c.JSON(schema.SuccessResponse(v2Resp))
}

// DeleteInboundLeadForm handles DELETE /v2/inbound-lead-forms/{id}
// @Summary Delete inbound lead form
// @Tags Inbound Lead Forms V2
// @Param id path string true "Inbound Lead Form ID"
// @Success 204 {object} nil
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 404 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v2/inbound-lead-forms/{id} [delete]
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

// SubmitInboundLeadForm handles POST /v2/inbound-lead-forms/{id}/submit
// @Summary Submit inbound lead form
// @Description Submit the inbound lead form for final processing or lead creation.
// @Tags Inbound Lead Forms V2
// @Accept json
// @Produce json
// @Param id path string true "Inbound Lead Form ID"
// @Param body body map[string]any true "Submitted Lead Form Data"
// @Success 200 {object} schema.APIResponse[v2schema.LeadFormSubmitResponse]
// @Failure 400 {object} schema.APIResponse[any]
// @Failure 401 {object} schema.APIResponse[any]
// @Failure 404 {object} schema.APIResponse[any]
// @Failure 500 {object} schema.APIResponse[any]
// @Security BearerAuth
// @Router /v2/inbound-lead-forms/{id}/submit [post]
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

	// Convert v1 response to v2
	v2Resp := convertV1SubmitResponseToV2(resp)

	return c.JSON(schema.SuccessResponse(v2Resp))
}

// Helper functions
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

// Conversion functions between v1 and v2 schemas
func convertV2CreateRequestToV1(v2Req *v2schema.InboundLeadFormCreateRequest) *v1schema.InboundLeadFormCreateRequest {
	v1Req := &v1schema.InboundLeadFormCreateRequest{
		Name:           v2Req.Name,
		Slug:           v2Req.Slug,
		UIMetadata:     v2Req.UIMetadata,
		ListContactIDs: v2Req.ListContactIDs,
	}

	// Convert fields
	for _, field := range v2Req.Fields {
		v1Req.Fields = append(v1Req.Fields, v1schema.InboundLeadFormFieldRequest{
			Name:        field.Name,
			DisplayName: field.DisplayName,
			IsRequired:  field.IsRequired,
			UIMetadata:  field.UIMetadata,
		})
	}

	return v1Req
}

func convertV2UpdateRequestToV1(v2Req *v2schema.InboundLeadFormUpdateRequest) *v1schema.InboundLeadFormUpdateRequest {
	v1Req := &v1schema.InboundLeadFormUpdateRequest{
		Name:       v2Req.Name,
		UIMetadata: v2Req.UIMetadata,
	}

	// Convert fields
	for _, field := range v2Req.Fields {
		v1Req.Fields = append(v1Req.Fields, v1schema.InboundLeadFormFieldRequest{
			Name:        field.Name,
			DisplayName: field.DisplayName,
			IsRequired:  field.IsRequired,
			UIMetadata:  field.UIMetadata,
		})
	}

	return v1Req
}

func convertV1ResponseToV2(v1Resp *v1schema.InboundLeadFormResponse) v2schema.InboundLeadFormResponse {
	v2Resp := v2schema.InboundLeadFormResponse{
		ID:         v1Resp.ID,
		Name:       v1Resp.Name,
		Slug:       v1Resp.Slug,
		UIMetadata: v1Resp.UIMetadata,
	}

	// Convert fields
	for _, field := range v1Resp.Fields {
		v2Resp.Fields = append(v2Resp.Fields, v2schema.InboundLeadFormFieldResponse{
			Name:          field.Name,
			DisplayName:   field.DisplayName,
			FieldType:     field.FieldType,
			IsRequired:    field.IsRequired,
			SelectOptions: field.SelectOptions,
			FallbackValue: field.FallbackValue,
			UIMetadata:    field.UIMetadata,
		})
	}

	return v2Resp
}

func convertV1ListItemToV2(v1Item v1schema.InboundLeadFormListItem) v2schema.InboundLeadFormListItem {
	return v2schema.InboundLeadFormListItem{
		ID:   v1Item.ID,
		Name: v1Item.Name,
		Slug: v1Item.Slug,
	}
}

func convertV1SubmitResponseToV2(v1Resp *v1schema.LeadFormSubmitResponse) v2schema.LeadFormSubmitResponse {
	return v2schema.LeadFormSubmitResponse{
		ContactID: v1Resp.ContactID,
		Data:      v1Resp.Data,
	}
}
