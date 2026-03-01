package context

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// OrgContext stores organization-level context shared by all users in the org
// This includes ICP definitions, company knowledge, and shared configurations
type OrgContext struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OrganizationID uuid.UUID `json:"organization_id" gorm:"type:uuid;uniqueIndex;not null"`

	// ICP (Ideal Customer Profile) definition
	ICPDefinition string `json:"icp_definition" gorm:"type:text"`

	// Target industries, company sizes, etc.
	TargetCriteria domain.JSONB `json:"target_criteria" gorm:"type:jsonb;default:'{}'"`

	// Company knowledge base - product info, competitors, value props
	CompanyKnowledge domain.JSONB `json:"company_knowledge" gorm:"type:jsonb;default:'{}'"`

	// Common pain points and goals for this org's target market
	CommonPainPoints domain.JSONB `json:"common_pain_points" gorm:"type:jsonb;default:'[]'"`
	CommonGoals      domain.JSONB `json:"common_goals" gorm:"type:jsonb;default:'[]'"`

	// Competitors information
	Competitors domain.JSONB `json:"competitors" gorm:"type:jsonb;default:'[]'"`

	// Messaging guidelines and tone
	MessagingGuidelines string `json:"messaging_guidelines" gorm:"type:text"`

	// Custom instructions for AI agents
	AIInstructions string `json:"ai_instructions" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (OrgContext) TableName() string {
	return "org_contexts"
}

// UserContext stores user-specific context for each BD/sales rep
type UserContext struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         uuid.UUID `json:"user_id" gorm:"type:uuid;uniqueIndex;not null"`
	OrganizationID uuid.UUID `json:"organization_id" gorm:"type:uuid;index;not null"`

	// User preferences for AI responses
	Preferences domain.JSONB `json:"preferences" gorm:"type:jsonb;default:'{}'"`

	// Communication style preferences
	CommunicationStyle string `json:"communication_style" gorm:"type:varchar(50);default:'professional'"`

	// Preferred email length: short, medium, long
	PreferredEmailLength string `json:"preferred_email_length" gorm:"type:varchar(20);default:'medium'"`

	// User's personal notes and learnings
	PersonalNotes string `json:"personal_notes" gorm:"type:text"`

	// Recent contacts the user has been working with (for quick context)
	RecentContacts domain.JSONB `json:"recent_contacts" gorm:"type:jsonb;default:'[]'"`

	// Custom AI instructions specific to this user
	AIInstructions string `json:"ai_instructions" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func (UserContext) TableName() string {
	return "user_contexts"
}

// ConversationHistory stores chat history for context continuity
type ConversationHistory struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         uuid.UUID `json:"user_id" gorm:"type:uuid;index;not null"`
	OrganizationID uuid.UUID `json:"organization_id" gorm:"type:uuid;index;not null"`

	// Optional: link to specific contact being discussed
	ContactID *uuid.UUID `json:"contact_id" gorm:"type:uuid;index"`

	// Session ID to group related messages
	SessionID string `json:"session_id" gorm:"type:varchar(100);index"`

	// Message role: user, assistant, system
	Role string `json:"role" gorm:"type:varchar(20);not null"`

	// Message content
	Content string `json:"content" gorm:"type:text;not null"`

	// Tools used in this message (if assistant)
	ToolsUsed domain.JSONB `json:"tools_used" gorm:"type:jsonb;default:'[]'"`

	// Metadata (tokens used, latency, etc.)
	Metadata domain.JSONB `json:"metadata" gorm:"type:jsonb;default:'{}'"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime;index"`
}

func (ConversationHistory) TableName() string {
	return "conversation_histories"
}

// ===== Helper structs for JSON fields =====

type TargetCriteria struct {
	Industries   []string `json:"industries,omitempty"`
	CompanySizes []string `json:"company_sizes,omitempty"` // startup, smb, mid-market, enterprise
	Regions      []string `json:"regions,omitempty"`
	MinRevenue   int64    `json:"min_revenue,omitempty"`
	MaxRevenue   int64    `json:"max_revenue,omitempty"`
	MinEmployees int      `json:"min_employees,omitempty"`
	MaxEmployees int      `json:"max_employees,omitempty"`
	Technologies []string `json:"technologies,omitempty"`
	Keywords     []string `json:"keywords,omitempty"`
}

type CompanyKnowledge struct {
	ProductName        string   `json:"product_name,omitempty"`
	ProductDescription string   `json:"product_description,omitempty"`
	ValuePropositions  []string `json:"value_propositions,omitempty"`
	KeyFeatures        []string `json:"key_features,omitempty"`
	UseCases           []string `json:"use_cases,omitempty"`
	Pricing            string   `json:"pricing,omitempty"`
	Website            string   `json:"website,omitempty"`
}

type Competitor struct {
	Name            string   `json:"name"`
	Website         string   `json:"website,omitempty"`
	Strengths       []string `json:"strengths,omitempty"`
	Weaknesses      []string `json:"weaknesses,omitempty"`
	Differentiators []string `json:"differentiators,omitempty"` // How we're different/better
}

type UserPreferences struct {
	Language           string `json:"language,omitempty"` // en, vi, etc.
	Timezone           string `json:"timezone,omitempty"`
	ResponseDetail     string `json:"response_detail,omitempty"` // brief, detailed
	NotifyOnEnrichment bool   `json:"notify_on_enrichment,omitempty"`
	AutoEnroll         bool   `json:"auto_enroll,omitempty"`
}

type RecentContact struct {
	ContactID   uuid.UUID `json:"contact_id"`
	ContactName string    `json:"contact_name"`
	LastTouched time.Time `json:"last_touched"`
	Action      string    `json:"action"` // viewed, emailed, called, analyzed
}
