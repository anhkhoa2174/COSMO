package mail

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

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newStubOpenAIClient(t *testing.T) *openai.Client {
	t.Helper()
	messageContent := "<subject>Subj</subject><content>Body</content><corrected_template>Fixed</corrected_template><reply_email>ReplyBody</reply_email>"

	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			_, _ = io.ReadAll(r.Body)
			defer r.Body.Close()
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"id":"1",
					"object":"chat.completion",
					"created":123,
					"model":"gpt-4o-mini",
					"choices":[{"index":0,"message":{"role":"assistant","content":"` + messageContent + `"},"finish_reason":"stop"}]
				}`)),
				Request: r,
			}
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		}),
	}

	c := openai.NewClient(
		option.WithBaseURL("http://localhost"),
		option.WithAPIKey("test"),
		option.WithHTTPClient(client),
	)
	return &c
}

func TestMailWriter_GenerateSingleAndReply(t *testing.T) {
	logger := zerolog.New(io.Discard)
	client := newStubOpenAIClient(t)
	params := EmailParameters{
		Sender: SenderInfo{
			Name:  "Agent",
			Email: "agent@example.com",
		},
		CampaignType: CampaignTypeRevive,
		ClientFields: []string{"first_name", "company"},
	}
	writer := NewMailWriter(client, "", params, &logger)

	email, err := writer.GenerateSingleOutreach(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, EmailTypeReach, email.Type)
	assert.Equal(t, "Subj", email.Subject)
	assert.Contains(t, email.Content, "{agent_signature}")

	reply, err := writer.GenerateReply(context.Background(), []EmailTemplate{{FromEmail: "c@example.com", Content: "Hi"}}, IntentInterested)
	require.NoError(t, err)
	assert.Equal(t, EmailTypeReply, reply.Type)
	assert.Equal(t, "Subj", reply.Subject)
}

func TestMailWriter_FormatMergeTagsAndHelpers(t *testing.T) {
	logger := zerolog.New(io.Discard)
	client := newStubOpenAIClient(t)
	writer := NewMailWriter(client, "", EmailParameters{Sender: SenderInfo{Email: "agent@example.com"}, ClientFields: []string{"first_name"}}, &logger)

	corrected, err := writer.FormatMergeTags(context.Background(), "Hi {name}", []string{"first_name"})
	require.NoError(t, err)
	assert.Equal(t, "Fixed", corrected)

	convCtx := writer.buildConversationContext([]EmailTemplate{{FromEmail: "a", ToEmail: "b", Subject: "s", Content: "c"}})
	assert.Contains(t, convCtx, "conversation-0")

	system := writer.buildSystemMessage()
	assert.Contains(t, system, "{first_name}")

	replySystem := writer.buildReplySystemMessage(IntentOutOfOffice)
	assert.Contains(t, strings.ToLower(replySystem), "follow up")
}

func TestMailWriter_GenerateSampleResponse(t *testing.T) {
	logger := zerolog.New(io.Discard)
	client := newStubOpenAIClient(t)
	resp, err := GenerateSampleResponse(context.Background(), client, "", "Original", IntentReferral, map[string]interface{}{"first_name": "A"}, &logger)
	require.NoError(t, err)
	assert.Equal(t, "ReplyBody", resp)
}

func TestMailWriter_ParsingHelpers(t *testing.T) {
	logger := zerolog.New(io.Discard)
	writer := NewMailWriter(nil, "", EmailParameters{Sender: SenderInfo{Email: "from@example.com"}}, &logger)

	email := writer.parseEmailResponse("<subject>Hi</subject><content>Body</content>", EmailTypeReach)
	assert.Equal(t, "Hi", email.Subject)
	assert.Contains(t, email.Content, "{agent_signature}")

	assert.Equal(t, EmailTypeReach, writer.determineEmailType(nil))
	assert.Equal(t, EmailTypeFollowUp1, writer.determineEmailType([]EmailTemplate{{}}))
	assert.Equal(t, EmailTypeFollowUp2, writer.determineEmailType([]EmailTemplate{{}, {}}))
	assert.Equal(t, EmailType("Follow-up Email 3"), writer.determineEmailType([]EmailTemplate{{}, {}, {}}))

	assert.Equal(t, "content", extractInnerText("<tag>content</tag>", "tag"))
	assert.Equal(t, "", extractInnerText("no tag", "tag"))

	guidance := writer.getIntentGuidance(IntentRequestForPricing)
	assert.True(t, strings.Contains(strings.ToLower(guidance), "price"))

	writer.params.CampaignType = CampaignTypeUpsell
	campaignWho := writer.getCampaignWhoami()
	assert.NotEmpty(t, campaignWho)
}
