package agent

import (
	"context"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	user "github.com/rockship/cosmo-agents-go/internal/repository/user"

	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"gorm.io/gorm"
)

// AgentStatsCalculator handles statistics calculations for agents
type AgentStatsCalculator struct {
	convRepo  *conversationRepo.ConversationRepository
	emailRepo *emailRepo.Repository
	userRepo  *user.UserRepository
}

// NewAgentStatsCalculator creates a new statistics calculator
func NewAgentStatsCalculator(db *gorm.DB) *AgentStatsCalculator {
	return &AgentStatsCalculator{
		convRepo:  conversationRepo.NewConversationRepository(db),
		emailRepo: emailRepo.NewEmailRepository(db),
		userRepo:  user.NewUserRepository(db),
	}
}

// CalculateEmailStatistics computes email statistics for an agent
func (calc *AgentStatsCalculator) CalculateEmailStatistics(ctx context.Context, agentID uuid.UUID) (v1schema.AgentEmailStatistics, error) {
	var stats v1schema.AgentEmailStatistics

	totalSent, err := calc.convRepo.Count(ctx, baseRepo.Filter{
		"agent_id":   agentID,
		"is_deleted": false,
	})
	if err != nil {
		return stats, err
	}

	repliedCount, err := calc.convRepo.Count(ctx, baseRepo.Filter{
		"agent_id":   agentID,
		"replied":    true,
		"is_deleted": false,
	})
	if err != nil {
		return stats, err
	}

	sent := int(totalSent)
	stats.Sent = sent

	if sent > 0 {
		stats.ReplyRate = float64(repliedCount) / float64(sent)
	}

	if sent > 0 {
		if openConvCount, err := calc.emailRepo.CountConversationsWithLabelByAgent(ctx, agentID, EmailLabelOpened); err == nil && openConvCount > 0 {
			stats.OpenRate = float64(openConvCount) / float64(sent)
		}
		if bounceConvCount, err := calc.emailRepo.CountConversationsWithLabelByAgent(ctx, agentID, EmailLabelBounced); err == nil && bounceConvCount > 0 {
			stats.BounceRate = float64(bounceConvCount) / float64(sent)
		}
	}

	return stats, nil
}

// GetInboxDetail retrieves inbox detail information for an agent
func (calc *AgentStatsCalculator) GetInboxDetail(ctx context.Context, agent *domain.Agent) (v1schema.AgentInboxDetail, error) {
	inboxDetail := v1schema.AgentInboxDetail{
		ConnectedDate:  agent.CreatedAt,
		ConnectedInbox: "",
		EmailProvider:  "",
		AddedBy:        "",
	}

	// Load user for inbox detail fields
	user, err := calc.userRepo.FindWithOrganizations(ctx, agent.UserID)
	if err != nil {
		logger.Logger.Warn().Err(err).Str("agent_id", agent.ID.String()).Msg("Failed to load user for agent")
	}

	if user != nil {
		inboxDetail.ConnectedInbox = user.Email
		inboxDetail.EmailProvider = user.Provider
		inboxDetail.AddedBy = user.Name
	} else {
		inboxDetail.ConnectedInbox = agent.Email
		inboxDetail.EmailProvider = string(agent.EmailProvider)
	}

	return inboxDetail, nil
}
