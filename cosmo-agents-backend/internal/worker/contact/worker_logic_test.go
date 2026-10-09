package contact

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	v1 "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

func TestHandleContactEnrich(t *testing.T) {
	contactID := uuid.New()
	userID := uuid.New()
	orgID := uuid.New()
	fakeContactRepo := &stubContactRepo{
		contacts: map[uuid.UUID]*domain.Contact{
			contactID: {Base: domain.Base{ID: contactID}, Name: "Bob"},
		},
	}
	enricher := &stubEnrichmentService{}
	worker := New(nil, nil, enricher, nil, fakeContactRepo, nil, nil, nil, nil)

	payload := ContactEnrichPayload{
		ContactID: contactID.String(),
		UserID:    userID.String(),
		OrgID:     orgID.String(),
	}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeContactEnrich, data)

	if err := worker.HandleContactEnrich(context.Background(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if enricher.calls != 1 {
		t.Fatalf("expected the enrichment service to be called once, got %d", enricher.calls)
	}
	if enricher.contactID != contactID || enricher.userID != userID || enricher.orgID != orgID {
		t.Fatal("enrichment must be scoped to the contact, user and org from the payload")
	}
	if enricher.forceRefresh {
		t.Fatal("auto-enrichment must not force a refresh — a re-import would re-bill every row")
	}
}

func TestHandleContactEnrich_NoServiceConfigured(t *testing.T) {
	contactID := uuid.New()
	worker := New(nil, nil, nil, nil, &stubContactRepo{}, nil, nil, nil, nil)

	payload := ContactEnrichPayload{ContactID: contactID.String(), UserID: uuid.New().String()}
	data, _ := json.Marshal(payload)
	task := asynq.NewTask(TypeContactEnrich, data)

	if err := worker.HandleContactEnrich(context.Background(), task); err != nil {
		t.Fatalf("a missing enrichment service should be a no-op, got %v", err)
	}
}

func TestEnqueueAutoEnrichment_CapsPerImport(t *testing.T) {
	contacts := make([]*domain.Contact, maxAutoEnrichPerImport+10)
	for i := range contacts {
		contacts[i] = &domain.Contact{Base: domain.Base{ID: uuid.New()}}
	}
	queue := &stubEnqueuer{}
	worker := New(nil, nil, nil, nil, &stubContactRepo{}, nil, nil, nil, nil).WithQueue(queue)

	worker.enqueueAutoEnrichment(context.Background(), contacts, uuid.New().String())

	if queue.count != maxAutoEnrichPerImport {
		t.Fatalf("expected the import cap of %d enrichments, got %d", maxAutoEnrichPerImport, queue.count)
	}
	if queue.lastType != TypeContactEnrich {
		t.Fatalf("expected %s tasks, got %s", TypeContactEnrich, queue.lastType)
	}
}

func TestEnqueueAutoEnrichment_NoQueueIsNoOp(t *testing.T) {
	worker := New(nil, nil, nil, nil, &stubContactRepo{}, nil, nil, nil, nil)
	// Must not panic when no queue was configured.
	worker.enqueueAutoEnrichment(context.Background(), []*domain.Contact{{}}, uuid.New().String())
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
	calls        int
	userID       uuid.UUID
	orgID        uuid.UUID
	contactID    uuid.UUID
	forceRefresh bool
}

func (s *stubEnrichmentService) EnrichContact(ctx context.Context, userID, orgID, contactID uuid.UUID, forceRefresh bool) (*v1.ContactEnrichmentResponse, error) {
	s.calls++
	s.userID, s.orgID, s.contactID, s.forceRefresh = userID, orgID, contactID, forceRefresh
	return &v1.ContactEnrichmentResponse{}, nil
}

type stubEnqueuer struct {
	count    int
	lastType string
}

func (s *stubEnqueuer) EnqueueTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	s.count++
	s.lastType = taskType
	return &asynq.TaskInfo{}, nil
}
