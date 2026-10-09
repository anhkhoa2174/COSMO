package ai

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	openai "github.com/sashabaranov/go-openai"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
)

// capturingHTTPClient records the request body and answers with a fixed reply.
type capturingHTTPClient struct {
	body string
	sent *string
}

func (c capturingHTTPClient) Do(req *http.Request) (*http.Response, error) {
	b, _ := io.ReadAll(req.Body)
	*c.sent = string(b)
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header),
		Body: io.NopCloser(strings.NewReader(c.body)), Request: req}, nil
}

func newCapturingAIClient(t *testing.T, sent *string) *ai.OpenAIClient {
	t.Helper()
	client := ai.NewOpenAIClient(ai.Config{APIKey: "test-key", Model: "test-model"})
	clientField := reflect.ValueOf(client).Elem().FieldByName("client")
	clientPtr := reflect.NewAt(clientField.Type(), unsafe.Pointer(clientField.UnsafeAddr())).Elem()
	oaClient := clientPtr.Interface().(*openai.Client)
	cfgField := reflect.ValueOf(oaClient).Elem().FieldByName("config")
	cfgPtr := reflect.NewAt(cfgField.Type(), unsafe.Pointer(cfgField.UnsafeAddr())).Elem()
	cfg := cfgPtr.Interface().(openai.ClientConfig)
	cfg.BaseURL = "http://example.com"
	cfg.HTTPClient = capturingHTTPClient{
		body: `{"choices":[{"message":{"content":"{\"subject\":\"Re\",\"body\":\"ok\"}"}}]}`,
		sent: sent,
	}
	cfgPtr.Set(reflect.ValueOf(cfg))
	return client
}

func promptFor(t *testing.T, intent string) string {
	t.Helper()
	var sent string
	svc := &AIEmailService{openAIClient: newCapturingAIClient(t, &sent)}
	user := &domain.User{Email: "alex@cosmo.ai", Name: "Alex"}
	thread := []ConversationMessage{{FromEmail: "david@orion.com", ToEmail: "alex@cosmo.ai",
		Subject: "Re: intro", Content: "Stop emailing me."}}
	if _, _, err := svc.generateReplyWithOpenAI(context.Background(), user, thread,
		EmailIntentResult{Intent: intent}, "k", "Re: intro", "david@orion.com"); err != nil {
		t.Fatal(err)
	}
	// The body is JSON; unescape it so assertions read naturally.
	return strings.NewReplacer(`\n`, "\n", `\u003c`, "<", `\u003e`, ">", `\u0026`, "&").Replace(sent)
}

// The opt-out reply was once written in the prospect's own voice. The prompt
// now names who is writing to whom.
func TestReplyPrompt_NamesTheSenderAndForbidsEchoingTheProspect(t *testing.T) {
	p := promptFor(t, "Interested")
	if !strings.Contains(p, "You are writing as Alex <alex@cosmo.ai>") ||
		!strings.Contains(p, "replying to the prospect <david@orion.com>") {
		t.Fatalf("sender/recipient not stated:\n%s", p)
	}
	if !strings.Contains(p, "Never restate the prospect's own words") {
		t.Fatalf("no instruction against echoing the prospect:\n%s", p)
	}
}

func TestReplyPrompt_OptOutGetsAConfirmationOnly(t *testing.T) {
	p := promptFor(t, "Do not contact")
	if !strings.Contains(p, "will not be emailed again") || !strings.Contains(p, "No product information, links or call to action") {
		t.Fatalf("opt-out instruction missing:\n%s", p)
	}
}

func TestReplyInstructionFor(t *testing.T) {
	cases := map[string]string{
		"Out of office":  "follow up after they return",
		"Not interested": "Do not pitch",
		"Interested":     "",
	}
	for intent, want := range cases {
		got := replyInstructionFor(intent)
		if want == "" && got != "" || want != "" && !strings.Contains(got, want) {
			t.Fatalf("%s: got %q", intent, got)
		}
	}
}
