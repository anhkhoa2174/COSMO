package helper

import (
	"context"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/relations"
	"gorm.io/gorm"
)

// UserWithRelationsRepository provides methods to load users with their relationships
// This demonstrates how to use the relations package to get the data that was previously
// available as direct relationships on the User domain model
type UserWithRelationsRepository struct {
	db *gorm.DB
}

// NewUserWithRelationsRepository creates a new repository for loading users with relations
func NewUserWithRelationsRepository(db *gorm.DB) *UserWithRelationsRepository {
	return &UserWithRelationsRepository{db: db}
}

// GetUserWithOrganizations retrieves a user with their organizations using relations package
func (r *UserWithRelationsRepository) GetUserWithOrganizations(ctx context.Context, userID uuid.UUID) (*relations.UserWithOrganizations, error) {
	var userWithOrgs relations.UserWithOrganizations
	err := r.db.WithContext(ctx).
		Preload("Organizations").
		Where("id = ? AND is_deleted = ?", userID, false).
		First(&userWithOrgs).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &userWithOrgs, nil
}

// GetUserWithRoles retrieves a user with their roles using relations package
func (r *UserWithRelationsRepository) GetUserWithRoles(ctx context.Context, userID uuid.UUID) (*relations.UserWithRoles, error) {
	var userWithRoles relations.UserWithRoles
	err := r.db.WithContext(ctx).
		Preload("Roles").
		// Note: Roles.Organization preload removed as Role model no longer has direct relationships
		Where("id = ? AND is_deleted = ?", userID, false).
		First(&userWithRoles).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &userWithRoles, nil
}

// GetUserWithFullRelations retrieves a user with all their relationships using relations package
func (r *UserWithRelationsRepository) GetUserWithFullRelations(ctx context.Context, userID uuid.UUID) (*relations.UserWithFullRelations, error) {
	var userWithFullRelations relations.UserWithFullRelations
	err := r.db.WithContext(ctx).
		Preload("Organizations").
		Preload("Roles").
		// Note: Roles.Organization preload removed as Role model no longer has direct relationships
		Preload("Notifications").
		Preload("Agents").
		Where("id = ? AND is_deleted = ?", userID, false).
		First(&userWithFullRelations).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &userWithFullRelations, nil
}

// GetUsersWithOrganizations retrieves multiple users with their organizations
func (r *UserWithRelationsRepository) GetUsersWithOrganizations(ctx context.Context, userIDs []uuid.UUID) ([]*relations.UserWithOrganizations, error) {
	if len(userIDs) == 0 {
		return []*relations.UserWithOrganizations{}, nil
	}

	var usersWithOrgs []*relations.UserWithOrganizations
	err := r.db.WithContext(ctx).
		Preload("Organizations").
		Where("id IN ? AND is_deleted = ?", userIDs, false).
		Find(&usersWithOrgs).Error

	return usersWithOrgs, err
}

// GetUsersWithRoles retrieves multiple users with their roles
func (r *UserWithRelationsRepository) GetUsersWithRoles(ctx context.Context, userIDs []uuid.UUID) ([]*relations.UserWithRoles, error) {
	if len(userIDs) == 0 {
		return []*relations.UserWithRoles{}, nil
	}

	var usersWithRoles []*relations.UserWithRoles
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Where("id IN ? AND is_deleted = ?", userIDs, false).
		Find(&usersWithRoles).Error

	return usersWithRoles, err
}
