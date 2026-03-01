package customfield

import (
	"context"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
)

// CustomFieldRepository defines interface for custom field repository operations
type CustomFieldRepository interface {
	Create(ctx context.Context, field *domain.CustomField) (*domain.CustomField, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.CustomField, error)
	FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.CustomField], error)
	UpdateFields(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// UserRepository defines interface for user repository operations
type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

// RoleRepository defines interface for role repository operations
type RoleRepository interface {
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
}
