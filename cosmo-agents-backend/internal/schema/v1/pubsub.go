package v1

// PubSubPublishRequest represents a request to publish a message to a Redis topic
// Used only for Swagger documentation and request validation
type PubSubPublishRequest struct {
	// Topic is the Redis Pub/Sub channel name
	Topic string `json:"topic" validate:"required" example:"notifications"`
	// Message is the payload to publish; can be any JSON-serializable value
	Message interface{} `json:"message" validate:"required" example:"hello world"`
}
