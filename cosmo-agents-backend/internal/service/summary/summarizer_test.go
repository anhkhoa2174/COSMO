package summary

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newStubOpenAI(t *testing.T, responses []string) *openai.Client {
	t.Helper()
	idx := 0
	client := &http.Client{
		Transport: rtFunc(func(r *http.Request) (*http.Response, error) {
			if idx >= len(responses) {
				idx = len(responses) - 1
			}
			body := `{
				"id":"1",
				"object":"chat.completion",
				"created":123,
				"model":"gpt-4o-mini",
				"choices":[{"index":0,"message":{"role":"assistant","content":"` + responses[idx] + `"},"finish_reason":"stop"}]
			}`
			idx++
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	c := openai.NewClient(
		option.WithHTTPClient(client),
		option.WithBaseURL("http://localhost"),
		option.WithAPIKey("test"),
	)
	return &c
}

func TestSummarizer_SummarizeAndChunking(t *testing.T) {
	logger := zerolog.New(io.Discard)
	client := newStubOpenAI(t, []string{"summary1", "compressed"})
	s := NewSummarizer(client, "", &logger)

	content := "para1\n\npara2\n\npara3"
	pairs, err := s.Summarize(context.Background(), content, 0.5, true)
	require.NoError(t, err)
	require.Len(t, pairs, 1)
	assert.Equal(t, "compressed", pairs[0].Compressed)
	assert.Contains(t, pairs[0].Raw, "summary1")

	chunksLow := s.chunkContent("a\n\nb\n\nc", 0.0)
	assert.Len(t, chunksLow, 1)
	chunksHigh := s.chunkContent("a\n\nb\n\nc", 1.0)
	assert.Len(t, chunksHigh, 3)
}

func TestSummarizer_ErrorsAndHelpers(t *testing.T) {
	logger := zerolog.New(io.Discard)
	client := newStubOpenAI(t, []string{"summary", "summary"})
	s := NewSummarizer(client, "", &logger)

	_, err := s.Summarize(context.Background(), "", 0.5, false)
	assert.Error(t, err)

	// compressSummary should fall back when compressed longer than raw
	compressed, err := s.compressSummary(context.Background(), "short")
	require.NoError(t, err)
	assert.NotEmpty(t, compressed)
}
