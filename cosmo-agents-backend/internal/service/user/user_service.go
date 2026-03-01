package user

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/relations"
	helperRepo "github.com/rockship/cosmo-agents-go/internal/repository/helper"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// UserWithRelationsService provides business logic for working with users and their relationships
type UserWithRelationsService struct {
	userRepo          *user.UserRepository
	userRelationsRepo *helperRepo.UserWithRelationsRepository
}

// NewUserWithRelationsService creates a new service for user relations
func NewUserWithRelationsService(
	userRepo *user.UserRepository,
	userRelationsRepo *helperRepo.UserWithRelationsRepository,
) *UserWithRelationsService {
	return &UserWithRelationsService{
		userRepo:          userRepo,
		userRelationsRepo: userRelationsRepo,
	}
}

// GetUserWithOrganizationsForResponse gets a user with organizations and converts to API response
// This replaces the previous functionality where User.Organizations was a direct field
func (s *UserWithRelationsService) GetUserWithOrganizationsForResponse(ctx context.Context, userID uuid.UUID) (v1schema.UserResponse, error) {
	// Load user with organizations using relations package
	userWithOrgs, err := s.userRelationsRepo.GetUserWithOrganizations(ctx, userID)
	if err != nil || userWithOrgs == nil {
		// Check if user exists at all
		user, err := s.userRepo.FindByID(ctx, userID)
		if err != nil || user == nil {
			return v1schema.UserResponse{}, fmt.Errorf("user not found")
		}
		// User exists but has no organizations - return user without orgs
		return v1schema.ToUserResponse(user), nil
	}

	// Convert the relations.UserWithOrganizations to regular schema response
	return s.convertUserWithOrgsToResponse(userWithOrgs), nil
}

// GetUserWithFullRelationsForResponse gets a user with all relationships and converts to API response
func (s *UserWithRelationsService) GetUserWithFullRelationsForResponse(ctx context.Context, userID uuid.UUID) (v1schema.UserResponse, error) {
	// Load user with all relationships using relations package
	userWithFullRelations, err := s.userRelationsRepo.GetUserWithFullRelations(ctx, userID)
	if err != nil || userWithFullRelations == nil {
		// Check if user exists at all
		user, err := s.userRepo.FindByID(ctx, userID)
		if err != nil || user == nil {
			return v1schema.UserResponse{}, fmt.Errorf("user not found")
		}
		// User exists but has no full relations - return user without relations
		return v1schema.ToUserResponse(user), nil
	}

	// Convert the relations.UserWithFullRelations to regular schema response
	return s.convertUserWithFullRelationsToResponse(userWithFullRelations), nil
}

// convertUserWithOrgsToResponse converts UserWithOrganizations to schema UserResponse
func (s *UserWithRelationsService) convertUserWithOrgsToResponse(userWithOrgs *relations.UserWithOrganizations) v1schema.UserResponse {
	// Convert the base user
	response := v1schema.ToUserResponse(&userWithOrgs.User)

	// Convert organizations
	orgs := make([]v1schema.OrganizationResponse, len(userWithOrgs.Organizations))
	for i, org := range userWithOrgs.Organizations {
		orgResponse := v1schema.ToOrganizationResponse(&org)
		orgs[i] = *orgResponse
	}
	response.Organizations = orgs

	return response
}

// convertUserWithFullRelationsToResponse converts UserWithFullRelations to schema UserResponse
func (s *UserWithRelationsService) convertUserWithFullRelationsToResponse(userWithFull *relations.UserWithFullRelations) v1schema.UserResponse {
	// Convert the base user
	response := v1schema.ToUserResponse(&userWithFull.User)

	// Convert organizations
	orgs := make([]v1schema.OrganizationResponse, len(userWithFull.Organizations))
	for i, org := range userWithFull.Organizations {
		orgResponse := v1schema.ToOrganizationResponse(&org)
		orgs[i] = *orgResponse
	}
	response.Organizations = orgs

	// Convert roles
	roles := make([]v1schema.RoleResponse, len(userWithFull.Roles))
	for i, role := range userWithFull.Roles {
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
	notifications := make([]v1schema.NotificationResponse, len(userWithFull.Notifications))
	for i, notification := range userWithFull.Notifications {
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
