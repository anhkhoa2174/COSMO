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
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Knowledge, error)
	GetByUserID(ctx context.Context, userID string, limit, skip int) ([]domain.Knowledge, int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// UserRepository defines interface for user repository operations
type UserRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

// KnowledgeService defines upload operations required by V2 handler
type KnowledgeService interface {
	Upload(ctx context.Context, userID uuid.UUID, files []*multipart.FileHeader) ([]v1schema.UploadKnowledgeResponse, error)
	UploadOfType(ctx context.Context, userID uuid.UUID, files []*multipart.FileHeader, knowledgeType domain.KnowledgeType) ([]v1schema.UploadKnowledgeResponse, error)
}
