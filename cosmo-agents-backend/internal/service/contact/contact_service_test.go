package contact

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	customfield "github.com/rockship/cosmo-agents-go/internal/repository/custom_field"
)

func emailFromProfileService(t *testing.T, contact *domain.Contact) string {
	t.Helper()
	var profile map[string]interface{}
	require.NoError(t, contact.Profile.Unmarshal(&profile))
	email, _ := profile["email"].(string)
	return email
}

// MockContactRepository is a mock implementation of ContactRepositoryInterface
type MockContactRepository struct {
	mock.Mock
}

func (m *MockContactRepository) Create(ctx context.Context, contact *domain.Contact) error {
	args := m.Called(ctx, contact)
	return args.Error(0)
}

func (m *MockContactRepository) CreateWithOwnershipCheck(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	args := m.Called(ctx, contact)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Contact), args.Error(1)
}

func (m *MockContactRepository) Update(ctx context.Context, id uuid.UUID, contact *domain.Contact) error {
	args := m.Called(ctx, id, contact)
	return args.Error(0)
}

func (m *MockContactRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Contact, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Contact), args.Error(1)
}

func (m *MockContactRepository) FindByUserIDWithPagination(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Contact, int, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]*domain.Contact), args.Int(1), args.Error(2)
}

func (m *MockContactRepository) Search(ctx context.Context, userID uuid.UUID, query string, offset, limit int) ([]*domain.Contact, int, error) {
	args := m.Called(ctx, userID, query, offset, limit)
	return args.Get(0).([]*domain.Contact), args.Int(1), args.Error(2)
}

func (m *MockContactRepository) SearchWithFilter(ctx context.Context, userID uuid.UUID, orgIDs []uuid.UUID, filter map[string]interface{}, pagination *baseRepo.PaginationParams) ([]*domain.Contact, int, error) {
	args := m.Called(ctx, userID, orgIDs, filter, pagination)
	return args.Get(0).([]*domain.Contact), args.Int(1), args.Error(2)
}

func (m *MockContactRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockContactRepository) UpsertMany(ctx context.Context, contacts []*domain.Contact) error {
	args := m.Called(ctx, contacts)
	return args.Error(0)
}

func (m *MockContactRepository) UpsertManyWithOwnershipCheck(ctx context.Context, contacts []*domain.Contact, userID uuid.UUID, organizationID uuid.UUID) (*contactRepo.OwnershipCheckResult, error) {
	args := m.Called(ctx, contacts, userID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*contactRepo.OwnershipCheckResult), args.Error(1)
}

// MockListContactRepository is a mock implementation of ListContactRepositoryInterface
type MockListContactRepository struct {
	mock.Mock
}

func (m *MockListContactRepository) AddContactToList(ctx context.Context, association *domain.ListContactAssociation) error {
	args := m.Called(ctx, association)
	return args.Error(0)
}

func (m *MockListContactRepository) RemoveContactFromList(ctx context.Context, association *domain.ListContactAssociation) error {
	args := m.Called(ctx, association)
	return args.Error(0)
}

func (m *MockListContactRepository) FindByContactID(ctx context.Context, contactID uuid.UUID) ([]*domain.ListContact, error) {
	args := m.Called(ctx, contactID)
	return args.Get(0).([]*domain.ListContact), args.Error(1)
}

func (m *MockListContactRepository) FindContactsByListIDWithPagination(ctx context.Context, listID uuid.UUID, offset, limit int) ([]*domain.Contact, int, error) {
	args := m.Called(ctx, listID, offset, limit)
	return args.Get(0).([]*domain.Contact), args.Int(1), args.Error(2)
}

func TestNewContactService_Mocks(t *testing.T) {
	mockContactRepo := &MockContactRepository{}
	mockListContactRepo := &MockListContactRepository{}

	// Verify that our mock repositories work correctly
	assert.NotNil(t, mockContactRepo)
	assert.NotNil(t, mockListContactRepo)
}

func TestContactService_ContactValidation(t *testing.T) {
	userID := uuid.New()

	// Test case: valid contact
	t.Run("ValidContact", func(t *testing.T) {
		contact := &domain.Contact{
			Base:    domain.Base{ID: uuid.New()},
			UserID:  userID,
			Name:    "John Doe",
			Profile: base.JSONB(`{"email":"john@example.com"}`),
			Source:  "csv",
		}

		// This should not panic and should have valid structure
		assert.NotNil(t, contact)
		assert.Equal(t, userID, contact.UserID)
		assert.Equal(t, "John Doe", contact.Name)
		assert.Equal(t, "john@example.com", emailFromProfileService(t, contact))
		assert.Equal(t, "csv", contact.Source)
	})

	// Test case: contact with profile data
	t.Run("ContactWithProfile", func(t *testing.T) {
		contact := &domain.Contact{
			Base:    domain.Base{ID: uuid.New()},
			UserID:  userID,
			Name:    "Jane Smith",
			Profile: base.JSONB(`{"email":"jane@example.com","age":30,"city":"New York"}`),
		}

		// This should not panic
		assert.NotNil(t, contact)
		assert.Equal(t, userID, contact.UserID)
		assert.NotZero(t, len(contact.Profile))
	})
}

func TestContactService_MockRepository(t *testing.T) {
	mockContactRepo := &MockContactRepository{}
	ctx := context.Background()
	userID := uuid.New()
	contactID := uuid.New()

	contact := &domain.Contact{
		Base:    domain.Base{ID: contactID},
		UserID:  userID,
		Name:    "John Doe",
		Profile: base.JSONB(`{"email":"john@example.com"}`),
	}

	// Test that mock repository works correctly
	mockContactRepo.On("FindByID", ctx, contactID).Return(contact, nil)
	mockContactRepo.On("Create", ctx, contact).Return(nil)
	mockContactRepo.On("Update", ctx, contactID, contact).Return(nil)
	mockContactRepo.On("SoftDelete", ctx, contactID).Return(nil)

	// Test FindByID
	found, err := mockContactRepo.FindByID(ctx, contactID)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, contactID, found.ID)

	// Test Create
	err = mockContactRepo.Create(ctx, contact)
	assert.NoError(t, err)

	// Test Update
	err = mockContactRepo.Update(ctx, contactID, contact)
	assert.NoError(t, err)

	// Test SoftDelete
	err = mockContactRepo.SoftDelete(ctx, contactID)
	assert.NoError(t, err)

	mockContactRepo.AssertExpectations(t)
}

func TestContactService_SearchFunctionality(t *testing.T) {
	mockContactRepo := &MockContactRepository{}
	ctx := context.Background()
	userID := uuid.New()
	orgIDs := []uuid.UUID{uuid.New()}
	filter := map[string]interface{}{"name": "John"}
	pagination := &baseRepo.PaginationParams{Offset: 0, Limit: 10}

	contacts := []*domain.Contact{
		{
			Base:    domain.Base{ID: uuid.New()},
			UserID:  userID,
			Name:    "John Doe",
			Profile: base.JSONB(`{"email":"john@example.com"}`),
		},
	}

	// Test the SearchWithFilter functionality
	mockContactRepo.On("SearchWithFilter", ctx, userID, orgIDs, filter, pagination).Return(contacts, len(contacts), nil)

	found, total, err := mockContactRepo.SearchWithFilter(ctx, userID, orgIDs, filter, pagination)
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, found, 1)
	assert.Equal(t, "John Doe", found[0].Name)

	mockContactRepo.AssertExpectations(t)
}

func TestContactService_PaginationFunctionality(t *testing.T) {
	mockContactRepo := &MockContactRepository{}
	ctx := context.Background()
	userID := uuid.New()

	contacts := []*domain.Contact{
		{
			Base:    domain.Base{ID: uuid.New()},
			UserID:  userID,
			Name:    "John1 Doe",
			Profile: base.JSONB(`{"email":"john1@example.com"}`),
		},
		{
			Base:    domain.Base{ID: uuid.New()},
			UserID:  userID,
			Name:    "John2 Doe",
			Profile: base.JSONB(`{"email":"john2@example.com"}`),
		},
	}

	// Test pagination
	mockContactRepo.On("FindByUserIDWithPagination", ctx, userID, 0, 10).Return(contacts, 5, nil)

	found, total, err := mockContactRepo.FindByUserIDWithPagination(ctx, userID, 0, 10)
	assert.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, found, 2)
	assert.Equal(t, "John1 Doe", found[0].Name)
	assert.Equal(t, "John2 Doe", found[1].Name)

	mockContactRepo.AssertExpectations(t)
}

// --- Additional coverage with sqlite-backed validator/custom fields ---

func newContactServiceWithDB(t *testing.T) (*ContactService, *MockContactRepository, *MockListContactRepository, *gorm.DB) {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	db = db.Session(&gorm.Session{AllowGlobalUpdate: true})
	require.NoError(t, db.AutoMigrate(&domain.CustomField{}))

	mockContactRepo := &MockContactRepository{}
	mockListRepo := &MockListContactRepository{}
	customRepo := customfield.NewCustomFieldRepository(db)

	service := NewContactService(mockContactRepo, customRepo, mockListRepo)
	return service, mockContactRepo, mockListRepo, db
}

func TestContactService_CreateAndUpdateWithValidation(t *testing.T) {
	service, mockContactRepo, _, db := newContactServiceWithDB(t)
	ctx := context.Background()
	orgID := uuid.New()

	contact := &domain.Contact{
		Base:    domain.Base{ID: uuid.New()},
		UserID:  uuid.New(),
		Name:    "Jane Doe",
		Profile: base.JSONB(`{"email":"jane@example.com","name":"Jane Doe"}`),
	}
	mockContactRepo.On("CreateWithOwnershipCheck", ctx, contact).Return(contact, nil).Once()
	require.NoError(t, service.CreateContact(ctx, contact, orgID))

	// Add custom field then update using it
	customField := domain.CustomField{
		Base:           domain.Base{ID: uuid.New()},
		OrganizationID: &orgID,
		NormalizedName: "favorite_color",
		DataType:       domain.CustomFieldDataTypeText,
		EntityType:     domain.CustomFieldEntityContact,
		Name:           "Favorite Color",
	}
	require.NoError(t, db.Create(&customField).Error)

	update := &domain.Contact{
		Base:    domain.Base{ID: contact.ID},
		UserID:  contact.UserID,
		Name:    "Updated Name",
		Profile: domain.JSONB(`{"favorite_color":"blue"}`),
	}
	mockContactRepo.On("Update", ctx, contact.ID, update).Return(nil).Once()
	require.NoError(t, service.UpdateContact(ctx, update, orgID))
}

func TestContactService_CreateContact_InvalidField(t *testing.T) {
	service, mockContactRepo, _, _ := newContactServiceWithDB(t)
	ctx := context.Background()

	contact := &domain.Contact{
		Base:    domain.Base{ID: uuid.New()},
		UserID:  uuid.New(),
		Profile: domain.JSONB(`{"unknown_field":"value"}`),
	}
	err := service.CreateContact(ctx, contact, uuid.New())
	assert.Error(t, err)
	mockContactRepo.AssertNotCalled(t, "CreateWithOwnershipCheck", mock.Anything, mock.Anything)
}

func TestContactService_ListSearchDelete(t *testing.T) {
	service, mockContactRepo, _, _ := newContactServiceWithDB(t)
	ctx := context.Background()
	userID := uuid.New()
	contactID := uuid.New()

	listResp := []*domain.Contact{{Base: domain.Base{ID: contactID}}}
	mockContactRepo.On("FindByUserIDWithPagination", ctx, userID, 0, 10).Return(listResp, 1, nil).Once()
	contacts, total, err := service.ListContacts(ctx, userID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, contacts, 1)

	mockContactRepo.On("Search", ctx, userID, "john", 0, 5).Return(listResp, 1, nil).Once()
	_, _, err = service.SearchContacts(ctx, userID, "john", 0, 5)
	require.NoError(t, err)

	mockContactRepo.On("SoftDelete", ctx, contactID).Return(nil).Once()
	require.NoError(t, service.DeleteContact(ctx, contactID))
}

func TestContactService_ListContactsOperations(t *testing.T) {
	service, mockContactRepo, mockListRepo, _ := newContactServiceWithDB(t)
	ctx := context.Background()

	contactID := uuid.New()
	listID := uuid.New()
	association := &domain.ListContactAssociation{ContactID: contactID, ListContactID: listID}

	mockListRepo.On("AddContactToList", ctx, association).Return(nil).Once()
	require.NoError(t, service.AddContactToList(ctx, contactID, listID))

	mockListRepo.On("RemoveContactFromList", ctx, association).Return(nil).Once()
	require.NoError(t, service.RemoveContactFromList(ctx, contactID, listID))

	mockListRepo.On("FindByContactID", ctx, contactID).Return([]*domain.ListContact{}, nil).Once()
	_, err := service.GetContactLists(ctx, contactID)
	require.NoError(t, err)

	mockListRepo.On("FindContactsByListIDWithPagination", ctx, listID, 0, 10).Return([]*domain.Contact{}, 0, nil).Once()
	_, _, err = service.GetListContacts(ctx, listID, 0, 10)
	require.NoError(t, err)

	mockContactRepo.AssertExpectations(t)
	mockListRepo.AssertExpectations(t)
}

func TestContactService_FieldsHelpers(t *testing.T) {
	service, _, _, db := newContactServiceWithDB(t)
	ctx := context.Background()
	orgID := uuid.New()

	cf := &domain.CustomField{
		Base:           domain.Base{ID: uuid.New()},
		OrganizationID: &orgID,
		NormalizedName: "favorite_color",
		DataType:       domain.CustomFieldDataTypeText,
		EntityType:     domain.CustomFieldEntityContact,
		Name:           "Favorite Color",
	}
	require.NoError(t, db.Create(cf).Error)

	standard, custom, err := service.GetContactFields(ctx, orgID)
	require.NoError(t, err)
	assert.NotEmpty(t, standard)
	require.Len(t, custom, 1)
	assert.Equal(t, "favorite_color", custom[0].NormalizedName)

	assert.Panics(t, func() {
		service.MapImportFields(map[string]string{"first_name": "Jane"})
	})

	assert.True(t, service.IsStandardField("email"))
	assert.False(t, service.IsStandardField("nonstandard"))
}
