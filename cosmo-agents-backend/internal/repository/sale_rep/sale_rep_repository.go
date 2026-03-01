package salerep

import (
	"context"

	"gorm.io/gorm"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	filterPkg "github.com/rockship/cosmo-agents-go/internal/repository/filter"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
)

// SaleRepRepository handles sale rep data operations.
type SaleRepRepository struct {
	*gormpkg.GormRepository[domain.SaleRep]
	db *gorm.DB
}

// NewSaleRepRepository creates a new sale rep gormpkg.
func NewSaleRepRepository(db *gorm.DB) *SaleRepRepository {
	return &SaleRepRepository{
		GormRepository: gormpkg.NewGormRepository[domain.SaleRep](db),
		db:             db,
	}
}

// GetByUserIDAndEmail finds a sale rep by user ID and email
func (r *SaleRepRepository) GetByUserIDAndEmail(ctx context.Context, userID uuid.UUID, email string) (*domain.SaleRep, error) {
	var saleRep domain.SaleRep
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND email = ?", userID, email).
		First(&saleRep).Error

	if err != nil {
		return nil, err
	}

	return &saleRep, nil
}

// GetByIDs finds multiple sale reps by IDs.
func (r *SaleRepRepository) GetByIDs(ctx context.Context, ids []string) ([]domain.SaleRep, error) {
	var saleReps []domain.SaleRep

	err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&saleReps).Error

	if err != nil {
		return nil, err
	}

	return saleReps, nil
}

// GetByUserID finds all sale reps for a user with pagination
func (r *SaleRepRepository) GetByUserID(ctx context.Context, userID uuid.UUID, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) ([]domain.SaleRep, int64, error) {
	if pagination == nil {
		pagination = baseRepo.DefaultPagination()
	}
	pagination.Validate()

	// Build base query
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	filterBuilder := filterPkg.NewFilterBuilder(query)
	query = filterBuilder.Apply(filter)

	// Count total (before pagination)
	var total int64
	if err := query.Model(&domain.SaleRep{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	var salereps []domain.SaleRep
	err := query.
		Offset(pagination.Offset).
		Limit(pagination.Limit).
		Order("updated_at DESC").
		Find(&salereps).Error

	if err != nil {
		return nil, 0, err
	}

	return salereps, total, nil
}

// GetByEmail finds a sale rep by email.
func (r *SaleRepRepository) GetByEmail(ctx context.Context, email string) (*domain.SaleRep, error) {
	var saleRep domain.SaleRep

	err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&saleRep).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &saleRep, nil
}

// FindByID finds a sale rep by ID without soft delete
func (r *SaleRepRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.SaleRep, error) {
	var saleRep domain.SaleRep
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&saleRep).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &saleRep, nil
}

// Update updates a sale rep without soft delete
func (r *SaleRepRepository) Update(ctx context.Context, id uuid.UUID, saleRep *domain.SaleRep) error {
	return r.db.WithContext(ctx).
		Model(&domain.SaleRep{}).
		Where("id = ?", id).
		Updates(saleRep).Error
}

// Delete hard deletes a sale rep
func (r *SaleRepRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&domain.SaleRep{}).Error
}
