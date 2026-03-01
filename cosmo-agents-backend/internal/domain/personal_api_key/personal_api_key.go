package personal_api_key

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/user"

	"github.com/google/uuid"
)

// PersonalApiKey represents a user API key.
type PersonalApiKey struct {
	base.Base
	base.TimestampMixin

	UserID     uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Name       string     `gorm:"type:text;not null" json:"name"`
	HashedKey  string     `gorm:"type:text;not null" json:"hashed_key"`
	Prefix     string     `gorm:"type:text;not null" json:"prefix"`
	ExpiresAt  time.Time  `gorm:"type:timestamptz;not null" json:"expires_at"`
	LastUsedAt *time.Time `gorm:"type:timestamptz" json:"last_used_at,omitempty"`

	User *user.User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// CreatePersonalAPIKeyRequest represents API key creation request
type CreatePersonalAPIKeyRequest struct {
	Name      string        `json:"name" validate:"required"`
	ExpiresAt *FlexibleTime `json:"expires_at,omitempty"`
}

func (PersonalApiKey) TableName() string {
	return "personal_api_keys"
}

// FlexibleTime allows unmarshalling either RFC3339 strings or unix timestamps.
type FlexibleTime struct {
	time.Time
	Valid bool
}

// UnmarshalJSON implements json.Unmarshaler to support string or numeric formats.
func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		ft.Valid = false
		ft.Time = time.Time{}
		return nil
	}

	if data[0] == '"' {
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		if str == "" {
			ft.Valid = false
			ft.Time = time.Time{}
			return nil
		}
		parsed, err := time.Parse(time.RFC3339, str)
		if err != nil {
			return err
		}
		ft.Time = parsed
		ft.Valid = true
		return nil
	}

	var num float64
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	if num == 0 {
		ft.Valid = false
		ft.Time = time.Time{}
		return nil
	}

	timestamp := int64(num)
	var parsed time.Time
	switch {
	case timestamp > 1e12:
		parsed = time.UnixMilli(timestamp)
	default:
		parsed = time.Unix(timestamp, 0)
	}

	ft.Time = parsed
	ft.Valid = true
	return nil
}
