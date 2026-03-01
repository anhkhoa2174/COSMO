package relations

import (
	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/agent"
	"github.com/rockship/cosmo-agents-go/internal/domain/campaign"
	"github.com/rockship/cosmo-agents-go/internal/domain/contact"
	"github.com/rockship/cosmo-agents-go/internal/domain/conversation"
	"github.com/rockship/cosmo-agents-go/internal/domain/email"
	"github.com/rockship/cosmo-agents-go/internal/domain/knowledge"
	"github.com/rockship/cosmo-agents-go/internal/domain/notification"
	"github.com/rockship/cosmo-agents-go/internal/domain/organization"
	"github.com/rockship/cosmo-agents-go/internal/domain/role"
	"github.com/rockship/cosmo-agents-go/internal/domain/task"
	"github.com/rockship/cosmo-agents-go/internal/domain/template"
	"github.com/rockship/cosmo-agents-go/internal/domain/user"
)

// AgentWithConversations extends Agent with Conversation relationships
type AgentWithConversations struct {
	agent.Agent
	Conversations []conversation.Conversation `gorm:"foreignKey:AgentID;constraint:OnDelete:SET NULL" json:"conversations,omitempty"`
}

// AgentWithUser extends Agent with User relationship
type AgentWithUser struct {
	agent.Agent
	User *user.User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
}

// AgentWithOrganization extends Agent with Organization relationship
type AgentWithOrganization struct {
	agent.Agent
	Organization *organization.Organization `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"organization,omitempty"`
}

// AgentWithFullRelations extends Agent with all relationships
type AgentWithFullRelations struct {
	agent.Agent
	User          *user.User                  `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"user,omitempty"`
	Organization  *organization.Organization  `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"organization,omitempty"`
	Conversations []conversation.Conversation `gorm:"foreignKey:AgentID;constraint:OnDelete:SET NULL" json:"conversations,omitempty"`
}

// ConversationWithAgent extends Conversation with Agent relationship
type ConversationWithAgent struct {
	conversation.Conversation
	Agent *agent.Agent `gorm:"foreignKey:AgentID;constraint:OnDelete:SET NULL" json:"agent,omitempty"`
}

// ConversationWithEmails extends Conversation with Email relationships
type ConversationWithEmails struct {
	conversation.Conversation
	Emails []email.Email `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE" json:"emails,omitempty"`
}

// ConversationWithCampaign extends Conversation with Campaign relationship
type ConversationWithCampaign struct {
	conversation.Conversation
	Campaign *campaign.Campaign `gorm:"foreignKey:CampaignID;constraint:OnDelete:CASCADE" json:"campaign,omitempty"`
}

// ConversationWithFullRelations extends Conversation with all relationships
type ConversationWithFullRelations struct {
	conversation.Conversation
	Agent    *agent.Agent       `gorm:"foreignKey:AgentID;constraint:OnDelete:SET NULL" json:"agent,omitempty"`
	Emails   []email.Email      `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE" json:"emails,omitempty"`
	Campaign *campaign.Campaign `gorm:"foreignKey:CampaignID;constraint:OnDelete:CASCADE" json:"campaign,omitempty"`
}

// EmailWithConversation extends Email with Conversation relationship
type EmailWithConversation struct {
	email.Email
	Conversation *conversation.Conversation `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE" json:"conversation,omitempty"`
}

// EmailWithCampaign extends Email with Campaign relationship
type EmailWithCampaign struct {
	email.Email
	Campaign *campaign.Campaign `gorm:"foreignKey:CampaignID;constraint:OnDelete:SET NULL" json:"campaign,omitempty"`
}

// EmailWithFullRelations extends Email with all relationships
type EmailWithFullRelations struct {
	email.Email
	Conversation *conversation.Conversation `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE" json:"conversation,omitempty"`
	Campaign     *campaign.Campaign         `gorm:"foreignKey:CampaignID;constraint:OnDelete:SET NULL" json:"campaign,omitempty"`
}

// UserWithOrganizations extends User with Organization relationships
type UserWithOrganizations struct {
	user.User
	Organizations []organization.Organization `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"organizations,omitempty"`
}

// UserWithRoles extends User with Role relationships
type UserWithRoles struct {
	user.User
	Roles []role.Role `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"roles,omitempty"`
}

// UserWithNotifications extends User with Notification relationships
type UserWithNotifications struct {
	user.User
	Notifications []notification.Notification `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"notifications,omitempty"`
}

// UserWithAgents extends User with Agent relationships
type UserWithAgents struct {
	user.User
	Agents []agent.Agent `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"agents,omitempty"`
}

// UserWithFullRelations extends User with all relationships
type UserWithFullRelations struct {
	user.User
	Organizations []organization.Organization `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"organizations,omitempty"`
	Roles         []role.Role                 `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"roles,omitempty"`
	Notifications []notification.Notification `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"notifications,omitempty"`
	Agents        []agent.Agent               `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE" json:"agents,omitempty"`
}

// OrganizationWithUser extends Organization with User relationship
type OrganizationWithUser struct {
	organization.Organization
	User *user.User `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"user,omitempty"`
}

// OrganizationWithRoles extends Organization with Role relationships
type OrganizationWithRoles struct {
	organization.Organization
	Roles []role.Role `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"roles,omitempty"`
}

// OrganizationWithAgents extends Organization with Agent relationships
type OrganizationWithAgents struct {
	organization.Organization
	Agents []agent.Agent `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"agents,omitempty"`
}

// OrganizationWithFullRelations extends Organization with all relationships
type OrganizationWithFullRelations struct {
	organization.Organization
	User   *user.User    `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL" json:"user,omitempty"`
	Roles  []role.Role   `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"roles,omitempty"`
	Agents []agent.Agent `gorm:"foreignKey:OrganizationID;constraint:OnDelete:CASCADE" json:"agents,omitempty"`
}

// CampaignWithAgent extends Campaign with Agent relationship
type CampaignWithAgent struct {
	campaign.Campaign
	Agent *agent.Agent `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE" json:"agent,omitempty"`
}

// CampaignWithEmails extends Campaign with Email relationships
type CampaignWithEmails struct {
	campaign.Campaign
	Emails []email.Email `gorm:"foreignKey:CampaignID;constraint:OnDelete:CASCADE" json:"emails,omitempty"`
}

// CampaignWithConversations extends Campaign with Conversation relationships
type CampaignWithConversations struct {
	campaign.Campaign
	Conversations []conversation.Conversation `gorm:"foreignKey:CampaignID;constraint:OnDelete:CASCADE" json:"conversations,omitempty"`
}

// CampaignWithFullRelations extends Campaign with all relationships
type CampaignWithFullRelations struct {
	campaign.Campaign
	Agent         *agent.Agent                `gorm:"foreignKey:AgentID;constraint:OnDelete:CASCADE" json:"agent,omitempty"`
	Emails        []email.Email               `gorm:"foreignKey:CampaignID;constraint:OnDelete:CASCADE" json:"emails,omitempty"`
	Conversations []conversation.Conversation `gorm:"foreignKey:CampaignID;constraint:OnDelete:CASCADE" json:"conversations,omitempty"`
}

// TemplateWithCampaign extends Template with Campaign relationship
type TemplateWithCampaign struct {
	template.Template
	Campaign *campaign.Campaign `gorm:"foreignKey:CampaignID;constraint:OnDelete:CASCADE" json:"campaign,omitempty"`
}

// TemplateWithKnowledges extends Template with Knowledge relationships
type TemplateWithKnowledges struct {
	template.Template
	Knowledges []knowledge.Knowledge `gorm:"many2many:template_knowledges" json:"knowledges,omitempty"`
}

// TemplateWithFullRelations extends Template with all relationships
type TemplateWithFullRelations struct {
	template.Template
	Campaign   *campaign.Campaign    `gorm:"foreignKey:CampaignID;constraint:OnDelete:CASCADE" json:"campaign,omitempty"`
	Knowledges []knowledge.Knowledge `gorm:"many2many:template_knowledges" json:"knowledges,omitempty"`
}

// TaskWithContact extends Task with Contact relationship
type TaskWithContact struct {
	task.Task
	Contact *contact.Contact `gorm:"foreignKey:ContactID;constraint:OnDelete:CASCADE" json:"contact,omitempty"`
}

// TaskWithTemplate extends Task with Template relationship
type TaskWithTemplate struct {
	task.Task
	Template *template.Template `gorm:"foreignKey:TemplateID;constraint:OnDelete:CASCADE" json:"template,omitempty"`
}

// TaskWithCampaign extends Task with Campaign relationship
type TaskWithCampaign struct {
	task.Task
	Campaign *campaign.Campaign `gorm:"foreignKey:CampaignID;constraint:OnDelete:CASCADE" json:"campaign,omitempty"`
}

// TaskWithFullRelations extends Task with all relationships
type TaskWithFullRelations struct {
	task.Task
	Contact  *contact.Contact   `gorm:"foreignKey:ContactID;constraint:OnDelete:CASCADE" json:"contact,omitempty"`
	Template *template.Template `gorm:"foreignKey:TemplateID;constraint:OnDelete:CASCADE" json:"template,omitempty"`
	Campaign *campaign.Campaign `gorm:"foreignKey:CampaignID;constraint:OnDelete:CASCADE" json:"campaign,omitempty"`
}

// Helper functions for common queries

// GetAgentByID retrieves an agent by ID
func GetAgentByID(id uuid.UUID) *agent.Agent {
	var a agent.Agent
	// This would typically be implemented in a repository
	return &a
}

// GetConversationByID retrieves a conversation by ID
func GetConversationByID(id uuid.UUID) *conversation.Conversation {
	var c conversation.Conversation
	// This would typically be implemented in a repository
	return &c
}

// GetUserByID retrieves a user by ID
func GetUserByID(id uuid.UUID) *user.User {
	var u user.User
	// This would typically be implemented in a repository
	return &u
}

// GetOrganizationByID retrieves an organization by ID
func GetOrganizationByID(id uuid.UUID) *organization.Organization {
	var o organization.Organization
	// This would typically be implemented in a repository
	return &o
}
