package email

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
)

// Repository handles database operations for emails.
type Repository struct {
	*gormpkg.GormRepository[domain.Email]
	db *gorm.DB
}

// GetDB returns the underlying database connection
func (r *Repository) GetDB() *gorm.DB {
	return r.db
}

// NewEmailRepository creates a new email gormpkg.
func NewEmailRepository(db *gorm.DB) *Repository {
	return &Repository{
		GormRepository: gormpkg.NewGormRepository[domain.Email](db),
		db:             db,
	}
}

// CountConversationsWithLabelByAgent counts distinct conversations for a given agent with a label.
func (r *Repository) CountConversationsWithLabelByAgent(ctx context.Context, agentID uuid.UUID, label string) (int64, error) {
	var count int64
	err := r.GetDB().WithContext(ctx).
		Table("emails e").
		Joins("JOIN conversations c ON c.id = e.conversation_id").
		Where("c.agent_id = ? AND c.is_deleted = ? AND e.is_deleted = ?", agentID, false, false).
		Where("? = ANY(e.labels)", label).
		Distinct("e.conversation_id").
		Count(&count).Error
	return count, err
}

// FindByConversationID retrieves all emails in a conversation ordered by created_at asc.
func (r *Repository) FindByConversationID(ctx context.Context, conversationID uuid.UUID) ([]*domain.Email, error) {
	var emails []*domain.Email
	err := r.GetDB().WithContext(ctx).
		Where("conversation_id = ? AND is_deleted = ?", conversationID, false).
		Order("created_at ASC").
		Find(&emails).Error
	return emails, err
}

// FindByCampaignID retrieves paginated emails for a campaign.
func (r *Repository) FindByCampaignID(ctx context.Context, campaignID uuid.UUID, offset, limit int) ([]*domain.Email, int64, error) {
	var emails []*domain.Email
	var total int64

	db := r.GetDB().WithContext(ctx)
	if err := db.Model(&domain.Email{}).
		Where("campaign_id = ? AND is_deleted = ?", campaignID, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Where("campaign_id = ? AND is_deleted = ?", campaignID, false).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&emails).Error

	return emails, total, err
}

// FindByConversationAndUser retrieves recent emails in a conversation for a specific user.
func (r *Repository) FindByConversationAndUser(ctx context.Context, conversationID, userID uuid.UUID, limit int) ([]*domain.Email, error) {
	var emails []*domain.Email

	query := r.GetDB().WithContext(ctx).
		Where("conversation_id = ? AND user_id = ? AND is_deleted = ?", conversationID, userID, false).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&emails).Error; err != nil {
		return nil, err
	}

	return emails, nil
}

// FindByUserID retrieves paginated emails for a user.
func (r *Repository) FindByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Email, int64, error) {
	var emails []*domain.Email
	var total int64

	db := r.GetDB().WithContext(ctx)
	if err := db.Model(&domain.Email{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Where("user_id = ? AND is_deleted = ?", userID, false).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&emails).Error

	return emails, total, err
}

// FindByGmailMessageID retrieves an email by its Gmail message ID.
func (r *Repository) FindByGmailMessageID(ctx context.Context, gmailMessageID string) (*domain.Email, error) {
	var email domain.Email
	err := r.GetDB().WithContext(ctx).
		Where("gmail_message_id = ? AND is_deleted = ?", gmailMessageID, false).
		First(&email).Error
	if err != nil {
		return nil, err
	}
	return &email, nil
}

// FindByStatus retrieves paginated emails by status.
func (r *Repository) FindByStatus(ctx context.Context, userID uuid.UUID, status domain.EmailStatus, offset, limit int) ([]*domain.Email, int64, error) {
	var emails []*domain.Email
	var total int64

	db := r.GetDB().WithContext(ctx)
	if err := db.Model(&domain.Email{}).
		Where("user_id = ? AND status = ? AND is_deleted = ?", userID, status, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := db.Where("user_id = ? AND status = ? AND is_deleted = ?", userID, status, false).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&emails).Error

	return emails, total, err
}

// FindLatestByConversationID retrieves the latest email for a conversation.
func (r *Repository) FindLatestByConversationID(ctx context.Context, conversationID uuid.UUID) (*domain.Email, error) {
	var email domain.Email

	err := r.GetDB().WithContext(ctx).
		Where("conversation_id = ? AND is_deleted = ?", conversationID, false).
		Order("created_at DESC").
		First(&email).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}

	return &email, nil
}

// FindByGmailMessageIDs returns a map of gmail_message_id -> Email for the given IDs.
func (r *Repository) FindByGmailMessageIDs(ctx context.Context, ids []string) (map[string]*domain.Email, error) {
	result := make(map[string]*domain.Email)
	if len(ids) == 0 {
		return result, nil
	}

	const maxChunk = 500
	for start := 0; start < len(ids); start += maxChunk {
		end := start + maxChunk
		if end > len(ids) {
			end = len(ids)
		}

		var emails []domain.Email
		if err := r.GetDB().WithContext(ctx).
			Where("gmail_message_id IN ? AND is_deleted = ?", ids[start:end], false).
			Find(&emails).Error; err != nil {
			return nil, err
		}

		for i := range emails {
			result[emails[i].GmailMessageID] = &emails[i]
		}
	}

	return result, nil
}

// FindLatestByConversationIDs retrieves the latest email for each conversation in batch.
func (r *Repository) FindLatestByConversationIDs(ctx context.Context, conversationIDs []uuid.UUID) (map[uuid.UUID]*domain.Email, error) {
	if len(conversationIDs) == 0 {
		return make(map[uuid.UUID]*domain.Email), nil
	}

	var emails []*domain.Email

	subquery := r.GetDB().WithContext(ctx).
		Select("*, ROW_NUMBER() OVER (PARTITION BY conversation_id ORDER BY created_at DESC) as rn").
		Table("emails").
		Where("conversation_id IN ? AND is_deleted = ?", conversationIDs, false)

	err := r.GetDB().WithContext(ctx).
		Table("(?) as ranked_emails", subquery).
		Where("rn = 1").
		Find(&emails).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest emails with ROW_NUMBER subquery: %w", err)
	}

	result := make(map[uuid.UUID]*domain.Email, len(emails))
	for _, email := range emails {
		if email.ConversationID != nil {
			result[*email.ConversationID] = email
		}
	}

	return result, nil
}
