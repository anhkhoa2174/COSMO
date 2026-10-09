package ai

import (
	"github.com/openai/openai-go/option"
)

// CompatOptions builds the client options for the official OpenAI SDK,
// honouring an alternative API-compatible host when one is configured.
//
// The project uses two OpenAI SDKs — this one for the intent classifier and
// sashabaranov's elsewhere — so the switch has to be applied in both places or
// half the system would keep calling the exhausted account. See provider.go
// for why the switch exists at all.
func CompatOptions(apiKey string) []option.RequestOption {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if base := BaseURL(); base != "" {
		opts = append(opts, option.WithBaseURL(base))
	}
	return opts
}
