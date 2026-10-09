package daily_action

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/core"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
)

// ChatMessageRepository handles ChatMessage entity operations.
type ChatMessageRepository struct {
	*gormpkg.GormRepository[domain.ChatMessage]
	db *gorm.DB
}

// NewChatMessageRepository creates a new chat message repository.
func NewChatMessageRepository(db *gorm.DB) *ChatMessageRepository {
	return &ChatMessageRepository{
		GormRepository: gormpkg.NewGormRepository[domain.ChatMessage](db),
		db:             db,
	}
}

// FindByUserID finds chat messages for a user, ordered by creation time.
func (r *ChatMessageRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]domain.ChatMessage, error) {
	var messages []domain.ChatMessage
	err := core.DB(ctx, r.db).WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error
	return messages, err
}

// FindByUserAndGeneration finds chat messages for a user scoped to a generation.
func (r *ChatMessageRepository) FindByUserAndGeneration(ctx context.Context, userID uuid.UUID, generationID uuid.UUID) ([]domain.ChatMessage, error) {
	var messages []domain.ChatMessage
	err := core.DB(ctx, r.db).WithContext(ctx).
		Where("user_id = ? AND generation_id = ?", userID, generationID).
		Order("created_at ASC").
		Find(&messages).Error
	return messages, err
}
