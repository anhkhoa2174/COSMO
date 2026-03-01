package agents

import (
	"context"

	"github.com/google/uuid"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	campaignrepo "github.com/rockship/cosmo-agents-go/internal/repository/campaign"
	conversationrepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailrepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
)

// CampaignIntelligenceAgent aggregates campaign metrics and recommendations.
type CampaignIntelligenceAgent struct {
	campaignRepo     *campaignrepo.CampaignRepository
	conversationRepo *conversationrepo.ConversationRepository
	emailRepo        *emailrepo.Repository
}

func NewCampaignIntelligenceAgent(
	campaignRepo *campaignrepo.CampaignRepository,
	conversationRepo *conversationrepo.ConversationRepository,
	emailRepo *emailrepo.Repository,
) *CampaignIntelligenceAgent {
	return &CampaignIntelligenceAgent{
		campaignRepo:     campaignRepo,
		conversationRepo: conversationRepo,
		emailRepo:        emailRepo,
	}
}

type CampaignIntelligenceResult struct {
	CampaignID     uuid.UUID `json:"campaign_id"`
	TotalEmails    int64     `json:"total_emails"`
	TotalThreads   int64     `json:"total_threads"`
	Replies        int64     `json:"replies"`
	ReplyRate      float64   `json:"reply_rate"`
	Status         string    `json:"status"`
	Recommendation string    `json:"recommendation"`
}

func (a *CampaignIntelligenceAgent) Run(ctx context.Context, userID, orgID, campaignID uuid.UUID) (*CampaignIntelligenceResult, error) {
	campaign, err := a.campaignRepo.FindWithRelations(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if campaign == nil {
		return nil, ErrNotFound
	}
	if campaign.UserID != userID {
		if campaign.OrganizationID == nil || *campaign.OrganizationID != orgID {
			return nil, ErrUnauthorized
		}
	}

	_, totalEmails, err := a.emailRepo.FindByCampaignID(ctx, campaignID, 0, 1)
	if err != nil {
		return nil, err
	}
	_, totalThreads, err := a.conversationRepo.FindByCampaignID(ctx, campaignID, 0, 1)
	if err != nil {
		return nil, err
	}
	replies, err := a.countReplies(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	replyRate := 0.0
	if totalThreads > 0 {
		replyRate = float64(replies) / float64(totalThreads)
	}

	recommendation := "Monitor replies and continue"
	if replyRate < 0.02 {
		recommendation = "Low reply rate: consider revising subject lines or targeting"
	}

	return &CampaignIntelligenceResult{
		CampaignID:     campaignID,
		TotalEmails:    totalEmails,
		TotalThreads:   totalThreads,
		Replies:        replies,
		ReplyRate:      replyRate,
		Status:         string(campaign.Status),
		Recommendation: recommendation,
	}, nil
}

func (a *CampaignIntelligenceAgent) countReplies(ctx context.Context, campaignID uuid.UUID) (int64, error) {
	var count int64
	err := a.conversationRepo.GetDB().WithContext(ctx).
		Model(&domain.Conversation{}).
		Where("campaign_id = ? AND is_deleted = ?", campaignID, false).
		Where("replied = ?", true).
		Count(&count).Error
	return count, err
}
