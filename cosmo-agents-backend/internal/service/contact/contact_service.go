package contact

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	customFieldRepo "github.com/rockship/cosmo-agents-go/internal/repository/custom_field"
)

// ContactRepositoryInterface defines the interface for contact repository operations
type ContactRepositoryInterface interface {
	Create(ctx context.Context, contact *domain.Contact) error
	CreateWithOwnershipCheck(ctx context.Context, contact *domain.Contact) (*domain.Contact, error)
	Update(ctx context.Context, id uuid.UUID, contact *domain.Contact) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Contact, error)
	FindByUserIDWithPagination(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Contact, int, error)
	Search(ctx context.Context, userID uuid.UUID, query string, offset, limit int) ([]*domain.Contact, int, error)
	SearchWithFilter(ctx context.Context, userID uuid.UUID, orgIDs []uuid.UUID, filter map[string]interface{}, pagination *baseRepo.PaginationParams) ([]*domain.Contact, int, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	UpsertMany(ctx context.Context, contacts []*domain.Contact) error
	UpsertManyWithOwnershipCheck(ctx context.Context, contacts []*domain.Contact, userID uuid.UUID, organizationID uuid.UUID) (*contactRepo.OwnershipCheckResult, error)
}

// ListContactRepositoryInterface defines the interface for list contact repository operations
type ListContactRepositoryInterface interface {
	AddContactToList(ctx context.Context, association *domain.ListContactAssociation) error
	RemoveContactFromList(ctx context.Context, association *domain.ListContactAssociation) error
	FindByContactID(ctx context.Context, contactID uuid.UUID) ([]*domain.ListContact, error)
	FindContactsByListIDWithPagination(ctx context.Context, listID uuid.UUID, offset, limit int) ([]*domain.Contact, int, error)
}

// ContactService provides business logic for contact operations
type ContactService struct {
	contactRepo     ContactRepositoryInterface
	customFieldRepo *customFieldRepo.CustomFieldRepository
	listContactRepo ListContactRepositoryInterface
	fieldValidator  *ContactFieldValidator
	fieldMapper     *FieldMapper
	csvImporter     *CSVImporter
}

// NewContactService creates a new ContactService
func NewContactService(
	contactRepo ContactRepositoryInterface,
	customFieldRepo *customFieldRepo.CustomFieldRepository,
	listContactRepo ListContactRepositoryInterface,
) *ContactService {
	return &ContactService{
		contactRepo:     contactRepo,
		customFieldRepo: customFieldRepo,
		listContactRepo: listContactRepo,
		fieldValidator:  NewContactFieldValidator(customFieldRepo),
		fieldMapper:     NewFieldMapper(),
		csvImporter:     NewCSVImporter(contactRepo),
	}
}

// CreateContact creates a new contact with field validation and ownership check
func (s *ContactService) CreateContact(ctx context.Context, contact *domain.Contact, organizationID uuid.UUID) error {
	// Validate contact fields
	if len(contact.Profile) > 0 {
		var profileData map[string]interface{}
		if err := contact.Profile.Unmarshal(&profileData); err != nil {
			return fmt.Errorf("failed to unmarshal contact profile: %w", err)
		}
		if err := s.fieldValidator.ValidateContactFields(ctx, profileData, organizationID); err != nil {
			return fmt.Errorf("field validation failed: %w", err)
		}
	}

	// Set organization ID if provided
	if organizationID != uuid.Nil {
		contact.OrganizationID = &organizationID
	}

	// Create the contact with ownership check (will merge if same owner, reject if different owner)
	_, err := s.contactRepo.CreateWithOwnershipCheck(ctx, contact)
	return err
}

// UpdateContact updates an existing contact with field validation
func (s *ContactService) UpdateContact(ctx context.Context, contact *domain.Contact, organizationID uuid.UUID) error {
	// Validate contact fields
	if len(contact.Profile) > 0 {
		var profileData map[string]interface{}
		if err := contact.Profile.Unmarshal(&profileData); err != nil {
			return fmt.Errorf("failed to unmarshal contact profile: %w", err)
		}
		if err := s.fieldValidator.ValidateContactFields(ctx, profileData, organizationID); err != nil {
			return fmt.Errorf("field validation failed: %w", err)
		}
	}

	// Update the contact
	return s.contactRepo.Update(ctx, contact.ID, contact)
}

// GetContact retrieves a contact by ID
func (s *ContactService) GetContact(ctx context.Context, id uuid.UUID) (*domain.Contact, error) {
	return s.contactRepo.FindByID(ctx, id)
}

// ListContacts retrieves contacts for a user with pagination
func (s *ContactService) ListContacts(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Contact, int, error) {
	return s.contactRepo.FindByUserIDWithPagination(ctx, userID, offset, limit)
}

// SearchContacts searches contacts by various criteria
func (s *ContactService) SearchContacts(ctx context.Context, userID uuid.UUID, query string, offset, limit int) ([]*domain.Contact, int, error) {
	return s.contactRepo.Search(ctx, userID, query, offset, limit)
}

// DeleteContact soft deletes a contact
func (s *ContactService) DeleteContact(ctx context.Context, id uuid.UUID) error {
	return s.contactRepo.SoftDelete(ctx, id)
}

// ImportContactsFromCSV imports contacts from a CSV file
func (s *ContactService) ImportContactsFromCSV(ctx context.Context, filePath string, userID uuid.UUID, requiredFields []string) (*ImportResult, error) {
	config := ImportConfig{
		FilePath:       filePath,
		UserID:         userID,
		RequiredFields: requiredFields,
	}

	return s.csvImporter.Import(ctx, config)
}

// ValidateCSVFile validates a CSV file structure
func (s *ContactService) ValidateCSVFile(filePath string, requiredFields []string) error {
	return s.csvImporter.ValidateCSVFile(filePath, requiredFields)
}

// GetContactFields returns standard and custom fields for contact forms
func (s *ContactService) GetContactFields(ctx context.Context, organizationID uuid.UUID) (standardFields []string, customFields []domain.CustomField, err error) {
	standardFields = s.fieldValidator.GetStandardFields()
	customFields, err = s.fieldValidator.GetCustomFields(ctx, organizationID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get custom fields: %w", err)
	}

	return standardFields, customFields, nil
}

// MapImportFields maps import field names to standard contact field names
func (s *ContactService) MapImportFields(importFields map[string]string) map[string]string {
	return s.fieldMapper.FlattenMapping(importFields)
}

// IsStandardField checks if a field is a standard contact field
func (s *ContactService) IsStandardField(fieldName string) bool {
	return s.fieldValidator.IsStandardField(fieldName)
}

// AddContactToList adds a contact to a contact list
func (s *ContactService) AddContactToList(ctx context.Context, contactID, listID uuid.UUID) error {
	association := &domain.ListContactAssociation{
		ContactID:     contactID,
		ListContactID: listID,
	}

	return s.listContactRepo.AddContactToList(ctx, association)
}

// RemoveContactFromList removes a contact from a contact list
func (s *ContactService) RemoveContactFromList(ctx context.Context, contactID, listID uuid.UUID) error {
	association := &domain.ListContactAssociation{
		ContactID:     contactID,
		ListContactID: listID,
	}

	return s.listContactRepo.RemoveContactFromList(ctx, association)
}

// GetContactLists retrieves all lists that a contact belongs to
func (s *ContactService) GetContactLists(ctx context.Context, contactID uuid.UUID) ([]*domain.ListContact, error) {
	return s.listContactRepo.FindByContactID(ctx, contactID)
}

// GetListContacts retrieves all contacts in a specific list
func (s *ContactService) GetListContacts(ctx context.Context, listID uuid.UUID, offset, limit int) ([]*domain.Contact, int, error) {
	return s.listContactRepo.FindContactsByListIDWithPagination(ctx, listID, offset, limit)
}
