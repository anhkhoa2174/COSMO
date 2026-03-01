package operation

import (
	"context"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/core"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	"gorm.io/gorm"
)

// OperationRepository handles database operations for operations
type OperationRepository struct {
	*gormpkg.GormRepository[domain.Operation]
}

// NewOperationRepository creates a new operation repository
func NewOperationRepository(db *gorm.DB) *OperationRepository {
	return &OperationRepository{
		GormRepository: gormpkg.NewGormRepository[domain.Operation](db),
	}
}

// FindByID finds an operation by ID without soft delete
func (r *OperationRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Operation, error) {
	var operation domain.Operation
	err := r.GetDB().WithContext(ctx).
		Where("id = ?", id).
		First(&operation).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &operation, nil
}

// Update updates an operation without soft delete
func (r *OperationRepository) Update(ctx context.Context, id uuid.UUID, operation *domain.Operation) error {
	return r.GetDB().WithContext(ctx).
		Model(&domain.Operation{}).
		Where("id = ?", id).
		Updates(operation).Error
}

// Delete hard deletes an operation (operations don't have soft delete)
func (r *OperationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.GetDB().WithContext(ctx).
		Where("id = ?", id).
		Delete(&domain.Operation{}).Error
}

// FindByStatus retrieves all operations with a specific status
func (r *OperationRepository) FindByStatus(ctx context.Context, status domain.OperationStatus, offset, limit int) ([]*domain.Operation, int64, error) {
	var operations []*domain.Operation
	var total int64

	// Count total
	if err := r.GetDB().WithContext(ctx).
		Model(&domain.Operation{}).
		Where("status = ?", status).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get operations
	err := r.GetDB().WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&operations).Error

	return operations, total, err
}

// UpdateStatus updates the status of an operation
func (r *OperationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OperationStatus) error {
	tx := core.DB(ctx, r.GetDB())
	return tx.WithContext(ctx).
		Model(&domain.Operation{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdateOutput updates the output of an operation
func (r *OperationRepository) UpdateOutput(ctx context.Context, id uuid.UUID, output domain.JSONB) error {
	tx := core.DB(ctx, r.GetDB())
	return tx.WithContext(ctx).
		Model(&domain.Operation{}).
		Where("id = ?", id).
		Update("output", output).Error
}

func (r *OperationRepository) UpdateFailedStatusWithOutput(ctx context.Context, id uuid.UUID, errOutput domain.JSONB) error {
	tx := core.DB(ctx, r.GetDB())
	return tx.WithContext(ctx).
		Model(&domain.Operation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status": domain.OperationStatusFailed,
			"output": errOutput,
		}).
		Error
}
