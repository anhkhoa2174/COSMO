package organization

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/repository/organization"
	roleRepo "github.com/rockship/cosmo-agents-go/internal/repository/role"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"gorm.io/gorm"
)

// Service provides organization business logic
type Service struct {
	orgRepo  *organization.OrganizationRepository
	userRepo *user.UserRepository
	roleRepo *roleRepo.RoleRepository
	db       *gorm.DB
}

// NewService creates a new organization service
func NewService(
	orgRepo *organization.OrganizationRepository,
	userRepo *user.UserRepository,
	roleRepo *roleRepo.RoleRepository,
	db *gorm.DB,
) *Service {
	return &Service{
		orgRepo:  orgRepo,
		userRepo: userRepo,
		roleRepo: roleRepo,
		db:       db,
	}
}

// CreateOrganizationRequest represents the request to create an organization
type CreateOrganizationRequest struct {
	UserID                  uuid.UUID `json:"user_id" validate:"required"`
	Name                    string    `json:"name" validate:"required,min=1,max=255"`
	CompanyURL              string    `json:"company_url" validate:"omitempty,url,max=500"`
	CompanyDescription      string    `json:"company_description" validate:"omitempty,max=1000"`
	CompanyTargetingPersona []string  `json:"company_targeting_persona"`
	ValueOffering           string    `json:"value_offering" validate:"omitempty,max=1000"`
}

// UpdateOrganizationRequest represents the request to update an organization
type UpdateOrganizationRequest struct {
	Name                    string   `json:"name" validate:"omitempty,min=1,max=255"`
	CompanyURL              string   `json:"company_url" validate:"omitempty,url,max=500"`
	CompanyDescription      string   `json:"company_description" validate:"omitempty,max=1000"`
	CompanyTargetingPersona []string `json:"company_targeting_persona"`
	ValueOffering           string   `json:"value_offering" validate:"omitempty,max=1000"`
}

// CreateOrganization creates a new organization
func (s *Service) CreateOrganization(ctx context.Context, req CreateOrganizationRequest) (*domain.Organization, error) {
	// Validate user exists
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	// Create organization
	org := &domain.Organization{
		UserID:                  &req.UserID,
		Name:                    req.Name,
		CompanyURL:              req.CompanyURL,
		CompanyDescription:      req.CompanyDescription,
		CompanyTargetingPersona: pq.StringArray(req.CompanyTargetingPersona),
		ValueOffering:           req.ValueOffering,
	}

	if _, err := s.orgRepo.Create(ctx, org); err != nil {
		logger.Logger.Error().Err(err).Str("user_id", req.UserID.String()).Msg("failed to create organization")
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	logger.Logger.Info().Str("org_id", org.ID.String()).Str("user_id", req.UserID.String()).Str("name", req.Name).Msg("organization created successfully")
	return org, nil
}

// GetOrganizationByID gets an organization by ID
func (s *Service) GetOrganizationByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	org, err := s.orgRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("organization not found")
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return org, nil
}

// UpdateOrganization updates an organization
func (s *Service) UpdateOrganization(ctx context.Context, id uuid.UUID, req UpdateOrganizationRequest) (*domain.Organization, error) {
	// Get existing organization
	org, err := s.orgRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("organization not found")
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	// Update fields if provided
	if req.Name != "" {
		org.Name = req.Name
	}
	if req.CompanyURL != "" {
		org.CompanyURL = req.CompanyURL
	}
	if req.CompanyDescription != "" {
		org.CompanyDescription = req.CompanyDescription
	}
	if len(req.CompanyTargetingPersona) > 0 {
		org.CompanyTargetingPersona = pq.StringArray(req.CompanyTargetingPersona)
	}
	if req.ValueOffering != "" {
		org.ValueOffering = req.ValueOffering
	}

	// Update timestamp
	org.UpdatedAt = time.Now()

	if err := s.orgRepo.Update(ctx, id, org); err != nil {
		logger.Logger.Error().Err(err).Str("org_id", id.String()).Msg("failed to update organization")
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	logger.Logger.Info().Str("org_id", id.String()).Msg("organization updated successfully")
	return org, nil
}

// DeleteOrganization soft deletes an organization
func (s *Service) DeleteOrganization(ctx context.Context, id uuid.UUID) error {
	// Check if organization exists
	_, err := s.orgRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("organization not found")
		}
		return fmt.Errorf("failed to get organization: %w", err)
	}

	// Soft delete
	if err := s.orgRepo.Delete(ctx, id); err != nil {
		logger.FromContext(ctx).Error().Err(err).Str("org_id", id.String()).Msg("failed to delete organization")
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	logger.FromContext(ctx).Info().Str("org_id", id.String()).Msg("organization deleted successfully")
	return nil
}

// GetOrganizationsByUserID gets all organizations for a user
func (s *Service) GetOrganizationsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Organization, error) {
	orgs, err := s.orgRepo.FindByUserID(ctx, userID)
	if err != nil {
		logger.FromContext(ctx).Error().Err(err).Str("user_id", userID.String()).Msg("failed to get organizations by user ID")
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}
	return orgs, nil
}

// ListOrganizations lists organizations with pagination
func (s *Service) ListOrganizations(ctx context.Context, limit, offset int) ([]domain.Organization, int64, error) {
	pagination := &baseRepo.PaginationParams{
		Limit:  limit,
		Offset: offset,
	}
	result, err := s.orgRepo.FindAll(ctx, nil, pagination)
	if err != nil {
		logger.FromContext(ctx).Error().Err(err).Msg("failed to list organizations")
		return nil, 0, fmt.Errorf("failed to list organizations: %w", err)
	}
	return result.List, result.Total, nil
}

// AddUserToOrganization adds a user to an organization with a role
func (s *Service) AddUserToOrganization(ctx context.Context, orgID, userID uuid.UUID, roleName domain.RoleName) (*domain.Role, error) {
	// Validate organization exists
	_, err := s.orgRepo.FindByID(ctx, orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("organization not found")
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	// Validate user exists
	_, err = s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user already has a role in this organization
	existingRole, err := s.roleRepo.FindByUserAndOrganization(ctx, userID, orgID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing role: %w", err)
	}
	if existingRole != nil {
		return nil, errors.New("user already has a role in this organization")
	}

	// Create role
	role := &domain.Role{
		UserID:         userID,
		OrganizationID: orgID,
		Name:           roleName,
		Status:         domain.RoleStatusPending,
	}

	if _, err := s.roleRepo.Create(ctx, role); err != nil {
		logger.FromContext(ctx).Error().Err(err).Str("user_id", userID.String()).Str("org_id", orgID.String()).Str("role", string(roleName)).Msg("failed to create role")
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	logger.FromContext(ctx).Info().Str("user_id", userID.String()).Str("org_id", orgID.String()).Str("role_id", role.ID.String()).Msg("user added to organization")
	return role, nil
}

// RemoveUserFromOrganization removes a user from an organization
func (s *Service) RemoveUserFromOrganization(ctx context.Context, orgID, userID uuid.UUID) error {
	// Get user's role in organization
	role, err := s.roleRepo.FindByUserAndOrganization(ctx, userID, orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user role not found in organization")
		}
		return fmt.Errorf("failed to get user role: %w", err)
	}

	// Delete role
	if err := s.roleRepo.Delete(ctx, role.ID); err != nil {
		logger.FromContext(ctx).Error().Err(err).Str("user_id", userID.String()).Str("org_id", orgID.String()).Msg("failed to remove user from organization")
		return fmt.Errorf("failed to remove user from organization: %w", err)
	}

	logger.FromContext(ctx).Info().Str("user_id", userID.String()).Str("org_id", orgID.String()).Msg("user removed from organization")
	return nil
}

// GetOrganizationUsers gets all users in an organization
func (s *Service) GetOrganizationUsers(ctx context.Context, orgID uuid.UUID) ([]domain.User, error) {
	var users []domain.User
	var roles []domain.Role

	// Get all roles for this organization
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND is_deleted = ?", orgID, false).
		Find(&roles).Error

	if err != nil {
		logger.FromContext(ctx).Error().Err(err).Str("org_id", orgID.String()).Msg("failed to get organization roles")
		return nil, fmt.Errorf("failed to get organization users: %w", err)
	}

	// Get unique user IDs from roles
	userIDs := make([]uuid.UUID, 0, len(roles))
	userIDMap := make(map[uuid.UUID]bool)
	for _, role := range roles {
		if !userIDMap[role.UserID] {
			userIDs = append(userIDs, role.UserID)
			userIDMap[role.UserID] = true
		}
	}

	// Get users for those IDs
	if len(userIDs) > 0 {
		err = s.db.WithContext(ctx).
			Where("id IN ? AND is_deleted = ?", userIDs, false).
			Find(&users).Error

		if err != nil {
			logger.FromContext(ctx).Error().Err(err).Str("org_id", orgID.String()).Msg("failed to get organization users")
			return nil, fmt.Errorf("failed to get organization users: %w", err)
		}
	}

	return users, nil
}

// GetUserOrganizations gets all organizations for a user with their roles
func (s *Service) GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]domain.Organization, error) {
	pagination := &baseRepo.PaginationParams{
		Limit:  1000, // Large limit to get all organizations
		Offset: 0,
	}
	orgs, _, err := s.roleRepo.GetOrganizations(ctx, userID, pagination)
	if err != nil {
		logger.FromContext(ctx).Error().Err(err).Str("user_id", userID.String()).Msg("failed to get user organizations")
		return nil, fmt.Errorf("failed to get user organizations: %w", err)
	}
	return orgs, nil
}
