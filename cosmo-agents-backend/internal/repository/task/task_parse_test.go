package task

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/rockship/cosmo-agents-go/internal/domain"
)

// TestFindByID_ParsesTimestampsAndUUIDs ensures the custom FindByID copes with RFC3339 timestamps and invalid UUIDs.
func TestFindByID_ParsesTimestampsAndUUIDs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Task{}))

	repo := NewTaskRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	task := &domain.Task{
		ContactID:  uuid.New(),
		CampaignID: uuid.New(),
		TemplateID: uuid.New(),
		Status:     domain.TaskStatusPending,
		ScheduleAt: &now,
	}
	created, err := repo.Create(ctx, task)
	require.NoError(t, err)

	// Raw update timestamps to RFC3339Nano strings to exercise parser.
	rfcTime := now.Format(time.RFC3339Nano)
	require.NoError(t, db.Exec(`UPDATE tasks SET created_at=?, updated_at=?, schedule_at=? WHERE id=?`,
		rfcTime, rfcTime, rfcTime, created.ID).Error)

	found, err := repo.FindByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.NotNil(t, found.ScheduleAt)
	require.Equal(t, created.ID, found.ID)

	// Insert a bad UUID row (with distinct FK trio to avoid unique conflict) to ensure no panic
	require.NoError(t, db.Exec(`INSERT INTO tasks (id, created_at, updated_at, contact_id, campaign_id, template_id, status) VALUES (?,?,?,?,?,?,?)`,
		"bad-uuid", rfcTime, rfcTime, uuid.New().String(), uuid.New().String(), uuid.New().String(), domain.TaskStatusPending).Error)
	_, err = repo.FindByID(ctx, created.ID) // Should still work for valid rows
	require.NoError(t, err)
}
