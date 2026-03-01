package agent

import (
	"encoding/json"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

type AgentMapper struct{}

func NewAgentMapper() *AgentMapper {
	return &AgentMapper{}
}

// ToAgentResponse converts domain.Agent to v1schema.AgentResponse
func (m *AgentMapper) ToAgentResponse(agent *domain.Agent) v1schema.AgentResponse {
	metadata := make(map[string]any)
	if len(agent.CMetadata) > 0 {
		if err := json.Unmarshal(agent.CMetadata, &metadata); err != nil {
			// Log warning but continue - don't break response for metadata errors
			logger.Logger.Warn().Err(err).Str("agent_id", agent.ID.String()).Msg("Failed to unmarshal agent metadata")
		}
	}

	return v1schema.AgentResponse{
		ID:              agent.ID,
		CreatedAt:       agent.CreatedAt,
		UpdatedAt:       agent.UpdatedAt,
		UserID:          agent.UserID,
		OrganizationID:  agent.OrganizationID,
		Name:            agent.Name,
		Email:           agent.Email,
		Persona:         agent.Persona,
		EmailProvider:   string(agent.EmailProvider),
		Signature:       agent.Signature,
		Picture:         agent.Picture,
		Status:          string(agent.Status),
		DailyLimit:      agent.DailyLimit,
		MaxDailyLimit:   agent.MaxDailyLimit,
		ValidCred:       agent.ValidCred,
		EmailsSentToday: agent.EmailsSentToday,
		LastHistoryID:   agent.LastHistoryID,
		CMetadata:       metadata,
		IsDeleted:       agent.IsDeleted,
	}
}

// ToAgentListItem converts domain.Agent to v1schema.AgentListItem
func (m *AgentMapper) ToAgentListItem(agent *domain.Agent) v1schema.AgentListItem {
	return v1schema.AgentListItem{
		Entity: m.ToAgentResponse(agent),
	}
}

// ToConversationEntity converts domain.Conversation to v1schema.ConversationEntity
func (m *AgentMapper) ToConversationEntity(conv *domain.Conversation) v1schema.ConversationEntity {
	// Ensure slices are non-nil so JSON encodes empty arrays instead of null
	labels := conv.Labels
	if labels == nil {
		labels = []string{}
	}

	intents := conv.Intents
	if intents == nil {
		intents = []string{}
	}

	return v1schema.ConversationEntity{
		ID:         conv.ID,
		UserID:     conv.UserID,
		Labels:     labels,
		Replied:    conv.Replied,
		CampaignID: conv.CampaignID,
		AssigneeID: conv.AssigneeID,
		Intents:    intents,
		IsDeleted:  conv.IsDeleted,
		CreatedAt:  conv.CreatedAt,
		UpdatedAt:  conv.UpdatedAt,
	}
}

// ToConversationResponse converts domain.Conversation and Email to response format
func (m *AgentMapper) ToConversationResponse(conv *domain.Conversation, latestEmail *domain.Email) v1schema.AgentGetConversationsResponse {
	emailEntity := m.ToEmailEntityFromDomain(latestEmail)
	return v1schema.AgentGetConversationsResponse{
		Entity:      m.ToConversationEntity(conv),
		LatestEmail: emailEntity,
	}
}

// ToEmailEntityFromDomain converts domain.Email to v1schema.EmailEntity
func (m *AgentMapper) ToEmailEntityFromDomain(email *domain.Email) v1schema.EmailEntity {
	if email == nil {
		return v1schema.EmailEntity{}
	}

	// Ensure arrays are non-nil to avoid null in JSON
	labels := email.Labels
	if labels == nil {
		labels = []string{}
	}

	intents := email.Intents
	if intents == nil {
		intents = []string{}
	}

	attachments := email.Attachments

	return v1schema.EmailEntity{
		ID:          &email.ID,
		Subject:     &email.Subject,
		Content:     &email.Content,
		FromEmail:   &email.FromEmail,
		ToEmail:     &email.ToEmail,
		Attachments: attachments,
		Intents:     intents,
		CreatedAt:   &email.CreatedAt,
		UpdatedAt:   &email.UpdatedAt,
	}
}
