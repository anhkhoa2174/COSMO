package middleware

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"gorm.io/gorm"
)

var (
	ErrUnauthorized  = errors.New("unauthorized")
	ErrUserNotFound  = errors.New("user not found")
	ErrNoRolesFound  = errors.New("no roles found for user")
	ErrInvalidUserID = errors.New("invalid user_id format")
)

// AuthHelper provides helper functions for authentication and authorization
type AuthHelper struct {
	userRepo *user.UserRepository
	roleRepo *roleRepo.RoleRepository
}

// NewAuthHelper creates a new AuthHelper instance
func NewAuthHelper(userRepo *user.UserRepository, roleRepo *roleRepo.RoleRepository) *AuthHelper {
	return &AuthHelper{
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

func (h *AuthHelper) GetUserIDAndRoles(c fiber.Ctx) (uuid.UUID, []domain.Role, error) {
	userID, err := h.GetUserID(c)
	if err != nil {
		return uuid.Nil, nil, err
	}
	roles, err := h.roleRepo.FindByUserID(c.Context(), userID)
	if err != nil {
		return uuid.Nil, nil, err
	}
	return userID, roles, nil
}

// GetUserAndOrganization extracts user and organization from the request context
// This eliminates duplicate auth logic across handlers
func (h *AuthHelper) GetUserAndOrganization(c fiber.Ctx) (*domain.User, uuid.UUID, error) {
	// Get user_id from locals (set by auth middleware)
	userID := c.Locals("user_id")
	if userID == nil {
		return nil, uuid.Nil, ErrUnauthorized
	}

	// Type assertion
	uid, ok := userID.(uuid.UUID)
	if !ok {
		return nil, uuid.Nil, ErrInvalidUserID
	}

	// Find user
	user, err := h.userRepo.FindByID(c.Context(), uid)
	if err != nil {
		return nil, uuid.Nil, err
	} else if user == nil {
		return nil, uuid.Nil, ErrUserNotFound
	}

	// Get user's organization (prefer admin org, fallback to any org with role)
	orgID, err := h.getUserOrganization(c.Context(), user.ID)
	if err != nil {
		return nil, uuid.Nil, err
	}
	return user, orgID, nil
}

// getUserOrganization gets the organization for a user
// Priority: 1) Admin organization, 2) Created organization, 3) Any organization via role
func (h *AuthHelper) getUserOrganization(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	// Step 1: Try FindUserMainOrganization (admin org or created org)
	org, err := h.userRepo.FindUserMainOrganization(ctx, userID)
	if err == nil {
		// The admin lookup joins roles without checking is_deleted, so a
		// removed admin would still resolve here. Keep the org only if the
		// membership is live or the user created it.
		role, roleErr := h.roleRepo.FindByUserAndOrganization(ctx, userID, org.ID)
		if roleErr != nil {
			return uuid.Nil, roleErr
		}
		if role != nil || (org.UserID != nil && *org.UserID == userID) {
			return org.ID, nil
		}
		err = gorm.ErrRecordNotFound
	}

	// Step 2: Fallback to ANY organization where user has a live role.
	// Removed members keep a soft-deleted role row, which must not count.
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Admin roles first, then the oldest membership: taking the first row
		// of an unordered list could pick an org where the user is only a
		// member over one they administer.
		orgID, roleErr := h.roleRepo.FindPrimaryOrganization(ctx, userID)
		if roleErr != nil {
			return uuid.Nil, roleErr
		}
		if orgID == nil {
			return uuid.Nil, ErrNoRolesFound
		}
		return *orgID, nil
	}

	return uuid.Nil, err
}

// GetUserID extracts only the user ID from context
func (h *AuthHelper) GetUserID(c fiber.Ctx) (uuid.UUID, error) {
	userID := c.Locals("user_id")
	if userID == nil {
		return uuid.Nil, ErrUnauthorized
	}

	uid, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, ErrInvalidUserID
	}

	return uid, nil
}

// GetOrganizationID gets the organization ID for the authenticated user
func (h *AuthHelper) GetOrganizationID(c fiber.Ctx) (uuid.UUID, error) {
	_, orgID, err := h.GetUserAndOrganization(c)
	return orgID, err
}

// IsAdminInOrganization checks if a user has admin role in the specified organization
func (h *AuthHelper) IsAdminInOrganization(ctx context.Context, userID, organizationID uuid.UUID) bool {
	role, err := h.roleRepo.FindByUserAndOrganization(ctx, userID, organizationID)
	if err != nil || role == nil {
		return false
	}
	return role.Name == "admin"
}
