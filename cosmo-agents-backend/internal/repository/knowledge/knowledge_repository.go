package knowledge

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"

	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"

	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"gorm.io/gorm"
)

var ErrKnowledgeOwnershipConflict = errors.New("knowledge embedding_gid already exists for a different user")

// KnowledgeRepository handles knowledge data operations.
type KnowledgeRepository struct {
	*gormpkg.GormRepository[domain.Knowledge]
	db *gorm.DB
}

// NewKnowledgeRepository creates a new knowledge gormpkg.
func NewKnowledgeRepository(db *gorm.DB) *KnowledgeRepository {
	return &KnowledgeRepository{
		GormRepository: gormpkg.NewGormRepository[domain.Knowledge](db),
		db:             db,
	}
}

// FindByIDs finds multiple knowledge items by their IDs.
func (r *KnowledgeRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Knowledge, error) {
	var knowledges []*domain.Knowledge
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&knowledges).Error
	if err != nil {
		return nil, err
	}
	return knowledges, nil
}

// GetByEmbeddingGID finds a knowledge by embedding GID.
func (r *KnowledgeRepository) GetByEmbeddingGID(ctx context.Context, embeddingGID string) (*domain.Knowledge, error) {
	var knowledge domain.Knowledge

	err := r.db.WithContext(ctx).
		Where("embedding_gid = ?", embeddingGID).
		First(&knowledge).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &knowledge, nil
}

// GetByEmbeddingGIDUnscoped finds a knowledge by embedding GID including soft-deleted rows.
func (r *KnowledgeRepository) GetByEmbeddingGIDUnscoped(ctx context.Context, embeddingGID string) (*domain.Knowledge, error) {
	var knowledge domain.Knowledge

	err := r.db.WithContext(ctx).
		Unscoped().
		Where("embedding_gid = ?", embeddingGID).
		First(&knowledge).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &knowledge, nil
}

// UpdateByEmbeddingGID updates a knowledge by embedding GID.
func (r *KnowledgeRepository) UpdateByEmbeddingGID(ctx context.Context, embeddingGID string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).
		Model(&domain.Knowledge{}).
		Where("embedding_gid = ?", embeddingGID).
		Updates(updates).Error
}

// DeleteByEmbeddingGID deletes a knowledge by embedding GID (soft delete).
func (r *KnowledgeRepository) DeleteByEmbeddingGID(ctx context.Context, embeddingGID string) error {
	return r.db.WithContext(ctx).
		Where("embedding_gid = ?", embeddingGID).
		Delete(&domain.Knowledge{}).Error
}

// UpsertByEmbeddingGID creates or updates a knowledge entry based on embedding GID.
func (r *KnowledgeRepository) UpsertByEmbeddingGID(ctx context.Context, knowledge *domain.Knowledge) (*domain.Knowledge, error) {
	if knowledge == nil || knowledge.EmbeddingGID == nil {
		return nil, errors.New("knowledge or embedding_gid is nil")
	}

	knowledge.IsDeleted = false

	err := r.db.WithContext(ctx).Create(knowledge).Error
	if err == nil {
		return knowledge, nil
	}

	if !baseRepo.IsUniqueViolation(err) {
		return nil, err
	}

	existing, getErr := r.GetByEmbeddingGID(ctx, *knowledge.EmbeddingGID)
	if getErr != nil {
		return nil, getErr
	}

	if existing == nil {
		// Might be soft-deleted; check unscoped to allow resurrection
		softDeleted, getErr := r.GetByEmbeddingGIDUnscoped(ctx, *knowledge.EmbeddingGID)
		if getErr != nil {
			return nil, getErr
		}
		if softDeleted == nil {
			return nil, err
		}
		if softDeleted.UserID != knowledge.UserID {
			return nil, ErrKnowledgeOwnershipConflict
		}

		updateData := map[string]interface{}{
			"source_type": knowledge.SourceType,
			"cmetadata":   knowledge.CMetadata,
			"is_deleted":  false,
		}

		if knowledge.CozeDatasetID != nil {
			updateData["coze_dataset_id"] = knowledge.CozeDatasetID
		}

		if err := r.db.WithContext(ctx).
			Unscoped().
			Model(&domain.Knowledge{}).
			Where("embedding_gid = ? AND user_id = ?", *knowledge.EmbeddingGID, knowledge.UserID).
			Updates(updateData).Error; err != nil {
			return nil, err
		}

		return r.GetByEmbeddingGID(ctx, *knowledge.EmbeddingGID)
	}

	if existing.UserID != knowledge.UserID {
		return nil, ErrKnowledgeOwnershipConflict
	}

	updateData := map[string]interface{}{
		"source_type": knowledge.SourceType,
		"cmetadata":   knowledge.CMetadata,
		"is_deleted":  false,
	}

	if knowledge.CozeDatasetID != nil {
		updateData["coze_dataset_id"] = knowledge.CozeDatasetID
	}

	query := r.db.WithContext(ctx).Model(&domain.Knowledge{})
	if existing.IsDeleted {
		query = query.Unscoped()
	}

	err = query.
		Where("embedding_gid = ? AND user_id = ?", *knowledge.EmbeddingGID, knowledge.UserID).
		Updates(updateData).Error
	if err != nil {
		return nil, err
	}

	return r.GetByEmbeddingGID(ctx, *knowledge.EmbeddingGID)
}

// GetByUserID finds all knowledge for a user.
func (r *KnowledgeRepository) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]domain.Knowledge, int64, error) {
	var knowledges []domain.Knowledge
	var total int64

	// Count total
	if err := r.db.WithContext(ctx).
		Model(&domain.Knowledge{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get records
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Limit(limit).
		Offset(offset).
		Order("updated_at DESC").
		Find(&knowledges).Error

	if err != nil {
		return nil, 0, err
	}

	return knowledges, total, nil
}

// GetByCollection finds all knowledge in a collection.
func (r *KnowledgeRepository) GetByCollection(ctx context.Context, collection string, limit, offset int) ([]domain.Knowledge, int64, error) {
	var knowledges []domain.Knowledge
	var total int64

	// Count total
	if err := r.db.WithContext(ctx).
		Model(&domain.Knowledge{}).
		Where("collection = ? AND is_deleted = ?", collection, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get records
	err := r.db.WithContext(ctx).
		Where("collection = ? AND is_deleted = ?", collection, false).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&knowledges).Error

	if err != nil {
		return nil, 0, err
	}

	return knowledges, total, nil
}
