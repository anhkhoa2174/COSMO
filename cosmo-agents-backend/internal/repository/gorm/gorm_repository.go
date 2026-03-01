package gorm

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/core"
	base "github.com/rockship/cosmo-agents-go/internal/repository/base"
	filterPkg "github.com/rockship/cosmo-agents-go/internal/repository/filter"
)

// GormRepository is a generic GORM-based repository implementation
type GormRepository[T any] struct {
	db *gorm.DB
}

// NewGormRepository creates a new GORM repository
func NewGormRepository[T any](db *gorm.DB) *GormRepository[T] {
	return &GormRepository[T]{db: db}
}

// Create creates a new entity
func (r *GormRepository[T]) Create(ctx context.Context, entity *T) (*T, error) {
	err := core.DB(ctx, r.db).WithContext(ctx).Create(entity).Error
	if err != nil {
		return nil, err
	}
	return entity, nil
}

// FindByID finds an entity by ID (excludes soft-deleted records)
func (r *GormRepository[T]) FindByID(ctx context.Context, id uuid.UUID) (*T, error) {
	var entity T
	err := core.DB(ctx, r.db).WithContext(ctx).Where("id = ? AND is_deleted = ?", id, false).First(&entity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &entity, nil
}

// GetByID is an alias for FindByID for compatibility
func (r *GormRepository[T]) GetByID(ctx context.Context, id uuid.UUID) (*T, error) {
	return r.FindByID(ctx, id)
}

// FindAll finds all entities with filters and pagination (excludes soft-deleted records)
func (r *GormRepository[T]) FindAll(ctx context.Context, filter base.Filter, pagination *base.PaginationParams) (*base.PaginatedResult[T], error) {
	if pagination == nil {
		pagination = base.DefaultPagination()
	}
	pagination.Validate()

	// Build query with filters
	query := core.DB(ctx, r.db).WithContext(ctx).Where("is_deleted = ?", false)
	filterBuilder := filterPkg.NewFilterBuilder(query)
	query = filterBuilder.Apply(filter)

	// Count total (before pagination)
	var total int64
	if err := query.Model(new(T)).Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	var entities []T
	err := query.
		Offset(pagination.Offset).
		Limit(pagination.Limit).
		Order("created_at DESC").
		Find(&entities).Error

	if err != nil {
		return nil, err
	}

	return &base.PaginatedResult[T]{
		List:   entities,
		Total:  total,
		Offset: pagination.Offset,
		Limit:  pagination.Limit,
	}, nil
}

// Update updates an existing entity (only non-deleted records)
func (r *GormRepository[T]) Update(ctx context.Context, id uuid.UUID, entity *T) error {
	return core.DB(ctx, r.db).WithContext(ctx).
		Model(new(T)).
		Where("id = ? AND is_deleted = ?", id, false).
		Select("*").              // Select all fields to ensure zero values are included
		Omit("id", "created_at"). // Don't update ID and created_at
		Updates(entity).Error
}

// Delete soft deletes an entity
func (r *GormRepository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	return core.DB(ctx, r.db).WithContext(ctx).
		Model(new(T)).
		Where("id = ?", id).
		Update("is_deleted", true).Error
}

// HardDelete permanently deletes an entity
func (r *GormRepository[T]) HardDelete(ctx context.Context, id uuid.UUID) error {
	return core.DB(ctx, r.db).WithContext(ctx).
		Where("id = ?", id).
		Delete(new(T)).Error
}

// Count counts entities matching the filter (excludes soft-deleted records)
func (r *GormRepository[T]) Count(ctx context.Context, filter base.Filter) (int64, error) {
	query := core.DB(ctx, r.db).WithContext(ctx).Model(new(T)).Where("is_deleted = ?", false)
	filterBuilder := filterPkg.NewFilterBuilder(query)
	query = filterBuilder.Apply(filter)

	var count int64
	err := query.Count(&count).Error
	return count, err
}

// Transaction executes a function within a transaction
func (r *GormRepository[T]) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

// GetDB returns the underlying GORM DB instance
func (r *GormRepository[T]) GetDB() *gorm.DB {
	return r.db
}
