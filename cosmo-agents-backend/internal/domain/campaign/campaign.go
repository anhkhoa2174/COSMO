package campaign

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"gorm.io/gorm"
)

// CampaignStatus defines the status of a campaign
type CampaignStatus string

const (
	CampaignStatusActive    CampaignStatus = "active"
	CampaignStatusEnded     CampaignStatus = "ended"
	CampaignStatusPaused    CampaignStatus = "paused"
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusScheduled CampaignStatus = "scheduled"
)

// IntentType defines different types of user intent
type IntentType string

const (
	IntentInterested        IntentType = "Interested"
	IntentNotInterested     IntentType = "Not interested"
	IntentReferral          IntentType = "Referral"
	IntentRequestForPricing IntentType = "Request for pricing"
	IntentRequestForInfo    IntentType = "Request for information"
	IntentNurture           IntentType = "Nurture"
	IntentDoNotContact      IntentType = "Do not contact"
	IntentOutOfOffice       IntentType = "Out of office"
	IntentUnknown           IntentType = "Unknown intent"
)

// IsValid checks if the intent type is valid
func (i IntentType) IsValid() bool {
	switch i {
	case IntentInterested, IntentNotInterested, IntentReferral,
		IntentRequestForPricing, IntentRequestForInfo, IntentNurture,
		IntentDoNotContact, IntentOutOfOffice, IntentUnknown:
		return true
	}
	return false
}

// AllIntents returns all valid intent types
func AllIntents() []IntentType {
	return []IntentType{
		IntentInterested,
		IntentNotInterested,
		IntentReferral,
		IntentRequestForPricing,
		IntentRequestForInfo,
		IntentNurture,
		IntentDoNotContact,
		IntentOutOfOffice,
		IntentUnknown,
	}
}

// Handler defines who handles campaign responses
type Handler string

const (
	HandlerAI    Handler = "Let AI reply"
	HandlerHuman Handler = "Assign to a person"
	HandlerDraft Handler = "Draft an email"
)

// CampaignMember represents a handler configuration for an intent type
type CampaignMember struct {
	Who        Handler     `json:"who"`
	IntentType IntentType  `json:"intent_type"`
	Payload    interface{} `json:"payload"` // Can be AIPayload, HumanPayload, or DraftPayload
}

// EmailSequenceItem represents an email in the sequence
type EmailSequenceItem struct {
	Type      string `json:"type"`
	Content   string `json:"content"`
	Subject   string `json:"subject"`
	ToEmail   string `json:"to_email"`
	FromEmail string `json:"from_email"`
}

// CampaignMetadata holds campaign configuration and sequence
type CampaignMetadata struct {
	Config   []CampaignMember       `json:"config"`
	Sequence []EmailSequenceItem    `json:"sequence,omitempty"`
	Client   map[string]interface{} `json:"client,omitempty"`
}

// SaleRepNodeState tracks round-robin assignment
type SaleRepNodeState struct {
	RRobinCount int `json:"rrobin_count"`
}

// SaleRepNodeStates maps intent types to their state
type SaleRepNodeStates map[IntentType]SaleRepNodeState

// Campaign represents a marketing/outreach campaign
// Converted from machine/models/campaign.py
type Campaign struct {
	base.Base
	base.SoftDeleteMixin
	base.TimestampMixin

	Name           string           `json:"name"`
	Playbook       string           `json:"playbook"`
	UserID         uuid.UUID        `gorm:"type:uuid;not null;index:idx_campaigns_user_id" json:"user_id"`
	OrganizationID *uuid.UUID       `gorm:"type:uuid;index:idx_campaigns_organization_id" json:"organization_id,omitempty"`
	ListContactID  *uuid.UUID       `gorm:"type:uuid" json:"list_contact_id,omitempty"`
	Schedule       *time.Time       `gorm:"type:timestamptz" json:"schedule,omitempty"`
	Status         CampaignStatus   `gorm:"default:'draft'" json:"status"`
	AgentID        *uuid.UUID       `gorm:"type:uuid" json:"agent_id,omitempty"`
	CMetadata      CampaignMetadata `gorm:"type:jsonb;column:cmetadata;default:'{\"config\":[]}'" json:"cmetadata"`

	// Internal state (not exposed in API)
	SaleRepNodeStates SaleRepNodeStates `gorm:"type:jsonb;column:_sale_rep_node_states" json:"-"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.CampaignWithAgent
	// - relations.CampaignWithEmails
	// - relations.CampaignWithConversations
	// - relations.CampaignWithFullRelations
}

// TableName specifies the table name
func (Campaign) TableName() string {
	return "campaigns"
}

// BeforeCreate hook to set default name from playbook
func (c *Campaign) BeforeCreate(tx *gorm.DB) error {
	// Call Base.BeforeCreate first
	if err := c.Base.BeforeCreate(tx); err != nil {
		return err
	}

	// Generate name from playbook if not provided
	if c.Name == "" && c.Playbook != "" {
		// Replace dashes and underscores with spaces, then title case
		re := regexp.MustCompile(`[-_]`)
		c.Name = strings.Title(re.ReplaceAllString(c.Playbook, " "))
	}

	// Ensure CMetadata has non-nil default values so JSONB column is not inserted as NULL.
	if c.CMetadata.Config == nil {
		c.CMetadata.Config = []CampaignMember{}
	}
	if c.CMetadata.Sequence == nil {
		c.CMetadata.Sequence = []EmailSequenceItem{}
	}
	if c.CMetadata.Client == nil {
		c.CMetadata.Client = map[string]interface{}{}
	}

	return nil
}

// GetIntentAssignee retrieves the handler config for a specific intent
func (c *Campaign) GetIntentAssignee(intent IntentType) *CampaignMember {
	for _, cfg := range c.CMetadata.Config {
		if cfg.IntentType == intent {
			return &cfg
		}
	}
	return nil
}

// SetIntentAssignee adds or updates the handler for an intent
func (c *Campaign) SetIntentAssignee(assignee CampaignMember) {
	intent := assignee.IntentType

	// Find and replace existing config
	for i, cfg := range c.CMetadata.Config {
		if cfg.IntentType == intent {
			c.CMetadata.Config[i] = assignee
			return
		}
	}

	// Add new config if not found
	c.CMetadata.Config = append(c.CMetadata.Config, assignee)
}

// GetAndIncrSaleRepCounter gets and increments the round-robin counter
func (c *Campaign) GetAndIncrSaleRepCounter(intentType IntentType) int {
	if c.SaleRepNodeStates == nil {
		c.SaleRepNodeStates = make(SaleRepNodeStates)
	}

	state, exists := c.SaleRepNodeStates[intentType]
	if !exists {
		state = SaleRepNodeState{RRobinCount: 0}
	}

	currentCount := state.RRobinCount
	state.RRobinCount++
	c.SaleRepNodeStates[intentType] = state

	return currentCount
}

// Scan implements sql.Scanner for CampaignMetadata
func (cm *CampaignMetadata) Scan(value interface{}) error {
	if value == nil {
		cm.Config = []CampaignMember{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, cm)
}

// Value implements driver.Valuer for CampaignMetadata
func (cm CampaignMetadata) Value() (driver.Value, error) {
	return json.Marshal(cm)
}

// Scan implements sql.Scanner for SaleRepNodeStates
func (s *SaleRepNodeStates) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, s)
}

// Value implements driver.Valuer for SaleRepNodeStates
func (s SaleRepNodeStates) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}
