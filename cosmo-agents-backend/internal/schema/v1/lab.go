package v1

// GenerateSampleResponseRequest represents request for generating sample responses
type GenerateSampleResponseRequest struct {
	Content  string                 `json:"content" validate:"required"`
	Contact  map[string]interface{} `json:"contact" validate:"required"`
	IntentID *int                   `json:"intent_id,omitempty"` // Optional, if nil will try all intents
}

// SampleResponseItem represents a single sample response
type SampleResponseItem struct {
	Intent   string `json:"intent"`
	Response string `json:"response"`
}
