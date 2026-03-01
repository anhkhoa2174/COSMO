package ai

import (
	"context"

	"github.com/google/uuid"

	aiService "github.com/rockship/cosmo-agents-go/internal/service/ai"
)

// AIEmailUsecase defines the business operations for AI email utilities.
type AIEmailUsecase interface {
	ClassifyIntent(ctx context.Context, content string) (*aiService.EmailIntentResult, error)
	GenerateReply(ctx context.Context, userID uuid.UUID, userEmail string, conversationID *uuid.UUID, conversation []aiService.ConversationMessage) (*aiService.AIReplyEmail, error)
	GenerateDeprecatedOutreach(ctx context.Context, userID uuid.UUID, campaign string, clientData map[string]interface{}) ([]aiService.AIGeneratedTemplate, error)
}

// aiEmailUsecaseAdapter is a thin adapter that satisfies AIEmailUsecase by delegating to service.AIEmailService.
type aiEmailUsecaseAdapter struct {
	svc *aiService.AIEmailService
}

// NewAIEmailUsecaseAdapter returns an AIEmailUsecase that delegates to the provided service.
func NewAIEmailUsecaseAdapter(svc *aiService.AIEmailService) AIEmailUsecase {
	return &aiEmailUsecaseAdapter{svc: svc}
}

func (a *aiEmailUsecaseAdapter) ClassifyIntent(ctx context.Context, content string) (*aiService.EmailIntentResult, error) {
	return a.svc.ClassifyIntent(ctx, content)
}

func (a *aiEmailUsecaseAdapter) GenerateReply(ctx context.Context, userID uuid.UUID, userEmail string, conversationID *uuid.UUID, conversation []aiService.ConversationMessage) (*aiService.AIReplyEmail, error) {
	return a.svc.GenerateReply(ctx, userID, userEmail, conversationID, conversation)
}

func (a *aiEmailUsecaseAdapter) GenerateDeprecatedOutreach(ctx context.Context, userID uuid.UUID, campaign string, clientData map[string]interface{}) ([]aiService.AIGeneratedTemplate, error) {
	return a.svc.GenerateDeprecatedOutreach(ctx, userID, campaign, clientData)
}
