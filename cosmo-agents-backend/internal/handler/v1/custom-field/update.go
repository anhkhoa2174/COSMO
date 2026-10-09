package customfield

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/lib/pq"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// Update handles PATCH /v1/custom-fields/:id
// @Summary Update a custom field
// @Description Updates an existing custom field. Only provided fields will be updated. User must own the custom field.
// @Tags Custom Fields
// @Accept json
// @Produce json
// @Param id path string true "Custom field ID (UUID)"
// @Param request body v1schema.UpdateCustomFieldRequest true "Fields to update (all optional)"
// @Success 200 {object} schema.APIResponse[v1schema.CustomFieldResponse] "Custom field updated successfully"
// @Failure 400 {object} schema.APIResponse[any] "Invalid custom field ID or request body"
// @Failure 401 {object} schema.APIResponse[any] "User not authenticated"
// @Failure 403 {object} schema.APIResponse[any] "Access denied - user doesn't own this custom field"
// @Failure 404 {object} schema.APIResponse[any] "Custom field not found"
// @Failure 409 {object} schema.APIResponse[any] "A custom field with this name already exists"
// @Failure 500 {object} schema.APIResponse[any] "Failed to update custom field"
// @Router /v1/custom-fields/{id} [patch]
// @Security BearerAuth
func (h *Handler) Update(c fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return unauthorized(c, "User not authenticated")
	}

	customFieldID, err := parseUUIDParam(c, "id")
	if err != nil {
		return badRequest(c, "Invalid custom field ID", err)
	}

	var req v1schema.UpdateCustomFieldRequest
	if err := c.Bind().JSON(&req); err != nil {
		return badRequest(c, "Invalid request body", err)
	}

	if err := v1validation.ValidateStruct(&req); err != nil {
		return badRequest(c, "Validation failed", err)
	}

	// Get existing custom field
	customField, err := h.customFieldRepo.FindByID(c.Context(), customFieldID)
	if err != nil {
		return internalError(c, "Failed to fetch custom field", err)
	}

	if customField == nil {
		return notFound(c, "Custom field not found")
	}

	// Check ownership
	if customField.UserID != userID {
		return forbidden(c, "Access denied")
	}

	// Build update map with only fields that are being updated
	updates := make(map[string]interface{})

	// Validate the definition as it will look after the update, so that e.g.
	// switching to select without options or keeping a fallback that no longer
	// matches the new type is rejected.
	merged := *customField
	if req.Name != nil {
		merged.Name = strings.TrimSpace(*req.Name)
		if merged.Name == "" {
			return badRequest(c, "Name cannot be empty", nil)
		}
		updates["name"] = merged.Name
	}
	if req.DataType != nil {
		merged.DataType = *req.DataType
		updates["data_type"] = *req.DataType
	}
	if req.EntityType != nil {
		merged.EntityType = *req.EntityType
		updates["entity_type"] = *req.EntityType
	}
	if req.IsRequired != nil {
		updates["is_required"] = *req.IsRequired
	}
	if req.Options != nil {
		merged.Options = pq.StringArray(req.Options)
	}
	if req.SampleData != nil {
		merged.SampleData = req.SampleData
	}
	if req.FallbackValue != nil {
		merged.FallbackValue = req.FallbackValue
	}
	if err := validateDefinition(&merged); err != nil {
		return badRequest(c, err.Error(), nil)
	}
	if req.Options != nil || req.DataType != nil {
		updates["options"] = merged.Options
	}
	if req.Name != nil || req.EntityType != nil {
		if err := h.checkDuplicate(c.Context(), userID, customField.OrganizationID, merged.EntityType, merged.Name, customField.ID); err != nil {
			if errors.Is(err, errDuplicateName) {
				return conflict(c, err.Error())
			}
			return internalError(c, "Failed to check existing custom fields", err)
		}
	}
	// Moving a field to another entity keeps its stored key, so that key must
	// be free there too: a company field keyed "email" would otherwise become
	// a contact field colliding with the email column, or with another
	// contact field that already uses the key.
	if req.EntityType != nil && merged.EntityType != customField.EntityType {
		if err := validateNewKey(merged.EntityType, customField.NormalizedName); err != nil {
			return badRequest(c, err.Error(), nil)
		}
		if err := h.checkDuplicate(c.Context(), userID, customField.OrganizationID, merged.EntityType, customField.NormalizedName, customField.ID); err != nil {
			if errors.Is(err, errDuplicateName) {
				return conflict(c, err.Error())
			}
			return internalError(c, "Failed to check existing custom fields", err)
		}
	}
	if req.SampleData != nil {
		updates["sample_data"] = req.SampleData
	}
	if req.FallbackValue != nil {
		updates["fallback_value"] = req.FallbackValue
	}

	// If nothing to update, return current state
	if len(updates) == 0 {
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
		return c.JSON(schema.SuccessResponse(response))
	}

	// Perform update using map to avoid triggering BeforeUpdate on unchanged fields
	if err := h.customFieldRepo.UpdateFields(c.Context(), customFieldID, updates); err != nil {
		return internalError(c, "Failed to update custom field", err)
	}

	// Reload to get updated normalized_name and any other computed fields
	updatedField, err := h.customFieldRepo.FindByID(c.Context(), customFieldID)
	if err != nil || updatedField == nil {
		// If reload fails, use the current object
		updatedField = customField
	}

	response := v1schema.CustomFieldResponse{
		ID:             updatedField.ID,
		UserID:         updatedField.UserID,
		OrganizationID: updatedField.OrganizationID,
		Name:           updatedField.Name,
		NormalizedName: updatedField.NormalizedName,
		DataType:       updatedField.DataType,
		EntityType:     updatedField.EntityType,
		IsRequired:     updatedField.IsRequired,
		Options:        []string(updatedField.Options),
		SampleData:     updatedField.SampleData,
		FallbackValue:  updatedField.FallbackValue,
		CreatedAt:      updatedField.CreatedAt,
		UpdatedAt:      updatedField.UpdatedAt,
	}

	return c.JSON(schema.SuccessResponse(response))
}
