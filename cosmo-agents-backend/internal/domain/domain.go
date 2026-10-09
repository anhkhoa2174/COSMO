package domain

// Import all domain subpackages to re-export their types
import (
	"github.com/rockship/cosmo-agents-go/internal/domain/agent"
	"github.com/rockship/cosmo-agents-go/internal/domain/ai"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/campaign"
	"github.com/rockship/cosmo-agents-go/internal/domain/contact"
	"github.com/rockship/cosmo-agents-go/internal/domain/conversation"
	"github.com/rockship/cosmo-agents-go/internal/domain/custom_field"
	"github.com/rockship/cosmo-agents-go/internal/domain/draft_template"
	"github.com/rockship/cosmo-agents-go/internal/domain/email"
	"github.com/rockship/cosmo-agents-go/internal/domain/feedback"
	"github.com/rockship/cosmo-agents-go/internal/domain/file"
	"github.com/rockship/cosmo-agents-go/internal/domain/google_token_store"
	"github.com/rockship/cosmo-agents-go/internal/domain/inbound_lead_form"
	"github.com/rockship/cosmo-agents-go/internal/domain/integration"
	"github.com/rockship/cosmo-agents-go/internal/domain/interaction"
	"github.com/rockship/cosmo-agents-go/internal/domain/knowledge"
	"github.com/rockship/cosmo-agents-go/internal/domain/notification"
	"github.com/rockship/cosmo-agents-go/internal/domain/operation"
	"github.com/rockship/cosmo-agents-go/internal/domain/organization"
	"github.com/rockship/cosmo-agents-go/internal/domain/outreach"
	"github.com/rockship/cosmo-agents-go/internal/domain/personal_api_key"
	"github.com/rockship/cosmo-agents-go/internal/domain/pubsub"
	"github.com/rockship/cosmo-agents-go/internal/domain/role"
	"github.com/rockship/cosmo-agents-go/internal/domain/sale_rep"
	"github.com/rockship/cosmo-agents-go/internal/domain/segmentation"
	"github.com/rockship/cosmo-agents-go/internal/domain/task"
	"github.com/rockship/cosmo-agents-go/internal/domain/template"
	"github.com/rockship/cosmo-agents-go/internal/domain/template_knowledge"
	"github.com/rockship/cosmo-agents-go/internal/domain/user"
	"github.com/rockship/cosmo-agents-go/internal/domain/workflow"
)

// Re-export base types for backward compatibility
type (
	// Base provides the base UUID primary key field
	Base = base.Base
	// TimestampMixin provides created_at and updated_at fields
	TimestampMixin = base.TimestampMixin
	// SoftDeleteMixin provides soft delete functionality
	SoftDeleteMixin = base.SoftDeleteMixin
	// JSON provides JSON field handling
	JSON = base.JSON
	// JSONB provides PostgreSQL JSONB field handling
	JSONB = base.JSONB
)

// Re-export entity types for backward compatibility
type (
	// Agent represents an AI agent that can handle conversations
	Agent              = agent.Agent
	AgentStatus        = agent.AgentStatus
	AgentEmailProvider = agent.AgentEmailProvider

	// AI represents AI configuration
	CompanyInfo = ai.CompanyInfo

	// Organization represents a company/organization
	Organization = organization.Organization

	// User represents an application user
	User = user.User

	// Role represents user roles within organizations
	Role       = role.Role
	RoleName   = role.RoleName
	RoleStatus = role.RoleStatus

	// Campaign represents marketing campaigns
	Campaign          = campaign.Campaign
	CampaignStatus    = campaign.CampaignStatus
	IntentType        = campaign.IntentType
	Handler           = campaign.Handler
	CampaignMember    = campaign.CampaignMember
	EmailSequenceItem = campaign.EmailSequenceItem
	CampaignMetadata  = campaign.CampaignMetadata
	SaleRepNodeState  = campaign.SaleRepNodeState
	SaleRepNodeStates = campaign.SaleRepNodeStates

	// Contact represents contact information
	Contact                = contact.Contact
	ContactSource          = contact.ContactSource
	ListContact            = contact.ListContact
	ListContactAssociation = contact.ListContactAssociation

	// Conversation represents conversations between agents and contacts
	Conversation       = conversation.Conversation
	ConversationStatus = conversation.ConversationStatus
	ConversationType   = conversation.ConversationType
	// ConversationGroup is a user's personal label for AI Inbox conversations
	ConversationGroup       = conversation.Group
	ConversationGroupMember = conversation.GroupMember

	// Template represents email/message templates
	Template         = template.Template
	TemplateCategory = template.TemplateCategory

	// Task represents tasks in the system
	Task           = task.Task
	TaskStatus     = task.TaskStatus
	TaskPriority   = task.TaskPriority
	TaskUpdate     = task.TaskUpdate
	TaskUpdateType = task.TaskUpdateType
	TaskAttributes = task.TaskAttributes

	// Email represents email entities
	Email       = email.Email
	EmailStatus = email.EmailStatus

	// Segmentation entities
	Segmentation      = segmentation.Segmentation
	SegmentationScore = segmentation.ContactSegmentScore

	// CustomField represents custom field definitions
	CustomField         = custom_field.CustomField
	CustomFieldDataType = custom_field.CustomFieldDataType
	CustomFieldEntity   = custom_field.CustomFieldEntity

	// Knowledge represents knowledge base entries
	Knowledge           = knowledge.Knowledge
	KnowledgeSourceType = knowledge.KnowledgeSourceType
	KnowledgeType       = knowledge.KnowledgeType

	// Integration represents external service integrations
	Integration       = integration.Integration
	DuplicationOption = integration.DuplicationOption
	SourceIntegration = integration.SourceIntegration

	// Interaction represents logged interactions
	Interaction = interaction.Interaction

	// Outreach represents outreach state and interactions
	InteractionLog      = outreach.InteractionLog
	OutreachState       = outreach.OutreachState
	Meeting             = outreach.Meeting
	ConversationState   = outreach.ConversationState
	OutreachIntent      = outreach.OutreachIntent
	OutreachScenario    = outreach.Scenario
	OutreachNextStep    = outreach.NextStepAction
	OutreachLastOutcome = outreach.LastOutcome
	MeetingStatus       = outreach.MeetingStatus
	MeetingChannel      = outreach.MeetingChannel

	// Notification represents system notifications
	Notification = notification.Notification

	// Operation represents operations
	Operation       = operation.Operation
	OperationStatus = operation.OperationStatus

	// Workflow represents workflow definitions
	Workflow = workflow.Workflow

	// GoogleTokenStore represents OAuth token storage for Google services
	GoogleTokenStore = google_token_store.GoogleTokenStore

	// File represents file entities
	File = file.File

	// Feedback represents user feedback log
	UserFeedback = feedback.UserFeedback

	// PersonalApiKey represents user API keys
	PersonalApiKey              = personal_api_key.PersonalApiKey
	CreatePersonalAPIKeyRequest = personal_api_key.CreatePersonalAPIKeyRequest
	FlexibleTime                = personal_api_key.FlexibleTime

	// DraftTemplate represents draft templates
	DraftTemplate = draft_template.DraftTemplate

	// InboundLeadForm represents lead capture forms
	InboundLeadForm                       = inbound_lead_form.InboundLeadForm
	FormField                             = inbound_lead_form.FormField
	InboundLeadFormListContactAssociation = inbound_lead_form.InboundLeadFormListContactAssociation

	// SaleRep represents sales representatives
	SaleRep = sale_rep.SaleRep

	// TemplateKnowledge represents template-knowledge relationships
	TemplateKnowledge = template_knowledge.TemplateKnowledge

	// PubSub represents pubsub messaging
	PubSubMessage      = pubsub.PubSubMessage
	PubSubTopicInfo    = pubsub.PubSubTopicInfo
	PubSubPublishInput = pubsub.PubSubPublishInput
)

// Re-export constants for backward compatibility
const (
	// Agent constants
	AgentStatusActive         = agent.AgentStatusActive
	AgentStatusInactive       = agent.AgentStatusInactive
	AgentStatusMissingScopes  = agent.AgentStatusMissingScopes
	AgentStatusSyncRequired   = agent.AgentStatusSyncRequired
	AgentStatusInvalidGrant   = agent.AgentStatusInvalidGrant
	AgentEmailProviderGmail   = agent.AgentEmailProviderGmail
	AgentEmailProviderOutlook = agent.AgentEmailProviderOutlook

	// Campaign constants
	CampaignStatusActive    = campaign.CampaignStatusActive
	CampaignStatusEnded     = campaign.CampaignStatusEnded
	CampaignStatusPaused    = campaign.CampaignStatusPaused
	CampaignStatusDraft     = campaign.CampaignStatusDraft
	CampaignStatusScheduled = campaign.CampaignStatusScheduled
	IntentInterested        = campaign.IntentInterested
	IntentNotInterested     = campaign.IntentNotInterested
	IntentReferral          = campaign.IntentReferral
	IntentRequestForPricing = campaign.IntentRequestForPricing
	IntentRequestForInfo    = campaign.IntentRequestForInfo
	IntentNurture           = campaign.IntentNurture
	IntentDoNotContact      = campaign.IntentDoNotContact
	IntentOutOfOffice       = campaign.IntentOutOfOffice
	IntentUnknown           = campaign.IntentUnknown
	HandlerAI               = campaign.HandlerAI
	HandlerHuman            = campaign.HandlerHuman
	HandlerDraft            = campaign.HandlerDraft
	NOT_AVAILABLE           = contact.NOT_AVAILABLE

	// Contact constants
	ContactSourceCosmoAgents = contact.ContactSourceCosmoAgents
	ContactSourceGoogleAds   = contact.ContactSourceGoogleAds
	ContactSourceCSV         = contact.ContactSourceCSV
	ContactSourceHubspot     = contact.ContactSourceHubspot
	ContactSourceApollo      = contact.ContactSourceApollo
	ContactSourceLinkedIn    = contact.ContactSourceLinkedIn
	ContactSourceFacebookAds = contact.ContactSourceFacebookAds
	ContactSourceTiktokAds   = contact.ContactSourceTiktokAds

	// Conversation constants
	ConversationStatusRead   = conversation.ConversationStatusRead
	ConversationStatusUnread = conversation.ConversationStatusUnread
	ConversationTypeSent     = conversation.ConversationTypeSent
	ConversationTypeAI       = conversation.ConversationTypeAI
	ConversationTypeHuman    = conversation.ConversationTypeHuman

	// Template constants
	TemplateCategoryDraft       = template.TemplateCategoryDraft
	TemplateCategoryOutreach    = template.TemplateCategoryOutreach
	TemplateCategoryOutOfOffice = template.TemplateCategoryOutOfOffice

	// Task constants
	TaskStatusPending        = task.TaskStatusPending
	TaskStatusRunning        = task.TaskStatusRunning
	TaskStatusCancelled      = task.TaskStatusCancelled
	TaskStatusFailed         = task.TaskStatusFailed
	TaskStatusDone           = task.TaskStatusDone
	TaskPriorityLow          = task.TaskPriorityLow
	TaskPriorityMedium       = task.TaskPriorityMedium
	TaskPriorityHigh         = task.TaskPriorityHigh
	TaskPriorityCritical     = task.TaskPriorityCritical
	TaskUpdateTypeResponse   = task.TaskUpdateTypeResponse
	TaskUpdateTypeResolution = task.TaskUpdateTypeResolution
	TaskUpdateTypeUpdate     = task.TaskUpdateTypeUpdate

	// Email constants
	EmailStatusDraft   = email.EmailStatusDraft
	EmailStatusSending = email.EmailStatusSending
	EmailStatusSent    = email.EmailStatusSent
	EmailStatusInbox   = email.EmailStatusInbox

	// CustomField constants
	CustomFieldEntityContact  = custom_field.CustomFieldEntityContact
	CustomFieldEntityCompany  = custom_field.CustomFieldEntityCompany
	CustomFieldDataTypeText   = custom_field.CustomFieldDataTypeText
	CustomFieldDataTypeNumber = custom_field.CustomFieldDataTypeNumber
	CustomFieldDataTypeEmail  = custom_field.CustomFieldDataTypeEmail
	CustomFieldDataTypeSelect = custom_field.CustomFieldDataTypeSelect
	CustomFieldDataTypeDate   = custom_field.CustomFieldDataTypeDate
	CustomFieldDataTypeURL    = custom_field.CustomFieldDataTypeURL

	// Knowledge constants
	KnowledgeSourceWebsite = knowledge.KnowledgeSourceWebsite
	KnowledgeSourceUpload  = knowledge.KnowledgeSourceUpload
	KnowledgeTypePricing   = knowledge.KnowledgeTypePricing
	KnowledgeTypeProduct   = knowledge.KnowledgeTypeProduct
	KnowledgeTypeCaseStudy = knowledge.KnowledgeTypeCaseStudy
	KnowledgeTypeFAQ       = knowledge.KnowledgeTypeFAQ
	KnowledgeTypeOther     = knowledge.KnowledgeTypeOther

	// Integration constants
	DuplicationOptionSkip      = integration.DuplicationOptionSkip
	DuplicationOptionOverwrite = integration.DuplicationOptionOverwrite
	DuplicationOptionMerge     = integration.DuplicationOptionMerge
	DuplicationOptionKeepBoth  = integration.DuplicationOptionKeepBoth
	SourceIntegrationHubspot   = integration.SourceIntegrationHubspot
	SourceIntegrationFacebook  = integration.SourceIntegrationFacebook

	// Role constants
	RoleNameAdmin     = role.RoleNameAdmin
	RoleNameMember    = role.RoleNameMember
	RoleStatusPending = role.RoleStatusPending
	RoleStatusActive  = role.RoleStatusActive

	// Operation constants
	OperationStatusInProgress = operation.OperationStatusInProgress
	OperationStatusSuccess    = operation.OperationStatusSuccess
	OperationStatusFailed     = operation.OperationStatusFailed

	// Outreach constants - Conversation States
	OutreachStateCold        = outreach.StateCold
	OutreachStateNoReply     = outreach.StateNoReply
	OutreachStateReplied     = outreach.StateReplied
	OutreachStatePostMeeting = outreach.StatePostMeeting
	OutreachStateDropped     = outreach.StateDropped

	// Outreach constants - Intents
	OutreachIntentIntro       = outreach.IntentIntro
	OutreachIntentFollowUp    = outreach.IntentFollowUp
	OutreachIntentReEngage    = outreach.IntentReEngage
	OutreachIntentPostMeeting = outreach.IntentPostMeeting

	// Outreach constants - Next Steps
	OutreachNextStepSend       = outreach.NextStepSend
	OutreachNextStepFollowUp   = outreach.NextStepFollowUp
	OutreachNextStepWait       = outreach.NextStepWait
	OutreachNextStepSetMeeting = outreach.NextStepSetMeeting
	OutreachNextStepDrop       = outreach.NextStepDrop

	// Outreach constants - Meeting Status
	OutreachMeetingScheduled = outreach.MeetingScheduled
	OutreachMeetingCompleted = outreach.MeetingCompleted
	OutreachMeetingCancelled = outreach.MeetingCancelled
	OutreachMeetingNoShow    = outreach.MeetingNoShow
)

// Re-export constructor functions for backward compatibility
var (
	// NewGoogleTokenStoreFromMap creates a new Google token store from map
	NewGoogleTokenStoreFromMap = google_token_store.NewGoogleTokenStoreFromMap
)

// Re-export helper functions for backward compatibility
var (
	// Task helper functions
	EmailTypeForIndex        = template.EmailTypeForIndex
	CalculatePositionBetween = template.CalculatePositionBetween
	CalculateNextPosition    = template.CalculateNextPosition
	NormalizePosition        = template.NormalizePosition
	AllIntents               = campaign.AllIntents
	ParseKnowledgeType       = knowledge.ParseKnowledgeType
	NormalizeGroupName       = conversation.NormalizeGroupName
	NormalizeGroupColor      = conversation.NormalizeGroupColor
	KnowledgeTypes           = knowledge.KnowledgeTypes

	// PubSub helper functions
	NewPubSubMessage = pubsub.NewPubSubMessage
)
