package email

import (
	"errors"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// EmailStatus represents the delivery status of an email.
type EmailStatus string

const (
	EmailStatusSending EmailStatus = "sending"
	EmailStatusSent    EmailStatus = "sent"
	EmailStatusInbox   EmailStatus = "inbox"
)

// Email mirrors machine/models/email.py.
type Email struct {
	base.Base
	base.TimestampMixin
	base.SoftDeleteMixin

	UserID         uuid.UUID      `gorm:"type:uuid;not null;index:idx_emails_user_id" json:"user_id"`
	ConversationID *uuid.UUID     `gorm:"type:uuid" json:"conversation_id,omitempty"`
	GmailMessageID string         `gorm:"type:text" json:"gmail_message_id"`
	FromEmail      string         `gorm:"type:text" json:"from_email"`
	ToEmail        string         `gorm:"type:text" json:"to_email"`
	Subject        string         `gorm:"type:text" json:"subject"`
	Content        string         `gorm:"type:text" json:"content"`
	Attachments    pq.StringArray `gorm:"type:uuid[];default:'{}'" json:"attachments,omitempty"`
	Labels         pq.StringArray `gorm:"type:text[];default:'{}'" json:"labels,omitempty"`
	Status         EmailStatus    `gorm:"type:text;default:'sending';index:idx_emails_status" json:"status"`
	CampaignID     *uuid.UUID     `gorm:"type:uuid" json:"campaign_id,omitempty"`
	Intents        pq.StringArray `gorm:"type:text[];default:'{}'" json:"intents,omitempty"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.EmailWithConversation
	// - relations.EmailWithCampaign
	// - relations.EmailWithFullRelations
}

// TableName specifies the table name.
func (Email) TableName() string {
	return "emails"
}

// BeforeCreate sets defaults aligned with the Python implementation.
func (e *Email) BeforeCreate(tx *gorm.DB) error {
	if err := e.Base.BeforeCreate(tx); err != nil {
		return err
	}

	if len(e.Attachments) == 0 {
		e.Attachments = pq.StringArray{}
	}
	if len(e.Labels) == 0 {
		e.Labels = pq.StringArray{}
	}
	if len(e.Intents) == 0 {
		e.Intents = pq.StringArray{}
	}
	if e.Status == "" {
		e.Status = EmailStatusSending
	}
	if e.UpdatedAt.IsZero() {
		e.UpdatedAt = time.Now().UTC()
	}

	return nil
}

// ValidateStatus ensures a status belongs to the enum.
func ValidateStatus(status EmailStatus) error {
	switch status {
	case EmailStatusSending, EmailStatusSent, EmailStatusInbox:
		return nil
	default:
		return errors.New("invalid email status")
	}
}
