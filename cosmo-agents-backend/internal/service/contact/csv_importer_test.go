package contact

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
)

func emailPhoneFromProfile(t *testing.T, contact *domain.Contact) (string, string) {
	t.Helper()
	var profile map[string]interface{}
	require.NoError(t, contact.Profile.Unmarshal(&profile))
	email, _ := profile["email"].(string)
	phone, _ := profile["phone"].(string)
	return email, phone
}

// MockContactRepositoryForCSV is a mock implementation for CSV importer testing
type MockContactRepositoryForCSV struct {
	mock.Mock
}

func (m *MockContactRepositoryForCSV) Create(ctx context.Context, contact *domain.Contact) error {
	args := m.Called(ctx, contact)
	return args.Error(0)
}

func (m *MockContactRepositoryForCSV) CreateWithOwnershipCheck(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	args := m.Called(ctx, contact)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Contact), args.Error(1)
}

func (m *MockContactRepositoryForCSV) Update(ctx context.Context, id uuid.UUID, contact *domain.Contact) error {
	args := m.Called(ctx, id, contact)
	return args.Error(0)
}

func (m *MockContactRepositoryForCSV) FindByID(ctx context.Context, id uuid.UUID) (*domain.Contact, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Contact), args.Error(1)
}

func (m *MockContactRepositoryForCSV) FindByUserIDWithPagination(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*domain.Contact, int, error) {
	args := m.Called(ctx, userID, offset, limit)
	return args.Get(0).([]*domain.Contact), args.Int(1), args.Error(2)
}

func (m *MockContactRepositoryForCSV) Search(ctx context.Context, userID uuid.UUID, query string, offset, limit int) ([]*domain.Contact, int, error) {
	args := m.Called(ctx, userID, query, offset, limit)
	return args.Get(0).([]*domain.Contact), args.Int(1), args.Error(2)
}

func (m *MockContactRepositoryForCSV) SoftDelete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockContactRepositoryForCSV) SearchWithFilter(ctx context.Context, userID uuid.UUID, orgIDs []uuid.UUID, filter map[string]interface{}, pagination *baseRepo.PaginationParams) ([]*domain.Contact, int, error) {
	args := m.Called(ctx, userID, orgIDs, filter, pagination)
	return args.Get(0).([]*domain.Contact), args.Int(1), args.Error(2)
}

func (m *MockContactRepositoryForCSV) UpsertMany(ctx context.Context, contacts []*domain.Contact) error {
	args := m.Called(ctx, contacts)
	return args.Error(0)
}

func (m *MockContactRepositoryForCSV) UpsertManyWithOwnershipCheck(ctx context.Context, contacts []*domain.Contact, userID uuid.UUID, organizationID uuid.UUID) (*contactRepo.OwnershipCheckResult, error) {
	args := m.Called(ctx, contacts, userID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*contactRepo.OwnershipCheckResult), args.Error(1)
}

func TestNewCSVImporter(t *testing.T) {
	mockRepo := &MockContactRepositoryForCSV{}
	importer := NewCSVImporter(mockRepo)

	assert.NotNil(t, importer)
	assert.Equal(t, mockRepo, importer.contactRepo)
}

func TestCSVImporter_ValidateCSVFile(t *testing.T) {
	mockRepo := &MockContactRepositoryForCSV{}
	importer := NewCSVImporter(mockRepo)

	// Create a temporary CSV file for testing
	tmpFile, err := os.CreateTemp("", "test_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write test CSV content
	csvContent := "first_name,last_name,email,phone\nJohn,Doe,john@example.com,1234567890\n"
	_, err = tmpFile.WriteString(csvContent)
	require.NoError(t, err)
	tmpFile.Close()

	t.Run("ValidCSVFile", func(t *testing.T) {
		requiredFields := []string{"first_name", "last_name", "email"}
		err := importer.ValidateCSVFile(tmpFile.Name(), requiredFields)
		assert.NoError(t, err)
	})

	t.Run("MissingRequiredField", func(t *testing.T) {
		requiredFields := []string{"first_name", "last_name", "email", "address"}
		err := importer.ValidateCSVFile(tmpFile.Name(), requiredFields)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required header 'address'")
	})
}

func TestCSVImporter_Import(t *testing.T) {

	// Create a temporary CSV file for testing
	tmpFile, err := os.CreateTemp("", "test_import_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write test CSV content
	csvContent := "first_name,last_name,email,phone\nJohn,Doe,john@example.com,1234567890\nJane,Smith,jane@example.com,9876543210\n"
	_, err = tmpFile.WriteString(csvContent)
	require.NoError(t, err)
	tmpFile.Close()

	userID := uuid.New()
	ctx := context.Background()
	config := ImportConfig{
		FilePath: tmpFile.Name(),
		UserID:   userID,
	}

	t.Run("SuccessfulImport", func(t *testing.T) {
		mockRepo := &MockContactRepositoryForCSV{}
		importer := NewCSVImporter(mockRepo)

		// Mock repository call with flexible matching
		mockRepo.On("UpsertMany", mock.Anything, mock.Anything).Return(nil)

		result, err := importer.Import(ctx, config)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.TotalRows)
		assert.Equal(t, 2, result.ImportedRows)
		assert.Equal(t, 0, result.SkippedRows)
		assert.Empty(t, result.ErrorMessages)

		mockRepo.AssertExpectations(t)
	})

	t.Run("RepositoryError", func(t *testing.T) {
		mockRepo := &MockContactRepositoryForCSV{}
		importer := NewCSVImporter(mockRepo)

		// Mock repository error
		mockRepo.On("UpsertMany", mock.Anything, mock.Anything).Return(assert.AnError)

		result, err := importer.Import(ctx, config)

		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, err.Error(), "failed to upsert contacts")

		mockRepo.AssertExpectations(t)
	})
}

func TestCSVImporter_ImportEmptyFile(t *testing.T) {
	mockRepo := &MockContactRepositoryForCSV{}
	importer := NewCSVImporter(mockRepo)

	// Create an empty temporary CSV file
	tmpFile, err := os.CreateTemp("", "test_empty_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	userID := uuid.New()
	ctx := context.Background()
	config := ImportConfig{
		FilePath: tmpFile.Name(),
		UserID:   userID,
	}

	_, err = importer.Import(ctx, config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CSV file is empty")
}

func TestCSVImporter_ImportOnlyHeaders(t *testing.T) {
	mockRepo := &MockContactRepositoryForCSV{}
	importer := NewCSVImporter(mockRepo)

	// Create a temporary CSV file with only headers
	tmpFile, err := os.CreateTemp("", "test_headers_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	csvContent := "first_name,last_name,email,phone\n"
	_, err = tmpFile.WriteString(csvContent)
	require.NoError(t, err)
	tmpFile.Close()

	userID := uuid.New()
	ctx := context.Background()
	config := ImportConfig{
		FilePath: tmpFile.Name(),
		UserID:   userID,
	}

	result, err := importer.Import(ctx, config)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.TotalRows)
	assert.Equal(t, 0, result.ImportedRows)
	assert.Equal(t, 0, result.SkippedRows)
}

func TestCSVImporter_ParseCSVRecords(t *testing.T) {
	mockRepo := &MockContactRepositoryForCSV{}
	importer := NewCSVImporter(mockRepo)

	// Create test CSV records
	records := [][]string{
		{"first_name", "last_name", "email", "phone"},
		{"John", "Doe", "john@example.com", "1234567890"},
		{"Jane", "Smith", "jane@example.com", "9876543210"},
		{"", "", "", ""}, // Empty row
	}

	headerMap := map[string]int{
		"first_name": 0,
		"last_name":  1,
		"email":      2,
		"phone":      3,
	}

	userID := uuid.New()

	contacts, skipped := importer.parseCSVRecords(records, headerMap, userID)

	assert.Len(t, contacts, 2)
	assert.Equal(t, 1, skipped) // One empty row

	// Check first contact
	assert.Equal(t, "John Doe", contacts[0].Name)
	email0, phone0 := emailPhoneFromProfile(t, contacts[0])
	assert.Equal(t, "john@example.com", email0)
	assert.Equal(t, "1234567890", phone0)
	assert.Equal(t, userID, contacts[0].UserID)
	assert.Equal(t, string(domain.ContactSourceCSV), contacts[0].Source)

	// Check second contact
	assert.Equal(t, "Jane Smith", contacts[1].Name)
	email1, phone1 := emailPhoneFromProfile(t, contacts[1])
	assert.Equal(t, "jane@example.com", email1)
	assert.Equal(t, "9876543210", phone1)
}

func TestCSVImporter_IsEmptyRow(t *testing.T) {
	mockRepo := &MockContactRepositoryForCSV{}
	importer := NewCSVImporter(mockRepo)

	testCases := []struct {
		name     string
		row      []string
		expected bool
	}{
		{"EmptySlice", []string{}, true},
		{"SingleEmptyString", []string{""}, true},
		{"SingleWhitespace", []string{"   "}, true},
		{"NonEmptyRow", []string{"John", "Doe"}, false},
		{"MixedEmpty", []string{"", "", "John"}, false},
		{"AllWhitespace", []string{"   ", "\t", "\n"}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := importer.isEmptyRow(tc.row)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestImportConfig(t *testing.T) {
	userID := uuid.New()
	config := ImportConfig{
		FilePath:       "test.csv",
		UserID:         userID,
		RequiredFields: []string{"first_name", "last_name", "email"},
	}

	assert.Equal(t, "test.csv", config.FilePath)
	assert.Equal(t, userID, config.UserID)
	assert.Equal(t, []string{"first_name", "last_name", "email"}, config.RequiredFields)
}

func TestImportResult(t *testing.T) {
	result := ImportResult{
		TotalRows:     10,
		ImportedRows:  8,
		SkippedRows:   2,
		ErrorMessages: []string{"Row 3: invalid email"},
	}

	assert.Equal(t, 10, result.TotalRows)
	assert.Equal(t, 8, result.ImportedRows)
	assert.Equal(t, 2, result.SkippedRows)
	assert.Len(t, result.ErrorMessages, 1)
	assert.Equal(t, "Row 3: invalid email", result.ErrorMessages[0])
}

func TestCSVImporter_DefaultRequiredFields(t *testing.T) {
	mockRepo := &MockContactRepositoryForCSV{}
	importer := NewCSVImporter(mockRepo)

	// Create a temporary CSV file
	tmpFile, err := os.CreateTemp("", "test_defaults_*.csv")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write CSV with only default required fields
	csvContent := "first_name,last_name,email\nJohn,Doe,john@example.com\n"
	_, err = tmpFile.WriteString(csvContent)
	require.NoError(t, err)
	tmpFile.Close()

	userID := uuid.New()
	ctx := context.Background()

	// Test with empty required fields (should use defaults)
	config := ImportConfig{
		FilePath:       tmpFile.Name(),
		UserID:         userID,
		RequiredFields: []string{}, // Empty to test defaults
	}

	mockRepo.On("UpsertMany", mock.Anything, mock.Anything).Return(nil)

	result, err := importer.Import(ctx, config)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.ImportedRows)

	mockRepo.AssertExpectations(t)
}

// importCSVIntoPG runs a real import against Postgres, with the unique index
// production carries for the upsert, and returns the stored rows by name.
func importCSVIntoPG(t *testing.T, content string) (*ImportResult, map[string]domain.Contact) {
	t.Helper()
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Contact{}))
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_contacts_source_id_source_user_id ON contacts(source_id, source, user_id)`).Error)

	f, err := os.CreateTemp(t.TempDir(), "import_*.csv")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	orgID := uuid.New()
	result, err := NewCSVImporter(contactRepo.NewContactRepository(db)).Import(context.Background(), ImportConfig{
		FilePath:       f.Name(),
		UserID:         uuid.New(),
		OrganizationID: &orgID,
		RequiredFields: []string{"email"},
	})
	require.NoError(t, err)

	var rows []domain.Contact
	require.NoError(t, db.Find(&rows).Error)
	byName := make(map[string]domain.Contact, len(rows))
	for _, r := range rows {
		byName[r.Name] = r
	}
	return result, byName
}

func TestCSVImporter_Import_PG_RealWorldFiles(t *testing.T) {
	t.Run("BOM and name column", func(t *testing.T) {
		// Excel "CSV UTF-8" output: BOM, then a name column first.
		_, rows := importCSVIntoPG(t, "\ufeffname,email,company,job_title\nNguyễn Văn An,an@example.vn,Công ty Việt,CTO\n")
		require.Len(t, rows, 1)
		got, ok := rows["Nguyễn Văn An"]
		require.True(t, ok, "name column must fill the name, got %v", rows)
		assert.Equal(t, "Công ty Việt", got.Company)
		assert.NotContains(t, string(got.Profile), "name", "name must not be filed as a profile field")
		// Complete rows are ready to send and carry their dedupe key.
		assert.Equal(t, "an@example.vn", got.ContactInformation)
		assert.Equal(t, "ready", got.Status)
	})

	t.Run("repeated email in one file", func(t *testing.T) {
		result, rows := importCSVIntoPG(t, "name,email,company\nFirst,dup@example.com,A\nSecond,DUP@example.com,B\nOther,other@example.com,C\n")
		require.Len(t, rows, 2, "one contact per email, whatever its case")
		assert.Equal(t, "B", rows["Second"].Company, "the later row wins")
		assert.Equal(t, 2, result.ImportedRows)
		assert.Equal(t, 1, result.SkippedRows)
	})

	t.Run("rows without email are kept apart", func(t *testing.T) {
		_, rows := importCSVIntoPG(t, "name,email\nNo Mail One,\nNo Mail Two,\n")
		assert.Len(t, rows, 2)
	})
}
