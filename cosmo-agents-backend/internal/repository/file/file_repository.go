package file

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	base "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

// FileRepository handles file-related database operations
type FileRepository struct {
	db *gorm.DB
}

// NewFileRepository creates a new FileRepository
func NewFileRepository(db *gorm.DB) *FileRepository {
	return &FileRepository{db: db}
}

// Create creates a new file record
func (r *FileRepository) Create(ctx context.Context, file *domain.File) (*domain.File, error) {
	err := r.db.WithContext(ctx).Create(file).Error
	if err != nil {
		return nil, err
	}
	return file, nil
}

// FindByID finds a file by ID
func (r *FileRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.File, error) {
	var file domain.File
	err := r.db.WithContext(ctx).Where("id = ? AND is_deleted = ?", id, false).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// FindByS3Key finds a file by S3 key
func (r *FileRepository) FindByS3Key(ctx context.Context, s3Key string) (*domain.File, error) {
	var file domain.File
	err := r.db.WithContext(ctx).Where("s3_key = ? AND is_deleted = ?", s3Key, false).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// FindByUserID finds all files for a user
func (r *FileRepository) FindByUserID(ctx context.Context, userID uuid.UUID, pagination *base.PaginationParams) (*base.PaginatedResult[domain.File], error) {
	var files []domain.File
	var total int64

	query := r.db.WithContext(ctx).Where("user_id = ? AND is_deleted = ?", userID, false)

	// Count total
	if err := query.Model(&domain.File{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	if pagination != nil {
		pagination.Validate()
		query = query.Offset(pagination.Offset).Limit(pagination.Limit)
	}

	if err := query.Find(&files).Error; err != nil {
		return nil, err
	}

	// Use default pagination values if nil
	offset := 0
	limit := 0
	if pagination != nil {
		offset = pagination.Offset
		limit = pagination.Limit
	}

	return &base.PaginatedResult[domain.File]{
		List:   files,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}

// FindAll finds all files with filter and pagination
func (r *FileRepository) FindAll(ctx context.Context, filter base.Filter, pagination *base.PaginationParams) (*base.PaginatedResult[domain.File], error) {
	var files []domain.File
	var total int64

	query := r.db.WithContext(ctx).Where("is_deleted = ?", false)

	// Apply filters
	for key, value := range filter {
		query = query.Where(key+" = ?", value)
	}

	// Count total
	if err := query.Model(&domain.File{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	if pagination != nil {
		pagination.Validate()
		query = query.Offset(pagination.Offset).Limit(pagination.Limit)
	}

	if err := query.Find(&files).Error; err != nil {
		return nil, err
	}

	// Use default pagination values if nil
	offset := 0
	limit := 0
	if pagination != nil {
		offset = pagination.Offset
		limit = pagination.Limit
	}

	return &base.PaginatedResult[domain.File]{
		List:   files,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}

// Update updates a file record
func (r *FileRepository) Update(ctx context.Context, file *domain.File) error {
	return r.db.WithContext(ctx).Save(file).Error
}

// Delete soft deletes a file
func (r *FileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&domain.File{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error
}
