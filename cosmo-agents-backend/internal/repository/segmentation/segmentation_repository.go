package segmentation

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	"gorm.io/gorm"
)

// SegmentationRepository handles CRUD for segmentations and scores.
type SegmentationRepository struct {
	*gormpkg.GormRepository[domain.Segmentation]
	db *gorm.DB
}

func NewSegmentationRepository(db *gorm.DB) *SegmentationRepository {
	return &SegmentationRepository{
		GormRepository: gormpkg.NewGormRepository[domain.Segmentation](db),
		db:             db,
	}
}

// List by user with optional active filter.
func (r *SegmentationRepository) List(ctx context.Context, userID uuid.UUID, onlyActive bool) ([]*domain.Segmentation, error) {
	var segments []*domain.Segmentation
	q := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if onlyActive {
		q = q.Where("is_active = ?", true)
	}
	if err := q.Order("priority DESC, updated_at DESC").Find(&segments).Error; err != nil {
		return nil, err
	}
	return segments, nil
}

// GetByID fetches a segmentation by ID (segmentations table has no is_deleted column).
func (r *SegmentationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Segmentation, error) {
	var seg domain.Segmentation
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&seg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &seg, nil
}

// DeleteByID removes a segmentation owned by the given user.
func (r *SegmentationRepository) DeleteByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (bool, error) {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&domain.Segmentation{})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// ScoreRepository handles contact_segment_scores.
type ScoreRepository struct {
	db *gorm.DB
}

func NewScoreRepository(db *gorm.DB) *ScoreRepository {
	return &ScoreRepository{db: db}
}

func (r *ScoreRepository) UpsertScore(ctx context.Context, score *domain.SegmentationScore) error {
	return r.db.WithContext(ctx).
		Where("contact_id = ? AND segmentation_id = ?", score.ContactID, score.SegmentationID).
		Assign(score).
		FirstOrCreate(score).Error
}

func (r *ScoreRepository) ListByContact(ctx context.Context, contactID uuid.UUID) ([]*domain.SegmentationScore, error) {
	var scores []*domain.SegmentationScore
	if err := r.db.WithContext(ctx).Where("contact_id = ?", contactID).Order("fit_score DESC").Find(&scores).Error; err != nil {
		return nil, err
	}
	return scores, nil
}

func (r *ScoreRepository) DeleteByContact(ctx context.Context, contactID uuid.UUID) error {
	return r.db.WithContext(ctx).Where("contact_id = ?", contactID).Delete(&domain.SegmentationScore{}).Error
}

// ListBySegmentThreshold returns scores for a segment above fit score threshold.
func (r *ScoreRepository) ListBySegmentThreshold(ctx context.Context, segmentID uuid.UUID, minFitScore int) ([]*domain.SegmentationScore, error) {
	var scores []*domain.SegmentationScore
	if err := r.db.WithContext(ctx).
		Where("segmentation_id = ? AND fit_score >= ? AND passes_filters = ?", segmentID, minFitScore, true).
		Order("fit_score DESC").
		Find(&scores).Error; err != nil {
		return nil, err
	}
	return scores, nil
}

// MarkAsEnrolled marks a contact as enrolled in a segment
func (r *ScoreRepository) MarkAsEnrolled(ctx context.Context, contactID, segmentID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&domain.SegmentationScore{}).
		Where("contact_id = ? AND segmentation_id = ?", contactID, segmentID).
		Updates(map[string]interface{}{
			"status":               "enrolled",
			"enrolled_in_campaign": true,
		}).Error
}

// ListBySegment returns all contact IDs for a segment
func (r *ScoreRepository) ListBySegment(ctx context.Context, segmentID uuid.UUID, limit int) ([]*domain.SegmentationScore, error) {
	var scores []*domain.SegmentationScore
	q := r.db.WithContext(ctx).
		Where("segmentation_id = ?", segmentID).
		Order("fit_score DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&scores).Error; err != nil {
		return nil, err
	}
	return scores, nil
}
