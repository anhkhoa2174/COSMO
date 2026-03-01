package contact

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

func TestHandleContactEnrich(t *testing.T) {
	contactID := uuid.New()
	fakeContactRepo := &stubContactRepo{
		contacts: map[uuid.UUID]*domain.Contact{
			contactID: {Base: domain.Base{ID: contactID}, Name: "Bob"},
		},
	}
	enricher := &stubEnrichmentService{contact: &domain.Contact{Name: "Enriched"}}
	worker := New(nil, nil, enricher, nil, fakeContactRepo, nil, nil, nil, nil)

	payload := ContactEnrichPayload{ContactID: contactID.String(), UserID: uuid.New().String()}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeContactEnrich, data)

	if err := worker.HandleContactEnrich(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if fakeContactRepo.updated == nil || fakeContactRepo.updated.Name != "Enriched" {
		t.Fatalf("expected contact updated via enrichment")
	}
}

func TestHandleContactImportCSV(t *testing.T) {
	fakeContactRepo := &stubContactRepo{contacts: make(map[uuid.UUID]*domain.Contact)}
	fakeListRepo := &stubListContactRepo{}
	worker := New(nil, nil, nil, nil, fakeContactRepo, fakeListRepo, nil, nil, nil)

	payload := ContactImportCSVPayload{
		UserID:      uuid.New().String(),
		FileURL:     "http://example.com/file.csv",
		OperationID: uuid.New().String(),
	}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeContactImportCSV, data)

	if err := worker.HandleContactImportCSV(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// ---- stubs ----

type stubContactRepo struct {
	contacts map[uuid.UUID]*domain.Contact
	updated  *domain.Contact
}

func (s *stubContactRepo) UpsertMany(ctx context.Context, contacts []*domain.Contact) error {
	for _, c := range contacts {
		s.contacts[c.ID] = c
	}
	return nil
}

func (s *stubContactRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Contact, error) {
	return s.contacts[id], nil
}

func (s *stubContactRepo) Update(ctx context.Context, id uuid.UUID, contact *domain.Contact) error {
	s.updated = contact
	return nil
}

type stubListContactRepo struct{}

func (s *stubListContactRepo) AddContactsToList(ctx context.Context, listID uuid.UUID, contacts []*domain.Contact) error {
	return nil
}

type stubEnrichmentService struct {
	contact *domain.Contact
}

func (s *stubEnrichmentService) EnrichContact(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	return s.contact, nil
}
