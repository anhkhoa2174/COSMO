package daily_action

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
)

// ChatSessionRepository stores Ask COSMO conversations.
//
// Every method is scoped by user id rather than trusting the session id alone.
// A conversation is private, and an id is guessable in a way a scoped query is
// not: reading someone else's thread must fail because the row was never
// selected, not because a later check happened to run.
type ChatSessionRepository struct {
	db *gorm.DB
}

func NewChatSessionRepository(db *gorm.DB) *ChatSessionRepository {
	return &ChatSessionRepository{db: db}
}

// Create opens a session. The title is trimmed from the first question, which
// is the only thing available to name it at that point.
func (r *ChatSessionRepository) Create(
	ctx context.Context, userID uuid.UUID, firstQuestion string,
) (*domain.ChatSession, error) {
	session := &domain.ChatSession{
		UserID: userID,
		Title:  titleFrom(firstQuestion),
	}
	session.ID = uuid.New()
	if err := r.db.WithContext(ctx).Create(session).Error; err != nil {
		return nil, err
	}
	return session, nil
}

// titleFrom makes a short label out of a question.
func titleFrom(question string) string {
	title := strings.Join(strings.Fields(question), " ")
	if title == "" {
		return "New conversation"
	}
	// Cut on a rune boundary: a Vietnamese question truncated mid-rune renders
	// as a replacement character in the session list.
	const limit = 60
	runes := []rune(title)
	if len(runes) <= limit {
		return title
	}
	return strings.TrimSpace(string(runes[:limit])) + "…"
}

// List returns the user's sessions, most recently used first.
func (r *ChatSessionRepository) List(
	ctx context.Context, userID uuid.UUID, limit int,
) ([]domain.ChatSession, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var out []domain.ChatSession
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Limit(limit).
		Find(&out).Error
	return out, err
}

// Get returns one session, or nil when it does not exist or belongs to
// somebody else. Both cases are deliberately indistinguishable to the caller.
func (r *ChatSessionRepository) Get(
	ctx context.Context, userID, sessionID uuid.UUID,
) (*domain.ChatSession, error) {
	var session domain.ChatSession
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", sessionID, userID).
		First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// Messages returns a session's history in order.
func (r *ChatSessionRepository) Messages(
	ctx context.Context, userID, sessionID uuid.UUID,
) ([]domain.ChatMessage, error) {
	var out []domain.ChatMessage
	err := r.db.WithContext(ctx).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Order("created_at").
		Find(&out).Error
	return out, err
}

// Append writes a turn and bumps the session's activity time.
//
// Both happen in one transaction: a message stored against a session whose
// ordering timestamp did not move would sink down the list even though it is
// the most recent conversation.
func (r *ChatSessionRepository) Append(
	ctx context.Context, userID, sessionID uuid.UUID, messages []domain.ChatMessage,
) error {
	if len(messages) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var session domain.ChatSession
		if err := tx.Where("id = ? AND user_id = ?", sessionID, userID).
			First(&session).Error; err != nil {
			return err
		}

		for i := range messages {
			messages[i].ID = uuid.New()
			messages[i].UserID = userID
			id := sessionID
			messages[i].SessionID = &id
			if err := tx.Create(&messages[i]).Error; err != nil {
				return err
			}
		}

		return tx.Model(&domain.ChatSession{}).
			Where("id = ?", sessionID).
			Update("updated_at", gorm.Expr("now()")).Error
	})
}

// Delete removes a session. Its messages go with it through the foreign key's
// cascade, so a deleted conversation leaves nothing readable behind.
func (r *ChatSessionRepository) Delete(
	ctx context.Context, userID, sessionID uuid.UUID,
) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", sessionID, userID).
		Delete(&domain.ChatSession{}).Error
}
