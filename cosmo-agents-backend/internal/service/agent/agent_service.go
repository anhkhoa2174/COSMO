package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// AgentService handles agent business logic.
type AgentService struct {
	agentRepository *agentRepo.AgentRepository
	db              *gorm.DB
}

// NewAgentService creates a new agent service.
func NewAgentService(db *gorm.DB) *AgentService {
	return &AgentService{
		agentRepository: agentRepo.NewAgentRepository(db),
		db:              db,
	}
}

// CreateAgentRequest represents the request to create a new agent.
type CreateAgentRequest struct {
	UserID         uuid.UUID                 `json:"user_id" validate:"required"`
	OrganizationID *uuid.UUID                `json:"organization_id,omitempty"`
	Name           string                    `json:"name" validate:"required,min=1,max=255"`
	Email          string                    `json:"email" validate:"required,email"`
	EmailProvider  domain.AgentEmailProvider `json:"email_provider" validate:"required,oneof=gmail outlook"`
	Persona        []string                  `json:"persona,omitempty"`
	Signature      string                    `json:"signature,omitempty"`
	Picture        string                    `json:"picture,omitempty"`
	Credentials    map[string]interface{}    `json:"credentials,omitempty"`
	DailyLimit     *int                      `json:"daily_limit,omitempty"`
	MaxDailyLimit  *int                      `json:"max_daily_limit,omitempty"`
	Metadata       map[string]interface{}    `json:"metadata,omitempty"`
}

// UpdateAgentRequest represents the request to update an agent.
type UpdateAgentRequest struct {
	Name          *string                    `json:"name,omitempty"`
	Email         *string                    `json:"email,omitempty"`
	Status        *domain.AgentStatus        `json:"status,omitempty"`
	EmailProvider *domain.AgentEmailProvider `json:"email_provider,omitempty"`
	Persona       []string                   `json:"persona,omitempty"`
	Signature     *string                    `json:"signature,omitempty"`
	Picture       *string                    `json:"picture,omitempty"`
	Credentials   map[string]interface{}     `json:"credentials,omitempty"`
	DailyLimit    *int                       `json:"daily_limit,omitempty"`
	MaxDailyLimit *int                       `json:"max_daily_limit,omitempty"`
	ValidCred     *bool                      `json:"valid_cred,omitempty"`
	LastHistoryID *string                    `json:"last_history_id,omitempty"`
	Metadata      map[string]interface{}     `json:"metadata,omitempty"`
}

// CreateAgent creates a new agent with business logic validation.
func (s *AgentService) CreateAgent(ctx context.Context, req CreateAgentRequest) (*domain.Agent, error) {
	logger.FromContext(ctx).Info().
		Str("user_id", req.UserID.String()).
		Str("name", req.Name).
		Str("email", req.Email).
		Str("email_provider", string(req.EmailProvider)).
		Msg("Creating new agent")

	// Validate business rules
	if err := s.validateCreateRequest(ctx, req); err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to validate create agent request")
		return nil, err
	}

	// Check if agent with same email already exists for this user
	existingAgent, err := s.agentRepository.FindByUserAndEmail(ctx, req.UserID, req.Email)
	if err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to check existing agent")
		return nil, fmt.Errorf("failed to check existing agent: %w", err)
	}
	if existingAgent != nil {
		logger.FromContext(ctx).Warn().
			Str("user_id", req.UserID.String()).
			Str("email", req.Email).
			Msg("Agent with email already exists for user")
		return nil, errors.New("agent with this email already exists for this user")
	}

	// Create agent entity
	agent := &domain.Agent{
		UserID:         req.UserID,
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Email:          req.Email,
		EmailProvider:  req.EmailProvider,
		Signature:      req.Signature,
		Picture:        req.Picture,
		DailyLimit:     req.DailyLimit,
		MaxDailyLimit:  req.MaxDailyLimit,
		Status:         domain.AgentStatusActive,
		ValidCred:      boolPtr(true),
	}

	if len(req.Persona) > 0 {
		agent.Persona = req.Persona
	}

	if req.Credentials != nil {
		if err := agent.SetCredentials(req.Credentials); err != nil {
			logger.FromContext(ctx).Err(err).Msg("Failed to set agent credentials")
			return nil, fmt.Errorf("failed to set credentials: %w", err)
		}
	}

	if req.Metadata != nil {
		if err := agent.SetMetadata(req.Metadata); err != nil {
			logger.FromContext(ctx).Err(err).Msg("Failed to set agent metadata")
			return nil, fmt.Errorf("failed to set metadata: %w", err)
		}
	}

	createdAgent, err := s.agentRepository.Create(ctx, agent)
	if err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to create agent in database")
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("agent_id", createdAgent.ID.String()).
		Str("user_id", createdAgent.UserID.String()).
		Str("email", createdAgent.Email).
		Msg("Successfully created agent")
	return createdAgent, nil
}

// GetAgent retrieves an agent by ID.
func (s *AgentService) GetAgent(ctx context.Context, agentID uuid.UUID) (*domain.Agent, error) {
	logger.FromContext(ctx).Info().Str("agent_id", agentID.String()).Msg("Retrieving agent")

	agent, err := s.agentRepository.GetByID(ctx, agentID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Str("agent_id", agentID.String()).Msg("Failed to retrieve agent")
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if agent == nil {
		logger.FromContext(ctx).Info().Str("agent_id", agentID.String()).Msg("Agent not found")
		return nil, errors.New("agent not found")
	}

	logger.FromContext(ctx).Info().
		Str("agent_id", agent.ID.String()).
		Str("email", agent.Email).
		Msg("Successfully retrieved agent")

	return agent, nil
}

// GetAgentsByUserID retrieves all agents for a user.
func (s *AgentService) GetAgentsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Agent, error) {
	logger.FromContext(ctx).Info().Str("user_id", userID.String()).Msg("Retrieving agents by user ID")

	agents, err := s.agentRepository.GetByUserID(ctx, userID.String())
	if err != nil {
		logger.FromContext(ctx).Err(err).Str("user_id", userID.String()).Msg("Failed to retrieve agents by user ID")
		return nil, fmt.Errorf("failed to get agents: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("user_id", userID.String()).
		Int("count", len(agents)).
		Msg("Successfully retrieved agents by user ID")
	return agents, nil
}

// UpdateAgent updates an existing agent.
func (s *AgentService) UpdateAgent(ctx context.Context, agentID uuid.UUID, req UpdateAgentRequest) (*domain.Agent, error) {
	logger.FromContext(ctx).Info().Str("agent_id", agentID.String()).Msg("Updating agent")

	// Get existing agent
	agent, err := s.agentRepository.GetByID(ctx, agentID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Str("agent_id", agentID.String()).Msg("Failed to get existing agent")
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	if agent == nil {
		logger.FromContext(ctx).Info().Str("agent_id", agentID.String()).Msg("Agent not found")
		return nil, errors.New("agent not found")
	}

	// Update fields if provided
	if req.Name != nil {
		agent.Name = *req.Name
	}
	if req.Email != nil {
		// Check if new email conflicts with existing agents for this user
		existingAgent, err := s.agentRepository.FindByUserAndEmail(ctx, agent.UserID, *req.Email)
		if err != nil {
			logger.FromContext(ctx).Err(err).Msg("Failed to check email uniqueness")
			return nil, fmt.Errorf("failed to check email uniqueness: %w", err)
		}
		if existingAgent != nil && existingAgent.ID != agentID {
			logger.FromContext(ctx).Warn().
				Str("email", *req.Email).
				Str("user_id", agent.UserID.String()).
				Msg("Email already exists for user")
			return nil, errors.New("email already exists for this user")
		}
		agent.Email = *req.Email
	}
	if req.Status != nil {
		agent.Status = *req.Status
	}
	if req.EmailProvider != nil {
		agent.EmailProvider = *req.EmailProvider
	}
	if req.Persona != nil {
		agent.Persona = req.Persona
	}
	if req.Signature != nil {
		agent.Signature = *req.Signature
	}
	if req.Picture != nil {
		agent.Picture = *req.Picture
	}
	if req.DailyLimit != nil {
		agent.DailyLimit = req.DailyLimit
	}
	if req.MaxDailyLimit != nil {
		agent.MaxDailyLimit = req.MaxDailyLimit
	}
	if req.ValidCred != nil {
		agent.ValidCred = req.ValidCred
	}
	if req.LastHistoryID != nil {
		agent.LastHistoryID = *req.LastHistoryID
	}

	// Update credentials if provided
	if req.Credentials != nil {
		if err := agent.SetCredentials(req.Credentials); err != nil {
			logger.FromContext(ctx).Err(err).Msg("Failed to update agent credentials")
			return nil, fmt.Errorf("failed to update credentials: %w", err)
		}
	}

	// Update metadata if provided
	if req.Metadata != nil {
		if err := agent.SetMetadata(req.Metadata); err != nil {
			logger.FromContext(ctx).Err(err).Msg("Failed to update agent metadata")
			return nil, fmt.Errorf("failed to update metadata: %w", err)
		}
	}

	// Save changes
	err = s.agentRepository.Update(ctx, agentID, agent)
	if err != nil {
		logger.FromContext(ctx).Err(err).Str("agent_id", agentID.String()).Msg("Failed to update agent")
		return nil, fmt.Errorf("failed to update agent: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("agent_id", agent.ID.String()).
		Msg("Successfully updated agent")

	return agent, nil
}

// DeleteAgent soft deletes an agent.
func (s *AgentService) DeleteAgent(ctx context.Context, agentID uuid.UUID) error {
	logger.FromContext(ctx).Info().Str("agent_id", agentID.String()).Msg("Deleting agent")

	// Check if agent exists
	agent, err := s.agentRepository.GetByID(ctx, agentID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Str("agent_id", agentID.String()).Msg("Failed to get agent for deletion")
		return fmt.Errorf("failed to get agent: %w", err)
	}
	if agent == nil {
		logger.FromContext(ctx).Info().Str("agent_id", agentID.String()).Msg("Agent not found for deletion")
		return errors.New("agent not found")
	}

	// Soft delete
	err = s.agentRepository.Delete(ctx, agentID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Str("agent_id", agentID.String()).Msg("Failed to delete agent")
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("agent_id", agentID.String()).
		Msg("Successfully deleted agent")
	return nil
}

// GetActiveAgents retrieves all active agents.
func (s *AgentService) GetActiveAgents(ctx context.Context) ([]domain.Agent, error) {
	logger.FromContext(ctx).Info().Msg("Retrieving active agents")

	agents, err := s.agentRepository.GetActiveAgents(ctx)
	if err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to retrieve active agents")
		return nil, fmt.Errorf("failed to get active agents: %w", err)
	}

	logger.FromContext(ctx).Info().
		Int("count", len(agents)).
		Msg("Successfully retrieved active agents")
	return agents, nil
}

// ValidateCreateRequest validates the create agent request.
func (s *AgentService) validateCreateRequest(ctx context.Context, req CreateAgentRequest) error {
	// Validate daily limits
	if req.DailyLimit != nil && req.MaxDailyLimit != nil {
		if *req.DailyLimit > *req.MaxDailyLimit {
			return errors.New("daily limit cannot exceed max daily limit")
		}
	}

	// Validate email provider
	if req.EmailProvider != domain.AgentEmailProviderGmail && req.EmailProvider != domain.AgentEmailProviderOutlook {
		return errors.New("invalid email provider")
	}

	return nil
}

// Helper function
func boolPtr(v bool) *bool {
	return &v
}
