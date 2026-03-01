package resend

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/resend/resend-go/v3"
)

type resendTransport struct {
	statusCode int
	body       string
	lastBody   string
}

func (rt *resendTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	data, _ := io.ReadAll(req.Body)
	rt.lastBody = string(data)

	status := rt.statusCode
	if status == 0 {
		status = http.StatusOK
	}

	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(rt.body)),
		Header:     make(http.Header),
	}, nil
}

func TestSendEmailSuccess(t *testing.T) {
	transport := &resendTransport{
		body: `{"id":"email_123"}`,
	}
	client := resend.NewCustomClient(&http.Client{Transport: transport}, "test-key")
	svc := &ResendService{client: client}

	err := svc.SendEmail(
		"sender@example.com",
		[]string{"to@example.com"},
		"Subject",
		EmailContent{PlainText: "Hello plain"},
		nil, nil,
	)
	if err != nil {
		t.Fatalf("expected send email success, got %v", err)
	}

	if !strings.Contains(transport.lastBody, "Hello plain") {
		t.Fatalf("expected plain text fallback in payload, got %s", transport.lastBody)
	}
}

func TestSendEmailUsesHTMLAndHandlesError(t *testing.T) {
	transport := &resendTransport{
		statusCode: http.StatusBadRequest,
		body:       `{"message":"bad"}`,
	}
	client := resend.NewCustomClient(&http.Client{Transport: transport}, "test-key")
	svc := &ResendService{client: client}

	err := svc.SendEmail(
		"sender@example.com",
		[]string{"to@example.com"},
		"Subject",
		EmailContent{HTML: "<p>Hello</p>"},
		[]string{"cc@example.com"},
		[]string{"bcc@example.com"},
	)
	if err == nil {
		t.Fatal("expected error on non-2xx response")
	}

	if !strings.Contains(transport.lastBody, "Hello") {
		t.Fatalf("expected html body to be sent, got %s", transport.lastBody)
	}
	if !strings.Contains(transport.lastBody, "bcc@example.com") {
		t.Fatalf("expected bcc recipients in payload, got %s", transport.lastBody)
	}
}
