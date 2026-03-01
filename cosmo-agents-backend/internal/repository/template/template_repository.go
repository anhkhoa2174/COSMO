package template

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"

	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"

	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"gorm.io/gorm"
)

// TemplateRepository handles database operations for templates
type TemplateRepository struct {
	*gormpkg.GormRepository[domain.Template]
	db *gorm.DB
}

// NewTemplateRepository creates a new template repository
func NewTemplateRepository(db *gorm.DB) *TemplateRepository {
	return &TemplateRepository{
		GormRepository: gormpkg.NewGormRepository[domain.Template](db),
		db:             db,
	}
}

// FindByCampaignID retrieves all templates for a campaign
func (r *TemplateRepository) FindByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*domain.Template, error) {
	var templates []*domain.Template
	err := r.db.WithContext(ctx).
		Where("campaign_id = ?", campaignID).
		Order("position ASC").
		Find(&templates).Error
	return templates, err
}

// FindByUserID retrieves all templates for a user
func (r *TemplateRepository) FindByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Template, int64, error) {
	var templates []*domain.Template
	var total int64

	// Count total
	if err := r.db.WithContext(ctx).
		Model(&domain.Template{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get templates
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&templates).Error

	return templates, total, err
}

// ReorderTemplates updates positions for multiple templates
func (r *TemplateRepository) ReorderTemplates(ctx context.Context, positions map[uuid.UUID]float64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for templateID, position := range positions {
			if err := tx.Model(&domain.Template{}).
				Where("id = ?", templateID).
				Update("position", position).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// FindByIDWithKnowledges retrieves a template by ID.
// Note: Template relations were removed from the struct, so knowledges must be loaded via the relations package if needed.
func (r *TemplateRepository) FindByIDWithKnowledges(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	var template domain.Template
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_deleted = ?", id, false).
		First(&template).Error
	if isUndefinedColumnErr(err, "is_deleted") {
		err = r.db.WithContext(ctx).
			Where("id = ?", id).
			First(&template).Error
	}
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
}

// FindByIDAndUserID finds a template by ID and user ID (authorization check)
func (r *TemplateRepository) FindByIDAndUserID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Template, error) {
	var template domain.Template
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&template).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
}

// FindByIDWithKnowledgesAndUserID finds a template by ID and user ID.
// Note: Template relations were removed from the struct, so knowledges must be loaded via the relations package if needed.
func (r *TemplateRepository) FindByIDWithKnowledgesAndUserID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Template, error) {
	var template domain.Template
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND is_deleted = ?", id, userID, false).
		First(&template).Error
	if isUndefinedColumnErr(err, "is_deleted") {
		err = r.db.WithContext(ctx).
			Where("id = ? AND user_id = ?", id, userID).
			First(&template).Error
	}
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
}

// AddKnowledge adds a knowledge to a template using atomic transaction with authorization check
func (r *TemplateRepository) AddKnowledge(ctx context.Context, templateID uuid.UUID, knowledge *domain.Knowledge) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// CRITICAL SECURITY FIX: Get template and verify ownership in one query
		var template domain.Template
		if err := tx.First(&template, templateID).Error; err != nil {
			return err
		}

		// CRITICAL SECURITY FIX: Verify knowledge exists, is not deleted, AND belongs to template owner
		// DO NOT trust knowledge.UserID from user input - fetch from database
		var dbKnowledge domain.Knowledge
		if err := tx.Where("id = ? AND user_id = ? AND is_deleted = ?",
			knowledge.ID, template.UserID, false).First(&dbKnowledge).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("knowledge not found or unauthorized")
			}
			return err
		}

		// Create explicit TemplateKnowledge record with proper ID
		templateKnowledge := &domain.TemplateKnowledge{
			Base:        domain.Base{ID: uuid.New()}, // Explicitly set ID
			TemplateID:  templateID,
			KnowledgeID: dbKnowledge.ID,
		}

		// Handle duplicate key errors - if the association already exists, treat as success
		if err := tx.Create(templateKnowledge).Error; err != nil {
			// Check for unique constraint violation using proper error handling
			if baseRepo.IsUniqueViolation(err) {
				// Association already exists - treat as success
				return nil
			}
			// Other error - return as-is
			return err
		}

		// Security verified: knowledge exists and belongs to template owner
		return nil
	})
}

// FindByID finds a template by ID without soft delete
func (r *TemplateRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	var template domain.Template
	err := r.GetDB().WithContext(ctx).
		Where("id = ?", id).
		First(&template).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &template, nil
}

// Delete hard deletes a template (templates table has no is_deleted column)
func (r *TemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.GetDB().WithContext(ctx).
		Where("id = ?", id).
		Delete(&domain.Template{}).Error
}

// isUndefinedColumnErr detects missing column errors so we can gracefully fall back on older schemas.
func isUndefinedColumnErr(err error, column string) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "column") && strings.Contains(msg, column) && strings.Contains(msg, "does not exist")
}

// FindByIDs returns templates mapped by ID.
func (r *TemplateRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Template, error) {
	result := make(map[uuid.UUID]*domain.Template)
	if len(ids) == 0 {
		return result, nil
	}

	var templates []domain.Template
	if err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&templates).Error; err != nil {
		return nil, err
	}

	for i := range templates {
		result[templates[i].ID] = &templates[i]
	}
	return result, nil
}

// Update updates a template without soft delete
func (r *TemplateRepository) Update(ctx context.Context, id uuid.UUID, template *domain.Template) error {
	return r.GetDB().WithContext(ctx).
		Model(&domain.Template{}).
		Where("id = ?", id).
		Updates(template).Error
}
