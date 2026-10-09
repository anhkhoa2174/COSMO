package conversation

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrGroupNameTaken is returned when a user already has a group by that name.
var ErrGroupNameTaken = errors.New("a group with this name already exists")

// GroupWithCount is a group together with how many live conversations it holds.
type GroupWithCount struct {
	domain.ConversationGroup
	ConversationCount int64 `json:"conversation_count"`
}

// GroupRepository stores users' personal conversation groups.
//
// Every read and write takes the owning user's ID and filters on it, so a
// group ID alone never reaches another user's group: a guessed or leaked ID
// finds nothing rather than someone else's data.
type GroupRepository struct {
	db *gorm.DB
}

// NewGroupRepository creates a GroupRepository.
func NewGroupRepository(db *gorm.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

// ListByUser returns the user's groups in name order, each with the number of
// conversations in it that are not in the trash.
func (r *GroupRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]GroupWithCount, error) {
	var groups []GroupWithCount
	err := r.db.WithContext(ctx).
		Table("conversation_groups AS g").
		Select(`g.*, (
			SELECT COUNT(*) FROM conversation_group_members m
			JOIN conversations c ON c.id = m.conversation_id
			WHERE m.group_id = g.id AND c.is_deleted = false
		) AS conversation_count`).
		Where("g.user_id = ?", userID).
		Order("lower(g.name)").
		Scan(&groups).Error
	if err != nil {
		return nil, fmt.Errorf("list conversation groups: %w", err)
	}
	return groups, nil
}

// Create inserts a group, returning ErrGroupNameTaken when the user already
// has one with the same name.
func (r *GroupRepository) Create(ctx context.Context, group *domain.ConversationGroup) error {
	if err := r.db.WithContext(ctx).Create(group).Error; err != nil {
		if baseRepo.IsUniqueViolation(err) {
			return ErrGroupNameTaken
		}
		return fmt.Errorf("create conversation group: %w", err)
	}
	return nil
}

// FindOwned returns the group only if it belongs to the user; otherwise nil.
func (r *GroupRepository) FindOwned(ctx context.Context, groupID, userID uuid.UUID) (*domain.ConversationGroup, error) {
	var group domain.ConversationGroup
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", groupID, userID).
		First(&group).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find conversation group: %w", err)
	}
	return &group, nil
}

// Update saves a group's name and colour. The caller must have loaded it with
// FindOwned.
func (r *GroupRepository) Update(ctx context.Context, group *domain.ConversationGroup) error {
	err := r.db.WithContext(ctx).
		Model(&domain.ConversationGroup{}).
		Where("id = ? AND user_id = ?", group.ID, group.UserID).
		Updates(map[string]interface{}{"name": group.Name, "color": group.Color}).Error
	if err != nil {
		if baseRepo.IsUniqueViolation(err) {
			return ErrGroupNameTaken
		}
		return fmt.Errorf("update conversation group: %w", err)
	}
	return nil
}

// Delete removes the user's group and its memberships. The conversations
// themselves are untouched. It reports whether a group was removed.
func (r *GroupRepository) Delete(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	deleted := false
	// Memberships are removed explicitly rather than left to the foreign
	// key's cascade, so a schema without it cannot keep orphans around.
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND user_id = ?", groupID, userID).
			Delete(&domain.ConversationGroup{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		deleted = true
		return tx.Where("group_id = ?", groupID).Delete(&domain.ConversationGroupMember{}).Error
	})
	if err != nil {
		return false, fmt.Errorf("delete conversation group: %w", err)
	}
	return deleted, nil
}

// AddConversation puts a conversation in a group. Adding it twice is not an
// error. The caller must have checked that the group is the user's and that
// the user may see the conversation.
func (r *GroupRepository) AddConversation(ctx context.Context, groupID, conversationID uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&domain.ConversationGroupMember{GroupID: groupID, ConversationID: conversationID}).Error
	if err != nil {
		return fmt.Errorf("add conversation to group: %w", err)
	}
	return nil
}

// RemoveConversation takes a conversation out of a group. Removing one that is
// not there is not an error.
func (r *GroupRepository) RemoveConversation(ctx context.Context, groupID, conversationID uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND conversation_id = ?", groupID, conversationID).
		Delete(&domain.ConversationGroupMember{}).Error
	if err != nil {
		return fmt.Errorf("remove conversation from group: %w", err)
	}
	return nil
}

// GroupIDsByConversation returns, for each of the given conversations, the IDs
// of the user's groups that contain it. Other users' groups never appear, even
// when they hold the same conversation.
func (r *GroupRepository) GroupIDsByConversation(ctx context.Context, userID uuid.UUID, conversationIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	out := make(map[uuid.UUID][]uuid.UUID, len(conversationIDs))
	if len(conversationIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ConversationID uuid.UUID
		GroupID        uuid.UUID
	}
	err := r.db.WithContext(ctx).
		Table("conversation_group_members AS m").
		Select("m.conversation_id, m.group_id").
		Joins("JOIN conversation_groups g ON g.id = m.group_id").
		Where("g.user_id = ? AND m.conversation_id IN ?", userID, conversationIDs).
		Order("lower(g.name)").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("load conversation groups: %w", err)
	}
	for _, row := range rows {
		out[row.ConversationID] = append(out[row.ConversationID], row.GroupID)
	}
	return out, nil
}
