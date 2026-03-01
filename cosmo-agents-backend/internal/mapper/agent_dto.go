package mapper

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// HTTP Request DTOs

// AgentCreateRequest represents HTTP request for creating an agent.
type AgentCreateRequest struct {
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

// AgentUpdateRequest represents HTTP request for updating an agent.
type AgentUpdateRequest struct {
	UserID        *uuid.UUID                 `json:"user_id,omitempty"` // For authorization
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

// AgentResponse represents HTTP response for an agent.
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
	Persona         pq.StringArray            `json:"persona,omitempty"`
	LastHistoryID   string                    `json:"last_history_id"`
	DailyLimit      *int                      `json:"daily_limit,omitempty"`
	MaxDailyLimit   *int                      `json:"max_daily_limit,omitempty"`
	ValidCred       *bool                     `json:"valid_cred,omitempty"`
	EmailsSentToday *int                      `json:"emails_sent_today,omitempty"`
	Metadata        map[string]interface{}    `json:"metadata,omitempty"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
}

// AgentsListResponse represents HTTP response for a list of agents.
type AgentsListResponse struct {
	Agents []*AgentResponse `json:"agents"`
	Total  int              `json:"total"`
}

// AgentStatsResponse represents agent statistics response.
type AgentStatsResponse struct {
	TotalAgents      int `json:"total_agents"`
	ActiveAgents     int `json:"active_agents"`
	InactiveAgents   int `json:"inactive_agents"`
	AgentsWithIssues int `json:"agents_with_issues"`
	TotalEmailsSent  int `json:"total_emails_sent"`
	EmailsSentToday  int `json:"emails_sent_today"`
}

// Query Parameters

// AgentListQuery represents query parameters for listing agents.
type AgentListQuery struct {
	UserID   *string `form:"user_id,omitempty"`
	Status   *string `form:"status,omitempty"`
	Provider *string `form:"provider,omitempty"`
	Page     *int    `form:"page,omitempty"`
	Limit    *int    `form:"limit,omitempty"`
	Search   *string `form:"search,omitempty"`
}

// Helper functions

// ParseUUID safely parses a UUID string.
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// ParseStringPtr safely returns string pointer from string.
func ParseStringPtr(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

// ParseIntPtr safely returns int pointer from int.
func ParseIntPtr(i *int) *int {
	if i == nil {
		return nil
	}
	return i
}

// ParseBoolPtr safely returns bool pointer from bool.
func ParseBoolPtr(b *bool) *bool {
	if b == nil {
		return nil
	}
	return b
}

// ValidateAgentStatus validates agent status.
func ValidateAgentStatus(status *string) (*domain.AgentStatus, error) {
	if status == nil {
		return nil, nil
	}

	switch *status {
	case string(domain.AgentStatusActive):
		active := domain.AgentStatusActive
		return &active, nil
	case string(domain.AgentStatusInactive):
		inactive := domain.AgentStatusInactive
		return &inactive, nil
	case string(domain.AgentStatusMissingScopes):
		missing := domain.AgentStatusMissingScopes
		return &missing, nil
	case string(domain.AgentStatusSyncRequired):
		sync := domain.AgentStatusSyncRequired
		return &sync, nil
	case string(domain.AgentStatusInvalidGrant):
		invalid := domain.AgentStatusInvalidGrant
		return &invalid, nil
	default:
		return nil, nil
	}
}

// ValidateAgentEmailProvider validates agent email provider.
func ValidateAgentEmailProvider(provider *string) (*domain.AgentEmailProvider, error) {
	if provider == nil {
		return nil, nil
	}

	switch *provider {
	case string(domain.AgentEmailProviderGmail):
		gmail := domain.AgentEmailProviderGmail
		return &gmail, nil
	case string(domain.AgentEmailProviderOutlook):
		outlook := domain.AgentEmailProviderOutlook
		return &outlook, nil
	default:
		return nil, nil
	}
}

// DomainToAgentResponse converts domain entity to HTTP response.
func DomainToAgentResponse(agent *domain.Agent) *AgentResponse {
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

// DomainListToAgentResponse converts domain entity list to HTTP response.
func DomainListToAgentResponse(agents []domain.Agent) *AgentsListResponse {
	responses := make([]*AgentResponse, len(agents))
	for i, agent := range agents {
		responses[i] = DomainToAgentResponse(&agent)
	}

	return &AgentsListResponse{
		Agents: responses,
		Total:  len(agents),
	}
}
