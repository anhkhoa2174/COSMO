package google_token_store

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// GoogleTokenStore represents OAuth token data persisted in the credentials JSON field.
type GoogleTokenStore struct {
	AccessToken  string
	RefreshToken string
	Scopes       []string
	Expiry       time.Time
}

// googleTokenStoreDTO mirrors the serialized shape stored in the database.
type googleTokenStoreDTO struct {
	Token        string   `json:"token"`
	RefreshToken string   `json:"refresh_token"`
	Scopes       []string `json:"scopes"`
	Expiry       string   `json:"expiry"`
}

// NewGoogleTokenStoreFromMap reconstructs a token store from raw credential data.
func NewGoogleTokenStoreFromMap(data map[string]interface{}) (*GoogleTokenStore, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var dto googleTokenStoreDTO
	if err := json.Unmarshal(payload, &dto); err != nil {
		return nil, err
	}

	if dto.Token == "" || dto.RefreshToken == "" || dto.Expiry == "" {
		return nil, errors.New("missing required token fields")
	}

	expiry := dto.Expiry
	if strings.HasSuffix(expiry, "Z") {
		expiry = strings.TrimSuffix(expiry, "Z") + "+00:00"
	}

	parsed, err := time.Parse(time.RFC3339, expiry)
	if err != nil {
		return nil, err
	}

	return &GoogleTokenStore{
		AccessToken:  dto.Token,
		RefreshToken: dto.RefreshToken,
		Scopes:       dto.Scopes,
		Expiry:       parsed.UTC(),
	}, nil
}

// ToMap converts the token store to the on-disk representation.
func (g GoogleTokenStore) ToMap() map[string]interface{} {
	expiry := g.Expiry.UTC().Truncate(time.Second).Format(time.RFC3339)
	if strings.HasSuffix(expiry, "+00:00") {
		expiry = strings.TrimSuffix(expiry, "+00:00") + "Z"
	}

	return map[string]interface{}{
		"token":         g.AccessToken,
		"refresh_token": g.RefreshToken,
		"scopes":        g.Scopes,
		"expiry":        expiry,
	}
}
