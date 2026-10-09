package salerep

import (
	"context"
	"fmt"
	"github.com/lib/pq"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	base "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the SaleRep schema
	err = db.AutoMigrate(&domain.User{}, &domain.Organization{}, &domain.SaleRep{})
	require.NoError(t, err)

	return db
}

// seedOwner inserts the user and organization a sale rep belongs to. sale_reps
// carries foreign keys to both (migration 000005). SQLite did not enforce them,
// so these tests used to insert reps that pointed at rows that did not exist -
// a state production's schema rejects.
func seedOwner(t *testing.T, db *gorm.DB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	user := domain.User{Email: uuid.NewString() + "@example.com", Name: "owner", PhoneNumber: pq.StringArray{"123"}}
	require.NoError(t, db.Create(&user).Error)
	org := domain.Organization{
		UserID:                  &user.ID,
		Name:                    "org",
		CompanyDescription:      "desc",
		ValueOffering:           "value",
		CompanyTargetingPersona: pq.StringArray{"buyer"},
	}
	require.NoError(t, db.Create(&org).Error)
	return user.ID, org.ID
}

// TestSaleRepRepository_Create tests creating a new sale rep
func TestSaleRepRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	saleRepRepo := NewSaleRepRepository(db)
	ctx := context.Background()

	// Test data
	userID, orgID := seedOwner(t, db)
	saleRep := &domain.SaleRep{
		FirstName:      "John",
		LastName:       "Doe",
		Email:          "john.doe@example.com",
		CalendarLink:   "https://calendar.example.com/john",
		Picture:        "https://example.com/avatars/john.jpg",
		UserID:         userID,
		OrganizationID: orgID,
	}

	// Test Create
	result, err := saleRepRepo.Create(ctx, saleRep)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.Equal(t, "John", result.FirstName)
	assert.Equal(t, "Doe", result.LastName)
	assert.Equal(t, "john.doe@example.com", result.Email)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, orgID, result.OrganizationID)
}

// TestSaleRepRepository_GetByUserIDAndEmail tests finding a sale rep by user ID and email
func TestSaleRepRepository_GetByUserIDAndEmail(t *testing.T) {
	db := setupTestDB(t)
	saleRepRepo := NewSaleRepRepository(db)
	ctx := context.Background()

	// Create test sale rep
	userID, orgID := seedOwner(t, db)
	saleRep := &domain.SaleRep{
		FirstName:      "Jane",
		LastName:       "Smith",
		Email:          "jane.smith@example.com",
		UserID:         userID,
		OrganizationID: orgID,
	}
	created, err := saleRepRepo.Create(ctx, saleRep)
	require.NoError(t, err)

	// Test GetByUserIDAndEmail
	found, err := saleRepRepo.GetByUserIDAndEmail(ctx, userID, "jane.smith@example.com")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Jane", found.FirstName)
	assert.Equal(t, "jane.smith@example.com", found.Email)

	// Test with non-existent user/email
	notFound, err := saleRepRepo.GetByUserIDAndEmail(ctx, uuid.New(), "nonexistent@example.com")
	assert.Error(t, err)
	assert.Nil(t, notFound)
}

// TestSaleRepRepository_GetByIDs tests finding multiple sale reps by IDs
func TestSaleRepRepository_GetByIDs(t *testing.T) {
	db := setupTestDB(t)
	saleRepRepo := NewSaleRepRepository(db)
	ctx := context.Background()

	// Create test sale reps
	userID, orgID := seedOwner(t, db)
	var saleRepIDs []string

	for i := 0; i < 3; i++ {
		saleRep := &domain.SaleRep{
			FirstName:      fmt.Sprintf("SaleRep%d", i),
			LastName:       "Test",
			Email:          fmt.Sprintf("salerep%d@example.com", i),
			UserID:         userID,
			OrganizationID: orgID,
		}
		created, err := saleRepRepo.Create(ctx, saleRep)
		require.NoError(t, err)
		saleRepIDs = append(saleRepIDs, created.ID.String())
	}

	// Test GetByIDs
	saleReps, err := saleRepRepo.GetByIDs(ctx, saleRepIDs)
	require.NoError(t, err)
	assert.Len(t, saleReps, 3)

	// Verify all sale reps are found
	foundIDs := make(map[string]bool)
	for _, sr := range saleReps {
		foundIDs[sr.ID.String()] = true
	}

	for _, id := range saleRepIDs {
		assert.True(t, foundIDs[id])
	}

	// Test with empty IDs
	emptyReps, err := saleRepRepo.GetByIDs(ctx, []string{})
	require.NoError(t, err)
	assert.Len(t, emptyReps, 0)
}

// TestSaleRepRepository_GetByEmail tests finding a sale rep by email
func TestSaleRepRepository_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	saleRepRepo := NewSaleRepRepository(db)
	ctx := context.Background()

	// Create test sale rep
	userID, orgID := seedOwner(t, db)
	saleRep := &domain.SaleRep{
		FirstName:      "Alice",
		LastName:       "Johnson",
		Email:          "alice.johnson@example.com",
		UserID:         userID,
		OrganizationID: orgID,
	}
	created, err := saleRepRepo.Create(ctx, saleRep)
	require.NoError(t, err)

	// Test GetByEmail
	found, err := saleRepRepo.GetByEmail(ctx, "alice.johnson@example.com")
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Alice", found.FirstName)
	assert.Equal(t, "alice.johnson@example.com", found.Email)

	// Test with non-existent email
	notFound, err := saleRepRepo.GetByEmail(ctx, "nonexistent@example.com")
	require.NoError(t, err)
	assert.Nil(t, notFound)
}

// TestSaleRepRepository_GetByUserID tests finding sale reps by user ID with pagination
func TestSaleRepRepository_GetByUserID(t *testing.T) {
	db := setupTestDB(t)
	saleRepRepo := NewSaleRepRepository(db)
	ctx := context.Background()

	// Create test sale reps
	userID, orgID := seedOwner(t, db)

	for i := 0; i < 5; i++ {
		saleRep := &domain.SaleRep{
			FirstName:      fmt.Sprintf("SaleRep%d", i),
			LastName:       "Test",
			Email:          fmt.Sprintf("salerep%d@example.com", i),
			UserID:         userID,
			OrganizationID: orgID,
		}
		_, err := saleRepRepo.Create(ctx, saleRep)
		require.NoError(t, err)
	}

	// Test GetByUserID with pagination
	saleReps, total, err := saleRepRepo.GetByUserID(ctx, userID, nil, nil)
	require.NoError(t, err)
	assert.Len(t, saleReps, 5)
	assert.Equal(t, int64(5), total)

	// Verify all sale reps belong to the user
	for _, sr := range saleReps {
		assert.Equal(t, userID, sr.UserID)
	}

	// Test with pagination limit
	pagination := &base.PaginationParams{
		Offset: 0,
		Limit:  3,
	}

	saleReps, total, err = saleRepRepo.GetByUserID(ctx, userID, nil, pagination)
	require.NoError(t, err)
	assert.Len(t, saleReps, 3)
	assert.Equal(t, int64(5), total)
}

// TestSaleRepRepository_Update tests updating a sale rep
func TestSaleRepRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	saleRepRepo := NewSaleRepRepository(db)
	ctx := context.Background()

	// Create test sale rep
	userID, orgID := seedOwner(t, db)
	saleRep := &domain.SaleRep{
		FirstName:      "Bob",
		LastName:       "Wilson",
		Email:          "bob.wilson@example.com",
		CalendarLink:   "https://calendar.example.com/bob",
		UserID:         userID,
		OrganizationID: orgID,
	}
	created, err := saleRepRepo.Create(ctx, saleRep)
	require.NoError(t, err)

	// Update sale rep
	created.FirstName = "Robert"
	created.LastName = "Williams"
	created.CalendarLink = "https://calendar.example.com/robert"

	err = saleRepRepo.Update(ctx, created.ID, created)
	require.NoError(t, err)

	// Verify update
	found, err := saleRepRepo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Robert", found.FirstName)
	assert.Equal(t, "Williams", found.LastName)
	assert.Equal(t, "https://calendar.example.com/robert", found.CalendarLink)
}

// TestSaleRepRepository_Delete tests deleting a sale rep (soft delete)
func TestSaleRepRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	saleRepRepo := NewSaleRepRepository(db)
	ctx := context.Background()

	// Create test sale rep
	userID, orgID := seedOwner(t, db)
	saleRep := &domain.SaleRep{
		FirstName:      "Charlie",
		LastName:       "Brown",
		Email:          "charlie.brown@example.com",
		UserID:         userID,
		OrganizationID: orgID,
	}
	created, err := saleRepRepo.Create(ctx, saleRep)
	require.NoError(t, err)

	// Delete sale rep
	err = saleRepRepo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// Verify soft deletion - should not be found with normal FindByID
	found, err := saleRepRepo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, found) // Soft deleted records should not be found
}

// TestSaleRepRepository_ComplexSaleRep tests creating a complex sale rep
func TestSaleRepRepository_ComplexSaleRep(t *testing.T) {
	db := setupTestDB(t)
	saleRepRepo := NewSaleRepRepository(db)
	ctx := context.Background()

	// Create test data
	userID, orgID := seedOwner(t, db)

	// Test complex sale rep
	saleRep := &domain.SaleRep{
		FirstName:      "Michael",
		LastName:       "Anderson",
		Email:          "michael.anderson@company.com",
		CalendarLink:   "https://calendar.google.com/company/michael.anderson",
		Picture:        "https://company.com/avatars/michael-anderson.jpg",
		UserID:         userID,
		OrganizationID: orgID,
	}

	// Create
	result, err := saleRepRepo.Create(ctx, saleRep)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)

	// Verify all fields were set correctly
	assert.Equal(t, "Michael", result.FirstName)
	assert.Equal(t, "Anderson", result.LastName)
	assert.Equal(t, "michael.anderson@company.com", result.Email)
	assert.Equal(t, "https://calendar.google.com/company/michael.anderson", result.CalendarLink)
	assert.Equal(t, "https://company.com/avatars/michael-anderson.jpg", result.Picture)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, orgID, result.OrganizationID)

	// Test email template context functionality
	context := result.ToEmailTemplateContext()
	assert.NotNil(t, context)
	assert.Equal(t, "Michael", context["sale_rep_first_name"])
	assert.Equal(t, "Anderson", context["sale_rep_last_name"])
	assert.Equal(t, "michael.anderson@company.com", context["sale_rep_email"])
	assert.Equal(t, "https://calendar.google.com/company/michael.anderson", context["sale_rep_calendar_link"])
}
