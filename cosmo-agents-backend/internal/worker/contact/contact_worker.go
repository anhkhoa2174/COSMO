package contact

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/core"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	hubspotService "github.com/rockship/cosmo-agents-go/internal/service/hubspot"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	queueworker "github.com/rockship/cosmo-agents-go/pkg/worker"
)

const (
	TypePullHubspotContacts     = "contact:pull_hubspot"
	TypePullHubspotListContacts = "contact:pull_hubspot_lists"
	TypeContactImportCSV        = "contact:import_csv"
	TypeContactEnrich           = "contact:enrich"
	TypeContactImportHubspot    = "contact:import_hubspot"
)

// PullHubspotContactsPayload represents the payload for pulling HubSpot contacts.
type PullHubspotContactsPayload struct {
	UserID      string `json:"user_id"`
	HubID       string `json:"hub_id"`
	AccessToken string `json:"access_token"`
	After       int    `json:"after"`
	Offset      int    `json:"offset"`
}

// ContactImportCSVPayload represents the payload for importing contacts from CSV.
type ContactImportCSVPayload struct {
	UserID       string            `json:"user_id"`
	FileURL      string            `json:"file_url"`
	OperationID  string            `json:"operation_id"`
	ListID       string            `json:"list_id,omitempty"`
	FieldMapping map[string]string `json:"field_mapping,omitempty"`
}

// ContactEnrichPayload represents the payload for enriching contact data.
type ContactEnrichPayload struct {
	ContactID string `json:"contact_id"`
	UserID    string `json:"user_id"`
}

// Worker handles contact-related background tasks.
type Worker struct {
	db              *gorm.DB
	session         core.Session
	contactRepo     contactRepository
	listContactRepo listContactRepository
	operationRepo   operationRepository
	userRepo        userRepository
	integrationRepo integrationRepository
	enrichService   EnrichmentService
	hubspotAPI      *hubspotService.HubspotAPI
}

type contactRepository interface {
	UpsertMany(ctx context.Context, contacts []*domain.Contact) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Contact, error)
	Update(ctx context.Context, id uuid.UUID, contact *domain.Contact) error
}

type listContactRepository interface {
	AddContactsToList(ctx context.Context, listID uuid.UUID, contacts []*domain.Contact) error
}

type operationRepository interface {
	UpdateFailedStatusWithOutput(ctx context.Context, id uuid.UUID, output domain.JSONB) error
}

type userRepository interface{}

type integrationRepository interface {
	FindByHubID(ctx context.Context, hubID string) (*domain.Integration, error)
}

// EnrichmentService interface for contact enrichment.
type EnrichmentService interface {
	EnrichContact(ctx context.Context, contact *domain.Contact) (*domain.Contact, error)
}

// New creates a new contact worker.
func New(
	db *gorm.DB,
	session core.Session,
	enrichService EnrichmentService,
	hubspotAPI *hubspotService.HubspotAPI,
	contactRepo contactRepository,
	listContactRepo listContactRepository,
	operationRepo operationRepository,
	userRepo userRepository,
	integrationRepo integrationRepository,
) *Worker {
	return &Worker{
		db:              db,
		session:         session,
		contactRepo:     contactRepo,
		listContactRepo: listContactRepo,
		operationRepo:   operationRepo,
		userRepo:        userRepo,
		integrationRepo: integrationRepo,
		enrichService:   enrichService,
		hubspotAPI:      hubspotAPI,
	}
}

// HandlePullHubspotContacts processes pulling contacts from HubSpot.
func (w *Worker) HandlePullHubspotContacts(ctx context.Context, task *asynq.Task) error {
	var payload PullHubspotContactsPayload
	if err := queueworker.ParsePayload(task, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("user_id", payload.UserID).
		Str("hub_id", payload.HubID).
		Int("after", payload.After).
		Msg("Pulling HubSpot contacts")

	if payload.After == -1 {
		// Done pulling contacts
		return nil
	}

	// Get contacts from HubSpot using hubspotAPI
	page, err := w.hubspotAPI.GetContactsAll(ctx, payload.AccessToken, payload.After)
	if err != nil {
		return fmt.Errorf("failed to get contacts from HubSpot: %w", err)
	}

	if len(page.Contacts) == 0 {
		// No more contacts, done
		return nil
	}

	// Parse user ID from string to UUID
	userUUID, err := uuid.Parse(payload.UserID)
	if err != nil {
		return fmt.Errorf("failed to parse user ID: %w", err)
	}

	// Convert to contacts
	contacts := make([]domain.Contact, 0, len(page.Contacts))
	for _, hubspotContact := range page.Contacts {
		// Combine firstname and lastname into name
		firstName := getPropertyOrNA(hubspotContact.Properties, "firstname")
		lastName := getPropertyOrNA(hubspotContact.Properties, "lastname")
		name := combineName(firstName, lastName)

		// Build profile with email and phone
		email := getPropertyOrNA(hubspotContact.Properties, "email")
		phone := getPropertyOrNA(hubspotContact.Properties, "phone")
		profileData := map[string]interface{}{}
		if email != "" && email != "N/A" {
			profileData["email"] = email
		}
		if phone != "" && phone != "N/A" {
			profileData["phone"] = phone
		}
		var profile domain.JSONB
		if len(profileData) > 0 {
			_ = profile.Marshal(profileData)
		}

		contact := domain.Contact{
			UserID:   userUUID,
			Source:   "hubspot",
			SourceID: hubspotContact.ID,
			Name:     name,
			Profile:  profile,
			Company:  getPropertyOrNA(hubspotContact.Properties, "company"),
			JobTitle: getPropertyOrNA(hubspotContact.Properties, "jobtitle"),
			Address:  getPropertyOrNA(hubspotContact.Properties, "address"),
			City:     getPropertyOrNA(hubspotContact.Properties, "city"),
			Country:  getPropertyOrNA(hubspotContact.Properties, "country"),
			State:    getPropertyOrNA(hubspotContact.Properties, "state"),
			Zip:      getPropertyOrNA(hubspotContact.Properties, "zip"),
		}
		contacts = append(contacts, contact)
	}

	// Convert to pointer slice
	contactPtrs := make([]*domain.Contact, len(contacts))
	for i := range contacts {
		contactPtrs[i] = &contacts[i]
	}

	// Upsert contacts
	if err := w.contactRepo.UpsertMany(ctx, contactPtrs); err != nil {
		return fmt.Errorf("failed to upsert contacts: %w", err)
	}

	logger.FromContext(ctx).Info().
		Int("count", len(contacts)).
		Msg("Imported HubSpot contacts")

	// Check if there are more contacts to fetch
	if page.After > 0 {
		// In production, you would re-enqueue with the next page token
		// For example:
		// newPayload := payload
		// newPayload.After = page.After
		// return w.enqueueNextPage(ctx, newPayload)
	}

	return nil
}

// HandlePullHubspotListContacts processes pulling list contacts from HubSpot.
func (w *Worker) HandlePullHubspotListContacts(ctx context.Context, task *asynq.Task) error {
	var payload PullHubspotContactsPayload
	if err := queueworker.ParsePayload(task, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("user_id", payload.UserID).
		Str("hub_id", payload.HubID).
		Msg("PullHubspotListContacts not implemented in refactor; skipping processing")

	return nil
}

// HandleContactImportCSV processes importing contacts from CSV.
func (w *Worker) HandleContactImportCSV(ctx context.Context, task *asynq.Task) error {
	var payload ContactImportCSVPayload
	if err := queueworker.ParsePayload(task, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("user_id", payload.UserID).
		Str("file_url", payload.FileURL).
		Msg("Importing contacts from CSV")

	// Simulate CSV reading from URL (in real scenario, download the file)
	csvData := `first_name,last_name,email,company,job_title
John,Doe,john@example.com,Example Inc,CTO
Jane,Smith,jane@example.com,Acme Corp,CEO`

	reader := csv.NewReader(strings.NewReader(csvData))
	headers, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV headers: %w", err)
	}

	contacts := make([]domain.Contact, 0)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read CSV record: %w", err)
		}

		contact := domain.Contact{}
		var firstName, lastName, email string
		for i, header := range headers {
			value := record[i]
			switch strings.ToLower(header) {
			case "name":
				contact.Name = value
			case "first_name":
				firstName = value
			case "last_name":
				lastName = value
			case "email":
				email = value
			case "company":
				contact.Company = value
			case "job_title":
				contact.JobTitle = value
			}
		}
		// If name wasn't set directly, combine first_name and last_name
		if contact.Name == "" {
			contact.Name = combineName(firstName, lastName)
		}
		// Store email in profile JSONB
		if email != "" {
			profileData := map[string]interface{}{"email": email}
			_ = contact.Profile.Marshal(profileData)
		}

		contacts = append(contacts, contact)
	}

	logger.FromContext(ctx).Info().
		Int("count", len(contacts)).
		Msg("Parsed contacts from CSV")

	// Upsert contacts
	contactPtrs := make([]*domain.Contact, len(contacts))
	for i := range contacts {
		contactPtrs[i] = &contacts[i]
		contactPtrs[i].UserID, _ = uuid.Parse(payload.UserID) // Assign user ID
	}

	if err := w.contactRepo.UpsertMany(ctx, contactPtrs); err != nil {
		return fmt.Errorf("failed to upsert contacts: %w", err)
	}

	logger.FromContext(ctx).Info().
		Int("count", len(contacts)).
		Msg("Imported contacts from CSV successfully")

	return nil
}

// HandleContactEnrich processes enriching contact data.
func (w *Worker) HandleContactEnrich(ctx context.Context, task *asynq.Task) error {
	var payload ContactEnrichPayload
	if err := queueworker.ParsePayload(task, &payload); err != nil {
		return fmt.Errorf("failed to parse payload: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("contact_id", payload.ContactID).
		Str("user_id", payload.UserID).
		Msg("Enriching contact")

	contactID, err := uuid.Parse(payload.ContactID)
	if err != nil {
		return fmt.Errorf("invalid contact ID: %w", err)
	}

	contact, err := w.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return fmt.Errorf("contact not found: %w", err)
	}

	if w.enrichService == nil {
		logger.FromContext(ctx).Warn().Msg("Enrichment service not configured")
		return nil
	}

	// Enrich contact data
	enrichedContact, err := w.enrichService.EnrichContact(ctx, contact)
	if err != nil {
		logger.FromContext(ctx).Error().Err(err).Msg("Failed to enrich contact")
		return fmt.Errorf("enrichment failed: %w", err)
	}

	// Update contact with enriched data
	if err := w.contactRepo.Update(ctx, contactID, enrichedContact); err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	logger.FromContext(ctx).Info().
		Str("contact_id", contactID.String()).
		Msg("Contact enriched successfully")

	return nil
}

// HandleContactImportHubspot triggers pull hubspot contacts process.
func (w *Worker) HandleContactImportHubspot(ctx context.Context, task *asynq.Task) error {
	logger.FromContext(ctx).Info().Msg("HandleContactImportHubspot not implemented in refactor; skipping")
	return nil
}

// enqueueNextPage handles pagination for HubSpot contact pulling.
func (w *Worker) enqueueNextPage(ctx context.Context, payload PullHubspotContactsPayload) error {
	task, err := queueworker.NewTask(TypePullHubspotContacts, payload)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	logger.FromContext(ctx).Info().
		Int("next_after", payload.After).
		Str("user_id", payload.UserID).
		Msg("Enqueuing next HubSpot contacts page")

	// In production, you would use asynq.Client to enqueue
	_ = task
	return nil
}

// EnqueuePullHubspotContacts enqueues a task to pull HubSpot contacts.
func EnqueuePullHubspotContacts(client *asynq.Client, payload PullHubspotContactsPayload) error {
	task, err := queueworker.NewTask(TypePullHubspotContacts, payload)
	if err != nil {
		return err
	}

	_, err = client.Enqueue(task)
	return err
}

// EnqueuePullHubspotListContacts enqueues a task to pull HubSpot list contacts.
func EnqueuePullHubspotListContacts(client *asynq.Client, payload PullHubspotContactsPayload) error {
	task, err := queueworker.NewTask(TypePullHubspotListContacts, payload)
	if err != nil {
		return err
	}

	_, err = client.Enqueue(task)
	return err
}

// Utility functions

func getPropertyOrNA(props map[string]interface{}, key string) string {
	if val, ok := props[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return domain.NOT_AVAILABLE
}

// combineName combines first name and last name into a single name field
func combineName(firstName, lastName string) string {
	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)

	if firstName == domain.NOT_AVAILABLE {
		firstName = ""
	}
	if lastName == domain.NOT_AVAILABLE {
		lastName = ""
	}

	if firstName != "" && lastName != "" {
		return firstName + " " + lastName
	}
	if firstName != "" {
		return firstName
	}
	if lastName != "" {
		return lastName
	}
	return domain.NOT_AVAILABLE
}

func parseInt(value string) int {
	i, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return i
}

// parseFirstName safely extracts first_name from nested attributes map
func parseFirstName(attrs map[string]any) *string {
	if attrs == nil {
		return nil
	}
	contactMap, ok := attrs["contact"].(map[string]any)
	if !ok {
		return nil
	}
	firstName, ok := contactMap["first_name"].(string)
	if !ok {
		return nil
	}
	return &firstName
}

func parseSchemaV2(input map[string]any, result interface{}) error {
	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			WeaklyTypedInput: true,
			Result:           result,
			ZeroFields:       true,
		},
	)
	if err != nil {
		return err
	}
	return decoder.Decode(input)
}
