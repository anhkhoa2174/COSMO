package common

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
)

// AuthHelper provides common authentication utilities
type AuthHelper struct {
	userRepo       *user.UserRepository
	roleRepo       *roleRepo.RoleRepository
	responseHelper *ResponseHelper
}

// NewAuthHelper creates a new auth helper
func NewAuthHelper(userRepo *user.UserRepository, roleRepo *roleRepo.RoleRepository) *AuthHelper {
	return &AuthHelper{
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		responseHelper: NewResponseHelper(),
	}
}

// UserContext represents user information extracted from request context
type UserContext struct {
	UserID         uuid.UUID
	OrganizationID *uuid.UUID
	User           *User
	Role           *Role
}

// User represents basic user information for auth context
type User struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
}

// Role represents user role in organization
type Role struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// GetUserID extracts user ID from Fiber context
func (h *AuthHelper) GetUserID(c fiber.Ctx) (uuid.UUID, error) {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return uuid.Nil, fiber.ErrUnauthorized
	}
	return userID, nil
}

// GetUserAndOrganization extracts user and organization ID from Fiber context
func (h *AuthHelper) GetUserAndOrganization(c fiber.Ctx) (uuid.UUID, *uuid.UUID, error) {
	userID, err := h.GetUserID(c)
	if err != nil {
		return uuid.Nil, nil, err
	}

	var orgID *uuid.UUID
	if orgIDValue, ok := c.Locals("organization_id").(uuid.UUID); ok && orgIDValue != uuid.Nil {
		orgID = &orgIDValue
	}

	return userID, orgID, nil
}

// GetFullUserContext loads full user context with organization role
func (h *AuthHelper) GetFullUserContext(c fiber.Ctx) (*UserContext, error) {
	userID, err := h.GetUserID(c)
	if err != nil {
		return nil, err
	}

	var orgID *uuid.UUID
	if orgIDValue, ok := c.Locals("organization_id").(uuid.UUID); ok && orgIDValue != uuid.Nil {
		orgID = &orgIDValue
	}

	// Load user
	user, err := h.userRepo.FindByID(c.Context(), userID)
	if err != nil {
		return nil, err
	}

	var role *domain.Role
	if orgID != nil {
		// Load user role in organization
		role, err = h.roleRepo.FindByUserAndOrganization(c.Context(), userID, *orgID)
		if err != nil {
			return nil, err
		}
	}

	var userPtr *User
	if user != nil {
		userPtr = &User{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		}
	}

	var rolePtr *Role
	if role != nil {
		rolePtr = &Role{
			ID:   role.ID,
			Name: string(role.Name),
		}
	}

	return &UserContext{
		UserID:         userID,
		OrganizationID: orgID,
		User:           userPtr,
		Role:           rolePtr,
	}, nil
}

// RequireAuthentication middleware helper
func (h *AuthHelper) RequireAuthentication(c fiber.Ctx) (uuid.UUID, error) {
	userID, err := h.GetUserID(c)
	if err != nil {
		_ = h.responseHelper.Unauthorized(c, "User not authenticated", nil)
		return uuid.Nil, err
	}
	return userID, nil
}

// RequireOrganizationAccess validates user has access to organization
func (h *AuthHelper) RequireOrganizationAccess(c fiber.Ctx, orgID uuid.UUID) (*UserContext, error) {
	userCtx, err := h.GetFullUserContext(c)
	if err != nil {
		_ = h.responseHelper.Unauthorized(c, "Failed to authenticate user", nil)
		return nil, err
	}

	// Check if user is member of the organization
	if userCtx.OrganizationID == nil || *userCtx.OrganizationID != orgID {
		_ = h.responseHelper.Forbidden(c, "Access denied to this organization", nil)
		return nil, fiber.ErrForbidden
	}

	return userCtx, nil
}

// ValidateOrganizationMembership checks if user belongs to specified organization
func (h *AuthHelper) ValidateOrganizationMembership(c fiber.Ctx, userID, orgID uuid.UUID) error {
	role, err := h.roleRepo.FindByUserAndOrganization(c.Context(), userID, orgID)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to verify organization access", err.Error())
	}

	if role == nil {
		return h.responseHelper.Forbidden(c, "Access denied to this organization", nil)
	}

	return nil
}

// HasRole checks if user has specific role in organization
func (h *AuthHelper) HasRole(userCtx *UserContext, requiredRole string) bool {
	if userCtx.Role == nil {
		return false
	}
	return userCtx.Role.Name == requiredRole
}

// HasAnyRole checks if user has any of the specified roles
func (h *AuthHelper) HasAnyRole(userCtx *UserContext, requiredRoles []string) bool {
	if userCtx.Role == nil {
		return false
	}

	for _, role := range requiredRoles {
		if userCtx.Role.Name == role {
			return true
		}
	}

	return false
}

// IsAdmin checks if user has admin role
func (h *AuthHelper) IsAdmin(userCtx *UserContext) bool {
	return h.HasRole(userCtx, "admin")
}

// IsOwner checks if user is owner of the resource
func (h *AuthHelper) IsOwner(resourceUserID, requestUserID uuid.UUID) bool {
	return resourceUserID == requestUserID
}

// CanAccessResource checks if user can access a resource (owner or organization member)
func (h *AuthHelper) CanAccessResource(resourceUserID, requestUserID uuid.UUID, resourceOrgID *uuid.UUID, requestOrgID *uuid.UUID) bool {
	// Owner can always access
	if h.IsOwner(resourceUserID, requestUserID) {
		return true
	}

	// Check organization access
	if resourceOrgID != nil && requestOrgID != nil {
		return *resourceOrgID == *requestOrgID
	}

	return false
}
