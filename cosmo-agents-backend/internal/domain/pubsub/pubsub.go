package pubsub

import (
	"time"
)

// PubSubMessage represents a message published to or received from a Redis Pub/Sub topic
type PubSubMessage struct {
	Topic     string      `json:"topic"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

// PubSubTopicInfo represents information about a Redis Pub/Sub topic
type PubSubTopicInfo struct {
	Topic       string `json:"topic"`
	Subscribers int64  `json:"subscribers"`
}

// PubSubPublishInput represents input for publishing a message
type PubSubPublishInput struct {
	Topic   string      `json:"topic" validate:"required"`
	Message interface{} `json:"message" validate:"required"`
}

// NewPubSubMessage creates a new PubSubMessage
func NewPubSubMessage(topic string, payload interface{}) *PubSubMessage {
	return &PubSubMessage{
		Topic:     topic,
		Payload:   payload,
		Timestamp: time.Now(),
	}
}
