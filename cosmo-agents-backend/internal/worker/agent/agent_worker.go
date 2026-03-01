package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	agentRepo "github.com/rockship/cosmo-agents-go/internal/repository/agent"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

const (
	TypeAgentCreate       = "agent:create"
	TypeAgentUpdate       = "agent:update"
	TypeAgentDelete       = "agent:delete"
	TypeAgentSyncEmails   = "agent:sync_emails"
	TypeAgentRefreshToken = "agent:refresh_token"
)

// AgentCreatePayload represents the payload for creating an agent.
type AgentCreatePayload struct {
	UserID         uuid.UUID              `json:"user_id"`
	OrganizationID *uuid.UUID             `json:"organization_id,omitempty"`
	Name           string                 `json:"name"`
	Email          string                 `json:"email"`
	EmailProvider  string                 `json:"email_provider"`
	Credentials    map[string]interface{} `json:"credentials,omitempty"`
}

// AgentUpdatePayload represents the payload for updating an agent.
type AgentUpdatePayload struct {
	AgentID        uuid.UUID              `json:"agent_id"`
	UserID         *uuid.UUID             `json:"user_id,omitempty"`
	OrganizationID *uuid.UUID             `json:"organization_id,omitempty"`
	Name           *string                `json:"name,omitempty"`
	Email          *string                `json:"email,omitempty"`
	Status         *string                `json:"status,omitempty"`
	EmailProvider  *string                `json:"email_provider,omitempty"`
	Credentials    map[string]interface{} `json:"credentials,omitempty"`
	DailyLimit     *int                   `json:"daily_limit,omitempty"`
	MaxDailyLimit  *int                   `json:"max_daily_limit,omitempty"`
}

// AgentDeletePayload represents the payload for deleting an agent.
type AgentDeletePayload struct {
	AgentID uuid.UUID `json:"agent_id"`
	UserID  uuid.UUID `json:"user_id"`
}

// AgentSyncEmailsPayload represents the payload for syncing agent emails.
type AgentSyncEmailsPayload struct {
	AgentID   uuid.UUID  `json:"agent_id"`
	UserID    uuid.UUID  `json:"user_id"`
	SyncAll   bool       `json:"sync_all"`
	SyncSince *time.Time `json:"sync_since,omitempty"`
}

// AgentRefreshTokenPayload represents the payload for refreshing agent tokens.
type AgentRefreshTokenPayload struct {
	AgentID uuid.UUID `json:"agent_id"`
	UserID  uuid.UUID `json:"user_id"`
}

// AgentWorker handles agent-specific background tasks.
type AgentWorker struct {
	db              *gorm.DB
	agentRepository agentRepository
}

type agentRepository interface {
	Create(ctx context.Context, agent *domain.Agent) (*domain.Agent, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Agent, error)
	Update(ctx context.Context, id uuid.UUID, agent *domain.Agent) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateLastHistoryID(ctx context.Context, agentID uuid.UUID, historyID string) error
}

// NewAgentWorker creates a new agent worker.
func NewAgentWorker(db *gorm.DB) *AgentWorker {
	return &AgentWorker{
		db:              db,
		agentRepository: agentRepo.NewAgentRepository(db),
	}
}

// HandleAgentCreate processes agent creation.
func (w *AgentWorker) HandleAgentCreate(ctx context.Context, task *asynq.Task) error {

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	var payload AgentCreatePayload
	if err := worker.ParsePayload(task, &payload); err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to parse agent create payload")
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("user_id", payload.UserID.String()).
		Str("name", payload.Name).
		Str("email", payload.Email).
		Msg("Processing agent creation")
	if payload.Name == "" {
		err := fmt.Errorf("agent name is required")
		logger.FromContext(ctx).Err(err).Msg("Agent creation validation failed")
		return err
	}

	if payload.Email == "" {
		err := fmt.Errorf("agent email is required")
		logger.FromContext(ctx).Err(err).Msg("Agent creation validation failed")
		return err
	}

	// Create agent
	agent := &domain.Agent{
		Base:           domain.Base{ID: uuid.New()},
		UserID:         payload.UserID,
		OrganizationID: payload.OrganizationID,
		Name:           payload.Name,
		Email:          payload.Email,
		EmailProvider:  domain.AgentEmailProvider(payload.EmailProvider),
		Status:         domain.AgentStatusActive,
		ValidCred:      boolPtr(true),
	}

	if len(payload.Credentials) > 0 {
		// Create a copy of credentials for sanitization to avoid mutating the payload
		credentialsCopy := make(map[string]interface{})
		for k, v := range payload.Credentials {
			credentialsCopy[k] = v
		}
		sanitizeAgentCredentialsForLogging(credentialsCopy)

		if err := agent.SetCredentials(payload.Credentials); err != nil {
			logger.FromContext(ctx).Err(err).Msg("Failed to set agent credentials")
			return fmt.Errorf("failed to set credentials: %w", err)
		}
	}

	createdAgent, err := w.agentRepository.Create(ctx, agent)
	if err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to create agent")
		return fmt.Errorf("failed to create agent: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("agent_id", createdAgent.ID.String()).
		Msg("Successfully created agent")
	return nil
}

// HandleAgentUpdate processes agent updates.
func (w *AgentWorker) HandleAgentUpdate(ctx context.Context, task *asynq.Task) error {

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	var payload AgentUpdatePayload
	if err := worker.ParsePayload(task, &payload); err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to parse agent update payload")
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("agent_id", payload.AgentID.String()).
		Msg("Processing agent update")

	// Get existing agent
	agent, err := w.agentRepository.GetByID(ctx, payload.AgentID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Str("agent_id", payload.AgentID.String()).Msg("Failed to get agent")
		return fmt.Errorf("failed to get agent: %w", err)
	}
	if agent == nil {
		err := fmt.Errorf("agent not found: %s", payload.AgentID)
		logger.FromContext(ctx).Err(err).Str("agent_id", payload.AgentID.String()).Msg("Agent not found")
		return err
	}

	// Update fields if provided
	if payload.UserID != nil {
		agent.UserID = *payload.UserID
	}
	if payload.OrganizationID != nil {
		agent.OrganizationID = payload.OrganizationID
	}
	if payload.Name != nil {
		agent.Name = *payload.Name
	}
	if payload.Email != nil {
		agent.Email = *payload.Email
	}
	if payload.Status != nil {
		agent.Status = domain.AgentStatus(*payload.Status)
	}
	if payload.EmailProvider != nil {
		agent.EmailProvider = domain.AgentEmailProvider(*payload.EmailProvider)
	}
	if payload.DailyLimit != nil {
		agent.DailyLimit = payload.DailyLimit
	}
	if payload.MaxDailyLimit != nil {
		agent.MaxDailyLimit = payload.MaxDailyLimit
	}

	// Update credentials if provided
	if len(payload.Credentials) > 0 {
		// Create a copy of credentials for sanitization to avoid mutating the payload
		credentialsCopy := make(map[string]interface{})
		for k, v := range payload.Credentials {
			credentialsCopy[k] = v
		}
		sanitizeAgentCredentialsForLogging(credentialsCopy)

		if err := agent.SetCredentials(payload.Credentials); err != nil {
			logger.FromContext(ctx).Err(err).Msg("Failed to update agent credentials")
			return fmt.Errorf("failed to update credentials: %w", err)
		}
	}

	if err := w.agentRepository.Update(ctx, agent.ID, agent); err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to update agent")
		return fmt.Errorf("failed to update agent: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("agent_id", agent.ID.String()).
		Msg("Successfully updated agent")
	return nil
}

// HandleAgentDelete processes agent deletion.
func (w *AgentWorker) HandleAgentDelete(ctx context.Context, task *asynq.Task) error {

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	var payload AgentDeletePayload
	if err := worker.ParsePayload(task, &payload); err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to parse agent delete payload")
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("agent_id", payload.AgentID.String()).
		Str("user_id", payload.UserID.String()).
		Msg("Processing agent deletion")

	// Get agent to verify ownership
	agent, err := w.agentRepository.GetByID(ctx, payload.AgentID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Str("agent_id", payload.AgentID.String()).Msg("Failed to get agent")
		return fmt.Errorf("failed to get agent: %w", err)
	}
	if agent == nil {
		err := fmt.Errorf("agent not found: %s", payload.AgentID)
		logger.FromContext(ctx).Err(err).Str("agent_id", payload.AgentID.String()).Msg("Agent not found")
		return err
	}

	// Verify user ownership
	if agent.UserID != payload.UserID {
		err := fmt.Errorf("user does not own this agent")
		logger.FromContext(ctx).Err(err).Msg("Agent deletion authorization failed")
		return err
	}

	if err := w.agentRepository.Delete(ctx, payload.AgentID); err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to delete agent")
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("agent_id", payload.AgentID.String()).
		Msg("Successfully deleted agent")
	return nil
}

// HandleAgentSyncEmails processes agent email synchronization.
func (w *AgentWorker) HandleAgentSyncEmails(ctx context.Context, task *asynq.Task) error {

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	var payload AgentSyncEmailsPayload
	if err := worker.ParsePayload(task, &payload); err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to parse agent sync emails payload")
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("agent_id", payload.AgentID.String()).
		Bool("sync_all", payload.SyncAll).
		Msg("Processing agent email sync")

	// Get agent
	agent, err := w.agentRepository.GetByID(ctx, payload.AgentID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Str("agent_id", payload.AgentID.String()).Msg("Failed to get agent")
		return fmt.Errorf("failed to get agent: %w", err)
	}
	if agent == nil {
		err := fmt.Errorf("agent not found: %s", payload.AgentID)
		logger.FromContext(ctx).Err(err).Str("agent_id", payload.AgentID.String()).Msg("Agent not found")
		return err
	}

	// Verify user ownership
	if agent.UserID != payload.UserID {
		err := fmt.Errorf("user does not own this agent")
		logger.FromContext(ctx).Err(err).Msg("Agent sync authorization failed")
		return err
	}

	if err := w.agentRepository.UpdateLastHistoryID(ctx, agent.ID, agent.LastHistoryID); err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to update last history ID")
		return fmt.Errorf("failed to update last history ID: %w", err)
	}

	// TODO: Implement actual email synchronization logic
	// This would involve:
	// 1. Using agent's Gmail credentials to fetch emails
	// 2. Creating conversation and email records
	// 3. Handling pagination and rate limiting

	logger.FromContext(ctx).Info().
		Str("agent_id", payload.AgentID.String()).
		Msg("Email sync completed (placeholder)")

	return nil
}

// HandleAgentRefreshToken processes agent token refresh.
func (w *AgentWorker) HandleAgentRefreshToken(ctx context.Context, task *asynq.Task) error {

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	var payload AgentRefreshTokenPayload
	if err := worker.ParsePayload(task, &payload); err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to parse agent refresh token payload")
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("agent_id", payload.AgentID.String()).
		Msg("Processing agent token refresh")

	// Get agent
	agent, err := w.agentRepository.GetByID(ctx, payload.AgentID)
	if err != nil {
		logger.FromContext(ctx).Err(err).Str("agent_id", payload.AgentID.String()).Msg("Failed to get agent")
		return fmt.Errorf("failed to get agent: %w", err)
	}
	if agent == nil {
		err := fmt.Errorf("agent not found: %s", payload.AgentID)
		logger.FromContext(ctx).Err(err).Str("agent_id", payload.AgentID.String()).Msg("Agent not found")
		return err
	}

	// Verify user ownership
	if agent.UserID != payload.UserID {
		err := fmt.Errorf("user does not own this agent")
		logger.FromContext(ctx).Err(err).Msg("Agent token refresh authorization failed")
		return err
	}

	if err := w.agentRepository.Update(ctx, agent.ID, agent); err != nil {
		logger.FromContext(ctx).Err(err).Msg("Failed to persist refreshed token")
		return fmt.Errorf("failed to persist refreshed token: %w", err)
	}

	// TODO: Implement actual token refresh logic
	// This would involve:
	// 1. Using OAuth2 service to refresh the agent's credentials
	// 2. Updating the agent's credentials in the database
	// 3. Handling invalid_grant errors

	logger.FromContext(ctx).Info().
		Str("agent_id", payload.AgentID.String()).
		Msg("Token refresh completed (placeholder)")
	return nil
}

// RegisterAgentTasks registers all agent-related tasks.
func (w *AgentWorker) RegisterAgentTasks(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeAgentCreate, w.HandleAgentCreate)
	mux.HandleFunc(TypeAgentUpdate, w.HandleAgentUpdate)
	mux.HandleFunc(TypeAgentDelete, w.HandleAgentDelete)
	mux.HandleFunc(TypeAgentSyncEmails, w.HandleAgentSyncEmails)
	mux.HandleFunc(TypeAgentRefreshToken, w.HandleAgentRefreshToken)
}

// Helper functions

func boolPtr(v bool) *bool {
	return &v
}

// Enqueue functions

func EnqueueAgentCreate(client *asynq.Client, payload AgentCreatePayload) error {
	task, err := worker.NewTask(TypeAgentCreate, payload)
	if err != nil {
		return err
	}
	_, err = client.Enqueue(task)
	return err
}

func EnqueueAgentUpdate(client *asynq.Client, payload AgentUpdatePayload) error {
	task, err := worker.NewTask(TypeAgentUpdate, payload)
	if err != nil {
		return err
	}
	_, err = client.Enqueue(task)
	return err
}

func EnqueueAgentDelete(client *asynq.Client, payload AgentDeletePayload) error {
	task, err := worker.NewTask(TypeAgentDelete, payload)
	if err != nil {
		return err
	}
	_, err = client.Enqueue(task)
	return err
}

func EnqueueAgentSyncEmails(client *asynq.Client, payload AgentSyncEmailsPayload) error {
	task, err := worker.NewTask(TypeAgentSyncEmails, payload)
	if err != nil {
		return err
	}
	_, err = client.Enqueue(task)
	return err
}

// sanitizeAgentCredentialsForLogging removes sensitive information from credential maps
func sanitizeAgentCredentialsForLogging(credMap map[string]interface{}) {
	// Remove sensitive fields that might be logged accidentally
	sensitiveFields := []string{"access_token", "refresh_token", "client_secret", "password", "client_id"}
	for _, field := range sensitiveFields {
		delete(credMap, field)
	}
}

func EnqueueAgentRefreshToken(client *asynq.Client, payload AgentRefreshTokenPayload) error {
	task, err := worker.NewTask(TypeAgentRefreshToken, payload)
	if err != nil {
		return err
	}
	_, err = client.Enqueue(task)
	return err
}
