package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/rockship/cosmo-agents-go/internal/core"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	"gorm.io/gorm"
)

// ErrLockTimeout indicates a row lock could not be acquired
var ErrLockTimeout = errors.New("could not obtain row lock - concurrent operation in progress")

// IntegrationRepository handles database operations for integrations
type IntegrationRepository struct {
	*gormpkg.GormRepository[domain.Integration]
	db *gorm.DB
}

// NewIntegrationRepository creates a new integration repository
func NewIntegrationRepository(db *gorm.DB) *IntegrationRepository {
	return &IntegrationRepository{
		GormRepository: gormpkg.NewGormRepository[domain.Integration](db),
		db:             db,
	}
}

// FindByUserIDAndSource retrieves an integration by user ID and source
func (r *IntegrationRepository) FindByUserIDAndSource(ctx context.Context, userID uuid.UUID, source domain.SourceIntegration) (*domain.Integration, error) {
	var integration domain.Integration
	err := core.DB(ctx, r.db).WithContext(ctx).
		Where("user_id = ? AND source = ?", userID, source).
		First(&integration).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &integration, nil
}

// UpdateConfig updates the config field of an integration
func (r *IntegrationRepository) UpdateConfig(ctx context.Context, id string, config map[string]interface{}) error {
	// Validate ID is a valid UUID to prevent injection
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid integration ID format: %w", err)
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return core.DB(ctx, r.db).WithContext(ctx).
		Model(&domain.Integration{}).
		Where("id = ?", id).
		Update("config", domain.JSONB(configBytes)).Error

}

// UpdateCredential updates the credential field of an integration
func (r *IntegrationRepository) UpdateCredential(ctx context.Context, id string, credential map[string]interface{}) error {
	// Validate ID is a valid UUID to prevent injection
	if _, err := uuid.Parse(id); err != nil {
		return fmt.Errorf("invalid integration ID format: %w", err)
	}

	credentialBytes, err := json.Marshal(credential)
	if err != nil {
		return err
	}
	return core.DB(ctx, r.db).WithContext(ctx).
		Model(&domain.Integration{}).
		Where("id = ?", id).
		Update("credential", domain.JSONB(credentialBytes)).Error
}

// FindByUserIDAndSourceForUpdate retrieves an integration by user ID and source with row lock
// This prevents concurrent modifications by locking the row until the transaction completes
func (r *IntegrationRepository) FindByUserIDAndSourceForUpdate(ctx context.Context, userID uuid.UUID, source domain.SourceIntegration) (*domain.Integration, error) {
	var integration domain.Integration
	err := core.DB(ctx, r.db).WithContext(ctx).
		Set("gorm:query_option", "FOR UPDATE NOWAIT").
		Where("user_id = ? AND source = ?", userID, source).
		First(&integration).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			// 55P03: lock_not_available, 40P01: deadlock_detected
			if pgErr.Code == "55P03" || pgErr.Code == "40P01" {
				return nil, ErrLockTimeout
			}
		} else {
			// Fallback string check for other drivers (e.g., MySQL)
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "could not obtain lock") ||
				strings.Contains(errMsg, "lock wait timeout") ||
				strings.Contains(errMsg, "deadlock") {
				return nil, ErrLockTimeout
			}
		}
		return nil, err
	}
	return &integration, nil
}

// FindAll overrides the default FindAll to handle entities without SoftDeleteMixin
func (r *IntegrationRepository) FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.Integration], error) {
	var integrations []domain.Integration
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.Integration{})

	// Apply filter (basic implementation)
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

	if err := query.Find(&integrations).Error; err != nil {
		return nil, err
	}

	offset := 0
	limit := 0
	if pagination != nil {
		offset = pagination.Offset
		limit = pagination.Limit
	}

	return &baseRepo.PaginatedResult[domain.Integration]{
		List:   integrations,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}, nil
}
