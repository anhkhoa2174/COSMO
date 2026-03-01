package user

import (
	"encoding/json"
	"errors"

	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// User represents a user in the system
// Converted from machine/models/user.py
type User struct {
	base.Base
	base.TimestampMixin
	base.SoftDeleteMixin

	Email                 string         `gorm:"uniqueIndex;not null" json:"email"`
	Name                  string         `json:"name"`
	Picture               string         `json:"picture"`
	Credentials           base.JSON      `gorm:"type:json" json:"-"` // Excluded from JSON
	LastHistoryID         string         `json:"last_history_id"`
	Provider              string         `json:"provider"`
	PhoneNumber           pq.StringArray `gorm:"type:text[]" json:"phone_number"`
	StaffEmails           pq.StringArray `gorm:"type:text[]" json:"staff_emails"`
	HubspotCredentials    base.JSON      `gorm:"type:json" json:"hubspot_credentials,omitempty"`
	JobTitle              string         `json:"job_title"`
	HubspotFieldMapping   base.JSON      `gorm:"type:json" json:"hubspot_field_mapping,omitempty"`
	UIMetadata            base.JSONB     `gorm:"type:jsonb;default:'{}'" json:"ui_metadata,omitempty"`
	MetaLlivedAccessToken *string        `json:"meta_llived_access_token,omitempty"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.UserWithOrganizations
	// - relations.UserWithRoles
	// - relations.UserWithNotifications
	// - relations.UserWithAgents
	// - relations.UserWithFullRelations
}

// TableName specifies the table name
func (User) TableName() string {
	return "users"
}

// GetGoogleTokenStore retrieves Google OAuth token from credentials
func (u *User) GetGoogleTokenStore() (map[string]interface{}, error) {
	if u.Credentials == nil {
		return nil, errors.New("user.credentials is empty")
	}

	var creds map[string]interface{}
	if err := json.Unmarshal(u.Credentials, &creds); err != nil {
		return nil, err
	}

	return creds, nil
}

// UpdateGoogleTokenStore updates the credentials field with new token data
func (u *User) UpdateGoogleTokenStore(tokenStore map[string]interface{}) error {
	var existingCreds map[string]interface{}
	if u.Credentials != nil {
		if err := json.Unmarshal(u.Credentials, &existingCreds); err != nil {
			return err
		}
	} else {
		existingCreds = make(map[string]interface{})
	}

	// Merge tokenStore into existingCreds
	for k, v := range tokenStore {
		existingCreds[k] = v
	}

	updatedCreds, err := json.Marshal(existingCreds)
	if err != nil {
		return err
	}

	u.Credentials = base.JSON(updatedCreds)
	return nil
}
