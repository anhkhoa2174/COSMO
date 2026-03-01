package task

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskStatus represents the lifecycle status of a task.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusDone      TaskStatus = "done"
)

// TaskPriority represents the urgency of a task.
type TaskPriority string

const (
	TaskPriorityLow      TaskPriority = "low"
	TaskPriorityMedium   TaskPriority = "medium"
	TaskPriorityHigh     TaskPriority = "high"
	TaskPriorityCritical TaskPriority = "critical"
)

const defaultTaskPayload = `{"agent":null,"template":null}`

// Task mirrors machine/models/task.py.
type Task struct {
	base.Base
	base.TimestampMixin

	ContactID   uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_tasks_contact_campaign_template" json:"contact_id"`
	CampaignID  uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_tasks_contact_campaign_template" json:"campaign_id"`
	TemplateID  uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_tasks_contact_campaign_template" json:"template_id"`
	ScheduleAt  *time.Time   `gorm:"type:timestamptz" json:"schedule_at,omitempty"`
	RespondedAt *time.Time   `gorm:"type:timestamptz" json:"responded_at,omitempty"`
	TriggeredAt *time.Time   `gorm:"type:timestamptz" json:"triggered_at,omitempty"`
	DoneAt      *time.Time   `gorm:"type:timestamptz" json:"done_at,omitempty"`
	Priority    TaskPriority `gorm:"type:text;default:'medium'" json:"priority"`
	Payload     base.JSONB   `gorm:"type:jsonb;default:'{\"agent\":null,\"template\":null}'" json:"payload,omitempty"`
	Status      TaskStatus   `gorm:"type:text;not null;default:'pending'" json:"status"`
	Error       *string      `gorm:"type:text" json:"error,omitempty"`

	// Note: Direct relationships removed to avoid circular imports.
	// Use the relations package for relationship queries:
	// - relations.TaskWithContact
	// - relations.TaskWithTemplate
	// - relations.TaskWithCampaign
	// - relations.TaskWithFullRelations
	// Only Updates kept as it doesn't cause circular imports
	Updates []TaskUpdate `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE" json:"updates,omitempty"`
}

// TableName specifies the table name.
func (Task) TableName() string {
	return "tasks"
}

// BeforeCreate sets defaults that mirror the Python implementation.
func (t *Task) BeforeCreate(tx *gorm.DB) error {
	if err := t.Base.BeforeCreate(tx); err != nil {
		return err
	}

	if t.Status == "" {
		t.Status = TaskStatusPending
	}
	if t.Priority == "" {
		t.Priority = TaskPriorityMedium
	}
	if len(t.Payload) == 0 {
		t.Payload = base.JSONB([]byte(defaultTaskPayload))
	}

	return nil
}

// SetPayload merges the provided structure into the payload JSON.
func (t *Task) SetPayload(data map[string]any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	t.Payload = base.JSONB(raw)
	return nil
}

// GetPayload returns the payload as a generic map.
func (t *Task) GetPayload() (map[string]any, error) {
	if len(t.Payload) == 0 {
		return map[string]any{}, nil
	}

	var out map[string]any
	if err := json.Unmarshal(t.Payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ValidateStatusTransition ensures status changes follow expected workflow.
func (t *Task) ValidateStatusTransition(newStatus TaskStatus) error {
	switch newStatus {
	case TaskStatusPending, TaskStatusRunning, TaskStatusCancelled, TaskStatusFailed, TaskStatusDone:
	default:
		return errors.New("invalid task status")
	}

	if t.Status == TaskStatusDone && newStatus != TaskStatusDone {
		return errors.New("cannot transition a completed task")
	}

	return nil
}

// TaskAttributes represents the minimal fields required to create or upsert tasks.
type TaskAttributes struct {
	ContactID   uuid.UUID
	CampaignID  uuid.UUID
	TemplateID  uuid.UUID
	Status      TaskStatus
	ScheduleAt  *time.Time
	Priority    *TaskPriority
	Payload     map[string]any
	TriggeredAt *time.Time
	RespondedAt *time.Time
	DoneAt      *time.Time
	Error       *string
}

// TaskUpdateType categorizes the type of update for a task.
type TaskUpdateType string

const (
	TaskUpdateTypeResponse   TaskUpdateType = "response"
	TaskUpdateTypeResolution TaskUpdateType = "resolution"
	TaskUpdateTypeUpdate     TaskUpdateType = "update"
)

// TaskUpdate records lifecycle events for a task.
type TaskUpdate struct {
	base.Base
	base.TimestampMixin

	TaskID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"task_id"`
	UpdateType TaskUpdateType `gorm:"type:text;default:'update'" json:"update_type"`

	Task *Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
}

// TableName specifies the table name.
func (TaskUpdate) TableName() string {
	return "task_updates"
}
