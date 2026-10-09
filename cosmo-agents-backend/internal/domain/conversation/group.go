package conversation

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// GroupColors are the colours a conversation group may take. They are names
// rather than hex values so the palette stays the front end's to render.
var GroupColors = []string{"slate", "red", "orange", "amber", "green", "teal", "blue", "violet", "pink"}

// DefaultGroupColor is used when a group is created without a colour.
const DefaultGroupColor = "slate"

// MaxGroupNameLength bounds a group name; it is shown as a chip in the inbox.
const MaxGroupNameLength = 50

// Group is a user's own label for conversations in the AI Inbox. Groups are
// personal: they belong to one user and are never visible to anyone else.
type Group struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Name      string    `gorm:"type:text;not null" json:"name"`
	Color     string    `gorm:"type:text;not null;default:'slate'" json:"color"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name.
func (Group) TableName() string { return "conversation_groups" }

// GroupMember places one conversation in one group.
type GroupMember struct {
	GroupID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"group_id"`
	ConversationID uuid.UUID `gorm:"type:uuid;primaryKey" json:"conversation_id"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name.
func (GroupMember) TableName() string { return "conversation_group_members" }

// NormalizeGroupName trims a proposed name and reports whether it is usable.
func NormalizeGroupName(raw string) (string, bool) {
	name := strings.Join(strings.Fields(raw), " ")
	if name == "" || len([]rune(name)) > MaxGroupNameLength {
		return "", false
	}
	return name, true
}

// NormalizeGroupColor accepts a palette colour in any case; an empty value is
// the default, and anything outside the palette is rejected.
func NormalizeGroupColor(raw string) (string, bool) {
	color := strings.ToLower(strings.TrimSpace(raw))
	if color == "" {
		return DefaultGroupColor, true
	}
	for _, c := range GroupColors {
		if c == color {
			return c, true
		}
	}
	return "", false
}
