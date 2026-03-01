package user

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/relations"
	v1validation "github.com/rockship/cosmo-agents-go/internal/handler/v1/validation"
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	helperRepo "github.com/rockship/cosmo-agents-go/internal/repository/helper"
	"github.com/rockship/cosmo-agents-go/internal/repository/organization"
	personalApiKeyRepo "github.com/rockship/cosmo-agents-go/internal/repository/personal_api_key"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userRepo          *user.UserRepository
	apiKeyRepo        *personalApiKeyRepo.PersonalApiKeyRepository
	orgRepo           *organization.OrganizationRepository
	roleRepo          *roleRepo.RoleRepository
	userRelationsRepo *helperRepo.UserWithRelationsRepository
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userRepo *user.UserRepository, apiKeyRepo *personalApiKeyRepo.PersonalApiKeyRepository, orgRepo *organization.OrganizationRepository, db *gorm.DB) *UserHandler {
	return &UserHandler{
		userRepo:          userRepo,
		apiKeyRepo:        apiKeyRepo,
		orgRepo:           orgRepo,
		roleRepo:          roleRepo.NewRoleRepository(db),
		userRelationsRepo: helperRepo.NewUserWithRelationsRepository(db),
	}
}

// GetCurrentUser handles GET /v1/users/me
// @Summary Get current authenticated user
// @Description Get information about the currently authenticated user
// @Tags Users V1
// @Accept json
// @Produce json
// @Success 200 {object} schema.APIResponse[v1schema.UserResponse] "Current user information"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized - User not authenticated"
// @Failure 404 {object} schema.APIResponse[any] "User not found"
// @Security BearerAuth
// @Router /v1/users/me [get]
func (h *UserHandler) GetCurrentUser(c fiber.Ctx) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"User not authenticated",
		))
	}

	// Get the basic user
	user, err := h.userRepo.GetDetailByID(c.Context(), userID)
	if user == nil || err == gorm.ErrRecordNotFound {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound,
			"User not found",
			"",
		))
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to get user",
			err.Error(),
		))
	}

	// Query organizations directly from repository
	organizations, err := h.orgRepo.FindByUserID(c.Context(), userID)
	if err != nil {
		fmt.Printf("Failed to load user organizations: %v\n", err)
		organizations = []domain.Organization{}
	}

	// Query roles directly from repository
	roles, err := h.roleRepo.FindByUserID(c.Context(), userID)
	if err != nil {
		fmt.Printf("Failed to load user roles: %v\n", err)
		roles = []domain.Role{}
	}

	// Convert to response
	response := v1schema.ToUserResponse(user)

	// Add organizations
	orgs := make([]v1schema.OrganizationResponse, len(organizations))
	for i, org := range organizations {
		orgResponse := v1schema.ToOrganizationResponse(&org)
		orgs[i] = *orgResponse
	}
	response.Organizations = orgs

	// Add roles
	roleResponses := make([]v1schema.RoleResponse, len(roles))
	for i, role := range roles {
		roleResponses[i] = v1schema.RoleResponse{
			ID:             role.ID,
			UserID:         role.UserID,
			OrganizationID: role.OrganizationID,
			Name:           string(role.Name),
			JobTitle:       role.JobTitle,
			Status:         string(role.Status),
			CreatedAt:      role.CreatedAt,
			UpdatedAt:      role.UpdatedAt,
		}
	}
	response.Roles = roleResponses

	fmt.Printf("Found %d organizations, %d roles for user %s\n",
		len(response.Organizations), len(response.Roles), userID.String())
	return c.JSON(schema.SuccessResponse(response))
}

// UpdateCurrentUser handles PATCH /v1/users/me
// @Summary Update current authenticated user
// @Description Update information about the currently authenticated user
// @Tags Users V1
// @Accept json
// @Produce json
// @Param request body v1schema.UpdateUserRequest true "User update information"
// @Success 200 {object} schema.APIResponse[v1schema.UserResponse] "Updated user information"
// @Failure 400 {object} schema.APIResponse[any] "Bad request - Invalid input"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized - User not authenticated"
// @Failure 404 {object} schema.APIResponse[any] "User not found"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error - Failed to update user"
// @Security BearerAuth
// @Router /v1/users/me [patch]
func (h *UserHandler) UpdateCurrentUser(c fiber.Ctx) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"User not authenticated",
		))
	}

	var req v1schema.UpdateUserRequest
	if err := v1validation.ValidateRequest(c, &req); err != nil {
		return err
	}

	updateData := map[string]interface{}{}
	if req.Email != nil {
		updateData["email"] = *req.Email
	}
	if req.Name != nil {
		updateData["name"] = *req.Name
	}
	if req.Picture != nil {
		updateData["picture"] = *req.Picture
	}
	if req.Provider != nil {
		updateData["provider"] = *req.Provider
	}
	if req.Credentials != nil {
		bytes, err := json.Marshal(req.Credentials)
		if err != nil {
			return fmt.Errorf("invalid credentials: %w", err)
		}
		updateData["credentials"] = bytes
	}
	if req.UIMetadata != nil {
		bytes, err := json.Marshal(req.UIMetadata)
		if err != nil {
			return fmt.Errorf("invalid ui_metadata: %w", err)
		}
		updateData["ui_metadata"] = bytes
	}

	if len(updateData) > 0 {
		if err := h.userRepo.GetDB().WithContext(c.Context()).
			Model(&domain.User{}).
			Where("id = ? AND is_deleted = ?", userID, false).
			Updates(updateData).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
				fiber.StatusInternalServerError,
				"Failed to update user",
				err.Error(),
			))
		}
	}

	user, err := h.userRepo.FindByID(c.Context(), userID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound,
			"User not found",
			"",
		))
	}

	return c.JSON(schema.SuccessResponse(v1schema.ToUserResponse(user)))
}

// DeleteCurrentUser handles DELETE /v1/users/me
// @Summary Delete current authenticated user
// @Description Delete the currently authenticated user's account
// @Tags Users V1
// @Accept json
// @Produce json
// @Success 200 {object} schema.APIResponse[string] "User deleted successfully"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized - User not authenticated"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error - Failed to delete user"
// @Security BearerAuth
// @Router /v1/users/me [delete]
func (h *UserHandler) DeleteCurrentUser(c fiber.Ctx) error {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized,
			"Unauthorized",
			"User not authenticated",
		))
	}

	if err := h.userRepo.HardDelete(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError,
			"Failed to delete user",
			err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse([]string{userID.String()}))
}

// CreatePersonalAPIKey handles POST /v1/users/:user_id/personal-api-keys
// @Summary Create personal API key
// @Description Create a new personal API key for user
// @Tags Users V1
// @Accept json
// @Produce json
// @Param user_id path string true "User ID (UUID)"
// @Param body body domain.CreatePersonalAPIKeyRequest true "API key details"
// @Success 200 {object} map[string]interface{} "Created API key with raw key"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Security BearerAuth
// @Router /v1/users/{user_id}/personal-api-keys [post]
func (h *UserHandler) CreatePersonalAPIKey(c fiber.Ctx) error {
	// Get current user
	currentUserID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "Unauthorized", "User ID not found",
		))
	}

	// Parse user_id from path
	userIDParam := c.Params("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid user ID format", err.Error(),
		))
	}

	// Check authorization (user can only create keys for themselves)
	if currentUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "Not allowed", "Cannot create API keys for other users",
		))
	}

	// Parse request body
	var req domain.CreatePersonalAPIKeyRequest
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

	// Generate API key
	rawKey, hashedKey, err := h.apiKeyRepo.GenerateAPIKey()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to generate API key", err.Error(),
		))
	}

	// Create API key record
	expiresAt := time.Now().AddDate(0, 6, 0) // Default 6 months.
	if req.ExpiresAt != nil && req.ExpiresAt.Valid {
		expiresAt = req.ExpiresAt.Time
	}

	apiKey := &domain.PersonalApiKey{
		UserID:    userID,
		Name:      req.Name,
		HashedKey: hashedKey,
		Prefix:    rawKey[:20],
		ExpiresAt: expiresAt,
	}

	if _, err := h.apiKeyRepo.Create(c.Context(), apiKey); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to create API key", err.Error(),
		))
	}

	// Return response with raw key (only shown once!)
	response := fiber.Map{
		"raw_key": rawKey,
		"entity": fiber.Map{
			"id":         apiKey.ID,
			"user_id":    apiKey.UserID,
			"name":       apiKey.Name,
			"prefix":     apiKey.Prefix,
			"expires_at": apiKey.ExpiresAt,
			"created_at": apiKey.CreatedAt,
		},
	}

	return c.JSON(schema.SuccessResponse(response))
}

// ListPersonalAPIKeys handles GET /v1/users/:user_id/personal-api-keys
// @Summary List personal API keys
// @Description Get all personal API keys for a user
// @Tags Users V1
// @Accept json
// @Produce json
// @Param user_id path string true "User ID (UUID)"
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit" default(10)
// @Success 200 {object} schema.APIResponse[schema.PaginatedResponse[v1schema.PersonalApiKeyResponse]] "List of API keys"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Security BearerAuth
// @Router /v1/users/{user_id}/personal-api-keys [get]
func (h *UserHandler) ListPersonalAPIKeys(c fiber.Ctx) error {
	// Get current user
	currentUserID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "Unauthorized", "User ID not found",
		))
	}

	// Parse user_id from path
	userIDParam := c.Params("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid user ID format", err.Error(),
		))
	}

	// Check authorization
	if currentUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "Not allowed", "Cannot list API keys for other users",
		))
	}

	// Parse pagination
	offset := 0
	limit := 10
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil {
			offset = val
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val <= 50 {
			limit = val
		}
	}

	pagination := &baseRepo.PaginationParams{
		Offset: offset,
		Limit:  limit,
	}

	result, err := h.apiKeyRepo.FindByUserID(c.Context(), userID, pagination)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to fetch API keys", err.Error(),
		))
	}

	// Convert to response (exclude hashed_key)
	keyResponses := make([]*v1schema.PersonalApiKeyResponse, len(result.List))
	for i, key := range result.List {
		resp := v1schema.ToPersonalApiKeyResponse(&key)
		keyResponses[i] = &resp
	}

	paginatedResponse := schema.PaginatedResponse[*v1schema.PersonalApiKeyResponse]{
		List:   keyResponses,
		Total:  result.Total,
		Offset: result.Offset,
		Limit:  result.Limit,
	}

	return c.JSON(schema.SuccessResponse(paginatedResponse))
}

// DeletePersonalAPIKey handles DELETE /v1/users/:user_id/personal-api-keys/:key_id
// @Summary Delete personal API key
// @Description Delete a personal API key
// @Tags Users V1
// @Accept json
// @Produce json
// @Param user_id path string true "User ID (UUID)"
// @Param key_id path string true "API Key ID (UUID)"
// @Success 200 {object} schema.APIResponse[string] "Deleted"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Security BearerAuth
// @Router /v1/users/{user_id}/personal-api-keys/{key_id} [delete]
func (h *UserHandler) DeletePersonalAPIKey(c fiber.Ctx) error {
	// Get current user
	currentUserID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(schema.ErrorResponse(
			fiber.StatusUnauthorized, "Unauthorized", "User ID not found",
		))
	}

	// Parse user_id from path
	userIDParam := c.Params("user_id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid user ID format", err.Error(),
		))
	}

	// Check authorization
	if currentUserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "Not allowed", "Cannot delete API keys for other users",
		))
	}

	// Parse key_id
	keyIDParam := c.Params("key_id")
	keyID, err := uuid.Parse(keyIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(schema.ErrorResponse(
			fiber.StatusBadRequest, "Invalid key ID format", err.Error(),
		))
	}

	// Verify key belongs to user
	key, err := h.apiKeyRepo.FindByID(c.Context(), keyID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(schema.ErrorResponse(
			fiber.StatusNotFound, "API key not found", err.Error(),
		))
	}

	if key.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(schema.ErrorResponse(
			fiber.StatusForbidden, "Not allowed", "API key does not belong to user",
		))
	}

	// Delete key. Because real database doesn't have is_deleted, therefore python version is hard deleting
	if err := h.apiKeyRepo.Delete(c.Context(), keyID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(schema.ErrorResponse(
			fiber.StatusInternalServerError, "Failed to delete API key", err.Error(),
		))
	}

	return c.JSON(schema.SuccessResponse("Deleted"))
}

// convertUserWithRelationsToResponse converts UserWithFullRelations to UserResponse with relations data
func (h *UserHandler) convertUserWithRelationsToResponse(userWithFullRelations *relations.UserWithFullRelations) v1schema.UserResponse {
	// Convert the base user
	response := v1schema.ToUserResponse(&userWithFullRelations.User)

	// Convert organizations
	orgs := make([]v1schema.OrganizationResponse, len(userWithFullRelations.Organizations))
	for i, org := range userWithFullRelations.Organizations {
		orgResponse := v1schema.ToOrganizationResponse(&org)
		orgs[i] = *orgResponse
	}
	response.Organizations = orgs

	// Convert roles
	roles := make([]v1schema.RoleResponse, len(userWithFullRelations.Roles))
	for i, role := range userWithFullRelations.Roles {
		roles[i] = v1schema.RoleResponse{
			ID:             role.ID,
			UserID:         role.UserID,
			OrganizationID: role.OrganizationID,
			Name:           string(role.Name),
			JobTitle:       role.JobTitle,
			Status:         string(role.Status),
			CreatedAt:      role.CreatedAt,
			UpdatedAt:      role.UpdatedAt,
		}
	}
	response.Roles = roles

	// Convert notifications
	notifications := make([]v1schema.NotificationResponse, len(userWithFullRelations.Notifications))
	for i, notification := range userWithFullRelations.Notifications {
		notifications[i] = v1schema.NotificationResponse{
			ID:         notification.ID,
			UserID:     notification.UserID,
			CampaignID: notification.CampaignID,
			CreatedAt:  notification.CreatedAt,
			UpdatedAt:  notification.UpdatedAt,
		}
	}
	response.Notifications = notifications

	return response
}

// getUserOrganizations loads organizations for a user
func (h *UserHandler) getUserOrganizations(ctx context.Context, userID uuid.UUID) ([]v1schema.OrganizationResponse, error) {
	org, err := h.userRepo.FindFirstOrganizationOfUser(ctx, userID)
	if err != nil || org == nil {
		return []v1schema.OrganizationResponse{}, nil
	}

	orgResponse := v1schema.ToOrganizationResponse(org)
	return []v1schema.OrganizationResponse{*orgResponse}, nil
}

// getUserRoles loads roles for a user
func (h *UserHandler) getUserRoles(ctx context.Context, userID uuid.UUID) ([]v1schema.RoleResponse, error) {
	// This is a simplified approach - in a real implementation you'd want to
	// query the role repository directly
	// For now, return empty since we don't have direct access to role queries
	return []v1schema.RoleResponse{}, nil
}
