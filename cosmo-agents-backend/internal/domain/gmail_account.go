package domain

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/google_token_store"
	"gorm.io/gorm"
)

// GmailAccountStatus defines the operational status of Gmail account
type GmailAccountStatus string

const (
	GmailAccountStatusActive        GmailAccountStatus = "active"
	GmailAccountStatusInactive      GmailAccountStatus = "inactive"
	GmailAccountStatusMissingScopes GmailAccountStatus = "insufficient scopes"
	GmailAccountStatusSyncRequired  GmailAccountStatus = "needs sync setup"
	GmailAccountStatusInvalidGrant  GmailAccountStatus = "invalid Google grant"
)

// Token expiry thresholds
const (
	TokenExpiryWarningThreshold  = 1 * time.Hour   // Warn 1 hour before expiry
	TokenExpiryCriticalThreshold = 5 * time.Minute // Mark as invalid 5 minutes before expiry
)

// GmailAccount represents a Gmail account with OAuth credentials and settings
// Separated from Agent to follow clean architecture and avoid circular imports
type GmailAccount struct {
	base.Base
	base.TimestampMixin
	base.SoftDeleteMixin

	AgentID       uuid.UUID          `gorm:"type:uuid;not null;uniqueIndex:idx_gmail_accounts_agent_id" json:"agent_id"`
	Email         string             `gorm:"type:text;not null;uniqueIndex:idx_gmail_accounts_email" json:"email"`
	Name          string             `gorm:"type:text" json:"name"`
	Picture       string             `gorm:"type:text" json:"picture"`
	Status        GmailAccountStatus `gorm:"type:text;default:'active';index:idx_gmail_accounts_status" json:"status"`
	Credentials   base.JSON          `gorm:"type:json" json:"-"` // OAuth credentials
	LastHistoryID string             `gorm:"type:text" json:"last_history_id"`
	IsWatching    bool               `gorm:"type:boolean;default:false" json:"is_watching"`
	WatchExpiry   *time.Time         `gorm:"type:timestamp" json:"watch_expiry,omitempty"`

	// Note: Relations handled via UUID references to avoid circular imports
	// Agent relation can be loaded through repository methods when needed
}

// TableName specifies the table name
func (GmailAccount) TableName() string {
	return "gmail_accounts"
}

// BeforeCreate ensures defaults before persisting
func (g *GmailAccount) BeforeCreate(tx *gorm.DB) error {
	if err := g.Base.BeforeCreate(tx); err != nil {
		return err
	}

	if g.Status == "" {
		g.Status = GmailAccountStatusActive
	}

	return nil
}

// GetGoogleTokenStore reconstructs OAuth credentials for the Gmail account
func (g *GmailAccount) GetGoogleTokenStore() (*google_token_store.GoogleTokenStore, error) {
	if len(g.Credentials) == 0 {
		return nil, errors.New("gmail account credentials is empty")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(g.Credentials, &data); err != nil {
		return nil, err
	}

	return google_token_store.NewGoogleTokenStoreFromMap(data)
}

// UpdateGoogleTokenStore merges new OAuth credentials into the credential blob
func (g *GmailAccount) UpdateGoogleTokenStore(tokenStore *google_token_store.GoogleTokenStore) error {
	if tokenStore == nil {
		return errors.New("token store cannot be nil")
	}

	var payload map[string]interface{}
	if len(g.Credentials) != 0 {
		if err := json.Unmarshal(g.Credentials, &payload); err != nil {
			return err
		}
	} else {
		payload = make(map[string]interface{})
	}

	for k, v := range tokenStore.ToMap() {
		payload[k] = v
	}

	updated, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	g.Credentials = base.JSON(updated)
	g.UpdatedAt = time.Now().UTC()

	return nil
}

// SetCredentials sets the credentials field from a map
func (g *GmailAccount) SetCredentials(credentials map[string]interface{}) error {
	return g.Credentials.Marshal(credentials)
}

// GetCredentials retrieves credentials as a map
func (g *GmailAccount) GetCredentials() (map[string]interface{}, error) {
	var creds map[string]interface{}
	err := g.Credentials.Unmarshal(&creds)
	return creds, err
}

// CheckTokenStatus checks if the Gmail account's OAuth token is valid
func (g *GmailAccount) CheckTokenStatus() (GmailAccountStatus, error) {
	if g.Status == GmailAccountStatusInvalidGrant || g.Status == GmailAccountStatusInactive {
		return g.Status, nil
	}

	store, err := g.GetGoogleTokenStore()
	if err != nil {
		return GmailAccountStatusInactive, err
	}

	if store.RefreshToken == "" {
		return GmailAccountStatusInvalidGrant, nil
	}

	now := time.Now()

	if store.Expiry.IsZero() {
		return GmailAccountStatusInvalidGrant, nil
	}

	if store.Expiry.Before(now.Add(TokenExpiryCriticalThreshold)) {
		return GmailAccountStatusInvalidGrant, nil
	}

	return GmailAccountStatusActive, nil
}

// IsWatchExpired checks if the Gmail watch has expired
func (g *GmailAccount) IsWatchExpired() bool {
	if !g.IsWatching || g.WatchExpiry == nil {
		return true
	}
	return time.Now().After(*g.WatchExpiry)
}

// SetWatching updates the watch status and expiry
func (g *GmailAccount) SetWatching(historyID string, expiration int64) {
	g.IsWatching = true
	g.LastHistoryID = historyID
	expiryTime := time.Unix(expiration/1000, 0)
	g.WatchExpiry = &expiryTime
}

// StopWatching stops the Gmail watch
func (g *GmailAccount) StopWatching() {
	g.IsWatching = false
	g.WatchExpiry = nil
}
