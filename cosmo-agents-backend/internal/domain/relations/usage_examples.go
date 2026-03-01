package relations

import (
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ExampleRepository demonstrates how to use the relations package in a repository pattern
type ExampleRepository struct {
	db *gorm.DB
}

// NewExampleRepository creates a new example repository
func NewExampleRepository(db *gorm.DB) *ExampleRepository {
	return &ExampleRepository{db: db}
}

// GetAgentWithConversations retrieves an agent with all their conversations
func (r *ExampleRepository) GetAgentWithConversations(agentID uuid.UUID) (*AgentWithConversations, error) {
	var agent AgentWithConversations
	err := r.db.Preload("Conversations").First(&agent, "id = ?", agentID).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

// GetAgentWithFullRelations retrieves an agent with all related data
func (r *ExampleRepository) GetAgentWithFullRelations(agentID uuid.UUID) (*AgentWithFullRelations, error) {
	var agent AgentWithFullRelations
	err := r.db.
		Preload("User").
		Preload("Organization").
		Preload("Conversations").
		First(&agent, "id = ?", agentID).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

// GetConversationWithAgentAndEmails retrieves a conversation with its agent and emails
func (r *ExampleRepository) GetConversationWithAgentAndEmails(conversationID uuid.UUID) (*ConversationWithFullRelations, error) {
	var conversation ConversationWithFullRelations
	err := r.db.
		Preload("Agent").
		Preload("Emails").
		Preload("Campaign").
		First(&conversation, "id = ?", conversationID).Error
	if err != nil {
		return nil, err
	}
	return &conversation, nil
}

// GetUserWithAgentsAndOrganizations retrieves a user with their agents and organizations
func (r *ExampleRepository) GetUserWithAgentsAndOrganizations(userID uuid.UUID) (*UserWithFullRelations, error) {
	var user UserWithFullRelations
	err := r.db.
		Preload("Agents").
		Preload("Organizations").
		Preload("Roles").
		First(&user, "id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetOrganizationWithUsersAndAgents retrieves an organization with its users and agents
func (r *ExampleRepository) GetOrganizationWithUsersAndAgents(orgID uuid.UUID) (*OrganizationWithFullRelations, error) {
	var org OrganizationWithFullRelations
	err := r.db.
		Preload("User").
		Preload("Agents").
		Preload("Roles").
		First(&org, "id = ?", orgID).Error
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// GetCampaignWithAllData retrieves a campaign with all related data
func (r *ExampleRepository) GetCampaignWithAllData(campaignID uuid.UUID) (*CampaignWithFullRelations, error) {
	var campaign CampaignWithFullRelations
	err := r.db.
		Preload("Agent").
		Preload("Emails").
		Preload("Conversations").
		First(&campaign, "id = ?", campaignID).Error
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

// Example usage in service layer
type AgentService struct {
	repo *ExampleRepository
}

func NewAgentService(repo *ExampleRepository) *AgentService {
	return &AgentService{repo: repo}
}

// GetAgentDashboardData gets all data needed for agent dashboard
func (s *AgentService) GetAgentDashboardData(agentID uuid.UUID) (*AgentWithFullRelations, error) {
	agent, err := s.repo.GetAgentWithFullRelations(agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// You can now access all the related data:
	// - agent.User (user information)
	// - agent.Organization (organization information)
	// - agent.Conversations (all conversations for this agent)

	return agent, nil
}

// ConversationService example
type ConversationService struct {
	repo *ExampleRepository
}

func NewConversationService(repo *ExampleRepository) *ConversationService {
	return &ConversationService{repo: repo}
}

// GetConversationDetails gets complete conversation information
func (s *ConversationService) GetConversationDetails(conversationID uuid.UUID) (*ConversationWithFullRelations, error) {
	conversation, err := s.repo.GetConversationWithAgentAndEmails(conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// You can now access all the related data:
	// - conversation.Agent (agent information)
	// - conversation.Emails (all emails in this conversation)
	// - conversation.Campaign (campaign information if applicable)

	return conversation, nil
}
