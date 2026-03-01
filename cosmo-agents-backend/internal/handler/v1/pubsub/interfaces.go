package pubsub

import (
	"context"
	"time"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// PubSubUseCase defines the interface for PubSub business logic
type PubSubUseCase interface {
	PublishMessage(ctx context.Context, input *domain.PubSubPublishInput) error
	SubscribeToTopic(ctx context.Context, topic string, timeout time.Duration) (*domain.PubSubMessage, error)
	GetTopicInfo(ctx context.Context, topic string) (*domain.PubSubTopicInfo, error)
}
