package drafttemplate

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"gorm.io/gorm"
)

// DraftTemplateRepository handles database operations for draft templates
type DraftTemplateRepository struct {
	db *gorm.DB
}

// NewDraftTemplateRepository creates a new draft template repository
func NewDraftTemplateRepository(db *gorm.DB) *DraftTemplateRepository {
	return &DraftTemplateRepository{
		db: db,
	}
}

// Create creates a new draft template
func (r *DraftTemplateRepository) Create(ctx context.Context, entity *domain.DraftTemplate) (*domain.DraftTemplate, error) {
	err := r.db.WithContext(ctx).Create(entity).Error
	if err != nil {
		return nil, err
	}
	return entity, nil
}

// FindByID finds a draft template by ID
func (r *DraftTemplateRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.DraftTemplate, error) {
	var draftTemplate domain.DraftTemplate
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&draftTemplate).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &draftTemplate, nil
}

// Update updates an existing draft template
func (r *DraftTemplateRepository) Update(ctx context.Context, id uuid.UUID, entity *domain.DraftTemplate) error {
	return r.db.WithContext(ctx).
		Model(&domain.DraftTemplate{}).
		Where("id = ?", id).
		Updates(entity).Error
}

// Delete hard deletes a draft template
func (r *DraftTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&domain.DraftTemplate{}).Error
}

// FindByIDWithTemplate retrieves a draft template with its associated template
func (r *DraftTemplateRepository) FindByIDWithTemplate(ctx context.Context, id uuid.UUID) (*domain.DraftTemplate, error) {
	var draftTemplate domain.DraftTemplate
	err := r.db.WithContext(ctx).
		Preload("Template").
		Where("id = ?", id).
		First(&draftTemplate).Error
	if err != nil {
		return nil, err
	}
	return &draftTemplate, nil
}

// FindByCampaignID retrieves all draft templates for a campaign
func (r *DraftTemplateRepository) FindByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*domain.DraftTemplate, error) {
	var draftTemplates []*domain.DraftTemplate
	err := r.db.WithContext(ctx).
		Preload("Template").
		Where("campaign_id = ?", campaignID).
		Find(&draftTemplates).Error
	return draftTemplates, err
}

// FindByCampaignAndIntent retrieves a draft template by campaign ID and intent
func (r *DraftTemplateRepository) FindByCampaignAndIntent(ctx context.Context, campaignID uuid.UUID, intent string) (*domain.DraftTemplate, error) {
	var draftTemplate domain.DraftTemplate
	err := r.db.WithContext(ctx).
		Preload("Template").
		Where("campaign_id = ? AND intent = ?", campaignID, intent).
		First(&draftTemplate).Error
	if err != nil {
		return nil, err
	}
	return &draftTemplate, nil
}

// FindAll finds draft templates with optional filtering and pagination
func (r *DraftTemplateRepository) FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.DraftTemplate], error) {
	var draftTemplates []domain.DraftTemplate
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.DraftTemplate{})

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

	if err := query.Find(&draftTemplates).Error; err != nil {
		return nil, err
	}

	offset := 0
	limit := 0
	if pagination != nil {
		offset = pagination.Offset
		limit = pagination.Limit
	}

	return &baseRepo.PaginatedResult[domain.DraftTemplate]{
		List:   draftTemplates,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}
