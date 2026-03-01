package facebook_token

import (
	"time"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"

	"github.com/google/uuid"
)

// TokenType enumerates facebook token types.
type TokenType string

const (
	TokenTypeFBUser TokenType = "fb_user"
	TokenTypeFBPage TokenType = "fb_page"
)

// FacebookToken stores user/page level Facebook tokens.
type FacebookToken struct {
	base.Base
	base.TimestampMixin

	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	FBUserID  string    `gorm:"type:text;not null" json:"fb_user_id"`
	TokenType TokenType `gorm:"type:text;not null" json:"token_type"`

	AccessToken    string     `gorm:"type:text;not null" json:"access_token"`
	TokenExpiresAt *time.Time `gorm:"type:timestamptz" json:"token_expires_at,omitempty"`

	PageID   *string `gorm:"type:text" json:"page_id,omitempty"`
	PageName *string `gorm:"type:text" json:"page_name,omitempty"`
}

func (FacebookToken) TableName() string {
	return "facebook_tokens"
}
