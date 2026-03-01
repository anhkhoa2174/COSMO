package agent

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/google_token_store"
	"gorm.io/gorm"
)

// AgentStatus defines the operational status of an agent.
type AgentStatus string

const (
	AgentStatusActive        AgentStatus = "active"
	AgentStatusInactive      AgentStatus = "inactive"
	AgentStatusMissingScopes AgentStatus = "insufficient scopes"
	AgentStatusSyncRequired  AgentStatus = "needs sync setup"
	AgentStatusInvalidGrant  AgentStatus = "invalid Google grant"
)

// Token expiry thresholds
const (
	TokenExpiryWarningThreshold  = 1 * time.Hour   // Warn 1 hour before expiry
	TokenExpiryCriticalThreshold = 5 * time.Minute // Mark as invalid 5 minutes before expiry
)

// AgentEmailProvider represents the supported email providers.
type AgentEmailProvider string

const (
	AgentEmailProviderGmail   AgentEmailProvider = "gmail"
	AgentEmailProviderOutlook AgentEmailProvider = "outlook"
)

const (
	defaultAgentSignature = "Best regard,\n\n{sender_name}\n\n{organization_name}"
	defaultDailyLimit     = 50
	defaultMaxDailyLimit  = 500
	defaultEmailsSent     = 0
)

// Agent mirrors machine/models/agent.py.
type Agent struct {
	base.Base
	base.TimestampMixin
	base.SoftDeleteMixin

	UserID          uuid.UUID          `gorm:"type:uuid;not null;index:idx_agents_user_id;uniqueIndex:idx_agents_user_id_email,priority:1" json:"user_id"`
	OrganizationID  *uuid.UUID         `gorm:"type:uuid;index:idx_agents_organization_id" json:"organization_id,omitempty"`
	Name            string             `json:"name"`
	CMetadata       base.JSONB         `gorm:"type:jsonb;default:'{}';column:cmetadata" json:"cmetadata,omitempty"`
	Persona         pq.StringArray     `gorm:"type:text[]" json:"persona,omitempty"`
	Email           string             `gorm:"type:text;uniqueIndex:idx_agents_user_id_email,priority:2" json:"email"`
	Status          AgentStatus        `gorm:"type:text;default:'active';index:idx_agents_status" json:"status"`
	EmailProvider   AgentEmailProvider `gorm:"type:text" json:"email_provider"`
	Signature       string             `gorm:"type:text" json:"signature"`
	Picture         string             `gorm:"type:text" json:"picture"`
	Credentials     base.JSON          `gorm:"type:json" json:"-"` // sensitive data
	LastHistoryID   string             `gorm:"type:text" json:"last_history_id"`
	DailyLimit      *int               `gorm:"type:int" json:"daily_limit,omitempty"`
	MaxDailyLimit   *int               `gorm:"type:int" json:"max_daily_limit,omitempty"`
	ValidCred       *bool              `gorm:"type:boolean" json:"valid_cred,omitempty"`
	EmailsSentToday *int               `gorm:"type:int" json:"emails_sent_today,omitempty"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.AgentWithUser
	// - relations.AgentWithOrganization
	// - relations.AgentWithConversations
	// - relations.AgentWithFullRelations
}

// TableName specifies the table name.
func (Agent) TableName() string {
	return "agents"
}

// BeforeCreate ensures defaults align with the Python model before persisting.
func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	if err := a.Base.BeforeCreate(tx); err != nil {
		return err
	}

	if len(a.CMetadata) == 0 {
		a.CMetadata = base.JSONB("{}")
	}
	if a.Persona == nil {
		a.Persona = pq.StringArray{}
	}
	if a.Signature == "" {
		a.Signature = defaultAgentSignature
	}
	if a.Status == "" {
		a.Status = AgentStatusActive
	}
	if a.DailyLimit == nil {
		a.DailyLimit = intPtr(defaultDailyLimit)
	}
	if a.MaxDailyLimit == nil {
		a.MaxDailyLimit = intPtr(defaultMaxDailyLimit)
	}
	if a.ValidCred == nil {
		a.ValidCred = boolPtr(true)
	}
	if a.EmailsSentToday == nil {
		a.EmailsSentToday = intPtr(defaultEmailsSent)
	}

	return nil
}

// SetMetadata sets the metadata from a map
func (a *Agent) SetMetadata(metadata map[string]any) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	a.CMetadata = base.JSONB(data)
	return nil
}

// GetMetadata returns the metadata as a map
func (a *Agent) GetMetadata() (map[string]any, error) {
	var metadata map[string]any
	if err := json.Unmarshal(a.CMetadata, &metadata); err != nil {
		return nil, err
	}
	return metadata, nil
}

// GetGoogleTokenStore reconstructs OAuth credentials for the agent.
func (a *Agent) GetGoogleTokenStore() (*google_token_store.GoogleTokenStore, error) {
	if len(a.Credentials) == 0 {
		return nil, errors.New("agent.credentials is empty")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(a.Credentials, &data); err != nil {
		return nil, err
	}

	return google_token_store.NewGoogleTokenStoreFromMap(data)
}

// UpdateGoogleTokenStore merges new OAuth credentials into the credential blob.
func (a *Agent) UpdateGoogleTokenStore(tokenStore *google_token_store.GoogleTokenStore) error {
	if tokenStore == nil {
		return errors.New("token store cannot be nil")
	}

	var payload map[string]interface{}
	if len(a.Credentials) != 0 {
		if err := json.Unmarshal(a.Credentials, &payload); err != nil {
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

	a.Credentials = base.JSON(updated)
	a.UpdatedAt = time.Now().UTC()

	return nil
}

// SetCredentials sets the credentials field from a map
func (a *Agent) SetCredentials(credentials map[string]interface{}) error {
	return a.Credentials.Marshal(credentials)
}

// GetCredentials retrieves credentials as a map
func (a *Agent) GetCredentials() (map[string]interface{}, error) {
	var creds map[string]interface{}
	err := a.Credentials.Unmarshal(&creds)
	return creds, err
}

func intPtr(v int) *int {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}

func (a *Agent) CheckTokenStatus() (AgentStatus, error) {
	if a.EmailProvider != AgentEmailProviderGmail {
		return a.Status, nil
	}

	if a.Status == AgentStatusInvalidGrant || a.Status == AgentStatusInactive {
		return a.Status, nil
	}

	store, err := a.GetGoogleTokenStore()
	if err != nil {
		return AgentStatusInactive, err
	}

	if store.RefreshToken == "" {
		return AgentStatusInvalidGrant, nil
	}

	now := time.Now()

	if store.Expiry.IsZero() {
		return AgentStatusInvalidGrant, nil
	}

	if store.Expiry.Before(now.Add(TokenExpiryCriticalThreshold)) {
		return AgentStatusInvalidGrant, nil
	}

	return AgentStatusActive, nil
}
