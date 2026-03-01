package task

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain"
	baseRepo "github.com/rockship/cosmo-agents-go/internal/repository/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates a test database
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // Help with SQLite compatibility
	})
	require.NoError(t, err)

	// Auto-migrate only the Task schema to avoid PostgreSQL-specific features
	err = db.AutoMigrate(&domain.Task{})
	require.NoError(t, err)

	return db
}

// TestTaskRepository_Create tests creating a new task
func TestTaskRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	// Test data
	task := &domain.Task{
		ContactID:  uuid.New(),
		CampaignID: uuid.New(),
		TemplateID: uuid.New(),
		Status:     domain.TaskStatusPending,
		Priority:   domain.TaskPriorityHigh,
	}

	// Test Create
	result, err := repo.Create(ctx, task)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEqual(t, uuid.Nil, result.ID)
	assert.Equal(t, domain.TaskStatusPending, result.Status)
	assert.Equal(t, domain.TaskPriorityHigh, result.Priority)
}

// TestTaskRepository_GetTask tests fetching a single task by ID
func TestTaskRepository_GetTask(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	// Create test task
	task := &domain.Task{
		ContactID:  uuid.New(),
		CampaignID: uuid.New(),
		TemplateID: uuid.New(),
		Status:     domain.TaskStatusPending,
	}
	created, err := repo.Create(ctx, task)
	require.NoError(t, err)

	// Test GetTask
	found, err := repo.GetTask(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, task.ContactID, found.ContactID)
	assert.Equal(t, task.CampaignID, found.CampaignID)
	assert.Equal(t, task.TemplateID, found.TemplateID)
}

// TestTaskRepository_CreateTasks tests creating multiple tasks with upsert
func TestTaskRepository_CreateTasks(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	// Test data
	now := time.Now()
	priority := domain.TaskPriorityHigh
	attrs := []domain.TaskAttributes{
		{
			ContactID:  uuid.New(),
			CampaignID: uuid.New(),
			TemplateID: uuid.New(),
			Status:     domain.TaskStatusPending,
			Priority:   &priority,
			ScheduleAt: &now,
			Payload:    map[string]any{"test": "data"},
		},
		{
			ContactID:   uuid.New(),
			CampaignID:  uuid.New(),
			TemplateID:  uuid.New(),
			Status:      domain.TaskStatusRunning,
			TriggeredAt: &now,
		},
	}

	// Test CreateTasks
	taskIDs, err := repo.CreateTasks(ctx, attrs)
	require.NoError(t, err)
	assert.Len(t, taskIDs, 2)

	// Verify tasks were created
	for _, id := range taskIDs {
		task, err := repo.GetTask(ctx, id)
		require.NoError(t, err)
		assert.NotNil(t, task)
	}
}

// TestTaskRepository_FindAllFilters tests FindAll with filters/pagination.
func TestTaskRepository_FindAllFilters(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	// Insert tasks with varying status
	t1 := &domain.Task{ContactID: uuid.New(), CampaignID: uuid.New(), TemplateID: uuid.New(), Status: domain.TaskStatusPending}
	t2 := &domain.Task{ContactID: uuid.New(), CampaignID: uuid.New(), TemplateID: uuid.New(), Status: domain.TaskStatusRunning}
	t3 := &domain.Task{ContactID: uuid.New(), CampaignID: uuid.New(), TemplateID: uuid.New(), Status: domain.TaskStatusRunning}
	require.NoError(t, db.Create([]*domain.Task{t1, t2, t3}).Error)

	page := baseRepo.PaginationParams{Offset: 0, Limit: 2}
	filter := baseRepo.Filter{"status": domain.TaskStatusRunning}
	res, err := repo.FindAll(ctx, filter, &page)
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.Total)
	assert.Len(t, res.List, 2)
	assert.Equal(t, domain.TaskStatusRunning, res.List[0].Status)

	// Update should work (override avoids is_deleted filter)
	t2.Status = domain.TaskStatusDone
	require.NoError(t, repo.Update(ctx, t2.ID, t2))
	var check domain.Task
	require.NoError(t, db.First(&check, "id = ?", t2.ID).Error)
	assert.Equal(t, domain.TaskStatusDone, check.Status)
}

// TestTaskRepository_CreateTasks_ValidationError tests validation errors
func TestTaskRepository_CreateTasks_ValidationError(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	// Test with missing required fields
	attrs := []domain.TaskAttributes{
		{
			ContactID:  uuid.New(),
			CampaignID: uuid.New(),
			// Missing TemplateID
		},
	}

	// Test CreateTasks should fail
	_, err := repo.CreateTasks(ctx, attrs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "contact_id, campaign_id, and template_id are required")
}

// TestTaskRepository_GetPendingTasksByIDs tests fetching pending tasks by IDs
func TestTaskRepository_GetPendingTasksByIDs(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	// Create test tasks
	taskIDs := []uuid.UUID{}
	for i := 0; i < 3; i++ {
		task := &domain.Task{
			ContactID:  uuid.New(),
			CampaignID: uuid.New(),
			TemplateID: uuid.New(),
			Status:     domain.TaskStatusPending,
		}
		created, err := repo.Create(ctx, task)
		require.NoError(t, err)
		taskIDs = append(taskIDs, created.ID)
	}

	// Create a non-pending task
	nonPendingTask := &domain.Task{
		ContactID:  uuid.New(),
		CampaignID: uuid.New(),
		TemplateID: uuid.New(),
		Status:     domain.TaskStatusDone,
	}
	nonPendingCreated, err := repo.Create(ctx, nonPendingTask)
	require.NoError(t, err)

	// Test GetPendingTasksByIDs
	tasks, err := repo.GetPendingTasksByIDs(ctx, append(taskIDs, nonPendingCreated.ID))
	require.NoError(t, err)
	assert.Len(t, tasks, 3) // Only pending tasks

	for _, task := range tasks {
		assert.Equal(t, domain.TaskStatusPending, task.Status)
	}
}

// TestTaskRepository_RespondTasks tests marking tasks as running
func TestTaskRepository_RespondTasks(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	// Create test tasks
	taskIDs := []uuid.UUID{}
	for i := 0; i < 3; i++ {
		task := &domain.Task{
			ContactID:  uuid.New(),
			CampaignID: uuid.New(),
			TemplateID: uuid.New(),
			Status:     domain.TaskStatusPending,
		}
		created, err := repo.Create(ctx, task)
		require.NoError(t, err)
		taskIDs = append(taskIDs, created.ID)
	}

	// Test RespondTasks
	respondedAt := time.Now()
	err := repo.RespondTasks(ctx, taskIDs, respondedAt)
	require.NoError(t, err)

	// Verify tasks were updated
	for _, id := range taskIDs {
		task, err := repo.GetTask(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, domain.TaskStatusRunning, task.Status)
		assert.NotNil(t, task.RespondedAt)
	}
}

// TestTaskRepository_FindEmailTaskByID tests finding a task with relations
func TestTaskRepository_FindEmailTaskByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	// Create test task
	task := &domain.Task{
		ContactID:  uuid.New(),
		CampaignID: uuid.New(),
		TemplateID: uuid.New(),
		Status:     domain.TaskStatusPending,
	}
	created, err := repo.Create(ctx, task)
	require.NoError(t, err)

	// Test FindEmailTaskByID
	found, err := repo.FindEmailTaskByID(ctx, created.ID)
	require.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, created.ID, found.ID)
}

// TestTaskRepository_FindEmailTasks tests finding tasks with filters
func TestTaskRepository_FindEmailTasks(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	// Create test tasks
	for i := 0; i < 5; i++ {
		task := &domain.Task{
			ContactID:  uuid.New(),
			CampaignID: uuid.New(),
			TemplateID: uuid.New(),
			Status:     domain.TaskStatusPending,
		}
		_, err := repo.Create(ctx, task)
		require.NoError(t, err)
	}

	// Test FindEmailTasks
	tasks, total, err := repo.FindEmailTasks(ctx, EmailTaskFilter{}, 0, 10)
	require.NoError(t, err)
	assert.Len(t, tasks, 5)
	assert.Equal(t, int64(5), total)
}

// TestTaskRepository_FindEmailTasks_WithStatusFilter tests filtering by status
func TestTaskRepository_FindEmailTasks_WithStatusFilter(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	// Create test tasks with different statuses
	pendingTask := &domain.Task{
		ContactID:  uuid.New(),
		CampaignID: uuid.New(),
		TemplateID: uuid.New(),
		Status:     domain.TaskStatusPending,
	}
	createdPending, err := repo.Create(ctx, pendingTask)
	require.NoError(t, err)

	doneTask := &domain.Task{
		ContactID:  uuid.New(),
		CampaignID: uuid.New(),
		TemplateID: uuid.New(),
		Status:     domain.TaskStatusDone,
	}
	createdDone, err := repo.Create(ctx, doneTask)
	require.NoError(t, err)

	// Test filtering by pending status
	tasks, total, err := repo.FindEmailTasks(ctx, EmailTaskFilter{Status: string(domain.TaskStatusPending)}, 0, 10)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, createdPending.ID, tasks[0].ID)

	// Test filtering by done status
	tasks, total, err = repo.FindEmailTasks(ctx, EmailTaskFilter{Status: string(domain.TaskStatusDone)}, 0, 10)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, createdDone.ID, tasks[0].ID)
}

// TestTaskRepository_Pagination tests pagination functionality
func TestTaskRepository_Pagination(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTaskRepository(db)
	ctx := context.Background()

	// Create test tasks
	for i := 0; i < 10; i++ {
		task := &domain.Task{
			ContactID:  uuid.New(),
			CampaignID: uuid.New(),
			TemplateID: uuid.New(),
			Status:     domain.TaskStatusPending,
		}
		_, err := repo.Create(ctx, task)
		require.NoError(t, err)
	}

	// Test first page
	tasks, total, err := repo.FindEmailTasks(ctx, EmailTaskFilter{}, 0, 3)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)
	assert.Equal(t, int64(10), total)

	// Test second page
	tasks, total, err = repo.FindEmailTasks(ctx, EmailTaskFilter{}, 3, 3)
	require.NoError(t, err)
	assert.Len(t, tasks, 3)
	assert.Equal(t, int64(10), total)

	// Test last page
	tasks, total, err = repo.FindEmailTasks(ctx, EmailTaskFilter{}, 9, 3)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, int64(10), total)
}
