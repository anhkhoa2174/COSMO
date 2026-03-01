package conversation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	gormpkg "github.com/rockship/cosmo-agents-go/internal/repository/gorm"
	"gorm.io/gorm"
)

type conversationContextKey string

const agentUserIDContextKey conversationContextKey = "conversation_agent_user_id"

// WithAgentUserID annotates a context with the agent owner's user ID for repository usage.
func WithAgentUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, agentUserIDContextKey, userID)
}

func agentUserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	if ctx == nil {
		return uuid.Nil, fmt.Errorf("context is required for conversation_type 'sent'")
	}
	val := ctx.Value(agentUserIDContextKey)
	userID, ok := val.(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("agent_user_id missing or invalid in context")
	}
	return userID, nil
}

// ConversationRepository handles database operations for conversations
type ConversationRepository struct {
	*gormpkg.GormRepository[domain.Conversation]
}

// Ensure interface compliance for reply mutations at compile time.
var _ interface {
	SetReplied(context.Context, uuid.UUID, bool) error
} = (*ConversationRepository)(nil)

// UpdateRepliedIfTrue sets replied=false only when replied is currently true.
func (r *ConversationRepository) UpdateRepliedIfTrue(ctx context.Context, id uuid.UUID) error {
	return r.GetDB().WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("id = ? AND replied = ?", id, true).
		Update("replied", false).Error
}

// NewConversationRepository creates a new conversation repository
func NewConversationRepository(db *gorm.DB) *ConversationRepository {
	return &ConversationRepository{
		GormRepository: gormpkg.NewGormRepository[domain.Conversation](db),
	}
}

// SetReplied updates the replied flag for a conversation.
func (r *ConversationRepository) SetReplied(ctx context.Context, id uuid.UUID, replied bool) error {
	return r.GetDB().WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("id = ?", id).
		Update("replied", replied).Error
}

// FindByGmailThreadID retrieves a conversation by Gmail thread ID
func (r *ConversationRepository) FindByGmailThreadID(ctx context.Context, gmailThreadID string) (*domain.Conversation, error) {
	var conversation domain.Conversation
	err := r.GetDB().WithContext(ctx).
		Where("gmail_thread_id = ? AND is_deleted = ?", gmailThreadID, false).
		First(&conversation).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

// FindByCampaignID retrieves all conversations for a campaign
func (r *ConversationRepository) FindByCampaignID(ctx context.Context, campaignID uuid.UUID, offset, limit int) ([]*domain.Conversation, int64, error) {
	var conversations []*domain.Conversation
	var total int64

	// Count total
	if err := r.GetDB().WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("campaign_id = ? AND is_deleted = ?", campaignID, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get conversations
	err := r.GetDB().WithContext(ctx).
		Where("campaign_id = ? AND is_deleted = ?", campaignID, false).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&conversations).Error

	return conversations, total, err
}

// FindByUserID retrieves all conversations for a user
func (r *ConversationRepository) FindByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Conversation, int64, error) {
	var conversations []*domain.Conversation
	var total int64

	// Count total
	if err := r.GetDB().WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get conversations
	err := r.GetDB().WithContext(ctx).
		Where("user_id = ? AND is_deleted = ?", userID, false).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&conversations).Error

	return conversations, total, err
}

// FindUnreadByUserID retrieves unread conversations for a user
func (r *ConversationRepository) FindUnreadByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Conversation, int64, error) {
	var conversations []*domain.Conversation
	var total int64

	// Count total
	if err := r.GetDB().WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("user_id = ? AND status = ? AND is_deleted = ?", userID, domain.ConversationStatusUnread, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get conversations
	err := r.GetDB().WithContext(ctx).
		Where("user_id = ? AND status = ? AND is_deleted = ?", userID, domain.ConversationStatusUnread, false).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&conversations).Error

	return conversations, total, err
}

// FindByAssigneeID retrieves conversations assigned to a sales rep
func (r *ConversationRepository) FindByAssigneeID(ctx context.Context, assigneeID uuid.UUID, offset, limit int) ([]*domain.Conversation, int64, error) {
	var conversations []*domain.Conversation
	var total int64

	// Count total
	if err := r.GetDB().WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("assignee_id = ? AND is_deleted = ?", assigneeID, false).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get conversations
	err := r.GetDB().WithContext(ctx).
		Where("assignee_id = ? AND is_deleted = ?", assigneeID, false).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&conversations).Error

	return conversations, total, err
}

// FindByGmailThreadIDs returns a map of gmail_thread_id -> Conversation for given IDs.
func (r *ConversationRepository) FindByGmailThreadIDs(ctx context.Context, threadIDs []string) (map[string]*domain.Conversation, error) {
	result := make(map[string]*domain.Conversation)
	if len(threadIDs) == 0 {
		return result, nil
	}

	const maxChunk = 500
	for start := 0; start < len(threadIDs); start += maxChunk {
		end := start + maxChunk
		if end > len(threadIDs) {
			end = len(threadIDs)
		}

		var conversations []domain.Conversation
		if err := r.GetDB().WithContext(ctx).
			Where("gmail_thread_id IN ? AND is_deleted = ?", threadIDs[start:end], false).
			Find(&conversations).Error; err != nil {
			return nil, err
		}

		for i := range conversations {
			result[conversations[i].GmailThreadID] = &conversations[i]
		}
	}

	return result, nil
}

// ConversationSearchFilter represents filter criteria for conversation search
type ConversationSearchFilter struct {
	Labels    []string
	Intents   []string
	Status    *string
	Replied   *bool
	IsDeleted *bool
}

// FindByAgentIDWithFilters retrieves conversations for an agent with filtering and pagination
func (r *ConversationRepository) FindByAgentIDWithFilters(
	ctx context.Context,
	agentID uuid.UUID,
	filter *ConversationSearchFilter,
	conversationType *domain.ConversationType,
	newest bool,
	offset, limit int,
) ([]*domain.Conversation, int64, error) {
	var conversations []*domain.Conversation
	var total int64

	// Build the base query scoped to the agent
	query := r.GetDB().WithContext(ctx).Model(&domain.Conversation{}).
		Where("agent_id = ?", agentID)

	// Default to non-deleted conversations unless explicitly requested
	isDeleted := false
	if filter != nil && filter.IsDeleted != nil {
		isDeleted = *filter.IsDeleted
	}
	query = query.Where("is_deleted = ?", isDeleted)

	// Apply filters
	if filter != nil {
		if len(filter.Labels) > 0 {
			// Use PostgreSQL array overlap operator with pq.Array
			query = query.Where("labels && ?", pq.Array(filter.Labels))
		}
		if len(filter.Intents) > 0 {
			// Use PostgreSQL array overlap operator with pq.Array
			query = query.Where("intents && ?", pq.Array(filter.Intents))
		}
		if filter.Status != nil {
			query = query.Where("status = ?", *filter.Status)
		}
		if filter.Replied != nil {
			query = query.Where("replied = ?", *filter.Replied)
		}
	}

	// Apply conversationType filtering if provided.
	// We don't have an explicit conversation_type column, but we can approximate
	// the three supported types using existing fields:
	// - "assign_to_ai" / ConversationTypeAI: assignee_id IS NULL and the agent owner hasn't sent an email
	// - "assign_to_human" / ConversationTypeHuman: assignee_id IS NOT NULL
	// - "sent": conversations where the user was the sender (derived from emails)
	var senderUserID *uuid.UUID
	if conversationType != nil {
		switch *conversationType {
		case domain.ConversationTypeSent, domain.ConversationTypeAI:
			uid, err := agentUserIDFromContext(ctx)
			if err != nil {
				return nil, 0, err
			}
			senderUserID = &uid
		}
	}

	if conversationType != nil {
		switch *conversationType {
		case domain.ConversationTypeAI:
			if senderUserID == nil || *senderUserID == uuid.Nil {
				return nil, 0, fmt.Errorf("agent_user_id is required for conversation_type 'assign_to_ai'")
			}
			query = query.
				Where("assignee_id IS NULL").
				Where(`
					id NOT IN (
						SELECT DISTINCT conversation_id
						FROM emails
						WHERE conversation_id IS NOT NULL
						  AND from_email IN (SELECT email FROM users WHERE id = ?)
					)
				`, *senderUserID)
		case domain.ConversationTypeHuman:
			query = query.Where("assignee_id IS NOT NULL")
		case domain.ConversationTypeSent:
			// Conversations where the agent's user sent the last email — approximate by checking
			// that there exists an email in the emails table from any of the user's addresses.
			// Note: this mirrors the logic used in SearchConversations but scoped to agent's context.
			if senderUserID == nil || *senderUserID == uuid.Nil {
				return nil, 0, fmt.Errorf("agent_user_id is required for conversation_type 'sent'")
			}
			query = query.Where("id IN (SELECT DISTINCT conversation_id FROM emails WHERE from_email IN (SELECT email FROM users WHERE id = ?))", *senderUserID)
		default:
			// Unknown conversationType — ignore the filter (handler should validate earlier)
		}
	}

	// Count total with filters applied
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Determine sort order
	orderClause := "updated_at DESC"
	if !newest {
		orderClause = "updated_at ASC"
	}

	// Get conversations with pagination and sorting
	err := query.
		Order(orderClause).
		Offset(offset).
		Limit(limit).
		Find(&conversations).Error

	return conversations, total, err
}

// SearchConversations retrieves conversations with optional filtering
func (r *ConversationRepository) SearchConversations(ctx context.Context, userID uuid.UUID, conversationType string, offset, limit int, filter map[string]interface{}) ([]*domain.Conversation, int64, error) {
	var conversations []*domain.Conversation
	var total int64

	// Build base query scoped to the user
	query := r.GetDB().WithContext(ctx).Model(&domain.Conversation{}).
		Where("user_id = ?", userID)

	// Unless caller provided an explicit is_deleted filter, default to active conversations.
	// When the filter is a simple boolean, apply it directly to avoid bypass issues.
	if filter != nil && filter["is_deleted"] != nil {
		if boolVal, ok := filter["is_deleted"].(bool); ok {
			query = query.Where("is_deleted = ?", boolVal)
		}
	} else {
		query = query.Where("is_deleted = ?", false)
	}

	// Validate conversationType before using in query
	allowedTypes := map[string]bool{
		string(domain.ConversationTypeSent):  true,
		string(domain.ConversationTypeAI):    true,
		string(domain.ConversationTypeHuman): true,
		"":                                   true, // allow empty for no filter
	}
	if !allowedTypes[conversationType] {
		return nil, 0, fmt.Errorf("invalid conversationType: %s", conversationType)
	}
	if conversationType == string(domain.ConversationTypeSent) {
		if userID == uuid.Nil {
			return nil, 0, fmt.Errorf("user_id is required for conversation_type 'sent'")
		}
		query = query.Where("id IN (SELECT DISTINCT conversation_id FROM emails WHERE from_email IN (SELECT email FROM users WHERE id = ?))", userID)
	} else if conversationType == string(domain.ConversationTypeAI) {
		query = query.Where("assignee_id IS NULL")
	} else if conversationType == string(domain.ConversationTypeHuman) {
		query = query.Where("assignee_id IS NOT NULL")
	}

	// Apply custom filters if provided (safely validated by handler)
	if filter != nil {
		if err := r.applySafeFilters(query, filter); err != nil {
			return nil, 0, fmt.Errorf("invalid filter: %w", err)
		}
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get conversations ordered by updated_at
	if err := query.Order("updated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&conversations).Error; err != nil {
		return nil, 0, err
	}

	return conversations, total, nil
}

// GetDistinctAssignees retrieves distinct assignee IDs for a user's conversations
func (r *ConversationRepository) GetDistinctAssignees(ctx context.Context, userID uuid.UUID, offset, limit int) ([]uuid.UUID, int64, error) {
	var assigneeIDs []uuid.UUID
	var total int64

	// Count distinct assignees
	if err := r.GetDB().WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("user_id = ? AND assignee_id IS NOT NULL AND is_deleted = ?", userID, false).
		Distinct("assignee_id").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get distinct assignee IDs
	err := r.GetDB().WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("user_id = ? AND assignee_id IS NOT NULL AND is_deleted = ?", userID, false).
		Distinct("assignee_id").
		Offset(offset).
		Limit(limit).
		Pluck("assignee_id", &assigneeIDs).Error

	return assigneeIDs, total, err
}

// SoftDelete marks a conversation as deleted
func (r *ConversationRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.GetDB().WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error
}

// applySafeFilters applies validated filters to the query safely
func (r *ConversationRepository) applySafeFilters(query *gorm.DB, filter map[string]interface{}) error {
	// Note: This method assumes filters have already been validated by the handler layer
	// using the ValidatebaseRepo.Filter function to prevent injection attacks

	for key, value := range filter {
		switch key {
		case "is_deleted":
			if valueMap, ok := value.(map[string]interface{}); ok {
				for operator, operatorValue := range valueMap {
					switch operator {
					case "$in":
						if inValues, ok := operatorValue.([]interface{}); ok {
							query = query.Where("is_deleted IN ?", inValues)
						} else {
							return fmt.Errorf("unsafe $in value for is_deleted")
						}
					case "$eq":
						query = query.Where("is_deleted = ?", operatorValue)
					case "$ne":
						query = query.Where("is_deleted != ?", operatorValue)
					default:
						return fmt.Errorf("unsupported operator for is_deleted: %s", operator)
					}
				}
			} else {
				query = query.Where("is_deleted = ?", value)
			}

		case "replied":
			if valueMap, ok := value.(map[string]interface{}); ok {
				for operator, operatorValue := range valueMap {
					switch operator {
					case "$eq":
						query = query.Where("replied = ?", operatorValue)
					case "$ne":
						query = query.Where("replied != ?", operatorValue)
					default:
						return fmt.Errorf("unsupported operator for replied: %s", operator)
					}
				}
			} else {
				query = query.Where("replied = ?", value)
			}

		case "status":
			if valueMap, ok := value.(map[string]interface{}); ok {
				for operator, operatorValue := range valueMap {
					switch operator {
					case "$in":
						if inValues, ok := operatorValue.([]interface{}); ok {
							query = query.Where("status IN ?", inValues)
						} else {
							return fmt.Errorf("unsafe $in value for status")
						}
					case "$eq":
						query = query.Where("status = ?", operatorValue)
					case "$ne":
						query = query.Where("status != ?", operatorValue)
					default:
						return fmt.Errorf("unsupported operator for status: %s", operator)
					}
				}
			} else {
				query = query.Where("status = ?", value)
			}
		default:
			return fmt.Errorf("unsupported filter key: %s", key)
		}
	}

	return nil
}
