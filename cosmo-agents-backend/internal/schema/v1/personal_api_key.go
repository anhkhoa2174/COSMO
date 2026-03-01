package v1

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// PersonalApiKeyResponse represents API key response (without hashed_key)
type PersonalApiKeyResponse struct {
	ID         uuid.UUID  `json:"id"`
	UserID     uuid.UUID  `json:"user_id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	ExpiresAt  time.Time  `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// ToPersonalApiKeyResponse converts domain model to response
func ToPersonalApiKeyResponse(key *domain.PersonalApiKey) PersonalApiKeyResponse {
	return PersonalApiKeyResponse{
		ID:         key.ID,
		UserID:     key.UserID,
		Name:       key.Name,
		Prefix:     key.Prefix,
		ExpiresAt:  key.ExpiresAt,
		LastUsedAt: key.LastUsedAt,
		CreatedAt:  key.CreatedAt,
		UpdatedAt:  key.UpdatedAt,
	}
}
