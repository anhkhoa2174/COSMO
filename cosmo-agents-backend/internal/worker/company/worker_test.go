package company

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hibiken/asynq"
)

func TestNewWorker(t *testing.T) {
	if New() == nil {
		t.Fatalf("expected company worker instance")
	}
}

func TestTypeConstant(t *testing.T) {
	if TypeCompanyIndexing == "" {
		t.Fatalf("type constant should not be empty")
	}
}

func TestHandleCompanyIndexing_Success(t *testing.T) {
	worker := New()
	payload := CompanyIndexingPayload{
		Collection:   "companies",
		Content:      " ACME corp profile ",
		EmbeddingGID: "gid-1",
		Metadata:     map[string]interface{}{"source": "test"},
	}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeCompanyIndexing, data)

	if err := worker.HandleCompanyIndexing(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestHandleCompanyIndexing_EmptyContent(t *testing.T) {
	worker := New()
	payload := CompanyIndexingPayload{
		Collection:   "companies",
		Content:      "   ",
		EmbeddingGID: "gid-1",
	}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeCompanyIndexing, data)

	if err := worker.HandleCompanyIndexing(context.Background(), task); err == nil {
		t.Fatalf("expected error for empty content")
	}
}
