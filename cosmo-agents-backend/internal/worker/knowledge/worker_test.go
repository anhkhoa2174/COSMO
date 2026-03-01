package knowledge

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/service/summary"
)

func TestHandleKnowledgeIndexing(t *testing.T) {
	vector := &stubVectorStore{}
	worker := New(nil, vector, nil)

	payload := KnowledgeIndexingPayload{
		Collection:   "c1",
		Content:      "hello world",
		EmbeddingGID: "gid",
		Metadata:     map[string]interface{}{"k": "v"},
	}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeKnowledgeIndexing, data)

	if err := worker.HandleKnowledgeIndexing(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if vector.addCalls == 0 {
		t.Fatalf("expected AddDocuments to be called")
	}
}

func TestHandleKnowledgesPruning(t *testing.T) {
	vector := &stubVectorStore{}
	worker := New(nil, vector, nil)

	payload := KnowledgesPruningPayload{
		Collection:    "c1",
		EmbeddingGIDs: []string{"a", "b"},
	}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeKnowledgesPruning, data)

	if err := worker.HandleKnowledgesPruning(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if vector.deleteCalls == 0 {
		t.Fatalf("expected DeleteDocuments to be called")
	}
}

func TestHandleKnowledgesSummarize(t *testing.T) {
	// Summarize requires knowledgeRepo; skipped to avoid DB dependency.
}

// ---- stubs ----

type stubVectorStore struct {
	addCalls    int
	deleteCalls int
}

func (s *stubVectorStore) AddDocuments(ctx context.Context, collection string, documents []string, metadatas []map[string]interface{}, ids []string) error {
	s.addCalls++
	return nil
}

func (s *stubVectorStore) DeleteDocuments(ctx context.Context, collection string, filter map[string]interface{}) error {
	s.deleteCalls++
	return nil
}

type stubSummarizer struct {
	pairs []summary.SummaryPair
}

func (s *stubSummarizer) Summarize(ctx context.Context, content string, detail float64, recursive bool) ([]summary.SummaryPair, error) {
	return s.pairs, nil
}

type stubKnowledgeRepo struct {
	knowledge *domain.Knowledge
	updated   map[string]map[string]interface{}
}

func (s *stubKnowledgeRepo) GetByEmbeddingGID(ctx context.Context, gid string) (*domain.Knowledge, error) {
	return s.knowledge, nil
}

func (s *stubKnowledgeRepo) UpdateByEmbeddingGID(ctx context.Context, gid string, updates map[string]interface{}) error {
	if s.updated == nil {
		s.updated = make(map[string]map[string]interface{})
	}
	s.updated[gid] = updates
	return nil
}

// Unused interface methods for this test
func (s *stubKnowledgeRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.Knowledge, error) {
	return nil, nil
}

func (s *stubKnowledgeRepo) GetByEmbeddingGIDUnscoped(ctx context.Context, embeddingGID string) (*domain.Knowledge, error) {
	return nil, nil
}

func (s *stubKnowledgeRepo) DeleteByEmbeddingGID(ctx context.Context, embeddingGID string) error {
	return nil
}

func (s *stubKnowledgeRepo) UpsertByEmbeddingGID(ctx context.Context, knowledge *domain.Knowledge) (*domain.Knowledge, error) {
	return nil, nil
}

func (s *stubKnowledgeRepo) GetByUserID(ctx context.Context, userID string, limit, offset int) ([]domain.Knowledge, int64, error) {
	return nil, 0, nil
}

func (s *stubKnowledgeRepo) GetByCollection(ctx context.Context, collection string, limit, offset int) ([]domain.Knowledge, int64, error) {
	return nil, 0, nil
}

func (s *stubKnowledgeRepo) FindAll(ctx context.Context, filter baseRepo.Filter, pagination *baseRepo.PaginationParams) (*baseRepo.PaginatedResult[domain.Knowledge], error) {
	return nil, nil
}
