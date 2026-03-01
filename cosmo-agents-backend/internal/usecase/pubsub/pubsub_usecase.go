package pubsub

import (
	"context"
	"time"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	pubsubRepo "github.com/rockship/cosmo-agents-go/internal/repository/pubsub"
	"github.com/rockship/cosmo-agents-go/internal/usecase"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rs/zerolog"
)

// PubSubUseCase implements business logic for Pub/Sub operations
type PubSubUseCase struct {
	pubsubRepo pubsubRepo.PubSubRepository
	log        *zerolog.Logger
}

// NewPubSubUseCase creates a new PubSubUseCase instance
func NewPubSubUseCase(pubsubRepo pubsubRepo.PubSubRepository) *PubSubUseCase {
	log := logger.Logger
	return &PubSubUseCase{
		pubsubRepo: pubsubRepo,
		log:        &log,
	}
}

// PublishMessage publishes a message to a topic
func (uc *PubSubUseCase) PublishMessage(ctx context.Context, input *domain.PubSubPublishInput) error {
	// Validate input
	if input.Topic == "" {
		return usecase.ErrTopicRequired
	}
	if input.Message == nil {
		return usecase.ErrMessageRequired
	}

	// Publish message
	if err := uc.pubsubRepo.Publish(ctx, input.Topic, input.Message); err != nil {
		uc.log.Error().Err(err).Str("topic", input.Topic).Msg("Failed to publish message")
		return err
	}

	uc.log.Info().Str("topic", input.Topic).Msg("Message published successfully")
	return nil
}

// SubscribeToTopic subscribes to a topic and waits for a message
func (uc *PubSubUseCase) SubscribeToTopic(ctx context.Context, topic string, timeout time.Duration) (*domain.PubSubMessage, error) {
	// Validate input
	if topic == "" {
		return nil, usecase.ErrTopicRequired
	}

	// Set default timeout if not provided
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Subscribe to topic
	message, err := uc.pubsubRepo.Subscribe(ctx, topic, timeout)
	if err != nil {
		if err == context.DeadlineExceeded {
			uc.log.Debug().Str("topic", topic).Msg("Subscription timeout")
			return nil, usecase.ErrSubscriptionTimeout
		}
		uc.log.Error().Err(err).Str("topic", topic).Msg("Failed to subscribe to topic")
		return nil, err
	}

	return message, nil
}

// GetTopicInfo retrieves information about a topic
func (uc *PubSubUseCase) GetTopicInfo(ctx context.Context, topic string) (*domain.PubSubTopicInfo, error) {
	// Validate input
	if topic == "" {
		return nil, usecase.ErrTopicRequired
	}

	// Get topic info
	info, err := uc.pubsubRepo.GetTopicInfo(ctx, topic)
	if err != nil {
		uc.log.Error().Err(err).Str("topic", topic).Msg("Failed to get topic info")
		return nil, err
	}

	return info, nil
}
