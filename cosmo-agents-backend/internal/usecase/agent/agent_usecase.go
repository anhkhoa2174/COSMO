package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/service/agent"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// AgentUseCase handles agent-related use cases and orchestrates the service layer.
type AgentUseCase struct {
	agentService *agent.AgentService
}

// NewAgentUseCase creates a new agent use case.
func NewAgentUseCase(agentService *agent.AgentService) *AgentUseCase {
	return &AgentUseCase{
		agentService: agentService,
	}
}

// CreateAgentUseCase represents the use case for creating an agent.
type CreateAgentUseCase struct {
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

// UpdateAgentUseCase represents the use case for updating an agent.
type UpdateAgentUseCase struct {
	AgentID       uuid.UUID                  `json:"agent_id" validate:"required"`
	UserID        *uuid.UUID                 `json:"user_id,omitempty"`
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

// GetAgentUseCase represents the use case for retrieving an agent.
type GetAgentUseCase struct {
	AgentID uuid.UUID  `json:"agent_id" validate:"required"`
	UserID  *uuid.UUID `json:"user_id,omitempty"`
}

// DeleteAgentUseCase represents the use case for deleting an agent.
type DeleteAgentUseCase struct {
	AgentID uuid.UUID  `json:"agent_id" validate:"required"`
	UserID  *uuid.UUID `json:"user_id,omitempty"`
}

// GetAgentsByUserUseCase represents the use case for retrieving agents by user.
type GetAgentsByUserUseCase struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`
}

// AgentResponse represents the response for agent operations.
type AgentResponse struct {
	ID              uuid.UUID                 `json:"id"`
	UserID          uuid.UUID                 `json:"user_id"`
	OrganizationID  *uuid.UUID                `json:"organization_id,omitempty"`
	Name            string                    `json:"name"`
	Email           string                    `json:"email"`
	Status          domain.AgentStatus        `json:"status"`
	EmailProvider   domain.AgentEmailProvider `json:"email_provider"`
	Signature       string                    `json:"signature"`
	Picture         string                    `json:"picture"`
	Persona         []string                  `json:"persona,omitempty"`
	LastHistoryID   string                    `json:"last_history_id"`
	DailyLimit      *int                      `json:"daily_limit,omitempty"`
	MaxDailyLimit   *int                      `json:"max_daily_limit,omitempty"`
	ValidCred       *bool                     `json:"valid_cred,omitempty"`
	EmailsSentToday *int                      `json:"emails_sent_today,omitempty"`
	Metadata        map[string]interface{}    `json:"metadata,omitempty"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
}

// AgentsListResponse represents a list of agents response.
type AgentsListResponse struct {
	Agents []AgentResponse `json:"agents"`
	Total  int             `json:"total"`
}

// CreateAgent executes the create agent use case.
func (uc *AgentUseCase) CreateAgent(ctx context.Context, useCase CreateAgentUseCase) (*AgentResponse, error) {
	logger.FromContext(ctx).Info().
		Str("user_id", useCase.UserID.String()).
		Str("name", useCase.Name).
		Str("email", useCase.Email).
		Msg("Executing create agent use case")

	// Convert to service request
	req := agent.CreateAgentRequest{
		UserID:         useCase.UserID,
		OrganizationID: useCase.OrganizationID,
		Name:           useCase.Name,
		Email:          useCase.Email,
		EmailProvider:  useCase.EmailProvider,
		Persona:        useCase.Persona,
		Signature:      useCase.Signature,
		Picture:        useCase.Picture,
		Credentials:    useCase.Credentials,
		DailyLimit:     useCase.DailyLimit,
		MaxDailyLimit:  useCase.MaxDailyLimit,
		Metadata:       useCase.Metadata,
	}

	// Call service
	createdAgent, err := uc.agentService.CreateAgent(ctx, req)
	if err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to create agent in use case")
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	response := uc.domainToResponse(createdAgent)
	logger.FromContext(ctx).Info().
		Str("agent_id", response.ID.String()).
		Msg("Successfully executed create agent use case")
	return response, nil
}

// GetAgent executes the get agent use case.
func (uc *AgentUseCase) GetAgent(ctx context.Context, useCase GetAgentUseCase) (*AgentResponse, error) {
	logger.FromContext(ctx).Info().
		Str("agent_id", useCase.AgentID.String()).
		Msg("Executing get agent use case")

	agent, err := uc.agentService.GetAgent(ctx, useCase.AgentID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to get agent in use case")
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if useCase.UserID != nil && agent.UserID != *useCase.UserID {
		logger.FromContext(ctx).Warn().
			Str("agent_id", useCase.AgentID.String()).
			Str("requested_user_id", useCase.UserID.String()).
			Str("agent_user_id", agent.UserID.String()).
			Msg("Authorization failed: user does not own this agent")
		return nil, fmt.Errorf("unauthorized: user does not own this agent")
	}

	response := uc.domainToResponse(agent)
	logger.FromContext(ctx).Info().
		Str("agent_id", response.ID.String()).
		Msg("Successfully executed get agent use case")

	return response, nil
}

// GetAgentsByUser executes the get agents by user use case.
func (uc *AgentUseCase) GetAgentsByUser(ctx context.Context, useCase GetAgentsByUserUseCase) (*AgentsListResponse, error) {
	logger.FromContext(ctx).Info().
		Str("user_id", useCase.UserID.String()).
		Msg("Executing get agents by user use case")

	agents, err := uc.agentService.GetAgentsByUserID(ctx, useCase.UserID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to get agents by user in use case")
		return nil, fmt.Errorf("failed to get agents by user: %w", err)
	}

	responses := make([]AgentResponse, len(agents))
	for i, agent := range agents {
		responses[i] = *uc.domainToResponse(&agent)
	}

	response := &AgentsListResponse{
		Agents: responses,
		Total:  len(agents),
	}

	logger.FromContext(ctx).Info().
		Str("user_id", useCase.UserID.String()).
		Int("count", response.Total).
		Msg("Successfully executed get agents by user use case")

	return response, nil
}

// UpdateAgent executes the update agent use case.
func (uc *AgentUseCase) UpdateAgent(ctx context.Context, useCase UpdateAgentUseCase) (*AgentResponse, error) {
	logger.FromContext(ctx).Info().
		Str("agent_id", useCase.AgentID.String()).
		Msg("Executing update agent use case")

	existingAgent, err := uc.agentService.GetAgent(ctx, useCase.AgentID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to get existing agent for update")
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Authorization check if user ID is provided
	if useCase.UserID != nil && existingAgent.UserID != *useCase.UserID {
		logger.FromContext(ctx).Warn().
			Str("agent_id", useCase.AgentID.String()).
			Str("requested_user_id", useCase.UserID.String()).
			Str("agent_user_id", existingAgent.UserID.String()).
			Msg("Authorization failed: user does not own this agent")
		return nil, fmt.Errorf("unauthorized: user does not own this agent")
	}

	req := agent.UpdateAgentRequest{
		Name:          useCase.Name,
		Email:         useCase.Email,
		Status:        useCase.Status,
		EmailProvider: useCase.EmailProvider,
		Persona:       useCase.Persona,
		Signature:     useCase.Signature,
		Picture:       useCase.Picture,
		Credentials:   useCase.Credentials,
		DailyLimit:    useCase.DailyLimit,
		MaxDailyLimit: useCase.MaxDailyLimit,
		ValidCred:     useCase.ValidCred,
		LastHistoryID: useCase.LastHistoryID,
		Metadata:      useCase.Metadata,
	}

	updatedAgent, err := uc.agentService.UpdateAgent(ctx, useCase.AgentID, req)
	if err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to update agent in use case")
		return nil, fmt.Errorf("failed to update agent: %w", err)
	}

	response := uc.domainToResponse(updatedAgent)
	logger.FromContext(ctx).Info().
		Str("agent_id", response.ID.String()).
		Msg("Successfully executed update agent use case")

	return response, nil
}

// DeleteAgent executes the delete agent use case.
func (uc *AgentUseCase) DeleteAgent(ctx context.Context, useCase DeleteAgentUseCase) error {
	logger.FromContext(ctx).Info().
		Str("agent_id", useCase.AgentID.String()).
		Msg("Executing delete agent use case")

	existingAgent, err := uc.agentService.GetAgent(ctx, useCase.AgentID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to get existing agent for deletion")
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Authorization check if user ID is provided
	if useCase.UserID != nil && existingAgent.UserID != *useCase.UserID {
		logger.FromContext(ctx).Warn().
			Str("agent_id", useCase.AgentID.String()).
			Str("requested_user_id", useCase.UserID.String()).
			Str("agent_user_id", existingAgent.UserID.String()).
			Msg("Authorization failed: user does not own this agent")
		return fmt.Errorf("unauthorized: user does not own this agent")
	}

	// Call service
	err = uc.agentService.DeleteAgent(ctx, useCase.AgentID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to delete agent in use case")
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("agent_id", useCase.AgentID.String()).
		Msg("Successfully executed delete agent use case")

	return nil
}

// GetActiveAgents executes the get active agents use case (admin operation).
func (uc *AgentUseCase) GetActiveAgents(ctx context.Context) (*AgentsListResponse, error) {
	logger.FromContext(ctx).Info().Msg("Executing get active agents use case")

	agents, err := uc.agentService.GetActiveAgents(ctx)
	if err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to get active agents in use case")
		return nil, fmt.Errorf("failed to get active agents: %w", err)
	}

	responses := make([]AgentResponse, len(agents))
	for i, agent := range agents {
		responses[i] = *uc.domainToResponse(&agent)
	}

	response := &AgentsListResponse{
		Agents: responses,
		Total:  len(agents),
	}

	logger.FromContext(ctx).Info().
		Int("count", response.Total).
		Msg("Successfully executed get active agents use case")

	return response, nil
}

// domainToResponse converts domain entity to response DTO.
func (uc *AgentUseCase) domainToResponse(agent *domain.Agent) *AgentResponse {
	metadata, _ := agent.GetMetadata()

	return &AgentResponse{
		ID:              agent.ID,
		UserID:          agent.UserID,
		OrganizationID:  agent.OrganizationID,
		Name:            agent.Name,
		Email:           agent.Email,
		Status:          agent.Status,
		EmailProvider:   agent.EmailProvider,
		Signature:       agent.Signature,
		Picture:         agent.Picture,
		Persona:         agent.Persona,
		LastHistoryID:   agent.LastHistoryID,
		DailyLimit:      agent.DailyLimit,
		MaxDailyLimit:   agent.MaxDailyLimit,
		ValidCred:       agent.ValidCred,
		EmailsSentToday: agent.EmailsSentToday,
		Metadata:        metadata,
		CreatedAt:       agent.CreatedAt,
		UpdatedAt:       agent.UpdatedAt,
	}
}
