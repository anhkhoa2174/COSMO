package pubsub

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// PubSubRepository defines the interface for Redis Pub/Sub operations
type PubSubRepository interface {
	Publish(ctx context.Context, topic string, message interface{}) error
	Subscribe(ctx context.Context, topic string, timeout time.Duration) (*domain.PubSubMessage, error)
	GetTopicInfo(ctx context.Context, topic string) (*domain.PubSubTopicInfo, error)
}

// RedisPubSubRepository implements PubSubRepository using Redis
type RedisPubSubRepository struct {
	client *redis.Client
}

// NewPubSubRepository creates a new PubSubRepository
func NewPubSubRepository(client *redis.Client) PubSubRepository {
	return &RedisPubSubRepository{
		client: client,
	}
}

// Publish publishes a message to a Redis topic
func (r *RedisPubSubRepository) Publish(ctx context.Context, topic string, message interface{}) error {
	// Marshal message to JSON
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish to Redis
	return r.client.Publish(ctx, topic, messageBytes).Err()
}

// Subscribe subscribes to a Redis topic and waits for a message
func (r *RedisPubSubRepository) Subscribe(ctx context.Context, topic string, timeout time.Duration) (*domain.PubSubMessage, error) {
	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Subscribe to the topic
	pubsub := r.client.Subscribe(timeoutCtx, topic)
	defer pubsub.Close()

	// Wait for a message
	ch := pubsub.Channel()
	select {
	case msg := <-ch:
		if msg == nil {
			return nil, context.DeadlineExceeded
		}
		return domain.NewPubSubMessage(msg.Channel, msg.Payload), nil
	case <-timeoutCtx.Done():
		return nil, timeoutCtx.Err()
	}
}

// GetTopicInfo returns information about a Redis topic
func (r *RedisPubSubRepository) GetTopicInfo(ctx context.Context, topic string) (*domain.PubSubTopicInfo, error) {
	// Get number of subscribers using PUBSUB NUMSUB command
	cmd := r.client.PubSubNumSub(ctx, topic)
	result, err := cmd.Result()
	if err != nil {
		return nil, err
	}

	subscribers := int64(0)
	if count, exists := result[topic]; exists {
		subscribers = count
	}

	return &domain.PubSubTopicInfo{
		Topic:       topic,
		Subscribers: subscribers,
	}, nil
}
