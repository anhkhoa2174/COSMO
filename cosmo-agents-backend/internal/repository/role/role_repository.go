package role

import (
	"context"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/core"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RoleRepository handles Role entity operations
type RoleRepository struct {
	*gormpkg.GormRepository[domain.Role]
	db *gorm.DB
}

// NewRoleRepository creates a new role repository
func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{
		GormRepository: gormpkg.NewGormRepository[domain.Role](db),
		db:             db,
	}
}

// Upsert creates or updates a role based on unique constraints
func (r *RoleRepository) Upsert(ctx context.Context, role *domain.Role) (*domain.Role, error) {
	tx := core.DB(ctx, r.db)
	role.IsDeleted = false

	// Use GORM's Clauses with OnConflict for upsert
	err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "organization_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"job_title",
			"status",
			"is_deleted",
			"updated_at",
		}),
	}).
		Create(role).Error

	if err != nil {
		return nil, err
	}

	return role, nil
}

// FindByUserID finds all roles for a user
func (r *RoleRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error) {
	var roles []domain.Role
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&roles).Error

	if err != nil {
		return nil, err
	}

	return roles, nil
}

// FindByUserAndOrganization finds a role by user ID and organization ID
func (r *RoleRepository) FindByUserAndOrganization(ctx context.Context, userID, orgID uuid.UUID) (*domain.Role, error) {
	var role domain.Role
	tx := core.DB(ctx, r.db)
	err := tx.Where("user_id = ? AND organization_id = ? AND is_deleted = ?", userID, orgID, false).
		First(&role).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &role, nil
}

// ExistsByOrgAndUser checks if a role exists for user in organization
func (r *RoleRepository) ExistsByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID, roleName domain.RoleName) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.Role{}).
		Where("organization_id = ? AND user_id = ? AND name = ? AND is_deleted = ?", orgID, userID, roleName, false).
		Count(&count).Error

	return count > 0, err
}

// FindPrimaryOrganization returns the organization ID where the user is admin or first active membership.
func (r *RoleRepository) FindPrimaryOrganization(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	var adminRole domain.Role
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND name = ? AND is_deleted = ?", userID, domain.RoleNameAdmin, false).
		Order("created_at ASC").
		First(&adminRole).Error

	if err == nil {
		return &adminRole.OrganizationID, nil
	}

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	var anyRole domain.Role
	err = r.db.WithContext(ctx).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Order("created_at ASC").
		First(&anyRole).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &anyRole.OrganizationID, nil
}

// GetOrganizations returns paginated organizations for a user with preloaded organization details
func (r *RoleRepository) GetOrganizations(ctx context.Context, userID uuid.UUID, pagination *baseRepo.PaginationParams) ([]domain.Organization, int64, error) {
	var roles []domain.Role
	var total int64

	// Count total records first
	err := r.db.WithContext(ctx).
		Model(&domain.Role{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&total).Error

	if err != nil {
		return nil, 0, err
	}

	// Get paginated results without preloaded organization
	// Note: Organization preload removed as Role model no longer has direct relationships
	err = r.db.WithContext(ctx).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Offset(pagination.Offset * pagination.Limit).
		Limit(pagination.Limit).
		Order("updated_at DESC").
		Find(&roles).Error

	if err != nil {
		return nil, 0, err
	}

	// Note: Direct Organization relationship removed from Role domain
	// Need to load organizations separately using relations package
	var organizations []domain.Organization
	orgIDs := make([]uuid.UUID, 0, len(roles))
	orgMap := make(map[uuid.UUID]domain.Organization)

	for _, role := range roles {
		if role.OrganizationID != uuid.Nil {
			orgIDs = append(orgIDs, role.OrganizationID)
		}
	}

	// Load organizations by their IDs if needed
	if len(orgIDs) > 0 {
		var orgs []domain.Organization
		err := r.db.WithContext(ctx).
			Where("id IN ? AND is_deleted = ?", orgIDs, false).
			Find(&orgs).Error
		if err == nil {
			for _, org := range orgs {
				orgMap[org.ID] = org
			}
		}
	}

	// Convert map to slice
	for _, org := range orgMap {
		organizations = append(organizations, org)
	}

	return organizations, total, nil
}
