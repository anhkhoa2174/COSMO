package workflow

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the Workflow schema
	err = db.AutoMigrate(&domain.Workflow{})
	require.NoError(t, err)

	return db
}

// TestWorkflowRepository_Create tests creating a new workflow
func TestWorkflowRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkflowRepository(db)
	ctx := context.Background()

	// Test data
	userID := uuid.New()
	workflow := &domain.Workflow{
		UserID:    userID,
		Nodes:     pq.StringArray{"node1", "node2", "node3"},
		Edges:     base.JSONB([]byte(`[{"from": "node1", "to": "node2"}]`)),
		State:     base.JSONB([]byte(`{"active": "node1"}`)),
		CMetadata: base.JSONB([]byte(`{"version": "1.0"}`)),
	}

	// Test Create
	result, err := repo.Create(ctx, workflow)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, pq.StringArray{"node1", "node2", "node3"}, result.Nodes)
	assert.Equal(t, base.JSONB([]byte(`[{"from": "node1", "to": "node2"}]`)), result.Edges)
	assert.Equal(t, base.JSONB([]byte(`{"active": "node1"}`)), result.State)
	assert.Equal(t, base.JSONB([]byte(`{"version": "1.0"}`)), result.CMetadata)
}

// TestWorkflowRepository_Create_WithDefaults tests creating workflow with default values
func TestWorkflowRepository_Create_WithDefaults(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkflowRepository(db)
	ctx := context.Background()

	// Test data with minimal fields
	userID := uuid.New()
	workflow := &domain.Workflow{
		UserID: userID,
		// Nodes, Edges, State, CMetadata should be set to defaults
	}

	// Test Create
	result, err := repo.Create(ctx, workflow)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, pq.StringArray{}, result.Nodes)
	assert.Equal(t, base.JSONB([]byte("[]")), result.Edges)
	assert.Equal(t, base.JSONB([]byte("{}")), result.State)
	assert.Equal(t, base.JSONB([]byte("{}")), result.CMetadata)
}

// TestWorkflowRepository_FindByID tests finding a workflow by ID
func TestWorkflowRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkflowRepository(db)
	ctx := context.Background()

	// Create test workflow
	userID := uuid.New()
	workflow := &domain.Workflow{
		UserID:    userID,
		Nodes:     pq.StringArray{"start", "process", "end"},
		Edges:     base.JSONB([]byte(`[{"from": "start", "to": "process"}, {"from": "process", "to": "end"}]`)),
		State:     base.JSONB([]byte(`{"current": "start"}`)),
		CMetadata: base.JSONB([]byte(`{"name": "Test Workflow"}`)),
	}
	created, err := repo.Create(ctx, workflow)
	require.NoError(t, err)

	// Test FindByID
	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, userID, found.UserID)
	assert.Equal(t, pq.StringArray{"start", "process", "end"}, found.Nodes)
}

// TestWorkflowRepository_Update tests updating a workflow
func TestWorkflowRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkflowRepository(db)
	ctx := context.Background()

	// Create test workflow
	workflow := &domain.Workflow{
		UserID:    uuid.New(),
		Nodes:     pq.StringArray{"node1"},
		Edges:     base.JSONB([]byte(`[]`)),
		State:     base.JSONB([]byte(`{}`)),
		CMetadata: base.JSONB([]byte(`{}`)),
	}
	created, err := repo.Create(ctx, workflow)
	require.NoError(t, err)

	// Update the workflow
	created.Nodes = pq.StringArray{"node1", "node2"}
	created.Edges = base.JSONB([]byte(`[{"from": "node1", "to": "node2"}]`))
	created.State = base.JSONB([]byte(`{"current": "node2"}`))

	// Test Update
	err = repo.Update(ctx, created.ID, created)
	require.NoError(t, err)

	// Verify the update was applied
	updated, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, updated)
	assert.Equal(t, pq.StringArray{"node1", "node2"}, updated.Nodes)
	assert.Equal(t, base.JSONB([]byte(`[{"from": "node1", "to": "node2"}]`)), updated.Edges)
	assert.Equal(t, base.JSONB([]byte(`{"current": "node2"}`)), updated.State)
}

// TestWorkflowRepository_Delete tests deleting a workflow (soft delete)
func TestWorkflowRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkflowRepository(db)
	ctx := context.Background()

	// Create test workflow
	workflow := &domain.Workflow{
		UserID:    uuid.New(),
		Nodes:     pq.StringArray{"node1"},
		Edges:     base.JSONB([]byte(`[]`)),
		State:     base.JSONB([]byte(`{}`)),
		CMetadata: base.JSONB([]byte(`{}`)),
	}
	created, err := repo.Create(ctx, workflow)
	require.NoError(t, err)

	// Delete workflow
	err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// Verify soft deletion - should not be found with normal FindByID
	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, found) // Soft deleted records should not be found
}

// TestWorkflowRepository_ComplexWorkflow tests creating a complex workflow
func TestWorkflowRepository_ComplexWorkflow(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkflowRepository(db)
	ctx := context.Background()

	// Test complex workflow data
	userID := uuid.New()
	workflow := &domain.Workflow{
		UserID: userID,
		Nodes:  pq.StringArray{"email_validator", "ai_enricher", "crm_updater", "notification_sender"},
		Edges: base.JSONB([]byte(`
			[
				{"from": "email_validator", "to": "ai_enricher"},
				{"from": "ai_enricher", "to": "crm_updater"},
				{"from": "crm_updater", "to": "notification_sender"}
			]
		`)),
		State: base.JSONB([]byte(`
			{
				"email_validator": {"status": "completed", "output": {"valid": true}},
				"ai_enricher": {"status": "running", "output": {}},
				"crm_updater": {"status": "pending", "output": {}},
				"notification_sender": {"status": "pending", "output": {}}
			}
		`)),
		CMetadata: base.JSONB([]byte(`
			{
				"name": "Contact Enrichment Workflow",
				"description": "Validates, enriches, and updates contact information",
				"version": "2.1",
				"tags": ["contact", "enrichment", "automation"]
			}
		`)),
	}

	// Test Create
	result, err := repo.Create(ctx, workflow)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Nodes, 4)
	assert.Equal(t, userID, result.UserID)
}
