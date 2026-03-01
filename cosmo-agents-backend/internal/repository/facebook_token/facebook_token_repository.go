package facebooktoken

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
)

// FacebookTokenRepository manages facebook token storage.
type FacebookTokenRepository struct {
	*gormpkg.GormRepository[domain.FacebookToken]
	db *gorm.DB
}

// NewFacebookTokenRepository creates gormpkg.
func NewFacebookTokenRepository(db *gorm.DB) *FacebookTokenRepository {
	return &FacebookTokenRepository{
		GormRepository: gormpkg.NewGormRepository[domain.FacebookToken](db),
		db:             db,
	}
}

// GetPageToken retrieves token by page ID.
func (r *FacebookTokenRepository) GetPageToken(ctx context.Context, pageID string) (*domain.FacebookToken, error) {
	var token domain.FacebookToken
	if err := r.db.WithContext(ctx).
		Where("page_id = ? AND token_type = ?", pageID, domain.TokenTypeFBPage).
		First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

// GetUserToken retrieves user token by user ID.
func (r *FacebookTokenRepository) GetUserToken(ctx context.Context, userID uuid.UUID) (*domain.FacebookToken, error) {
	var token domain.FacebookToken
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND token_type = ?", userID, domain.TokenTypeFBUser).
		First(&token).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

// SavePageToken stores a new page token.
func (r *FacebookTokenRepository) SavePageToken(ctx context.Context, userID uuid.UUID, fbUserID string, pageID string, token string) (*domain.FacebookToken, error) {
	pageToken := &domain.FacebookToken{
		UserID:      userID,
		FBUserID:    fbUserID,
		TokenType:   domain.TokenTypeFBPage,
		PageID:      &pageID,
		AccessToken: token,
	}
	if err := r.db.WithContext(ctx).Create(pageToken).Error; err != nil {
		return nil, err
	}
	return pageToken, nil
}

// FindByID finds a facebook token by ID without soft delete
func (r *FacebookTokenRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.FacebookToken, error) {
	var token domain.FacebookToken
	err := r.GetDB().WithContext(ctx).
		Where("id = ?", id).
		First(&token).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &token, nil
}

// Update updates a facebook token without soft delete
func (r *FacebookTokenRepository) Update(ctx context.Context, id uuid.UUID, token *domain.FacebookToken) error {
	return r.GetDB().WithContext(ctx).
		Model(&domain.FacebookToken{}).
		Where("id = ?", id).
		Updates(token).Error
}

// Delete hard deletes a facebook token (facebook tokens don't have soft delete)
func (r *FacebookTokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.GetDB().WithContext(ctx).
		Where("id = ?", id).
		Delete(&domain.FacebookToken{}).Error
}

// HardDelete hard deletes a facebook token (same as Delete since no soft delete)
func (r *FacebookTokenRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	return r.Delete(ctx, id)
}

// FindAll finds all facebook tokens without soft delete filter
func (r *FacebookTokenRepository) FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.FacebookToken], error) {
	query := r.GetDB().WithContext(ctx)

	// Apply filters
	if filter != nil {
		for key, value := range filter {
			query = query.Where(key+" = ?", value)
		}
	}

	// Count total
	var total int64
	var model domain.FacebookToken
	if err := query.Model(&model).Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	if pagination != nil {
		pagination.Validate()
		query = query.Offset(pagination.Offset).Limit(pagination.Limit)
	}

	var tokens []domain.FacebookToken
	err := query.Find(&tokens).Error
	if err != nil {
		return nil, err
	}

	// Use default pagination values if nil
	offset := 0
	limit := 0
	if pagination != nil {
		offset = pagination.Offset
		limit = pagination.Limit
	}

	return &baseRepo.PaginatedResult[domain.FacebookToken]{
		List:   tokens,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}
