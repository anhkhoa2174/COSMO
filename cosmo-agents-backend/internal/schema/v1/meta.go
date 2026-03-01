package v1

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// FacebookTokenResponse represents OAuth token response
type FacebookTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// FacebookTokenEntity represents stored token
type FacebookTokenEntity struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	FBUserID       string     `json:"fb_user_id"`
	TokenType      string     `json:"token_type"`
	TokenExpiresAt *time.Time `json:"token_expires_at,omitempty"`
	PageID         *string    `json:"page_id,omitempty"`
	PageName       *string    `json:"page_name,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ToFacebookTokenEntity converts domain model to response
func ToFacebookTokenEntity(token *domain.FacebookToken) FacebookTokenEntity {
	return FacebookTokenEntity{
		ID:             token.ID,
		UserID:         token.UserID,
		FBUserID:       token.FBUserID,
		TokenType:      string(token.TokenType),
		TokenExpiresAt: token.TokenExpiresAt,
		PageID:         token.PageID,
		PageName:       token.PageName,
		CreatedAt:      token.CreatedAt,
		UpdatedAt:      token.UpdatedAt,
	}
}
