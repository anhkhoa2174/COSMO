package pubsub

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPubSubMessage(t *testing.T) {
	topic := "test-topic"
	payload := "test payload"

	msg := NewPubSubMessage(topic, payload)

	assert.NotNil(t, msg)
	assert.Equal(t, topic, msg.Topic)
	assert.Equal(t, payload, msg.Payload)
	assert.WithinDuration(t, time.Now(), msg.Timestamp, time.Second)
}

func TestPubSubPublishInput(t *testing.T) {
	input := &PubSubPublishInput{
		Topic:   "test-topic",
		Message: map[string]string{"key": "value"},
	}

	assert.Equal(t, "test-topic", input.Topic)
	assert.NotNil(t, input.Message)
}

func TestPubSubTopicInfo(t *testing.T) {
	info := &PubSubTopicInfo{
		Topic:       "test-topic",
		Subscribers: 10,
	}

	assert.Equal(t, "test-topic", info.Topic)
	assert.Equal(t, int64(10), info.Subscribers)
}
