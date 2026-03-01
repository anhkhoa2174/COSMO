package customfield

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

// CustomFieldRepository handles CRUD for custom field definitions.
type CustomFieldRepository struct {
	db *gorm.DB
}

// NewCustomFieldRepository creates a new custom field repository.
func NewCustomFieldRepository(db *gorm.DB) *CustomFieldRepository {
	return &CustomFieldRepository{
		db: db,
	}
}

// Create creates a new custom field
func (r *CustomFieldRepository) Create(ctx context.Context, entity *domain.CustomField) (*domain.CustomField, error) {
	err := r.db.WithContext(ctx).Create(entity).Error
	if err != nil {
		return nil, err
	}
	return entity, nil
}

// FindByID finds a custom field by ID
func (r *CustomFieldRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.CustomField, error) {
	var field domain.CustomField
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&field).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &field, nil
}

// FindByNormalizedNames returns a map of normalized name to custom field for the given user.
func (r *CustomFieldRepository) FindByNormalizedNames(ctx context.Context, userID uuid.UUID, names []string) (map[string]domain.CustomField, error) {
	result := make(map[string]domain.CustomField)
	if len(names) == 0 {
		return result, nil
	}

	var fields []domain.CustomField
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND normalized_name IN ?", userID, names).
		Find(&fields).Error; err != nil {
		return nil, err
	}

	for _, field := range fields {
		result[field.NormalizedName] = field
	}

	return result, nil
}

// FindByOrganizationID returns all custom fields for an organization
func (r *CustomFieldRepository) FindByOrganizationID(ctx context.Context, organizationID uuid.UUID) ([]domain.CustomField, error) {
	var fields []domain.CustomField
	if err := r.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Find(&fields).Error; err != nil {
		return nil, err
	}
	return fields, nil
}

// UpdateFields updates specific fields of a custom field
// This method handles validation, normalization and timestamp updates manually
// to support partial updates without triggering BeforeUpdate hook on unchanged fields
func (r *CustomFieldRepository) UpdateFields(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	// Validate data_type if being updated
	if dataTypeValue, ok := updates["data_type"]; ok {
		var dataType domain.CustomFieldDataType
		switch v := dataTypeValue.(type) {
		case domain.CustomFieldDataType:
			dataType = v
		case string:
			dataType = domain.CustomFieldDataType(v)
		default:
			return errors.New("invalid data_type for custom field")
		}

		allowedTypes := map[domain.CustomFieldDataType]bool{
			domain.CustomFieldDataTypeText:   true,
			domain.CustomFieldDataTypeNumber: true,
			domain.CustomFieldDataTypeEmail:  true,
			domain.CustomFieldDataTypeSelect: true,
			domain.CustomFieldDataTypeDate:   true,
			domain.CustomFieldDataTypeURL:    true,
		}
		if !allowedTypes[dataType] {
			return errors.New("invalid data_type for custom field")
		}
		updates["data_type"] = dataType
	}

	// Validate entity_type if being updated
	if entityTypeValue, ok := updates["entity_type"]; ok {
		var entityType domain.CustomFieldEntity
		switch v := entityTypeValue.(type) {
		case domain.CustomFieldEntity:
			entityType = v
		case string:
			entityType = domain.CustomFieldEntity(v)
		default:
			return errors.New("invalid entity_type for custom field")
		}

		if entityType != domain.CustomFieldEntityContact && entityType != domain.CustomFieldEntityCompany {
			return errors.New("invalid entity_type for custom field")
		}
		updates["entity_type"] = entityType
	}

	// Normalize name if being updated
	if name, ok := updates["name"].(string); ok {
		name = strings.TrimSpace(name)
		if name == "" {
			return errors.New("custom field name cannot be empty")
		}
		updates["name"] = name
		updates["normalized_name"] = normalizeCustomFieldName(name)
	}

	// Ensure options is not nil if being updated
	if options, ok := updates["options"]; ok {
		if options == nil {
			updates["options"] = pq.StringArray{}
		}
	}

	// Always update updated_at timestamp
	updates["updated_at"] = time.Now()

	// Use UpdateColumns to avoid triggering BeforeUpdate hook
	// (which would validate ALL fields, not just the ones being updated)
	return r.db.WithContext(ctx).
		Model(&domain.CustomField{}).
		Where("id = ?", id).
		UpdateColumns(updates).Error
}

// normalizeCustomFieldName converts field name to normalized form (lowercase, underscores)
func normalizeCustomFieldName(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = strings.ReplaceAll(slug, " ", "_")
	return slug
}

// Delete hard deletes a custom field
// custom_fields table doesn't have soft delete columns, so we do a real DELETE
func (r *CustomFieldRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&domain.CustomField{}).Error
}

// FindAll finds custom fields with optional filtering and pagination
func (r *CustomFieldRepository) FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.CustomField], error) {
	var customFields []domain.CustomField
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.CustomField{})

	// Apply filter
	if filter != nil {
		for key, value := range filter {
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		}
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	if pagination != nil {
		pagination.Validate()
		query = query.Offset(pagination.Offset).Limit(pagination.Limit)
	}

	if err := query.Find(&customFields).Error; err != nil {
		return nil, err
	}

	offset := 0
	limit := 0
	if pagination != nil {
		offset = pagination.Offset
		limit = pagination.Limit
	}

	return &baseRepo.PaginatedResult[domain.CustomField]{
		List:   customFields,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}
