package customfield

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// Create handles POST /v1/custom-fields
// @Summary Create a custom field
// @Description Creates a new custom field for the authenticated user
// @Tags Custom Fields
// @Accept json
// @Produce json
// @Param request body v1schema.CreateCustomFieldRequest true "Custom field details"
// @Success 201 {object} schema.APIResponse[v1schema.CustomFieldResponse] "Custom field created successfully"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request body or validation failed"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 500 {object} schema.APIResponse[any] "Failed to create custom field"
// @Router /v1/custom-fields [post]
// @Security BearerAuth
func (h *Handler) Create(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return unauthorized(c, "User not authenticated")
	}

	var req v1schema.CreateCustomFieldRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return badRequest(c, "Validation failed", err)
	}

	// Get organization_id from user's role
	var organizationID *uuid.UUID
	roles, err := h.roleRepo.FindByUserID(c.Context(), userID)
	if err == nil && len(roles) > 0 {
		// Use the first active role's organization
		for _, role := range roles {
			if role.Status == "active" {
				organizationID = &role.OrganizationID
				break
			}
		}
	}

	// Create domain custom field
	customField := &domain.CustomField{
		UserID:         userID,
		Name:           req.Name,
		DataType:       req.DataType,
		EntityType:     req.EntityType,
		IsRequired:     req.IsRequired,
		SampleData:     req.SampleData,
		FallbackValue:  req.FallbackValue,
		OrganizationID: organizationID,
	}

	if req.Options != nil {
		customField.Options = pq.StringArray(req.Options)
	}

	if _, err := h.customFieldRepo.Create(c.Context(), customField); err != nil {
		return internalError(c, "Failed to create custom field", err)
	}

	response := v1schema.CustomFieldResponse{
		ID:             customField.ID,
		UserID:         customField.UserID,
		OrganizationID: customField.OrganizationID,
		Name:           customField.Name,
		NormalizedName: customField.NormalizedName,
		DataType:       customField.DataType,
		EntityType:     customField.EntityType,
		IsRequired:     customField.IsRequired,
		Options:        []string(customField.Options),
		SampleData:     customField.SampleData,
		FallbackValue:  customField.FallbackValue,
		CreatedAt:      customField.CreatedAt,
		UpdatedAt:      customField.UpdatedAt,
	}

	return c.Status(fiber.StatusCreated).JSON(schema.SuccessResponse(response))
}
