package intelligence

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain/outreach"
	v1 "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

func strp(s string) *string { return &s }

// Outreach and the email worker write to interaction_logs; the enrichment
// prompt must see that history, and must say who said what.
func TestHistoryFromLogsReachesThePrompt(t *testing.T) {
	contactID := uuid.New()
	when := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	logs := []*outreach.InteractionLog{
		{ID: uuid.New(), ContactID: contactID, Channel: "LinkedIn", Direction: "incoming",
			Content:   "Our 4 reps each keep their own Google Sheet; is there an on-premise option?",
			Sentiment: strp("neutral"), Timestamp: when.Add(time.Hour)},
		{ID: uuid.New(), ContactID: contactID, Channel: "LinkedIn", Direction: "outgoing",
			Content: "Hi Khang, saw your talk at Vietnam Fintech Summit.", Timestamp: when},
		nil,
	}

	history := historyFromLogs(logs)
	require.Len(t, history, 2)
	assert.Equal(t, "incoming", history[0].Direction)
	assert.Equal(t, "message", history[0].InteractionType)
	assert.Equal(t, when.Add(time.Hour).Format(time.RFC3339), history[0].OccurredAt)
	assert.Equal(t, "neutral", history[0].Content["sentiment"])

	prompt := (&Service{}).buildEnrichmentPrompt(map[string]interface{}{
		"first_name": "Khang", "company": "VietPay", "job_title": "CTO",
	}, history, when.Add(2*time.Hour))
	assert.Contains(t, prompt, "2 recent interactions")
	assert.Contains(t, prompt, "the contact, message via LinkedIn")
	assert.Contains(t, prompt, "us (sales rep), message via LinkedIn")
	assert.Contains(t, prompt, "on-premise option", "the prospect's own words reach the model")
	assert.False(t, strings.Contains(prompt, "map[text:"), "content is printed as text, not as a Go map")
}

func TestPromptWithoutHistory(t *testing.T) {
	prompt := (&Service{}).buildEnrichmentPrompt(map[string]interface{}{"company": "X"}, []*v1.InteractionResponse{}, time.Now())
	assert.Contains(t, prompt, "No interactions yet.")
}
