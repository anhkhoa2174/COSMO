package contact

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"gorm.io/gorm"
)

// ContactSource defines the source of a contact
type ContactSource string

const (
	ContactSourceCosmoAgents ContactSource = "cosmo-agents"
	ContactSourceGoogleAds   ContactSource = "google-ads"
	ContactSourceCSV         ContactSource = "csv"
	ContactSourceHubspot     ContactSource = "hubspot"
	ContactSourceApollo      ContactSource = "Apollo"
	ContactSourceLinkedIn    ContactSource = "LinkedIn"
	ContactSourceFacebookAds ContactSource = "facebook-ads"
	ContactSourceTiktokAds   ContactSource = "tiktok-ads"
)

// ContactStatus defines the readiness status of a contact
type ContactStatus string

const (
	// ContactStatusReady indicates the contact has all required fields
	ContactStatusReady ContactStatus = "ready"
	// ContactStatusPending indicates the contact is missing required fields
	ContactStatusPending ContactStatus = "pending"
)

// ContextLevel defines how much context we have about a contact
type ContextLevel string

const (
	ContextLevelLow    ContextLevel = "LOW"
	ContextLevelMedium ContextLevel = "MEDIUM"
	ContextLevelHigh   ContextLevel = "HIGH"
)

// OutreachDecision defines the recommended outreach action
type OutreachDecision string

const (
	OutreachDecisionIntro    OutreachDecision = "INTRO"
	OutreachDecisionFollowUp OutreachDecision = "FOLLOW-UP"
	OutreachDecisionNurture  OutreachDecision = "NURTURE"
	OutreachDecisionHold     OutreachDecision = "HOLD"
)

// NextStep defines the recommended next action
type NextStep string

const (
	NextStepSend       NextStep = "SEND"
	NextStepFollowUp   NextStep = "FOLLOW_UP"
	NextStepSetMeeting NextStep = "SET_MEETING"
	NextStepWait       NextStep = "WAIT"
	NextStepDrop       NextStep = "DROP"
)

// BusinessStage defines the lifecycle/pipeline stage for a contact
type BusinessStage string

const (
	// Lifecycle stages (industry standard)
	BusinessStageSubscriber  BusinessStage = "SUBSCRIBER"  // Opted in, no outreach yet
	BusinessStageLead        BusinessStage = "LEAD"        // New contact, ready for outreach
	BusinessStageQualified   BusinessStage = "QUALIFIED"   // Replied positive / met criteria
	BusinessStageOpportunity BusinessStage = "OPPORTUNITY" // Meeting booked / deal in progress
	BusinessStageCustomer    BusinessStage = "CUSTOMER"    // Deal closed
	BusinessStageAdvocate    BusinessStage = "ADVOCATE"    // Loyal customer, referrals
	// Legacy values (backward compatibility)
	BusinessStagePreSales  BusinessStage = "PRE_SALES"
	BusinessStageSales     BusinessStage = "SALES"
	BusinessStagePostSales BusinessStage = "POST_SALES"
)

const (
	// NOT_AVAILABLE is the default value for missing contact fields
	// Matches Python's _constants.NOT_AVAILABLE
	NOT_AVAILABLE = "N/A"
)

// Contact represents a contact in the CRM
// Converted from machine/models/contact.py
type Contact struct {
	base.Base
	base.TimestampMixin
	base.SoftDeleteMixin

	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_contacts_user_id" json:"user_id"`
	SourceID  string    `gorm:"not null" json:"source_id"`
	HubspotID *string   `json:"hubspot_id,omitempty"`
	Source    string    `gorm:"not null" json:"source"` // ContactSource value
	Name      string    `gorm:"default:'N/A'" json:"name"`
	Company   string    `gorm:"default:'N/A'" json:"company"`
	JobTitle  string    `gorm:"default:'N/A'" json:"job_title"`
	Address   string    `gorm:"default:'N/A'" json:"address"`
	City      string    `gorm:"default:'N/A'" json:"city"`
	Country   string    `gorm:"default:'N/A'" json:"country"`
	State     string    `gorm:"default:'N/A'" json:"state"`
	Zip       string    `gorm:"default:'N/A'" json:"zip"`
	// The operator class goes inside the index expression. As `option:` GORM
	// appended it after the column list (USING gin("profile") jsonb_path_ops),
	// which PostgreSQL rejects; migration 000028 builds the same index as
	// USING gin (profile jsonb_path_ops), and this tag now matches it.
	Profile           base.JSONB `gorm:"type:jsonb;default:'{}';index:idx_contacts_profile_gin,type:gin,expression:profile jsonb_path_ops" json:"profile"`
	ConfirmedFacts    base.JSONB `gorm:"type:jsonb;default:'{}'" json:"confirmed_facts"`
	AIInsights        base.JSONB `gorm:"type:jsonb;default:'{}'" json:"ai_insights"`
	InsightValidation base.JSONB `gorm:"type:jsonb;default:'{}'" json:"insight_validation"`
	Scores            base.JSONB `gorm:"type:jsonb;default:'{}'" json:"scores"`
	DoNotContact      bool       `gorm:"default:false" json:"do_not_contact"`
	OrganizationID    *uuid.UUID `gorm:"type:uuid" json:"organization_id,omitempty"`
	Tags              base.JSONB `gorm:"type:jsonb;default:'{}'" json:"tags"`

	// Status fields for contact readiness
	Status             string     `gorm:"default:'pending';index:idx_contacts_status" json:"status"`
	MissingFields      base.JSONB `gorm:"type:jsonb;default:'[]'" json:"missing_fields"`
	ContactInformation string     `gorm:"default:''" json:"contact_information"` // LinkedIn URL for LinkedIn source, email for others

	// Outreach context fields
	Industry         string `gorm:"default:''" json:"industry"`                                             // e.g., Fintech, SaaS
	ContactChannel   string `gorm:"default:''" json:"contact_channel"`                                      // e.g., LinkedIn, Email
	ContextLevel     string `gorm:"default:'LOW'" json:"context_level"`                                     // LOW, MEDIUM, HIGH
	OutreachDecision string `gorm:"default:'INTRO'" json:"outreach_decision"`                               // INTRO, FOLLOW-UP, NURTURE, HOLD
	Scenario         string `gorm:"default:''" json:"scenario"`                                             // Role-based, Post-reply, etc.
	MessageDraft     string `gorm:"type:text;default:''" json:"message_draft"`                              // Draft message for outreach
	LastOutcome      string `gorm:"default:''" json:"last_outcome"`                                         // Result of last outreach
	NextStep         string `gorm:"default:'SEND'" json:"next_step"`                                        // SEND, FOLLOW_UP, SET_MEETING, WAIT, DROP
	OutreachStage    string `gorm:"default:'COLD'" json:"outreach_stage"`                                   // COLD, NO_REPLY, REPLIED, POST_MEETING, DROPPED
	FollowupCount    int    `gorm:"default:0" json:"followup_count"`                                        // Number of follow-up messages sent
	Meeting          string `gorm:"default:''" json:"meeting"`                                              // Meeting details if scheduled
	BusinessStage    string `gorm:"default:'LEAD';index:idx_contacts_business_stage" json:"business_stage"` // SUBSCRIBER, LEAD, QUALIFIED, OPPORTUNITY, CUSTOMER, ADVOCATE

	// Latest decision of the next-step engine (migration 000058). Written only
	// when the engine is on; the cadence fields above are untouched by it.
	NextAction           *string    `gorm:"column:next_action" json:"next_action,omitempty"`
	NextActionArgs       base.JSONB `gorm:"column:next_action_args;type:jsonb" json:"next_action_args,omitempty"`
	NextActionReason     *string    `gorm:"column:next_action_reason" json:"next_action_reason,omitempty"`
	NextActionDueAt      *time.Time `gorm:"column:next_action_due_at" json:"next_action_due_at,omitempty"`
	NextActionDecisionID *uuid.UUID `gorm:"column:next_action_decision_id;type:uuid" json:"next_action_decision_id,omitempty"`

	// Relationships
	ListContacts []ListContact `gorm:"many2many:list_contact_association" json:"list_contacts,omitempty"`
}

// TableName specifies the table name
func (Contact) TableName() string {
	return "contacts"
}

// BeforeCreate hook to set default values
func (c *Contact) BeforeCreate(tx *gorm.DB) error {
	// Call Base.BeforeCreate first
	if err := c.Base.BeforeCreate(tx); err != nil {
		return err
	}

	// Set default source_id if not provided
	if c.SourceID == "" {
		c.SourceID = c.ID.String()
	}

	// Set default source if not provided
	if c.Source == "" {
		c.Source = string(ContactSourceCosmoAgents)
	}

	// Calculate and set status
	c.CalculateStatus()

	return nil
}

// CalculateStatus evaluates required fields and sets Status and MissingFields
// Required fields: name, company, job_title, source, contact_information
func (c *Contact) CalculateStatus() {
	missing := c.GetMissingFields()

	if len(missing) == 0 {
		c.Status = string(ContactStatusReady)
	} else {
		c.Status = string(ContactStatusPending)
	}

	// Store missing fields as JSON array
	c.MissingFields = nil
	if len(missing) > 0 {
		_ = c.MissingFields.Marshal(missing)
	}
}

// GetMissingFields returns a list of required fields that are missing or invalid
// Required fields: name, company, job_title, source, contact_information
func (c *Contact) GetMissingFields() []string {
	var missing []string

	// Check name
	if c.Name == "" || c.Name == NOT_AVAILABLE {
		missing = append(missing, "name")
	}

	// Check company
	if c.Company == "" || c.Company == NOT_AVAILABLE {
		missing = append(missing, "company")
	}

	// Check job_title
	if c.JobTitle == "" || c.JobTitle == NOT_AVAILABLE {
		missing = append(missing, "job_title")
	}

	// Check source
	if c.Source == "" {
		missing = append(missing, "source")
	}

	// Check contact_information - now stored as a dedicated field
	// For LinkedIn sources: contains linkedin_url
	// For other sources: contains email
	if c.ContactInformation != "" {
		// ContactInformation field is populated, no need to check profile
	} else {
		// Fallback: check profile for backwards compatibility
		hasContactInfo := false
		var profile map[string]interface{}
		if err := c.Profile.Unmarshal(&profile); err == nil {
			if c.Source == string(ContactSourceLinkedIn) || c.Source == "linkedin_extension" || c.Source == "linkedin_connections" {
				// Check linkedin_url in profile
				if linkedinURL, ok := profile["linkedin_url"].(string); ok && linkedinURL != "" {
					hasContactInfo = true
				}
			} else {
				// For Apollo or other sources, check email in profile
				if email, ok := profile["email"].(string); ok && email != "" && email != NOT_AVAILABLE && !isPlaceholderEmail(email) {
					hasContactInfo = true
				}
			}
		}
		if !hasContactInfo {
			missing = append(missing, "contact_information")
		}
	}

	return missing
}

// isPlaceholderEmail checks if email is a system-generated placeholder
func isPlaceholderEmail(email string) bool {
	return len(email) > 8 && email[:8] == "unknown-"
}

// ListContactAssociation is the many-to-many association table
type ListContactAssociation struct {
	base.Base
	ListContactID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_list_contact_association" json:"list_contact_id"`
	ContactID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_list_contact_association" json:"contact_id"`
}

// TableName specifies the table name
func (ListContactAssociation) TableName() string {
	return "list_contact_association"
}

// ListContact represents a list of contacts
// Converted from machine/models/contact.py ListContact class
type ListContact struct {
	base.Base
	base.TimestampMixin
	base.SoftDeleteMixin

	UserID         uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_list_contacts_source_id_source_user_id" json:"user_id"`
	Name           string     `json:"name"`
	Source         string     `gorm:"uniqueIndex:idx_list_contacts_source_id_source_user_id" json:"source"`
	SourceID       string     `gorm:"uniqueIndex:idx_list_contacts_source_id_source_user_id" json:"source_id"`
	HubspotID      *string    `json:"hubspot_id,omitempty"`
	OrganizationID *uuid.UUID `gorm:"type:uuid" json:"organization_id,omitempty"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.ListContactWithContacts
	// - relations.ListContactWithInboundLeadForms
	// - relations.ListContactWithFullRelations
	Contacts []Contact `gorm:"many2many:list_contact_association" json:"contacts,omitempty"`
}

// TableName specifies the table name
func (ListContact) TableName() string {
	return "list_contacts"
}

// BeforeCreate hook to set default values
func (lc *ListContact) BeforeCreate(tx *gorm.DB) error {
	// Call Base.BeforeCreate first
	if err := lc.Base.BeforeCreate(tx); err != nil {
		return err
	}

	// Set default source_id if not provided
	if lc.SourceID == "" {
		lc.SourceID = lc.ID.String()
	}

	// Set default source if not provided
	if lc.Source == "" {
		lc.Source = string(ContactSourceCosmoAgents)
	}

	return nil
}
