package knowledge

import (
	"context"
	"mime/multipart"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// KnowledgeRepository defines interface for knowledge repository operations
type KnowledgeRepository interface {
	Create(ctx context.Context, knowledge *domain.Knowledge) (*domain.Knowledge, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Knowledge, error)
	GetByUserID(ctx context.Context, userID string, limit, offset int) ([]domain.Knowledge, int64, error)
	Update(ctx context.Context, id uuid.UUID, knowledge *domain.Knowledge) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// UserRepository defines interface for user repository operations
type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

// KnowledgeService defines interface for knowledge service operations
type KnowledgeService interface {
	Search(ctx context.Context, userID uuid.UUID, query string, topK int, filter map[string]interface{}) ([]v1schema.KnowledgeSearchResult, error)
	Upload(ctx context.Context, userID uuid.UUID, files []*multipart.FileHeader) ([]v1schema.UploadKnowledgeResponse, error)
	EnqueuePruning(ctx context.Context, embeddingGIDs []string) error
}
