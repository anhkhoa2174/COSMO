package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/core"
	"github.com/rockship/cosmo-agents-go/internal/domain"

	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserRepository handles User entity operations
type UserRepository struct {
	*gormpkg.GormRepository[domain.User]
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		GormRepository: gormpkg.NewGormRepository[domain.User](db),
		db:             db,
	}
}

// GetDetailByID finds a user detail by ID
func (r *UserRepository) GetDetailByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	tx := core.DB(ctx, r.db)
	err := tx.Create(user).Error
	if err != nil {
		return nil, err
	}
	return user, nil
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	tx := core.DB(ctx, r.db)
	err := tx.Where("email = ? AND is_deleted = ?", email, false).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// FindWithOrganizations finds user (organizations should be loaded separately)
func (r *UserRepository) FindWithOrganizations(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&user).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// FindFirstOrganizationOfUser finds the first organization for a user
func (r *UserRepository) FindFirstOrganizationOfUser(ctx context.Context, userID uuid.UUID) (*domain.Organization, error) {
	var org domain.Organization
	tx := core.DB(ctx, r.db)

	err := tx.Where("user_id = ? AND is_deleted = ?", userID, false).
		Order("created_at ASC, id ASC").
		Limit(1).
		First(&org).Error

	if err == nil {
		return &org, nil
	}
	return nil, err
}

func (r *UserRepository) FindUserMainOrganization(ctx context.Context, userID uuid.UUID) (*domain.Organization, error) {
	var org domain.Organization
	tx := core.DB(ctx, r.db)

	// Step 1: Try to find organization where user is admin
	err := tx.Model(&domain.Organization{}).
		Joins("JOIN roles ON roles.organization_id = organizations.id").
		Where("roles.user_id = ? AND roles.name = ? AND organizations.is_deleted = ?", userID, domain.RoleNameAdmin, false).
		Limit(1).
		First(&org).Error

	if err == nil {
		return &org, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	} else {
		// Step 2: Fallback - find first organization created by user
		return r.FindFirstOrganizationOfUser(ctx, userID)
	}
}

// FindByIDs finds multiple users by their IDs (batch query to avoid N+1)
func (r *UserRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.User, error) {
	if len(ids) == 0 {
		return []*domain.User{}, nil
	}

	var users []*domain.User
	err := r.db.WithContext(ctx).
		Where("id IN ? AND is_deleted = ?", ids, false).
		Find(&users).Error

	return users, err
}

// UpdateByID performs a partial update on the user identified by ID.
func (r *UserRepository) UpdateByID(ctx context.Context, id uuid.UUID, attrs map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Updates(attrs).Error
}

// UpdateHubspotCredentials updates the HubSpot credentials for a user.
func (r *UserRepository) UpdateHubspotCredentials(ctx context.Context, id uuid.UUID, creds domain.JSON) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ? AND is_deleted = ?", id, false).
		Update("hubspot_credentials", creds).
		Error
}

// UpsertByEmail creates or updates a user identified by email
func (r *UserRepository) UpsertByEmail(ctx context.Context, user *domain.User) (*domain.User, error) {
	if user == nil || user.Email == "" {
		return nil, gorm.ErrInvalidData
	}

	tx := core.DB(ctx, r.db)

	err := tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "email"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"job_title", // Add more columns if you need
		}),
	}).
		Clauses(clause.Returning{}).
		Create(user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

// HardDelete hard deletes a user out of the database
func (r *UserRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Unscoped().
		Where("id = ?", id).
		Delete(&domain.User{}).Error
}
