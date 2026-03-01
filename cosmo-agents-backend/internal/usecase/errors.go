package usecase

import "errors"

var (
	// PubSub errors
	ErrTopicRequired       = errors.New("topic is required")
	ErrMessageRequired     = errors.New("message is required")
	ErrSubscriptionTimeout = errors.New("subscription timeout")
	ErrPublishFailed       = errors.New("failed to publish message")
	ErrSubscribeFailed     = errors.New("failed to subscribe to topic")
	ErrGetTopicInfoFailed  = errors.New("failed to get topic info")
)
