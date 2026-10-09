package operation

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/testutil/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTestDB creates a test database
// JSONB columns are compared with JSONEq: PostgreSQL normalises jsonb on write
// (whitespace, key order), so a byte-for-byte comparison tests formatting, not data.
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := pgtest.Open(t, &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the Operation schema
	err = db.AutoMigrate(&domain.Operation{})
	require.NoError(t, err)

	return db
}

// TestOperationRepository_Create tests creating a new operation
func TestOperationRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOperationRepository(db)
	ctx := context.Background()

	// Test data
	operation := &domain.Operation{
		Name:   "Test Operation",
		Status: domain.OperationStatusInProgress,
		Input:  base.JSONB([]byte(`{"param1": "value1"}`)),
		Output: base.JSONB([]byte(`{}`)),
	}

	// Test Create
	result, err := repo.Create(ctx, operation)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.Equal(t, "Test Operation", result.Name)
	assert.Equal(t, domain.OperationStatusInProgress, result.Status)
	assert.JSONEq(t, `{"param1": "value1"}`, string(result.Input))
}

// TestOperationRepository_FindByID tests finding an operation by ID
func TestOperationRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOperationRepository(db)
	ctx := context.Background()

	// Create test operation
	operation := &domain.Operation{
		Name:   "Find Test Operation",
		Status: domain.OperationStatusInProgress,
		Input:  base.JSONB([]byte(`{"test": "data"}`)),
	}
	created, err := repo.Create(ctx, operation)
	require.NoError(t, err)

	// Test FindByID
	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Find Test Operation", found.Name)
	assert.Equal(t, domain.OperationStatusInProgress, found.Status)
}

// TestOperationRepository_FindByStatus tests finding operations by status
func TestOperationRepository_FindByStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOperationRepository(db)
	ctx := context.Background()

	// Create test operations with different statuses
	inProgressOperation1 := &domain.Operation{
		Name:   "In Progress 1",
		Status: domain.OperationStatusInProgress,
		Input:  base.JSONB([]byte(`{}`)),
	}
	created1, err := repo.Create(ctx, inProgressOperation1)
	require.NoError(t, err)

	inProgressOperation2 := &domain.Operation{
		Name:   "In Progress 2",
		Status: domain.OperationStatusInProgress,
		Input:  base.JSONB([]byte(`{}`)),
	}
	created2, err := repo.Create(ctx, inProgressOperation2)
	require.NoError(t, err)

	successOperation := &domain.Operation{
		Name:   "Success Operation",
		Status: domain.OperationStatusSuccess,
		Input:  base.JSONB([]byte(`{}`)),
	}
	_, err = repo.Create(ctx, successOperation)
	require.NoError(t, err)

	// Test FindByStatus for in_progress operations
	operations, total, err := repo.FindByStatus(ctx, domain.OperationStatusInProgress, 0, 10)
	require.NoError(t, err)
	assert.Len(t, operations, 2)
	assert.Equal(t, int64(2), total)

	// Verify the operations are correct (ordered by created_at DESC)
	operationIDs := []uuid.UUID{operations[0].ID, operations[1].ID}
	assert.Contains(t, operationIDs, created1.ID)
	assert.Contains(t, operationIDs, created2.ID)

	// Test FindByStatus for success operations
	operations, total, err = repo.FindByStatus(ctx, domain.OperationStatusSuccess, 0, 10)
	require.NoError(t, err)
	assert.Len(t, operations, 1)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "Success Operation", operations[0].Name)
}

// TestOperationRepository_FindByStatus_Pagination tests pagination functionality
func TestOperationRepository_FindByStatus_Pagination(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOperationRepository(db)
	ctx := context.Background()

	// Create test operations
	for i := 0; i < 5; i++ {
		operation := &domain.Operation{
			Name:   "Test Operation",
			Status: domain.OperationStatusInProgress,
			Input:  base.JSONB([]byte(`{}`)),
		}
		_, err := repo.Create(ctx, operation)
		require.NoError(t, err)
	}

	// Test pagination
	operations, total, err := repo.FindByStatus(ctx, domain.OperationStatusInProgress, 0, 2)
	require.NoError(t, err)
	assert.Len(t, operations, 2)
	assert.Equal(t, int64(5), total)

	// Second page
	operations, total, err = repo.FindByStatus(ctx, domain.OperationStatusInProgress, 2, 2)
	require.NoError(t, err)
	assert.Len(t, operations, 2)
	assert.Equal(t, int64(5), total)

	// Last page
	operations, total, err = repo.FindByStatus(ctx, domain.OperationStatusInProgress, 4, 2)
	require.NoError(t, err)
	assert.Len(t, operations, 1)
	assert.Equal(t, int64(5), total)
}

// TestOperationRepository_UpdateStatus tests updating operation status
func TestOperationRepository_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOperationRepository(db)
	ctx := context.Background()

	// Create test operation
	operation := &domain.Operation{
		Name:   "Status Update Test",
		Status: domain.OperationStatusInProgress,
		Input:  base.JSONB([]byte(`{}`)),
	}
	created, err := repo.Create(ctx, operation)
	require.NoError(t, err)

	// Test UpdateStatus
	err = repo.UpdateStatus(ctx, created.ID, domain.OperationStatusSuccess)
	require.NoError(t, err)

	// Verify the update
	updated, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.OperationStatusSuccess, updated.Status)
}

// TestOperationRepository_UpdateOutput tests updating operation output
func TestOperationRepository_UpdateOutput(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOperationRepository(db)
	ctx := context.Background()

	// Create test operation
	operation := &domain.Operation{
		Name:   "Output Update Test",
		Status: domain.OperationStatusInProgress,
		Input:  base.JSONB([]byte(`{"input": "data"}`)),
		Output: base.JSONB([]byte(`{}`)),
	}
	created, err := repo.Create(ctx, operation)
	require.NoError(t, err)

	// Test UpdateOutput
	newOutput := base.JSONB([]byte(`{"result": "success", "data": [1, 2, 3]}`))
	err = repo.UpdateOutput(ctx, created.ID, newOutput)
	require.NoError(t, err)

	// Verify the update
	updated, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.JSONEq(t, string(newOutput), string(updated.Output))
}

// TestOperationRepository_UpdateFailedStatusWithOutput tests updating operation to failed status with output
func TestOperationRepository_UpdateFailedStatusWithOutput(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOperationRepository(db)
	ctx := context.Background()

	// Create test operation
	operation := &domain.Operation{
		Name:   "Failed Update Test",
		Status: domain.OperationStatusInProgress,
		Input:  base.JSONB([]byte(`{}`)),
	}
	created, err := repo.Create(ctx, operation)
	require.NoError(t, err)

	// Test UpdateFailedStatusWithOutput
	errorOutput := base.JSONB([]byte(`{"error": "Something went wrong", "code": 500}`))
	err = repo.UpdateFailedStatusWithOutput(ctx, created.ID, errorOutput)
	require.NoError(t, err)

	// Verify the update
	updated, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.OperationStatusFailed, updated.Status)
	assert.JSONEq(t, string(errorOutput), string(updated.Output))
}

// TestOperationRepository_Delete tests deleting an operation (soft delete)
func TestOperationRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOperationRepository(db)
	ctx := context.Background()

	// Create test operation
	operation := &domain.Operation{
		Name:   "Delete Test Operation",
		Status: domain.OperationStatusInProgress,
		Input:  base.JSONB([]byte(`{}`)),
	}
	created, err := repo.Create(ctx, operation)
	require.NoError(t, err)

	// Delete operation
	err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// Verify soft deletion - should not be found with normal FindByID
	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, found) // Soft deleted records should not be found
}

// TestOperationRepository_ComplexOperation tests a complex operation scenario
func TestOperationRepository_ComplexOperation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOperationRepository(db)
	ctx := context.Background()

	// Create a complex operation
	inputData := base.JSONB([]byte(`
		{
			"campaign_id": "123e4567-e89b-12d3-a456-426614174000",
			"contact_count": 1000,
			"template": "welcome_email",
			"settings": {
				"batch_size": 50,
				"delay": 5
			}
		}
	`))

	operation := &domain.Operation{
		Name:   "Campaign Email Batch Send",
		Status: domain.OperationStatusInProgress,
		Input:  inputData,
	}
	created, err := repo.Create(ctx, operation)
	require.NoError(t, err)

	// Simulate operation progress
	// Update status to success with output
	successOutput := base.JSONB([]byte(`
		{
			"processed": 1000,
			"sent": 950,
			"failed": 50,
			"duration_seconds": 180,
			"details": {
				"batches_sent": 20,
				"average_send_time": 0.18
			}
		}
	`))

	err = repo.UpdateFailedStatusWithOutput(ctx, created.ID, successOutput)
	require.NoError(t, err)

	// Override status to success
	err = repo.UpdateStatus(ctx, created.ID, domain.OperationStatusSuccess)
	require.NoError(t, err)

	// Verify final state
	final, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, domain.OperationStatusSuccess, final.Status)
	assert.JSONEq(t, string(successOutput), string(final.Output))
}
