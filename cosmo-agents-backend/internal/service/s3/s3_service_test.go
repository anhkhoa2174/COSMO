package s3

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type mockS3Transport struct {
	lastPutBody string
}

func (m *mockS3Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.Method {
	case http.MethodPut:
		body, _ := io.ReadAll(req.Body)
		m.lastPutBody = string(body)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	case http.MethodGet:
		if strings.Contains(req.URL.RawQuery, "list-type=2") {
			xml := `<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Contents><Key>sample.txt</Key><Size>5</Size></Contents></ListBucketResult>`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(xml)), Header: make(http.Header)}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("hello world")), Header: make(http.Header)}, nil
	case http.MethodDelete:
		return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
	default:
		return &http.Response{StatusCode: http.StatusTeapot, Body: io.NopCloser(strings.NewReader("unexpected"))}, nil
	}
}

func TestS3ServiceOperations(t *testing.T) {
	transport := &mockS3Transport{}
	client := s3.New(s3.Options{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("key", "secret", ""),
		HTTPClient:  &http.Client{Transport: transport},
	})

	svc := &S3Service{client: client}
	ctx := context.Background()

	// Upload
	if err := svc.UploadFile(ctx, "bucket", "key.txt", bytes.NewBufferString("data"), "text/plain", map[string]string{"meta": "yes"}); err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if transport.lastPutBody != "data" {
		t.Fatalf("expected uploaded data to be sent, got %s", transport.lastPutBody)
	}

	// Get
	content, err := svc.GetObject(ctx, "bucket", "key.txt")
	if err != nil {
		t.Fatalf("get object failed: %v", err)
	}
	if string(content) != "hello world" {
		t.Fatalf("unexpected object content: %s", string(content))
	}

	// List
	items, err := svc.ListObjects(ctx, "bucket", "key")
	if err != nil {
		t.Fatalf("list objects failed: %v", err)
	}
	if len(items) != 1 || items[0].Key == nil || *items[0].Key != "sample.txt" {
		t.Fatalf("unexpected list result: %+v", items)
	}

	// Presign
	url, err := svc.GeneratePresignedURL(ctx, "bucket", "key.txt", time.Minute)
	if err != nil {
		t.Fatalf("presign failed: %v", err)
	}
	if url == "" || !strings.Contains(url, "X-Amz-Expires=60") {
		t.Fatalf("unexpected presigned url: %s", url)
	}

	// Delete
	if err := svc.DeleteObject(ctx, "bucket", "key.txt"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestNewS3ServiceMissingRegion(t *testing.T) {
	t.Setenv("AWS_REGION", "")
	t.Setenv("AWS_DEFAULT_REGION", "")

	svc, err := NewS3Service(S3Config{
		Region:          "us-east-1",
		AccessKeyID:     "key",
		SecretAccessKey: "secret",
		EndpointURL:     "https://example.com",
	})
	if err != nil {
		t.Fatalf("expected service to initialize, got %v", err)
	}
	if svc == nil || svc.client == nil {
		t.Fatal("expected non-nil service and client")
	}
}
