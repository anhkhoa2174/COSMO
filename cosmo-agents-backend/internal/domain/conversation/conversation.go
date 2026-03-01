package conversation

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"gorm.io/gorm"
)

// ConversationType enumerates assignment targets for conversations.
type ConversationType string

const (
	ConversationTypeSent  ConversationType = "sent"
	ConversationTypeAI    ConversationType = "assign_to_ai"
	ConversationTypeHuman ConversationType = "assign_to_human"
)

// ConversationStatus represents the read state of a conversation.
type ConversationStatus string

const (
	ConversationStatusRead   ConversationStatus = "read"
	ConversationStatusUnread ConversationStatus = "unread"
)

// Conversation mirrors machine/models/conversation.py.
type Conversation struct {
	base.Base
	base.SoftDeleteMixin
	base.TimestampMixin

	UserID        uuid.UUID          `gorm:"type:uuid;not null;index:idx_conversations_user_id" json:"user_id"`
	GmailThreadID string             `gorm:"type:text;uniqueIndex:idx_conversations_gmail_thread_id" json:"gmail_thread_id"`
	Labels        pq.StringArray     `gorm:"type:text[];default:'{}'" json:"labels,omitempty"`
	Replied       bool               `gorm:"default:false" json:"replied"`
	Status        ConversationStatus `gorm:"type:text;default:'unread'" json:"status"`
	CampaignID    *uuid.UUID         `gorm:"type:uuid" json:"campaign_id,omitempty"`
	AssigneeID    *uuid.UUID         `gorm:"type:uuid" json:"assignee_id,omitempty"`
	Intents       pq.StringArray     `gorm:"type:text[];default:'{}'" json:"intents,omitempty"`
	CMetadata     base.JSONB         `gorm:"type:jsonb;default:'{}';column:cmetadata" json:"cmetadata,omitempty"`
	AgentID       *uuid.UUID         `gorm:"type:uuid" json:"agent_id,omitempty"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.ConversationWithAgent
	// - relations.ConversationWithEmails
	// - relations.ConversationWithCampaign
	// - relations.ConversationWithFullRelations
}

// TableName specifies the table name.
func (Conversation) TableName() string {
	return "conversations"
}

// BeforeCreate sets default values similar to the Python model.
func (c *Conversation) BeforeCreate(tx *gorm.DB) error {
	if err := c.Base.BeforeCreate(tx); err != nil {
		return err
	}

	if len(c.Labels) == 0 {
		c.Labels = pq.StringArray{}
	}
	if len(c.Intents) == 0 {
		c.Intents = pq.StringArray{}
	}
	if len(c.CMetadata) == 0 {
		c.CMetadata = base.JSONB([]byte("{}"))
	}
	if c.Status == "" {
		c.Status = ConversationStatusUnread
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = time.Now().UTC()
	}

	return nil
}

// Read marks the conversation as read.
func (c *Conversation) Read() {
	c.Status = ConversationStatusRead
}

// Unread marks the conversation as unread.
func (c *Conversation) Unread() {
	c.Status = ConversationStatusUnread
}
