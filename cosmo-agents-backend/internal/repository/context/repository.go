package context

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	baseDomain "github.com/rockship/cosmo-agents-go/internal/domain"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/context"
	"gorm.io/gorm"
)

var (
	ErrContextNotFound = errors.New("context not found")
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ============ Organization Context ============

func (r *Repository) GetOrgContext(ctx context.Context, orgID uuid.UUID) (*domain.OrgContext, error) {
	var orgCtx domain.OrgContext
	err := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		First(&orgCtx).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrContextNotFound
	}
	return &orgCtx, err
}

func (r *Repository) UpsertOrgContext(ctx context.Context, orgCtx *domain.OrgContext) error {
	return r.db.WithContext(ctx).
		Where("organization_id = ?", orgCtx.OrganizationID).
		Assign(orgCtx).
		FirstOrCreate(orgCtx).Error
}

func (r *Repository) UpdateOrgContextField(ctx context.Context, orgID uuid.UUID, field string, value interface{}) error {
	return r.db.WithContext(ctx).
		Model(&domain.OrgContext{}).
		Where("organization_id = ?", orgID).
		Update(field, value).Error
}

// ============ User Context ============

func (r *Repository) GetUserContext(ctx context.Context, userID uuid.UUID) (*domain.UserContext, error) {
	var userCtx domain.UserContext
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&userCtx).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrContextNotFound
	}
	return &userCtx, err
}

func (r *Repository) UpsertUserContext(ctx context.Context, userCtx *domain.UserContext) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userCtx.UserID).
		Assign(userCtx).
		FirstOrCreate(userCtx).Error
}

func (r *Repository) UpdateUserContextField(ctx context.Context, userID uuid.UUID, field string, value interface{}) error {
	return r.db.WithContext(ctx).
		Model(&domain.UserContext{}).
		Where("user_id = ?", userID).
		Update(field, value).Error
}

// AddRecentContact adds a contact to user's recent contacts list
func (r *Repository) AddRecentContact(ctx context.Context, userID uuid.UUID, contact domain.RecentContact) error {
	userCtx, err := r.GetUserContext(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrContextNotFound) {
			// Create new context with this contact
			userCtx = &domain.UserContext{
				UserID: userID,
			}
		} else {
			return err
		}
	}

	// Parse existing recent contacts
	var recentContacts []domain.RecentContact
	if userCtx.RecentContacts != nil {
		_ = json.Unmarshal(userCtx.RecentContacts, &recentContacts)
	}

	// Remove if already exists
	filtered := make([]domain.RecentContact, 0)
	for _, rc := range recentContacts {
		if rc.ContactID != contact.ContactID {
			filtered = append(filtered, rc)
		}
	}

	// Add to front
	contact.LastTouched = time.Now()
	filtered = append([]domain.RecentContact{contact}, filtered...)

	// Keep only last 20
	if len(filtered) > 20 {
		filtered = filtered[:20]
	}

	// Save back
	jsonData, _ := json.Marshal(filtered)
	userCtx.RecentContacts = baseDomain.JSONB(jsonData)

	return r.UpsertUserContext(ctx, userCtx)
}

// ============ Conversation History ============

func (r *Repository) SaveConversationMessage(ctx context.Context, msg *domain.ConversationHistory) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *Repository) GetConversationHistory(ctx context.Context, userID uuid.UUID, sessionID string, limit int) ([]domain.ConversationHistory, error) {
	var messages []domain.ConversationHistory

	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&messages).Error
	if err != nil {
		return nil, err
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (r *Repository) GetContactConversationHistory(ctx context.Context, userID, contactID uuid.UUID, limit int) ([]domain.ConversationHistory, error) {
	var messages []domain.ConversationHistory

	query := r.db.WithContext(ctx).
		Where("user_id = ? AND contact_id = ?", userID, contactID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&messages).Error
	if err != nil {
		return nil, err
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (r *Repository) ClearSessionHistory(ctx context.Context, userID uuid.UUID, sessionID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND session_id = ?", userID, sessionID).
		Delete(&domain.ConversationHistory{}).Error
}

// CleanupOldHistory removes conversation history older than the specified duration
func (r *Repository) CleanupOldHistory(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Delete(&domain.ConversationHistory{})
	return result.RowsAffected, result.Error
}
