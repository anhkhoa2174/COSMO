package task

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// TestTask_TableName tests the table name method
func TestTask_TableName(t *testing.T) {
	t.Run("Correct table name", func(t *testing.T) {
		task := Task{}
		expected := "tasks"
		assert.Equal(t, expected, task.TableName())
	})
}

// TestTask_Structure tests the task struct fields
func TestTask_Structure(t *testing.T) {
	t.Run("Task struct has correct fields", func(t *testing.T) {
		contactID := uuid.New()
		campaignID := uuid.New()
		templateID := uuid.New()
		scheduleAt := time.Now().Add(1 * time.Hour)
		respondedAt := time.Now()
		triggeredAt := time.Now().Add(-5 * time.Minute)
		doneAt := time.Now()
		errorMsg := "Something went wrong"

		task := Task{
			ContactID:   contactID,
			CampaignID:  campaignID,
			TemplateID:  templateID,
			ScheduleAt:  &scheduleAt,
			RespondedAt: &respondedAt,
			TriggeredAt: &triggeredAt,
			DoneAt:      &doneAt,
			Priority:    TaskPriorityHigh,
			Payload:     base.JSONB(`{"test": "data"}`),
			Status:      TaskStatusRunning,
			Error:       &errorMsg,
		}

		assert.Equal(t, contactID, task.ContactID)
		assert.Equal(t, campaignID, task.CampaignID)
		assert.Equal(t, templateID, task.TemplateID)
		require.NotNil(t, task.ScheduleAt)
		assert.Equal(t, scheduleAt, *task.ScheduleAt)
		require.NotNil(t, task.RespondedAt)
		assert.Equal(t, respondedAt, *task.RespondedAt)
		require.NotNil(t, task.TriggeredAt)
		assert.Equal(t, triggeredAt, *task.TriggeredAt)
		require.NotNil(t, task.DoneAt)
		assert.Equal(t, doneAt, *task.DoneAt)
		assert.Equal(t, TaskPriorityHigh, task.Priority)
		assert.Equal(t, base.JSONB(`{"test": "data"}`), task.Payload)
		assert.Equal(t, TaskStatusRunning, task.Status)
		require.NotNil(t, task.Error)
		assert.Equal(t, errorMsg, *task.Error)
	})

	t.Run("Task with nil optional fields", func(t *testing.T) {
		task := Task{
			ContactID:  uuid.New(),
			CampaignID: uuid.New(),
			TemplateID: uuid.New(),
			Status:     TaskStatusPending,
		}

		assert.Equal(t, TaskStatusPending, task.Status)
		assert.Nil(t, task.ScheduleAt)
		assert.Nil(t, task.RespondedAt)
		assert.Nil(t, task.TriggeredAt)
		assert.Nil(t, task.DoneAt)
		assert.Equal(t, TaskPriority(""), task.Priority) // Initial empty value
		assert.Nil(t, task.Error)

		// Test BeforeCreate sets defaults
		err := task.BeforeCreate(nil)
		require.NoError(t, err)
		assert.Equal(t, TaskPriorityMedium, task.Priority) // Set by BeforeCreate
	})
}

// TestTask_Inheritance tests that Task properly inherits from base structs
func TestTask_Inheritance(t *testing.T) {
	t.Run("Inherits Base fields", func(t *testing.T) {
		task := Task{}

		// Should have ID field from Base
		_ = task.ID // Just verify field exists

		// Should be able to use BeforeCreate method from Base
		err := task.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, task.ID, "BeforeCreate should set ID")
	})

	t.Run("Inherits TimestampMixin fields", func(t *testing.T) {
		task := Task{}

		// Should have CreatedAt and UpdatedAt fields
		assert.Equal(t, time.Time{}, task.CreatedAt, "CreatedAt should be zero time initially")
		assert.Equal(t, time.Time{}, task.UpdatedAt, "UpdatedAt should be zero time initially")
	})
}

// TestTaskBeforeCreate_Comprehensive tests comprehensive BeforeCreate behavior
func TestTaskBeforeCreate_Comprehensive(t *testing.T) {
	t.Run("BeforeCreate sets all defaults", func(t *testing.T) {
		task := Task{
			ContactID:  uuid.New(),
			CampaignID: uuid.New(),
			TemplateID: uuid.New(),
			Status:     "",           // Empty should default to pending
			Priority:   "",           // Empty should default to medium
			Payload:    base.JSONB{}, // Empty should default to default payload
		}

		err := task.BeforeCreate(nil)
		assert.NoError(t, err)

		assert.Equal(t, TaskStatusPending, task.Status)
		assert.Equal(t, TaskPriorityMedium, task.Priority)
		assert.Equal(t, base.JSONB(defaultTaskPayload), task.Payload)
		assert.NotEqual(t, uuid.Nil, task.ID)
	})

	t.Run("BeforeCreate preserves existing values", func(t *testing.T) {
		existingID := uuid.New()
		status := TaskStatusRunning
		priority := TaskPriorityCritical
		payload := base.JSONB(`{"custom": "data"}`)

		task := Task{
			Base:     base.Base{ID: existingID},
			Status:   status,
			Priority: priority,
			Payload:  payload,
		}

		err := task.BeforeCreate(nil)
		assert.NoError(t, err)
		assert.Equal(t, existingID, task.ID, "Existing ID should be preserved")
		assert.Equal(t, status, task.Status, "Existing status should be preserved")
		assert.Equal(t, priority, task.Priority, "Existing priority should be preserved")
		assert.Equal(t, payload, task.Payload, "Existing payload should be preserved")
	})

	t.Run("BeforeCreate handles Base.BeforeCreate error", func(t *testing.T) {
		task := Task{}
		// Force Base.BeforeCreate to fail by setting nil ID to invalid state
		task.ID = uuid.Nil // This should not cause error in current implementation

		err := task.BeforeCreate(nil)
		assert.NoError(t, err) // Should still succeed as Base.BeforeCreate handles this
	})
}

// TestTask_PayloadOperations tests payload JSON operations
func TestTask_PayloadOperations(t *testing.T) {
	t.Run("SetPayload with valid data", func(t *testing.T) {
		task := Task{}
		data := map[string]any{
			"agent": map[string]any{
				"id":    "agent-123",
				"name":  "Test Agent",
				"email": "test@example.com",
			},
			"template": map[string]any{
				"id":   "template-456",
				"name": "Test Template",
			},
			"metadata": map[string]any{
				"version": "1.0",
				"created": time.Now(),
			},
		}

		err := task.SetPayload(data)
		require.NoError(t, err)
		assert.NotEmpty(t, task.Payload)

		// Verify the data can be retrieved
		result, err := task.GetPayload()
		require.NoError(t, err)
		assert.Equal(t, "agent-123", result["agent"].(map[string]any)["id"])
		assert.Equal(t, "Test Agent", result["agent"].(map[string]any)["name"])
		assert.Equal(t, "template-456", result["template"].(map[string]any)["id"])
	})

	t.Run("SetPayload with nil values", func(t *testing.T) {
		task := Task{}
		data := map[string]any{
			"agent":    nil,
			"template": nil,
			"metadata": nil,
		}

		err := task.SetPayload(data)
		require.NoError(t, err)

		result, err := task.GetPayload()
		require.NoError(t, err)
		assert.Equal(t, 3, len(result))
		assert.Nil(t, result["agent"])
		assert.Nil(t, result["template"])
		assert.Nil(t, result["metadata"])

		// Test nested structure with type assertions for JSON number conversion
		if agent, ok := result["agent"].(map[string]any); ok {
			if settings, ok := agent["settings"].(map[string]any); ok {
				// JSON numbers are unmarshaled as float64
				if timeout, ok := settings["timeout"].(float64); ok {
					assert.Equal(t, float64(30), timeout)
				}
				if retries, ok := settings["retries"].(float64); ok {
					assert.Equal(t, float64(3), retries)
				}
			}
		}
	})

	t.Run("SetPayload with complex nested structure", func(t *testing.T) {
		task := Task{}
		data := map[string]any{
			"agent": map[string]any{
				"settings": map[string]any{
					"timeout": 30,
					"retries": 3,
					"config": map[string]any{
						"debug":   true,
						"verbose": false,
					},
				},
			},
			"custom_fields": []any{"field1", "field2", "field3"},
			"numbers": map[string]any{
				"count": 42,
				"rate":  0.95,
			},
		}

		err := task.SetPayload(data)
		require.NoError(t, err)

		result, err := task.GetPayload()
		require.NoError(t, err)
		agent := result["agent"].(map[string]any)
		settings := agent["settings"].(map[string]any)
		// JSON numbers are unmarshaled as float64
		assert.Equal(t, float64(30), settings["timeout"])
		assert.Equal(t, float64(3), settings["retries"])
	})

	t.Run("SetPayload error handling", func(t *testing.T) {
		task := Task{}

		// Create data that would cause JSON marshaling to fail
		invalidData := make(map[string]any)
		invalidData["invalid"] = make(chan int) // Channels cannot be marshaled to JSON

		err := task.SetPayload(invalidData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "json")
	})

	t.Run("GetPayload with empty payload", func(t *testing.T) {
		task := Task{}
		result, err := task.GetPayload()
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("GetPayload with invalid JSON", func(t *testing.T) {
		task := Task{
			Payload: base.JSONB([]byte("invalid json {")),
		}

		result, err := task.GetPayload()
		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character")
	})

	t.Run("Round trip payload", func(t *testing.T) {
		task := Task{}
		originalData := map[string]any{
			"test":    "value",
			"number":  123,
			"boolean": true,
		}

		err := task.SetPayload(originalData)
		require.NoError(t, err)

		retrievedData, err := task.GetPayload()
		require.NoError(t, err)
		assert.Equal(t, originalData["test"], retrievedData["test"])
		// JSON numbers are unmarshaled as float64
		assert.Equal(t, float64(123), retrievedData["number"])
		assert.Equal(t, originalData["boolean"], retrievedData["boolean"])
	})
}

// TestTask_ValidateStatusTransition_Comprehensive tests all status transition scenarios
func TestTask_ValidateStatusTransition_Comprehensive(t *testing.T) {
	t.Run("Valid transitions from pending", func(t *testing.T) {
		task := Task{Status: TaskStatusPending}

		validTransitions := []TaskStatus{
			TaskStatusPending, TaskStatusRunning, TaskStatusCancelled, TaskStatusFailed, TaskStatusDone,
		}

		for _, newStatus := range validTransitions {
			t.Run(string(newStatus), func(t *testing.T) {
				err := task.ValidateStatusTransition(newStatus)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("Valid transitions from running", func(t *testing.T) {
		task := Task{Status: TaskStatusRunning}

		validTransitions := []TaskStatus{
			TaskStatusRunning, TaskStatusCancelled, TaskStatusFailed, TaskStatusDone,
		}

		for _, newStatus := range validTransitions {
			t.Run(string(newStatus), func(t *testing.T) {
				err := task.ValidateStatusTransition(newStatus)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("Valid transitions from cancelled", func(t *testing.T) {
		task := Task{Status: TaskStatusCancelled}

		validTransitions := []TaskStatus{
			TaskStatusCancelled, TaskStatusPending, TaskStatusRunning, TaskStatusFailed, TaskStatusDone,
		}

		for _, newStatus := range validTransitions {
			t.Run(string(newStatus), func(t *testing.T) {
				err := task.ValidateStatusTransition(newStatus)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("Valid transitions from failed", func(t *testing.T) {
		task := Task{Status: TaskStatusFailed}

		validTransitions := []TaskStatus{
			TaskStatusFailed, TaskStatusPending, TaskStatusRunning, TaskStatusDone,
		}

		for _, newStatus := range validTransitions {
			t.Run(string(newStatus), func(t *testing.T) {
				err := task.ValidateStatusTransition(newStatus)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("Done task cannot transition to other statuses", func(t *testing.T) {
		task := Task{Status: TaskStatusDone}

		invalidTransitions := []TaskStatus{
			TaskStatusPending, TaskStatusRunning, TaskStatusCancelled, TaskStatusFailed,
		}

		for _, newStatus := range invalidTransitions {
			t.Run(string(newStatus), func(t *testing.T) {
				err := task.ValidateStatusTransition(newStatus)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cannot transition a completed task")
			})
		}
	})

	t.Run("Invalid status values", func(t *testing.T) {
		task := Task{Status: TaskStatusPending}

		invalidStatuses := []TaskStatus{
			TaskStatus("invalid"),
			TaskStatus(""),
			TaskStatus("PROCESSING"),
			TaskStatus("COMPLETED"), // Should be "done"
			TaskStatus("STARTED"),   // Should be "running"
		}

		for _, invalidStatus := range invalidStatuses {
			t.Run(string(invalidStatus), func(t *testing.T) {
				err := task.ValidateStatusTransition(invalidStatus)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid task status")
			})
		}
	})
}

// TestTaskStatus_Constants tests task status constants
func TestTaskStatus_Constants(t *testing.T) {
	t.Run("Task status constants are correct", func(t *testing.T) {
		assert.Equal(t, TaskStatus("pending"), TaskStatusPending)
		assert.Equal(t, TaskStatus("running"), TaskStatusRunning)
		assert.Equal(t, TaskStatus("cancelled"), TaskStatusCancelled)
		assert.Equal(t, TaskStatus("failed"), TaskStatusFailed)
		assert.Equal(t, TaskStatus("done"), TaskStatusDone)
	})

	t.Run("Task status constants are unique", func(t *testing.T) {
		statuses := []TaskStatus{
			TaskStatusPending, TaskStatusRunning, TaskStatusCancelled, TaskStatusFailed, TaskStatusDone,
		}

		seen := make(map[TaskStatus]bool)
		for _, status := range statuses {
			if seen[status] {
				t.Errorf("Duplicate status constant: %s", status)
			}
			seen[status] = true
		}
	})
}

// TestTaskPriority_Constants tests task priority constants
func TestTaskPriority_Constants(t *testing.T) {
	t.Run("Task priority constants are correct", func(t *testing.T) {
		assert.Equal(t, TaskPriority("low"), TaskPriorityLow)
		assert.Equal(t, TaskPriority("medium"), TaskPriorityMedium)
		assert.Equal(t, TaskPriority("high"), TaskPriorityHigh)
		assert.Equal(t, TaskPriority("critical"), TaskPriorityCritical)
	})

	t.Run("Task priority constants are unique", func(t *testing.T) {
		priorities := []TaskPriority{
			TaskPriorityLow, TaskPriorityMedium, TaskPriorityHigh, TaskPriorityCritical,
		}

		seen := make(map[TaskPriority]bool)
		for _, priority := range priorities {
			if seen[priority] {
				t.Errorf("Duplicate priority constant: %s", priority)
			}
			seen[priority] = true
		}
	})
}

// TestTask_RealWorldScenarios tests realistic task scenarios
func TestTask_RealWorldScenarios(t *testing.T) {
	t.Run("Scheduled task", func(t *testing.T) {
		contactID := uuid.New()
		campaignID := uuid.New()
		templateID := uuid.New()
		scheduleAt := time.Now().Add(2 * time.Hour)

		task := Task{
			ContactID:  contactID,
			CampaignID: campaignID,
			TemplateID: templateID,
			ScheduleAt: &scheduleAt,
			Priority:   TaskPriorityHigh,
			Status:     TaskStatusPending,
		}

		assert.Equal(t, TaskStatusPending, task.Status)
		require.NotNil(t, task.ScheduleAt)
		assert.True(t, task.ScheduleAt.After(time.Now()))
		assert.Equal(t, TaskPriorityHigh, task.Priority)
	})

	t.Run("Completed task with updates", func(t *testing.T) {
		contactID := uuid.New()
		campaignID := uuid.New()
		templateID := uuid.New()
		now := time.Now()

		task := Task{
			ContactID:   contactID,
			CampaignID:  campaignID,
			TemplateID:  templateID,
			TriggeredAt: &now,
			RespondedAt: &now,
			DoneAt:      &now,
			Priority:    TaskPriorityMedium,
			Status:      TaskStatusDone,
			Payload:     base.JSONB(`{"result": "success"}`),
		}

		assert.Equal(t, TaskStatusDone, task.Status)
		require.NotNil(t, task.TriggeredAt)
		require.NotNil(t, task.RespondedAt)
		require.NotNil(t, task.DoneAt)
		assert.Equal(t, TaskPriorityMedium, task.Priority)
	})

	t.Run("Failed task with error", func(t *testing.T) {
		contactID := uuid.New()
		campaignID := uuid.New()
		templateID := uuid.New()
		errorMsg := "Timeout exceeded"

		task := Task{
			ContactID:   contactID,
			CampaignID:  campaignID,
			TemplateID:  templateID,
			TriggeredAt: &time.Time{},
			Status:      TaskStatusFailed,
			Error:       &errorMsg,
		}

		assert.Equal(t, TaskStatusFailed, task.Status)
		require.NotNil(t, task.Error)
		assert.Equal(t, errorMsg, *task.Error)
	})

	t.Run("Critical priority task", func(t *testing.T) {
		task := Task{
			Priority: TaskPriorityCritical,
			Status:   TaskStatusRunning,
		}

		assert.Equal(t, TaskPriorityCritical, task.Priority)
		assert.Equal(t, TaskStatusRunning, task.Status)
	})
}

// TestTask_EdgeCases tests edge cases and unusual but valid scenarios
func TestTask_EdgeCases(t *testing.T) {
	t.Run("Task with all timestamps set", func(t *testing.T) {
		now := time.Now()
		past := now.Add(-1 * time.Hour)
		future := now.Add(1 * time.Hour)

		task := Task{
			ScheduleAt:  &future,
			TriggeredAt: &past,
			RespondedAt: &now,
			DoneAt:      &now,
		}

		assert.True(t, task.ScheduleAt.After(*task.TriggeredAt))
		assert.True(t, task.RespondedAt.Equal(*task.DoneAt))
	})

	t.Run("Task with past and future schedule", func(t *testing.T) {
		pastTime := time.Now().Add(-1 * time.Hour)
		futureTime := time.Now().Add(1 * time.Hour)

		pastTask := Task{ScheduleAt: &pastTime}
		futureTask := Task{ScheduleAt: &futureTime}

		assert.True(t, pastTask.ScheduleAt.Before(time.Now()))
		assert.True(t, futureTask.ScheduleAt.After(time.Now()))
	})

	t.Run("Task with zero values", func(t *testing.T) {
		task := Task{
			Status:   TaskStatusPending,
			Priority: TaskPriorityMedium,
			Payload:  base.JSONB(defaultTaskPayload),
		}

		assert.Equal(t, TaskStatusPending, task.Status)
		assert.Equal(t, TaskPriorityMedium, task.Priority)
		assert.Equal(t, base.JSONB(defaultTaskPayload), task.Payload)
	})

	t.Run("Task with very long payload", func(t *testing.T) {
		task := Task{}
		largeData := make(map[string]any)

		// Create a large payload
		for i := 0; i < 100; i++ {
			largeData[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
		}

		err := task.SetPayload(largeData)
		require.NoError(t, err)

		result, err := task.GetPayload()
		require.NoError(t, err)
		assert.Len(t, result, 100)
	})
}

// TestTaskUpdate_Entity tests TaskUpdate struct
func TestTaskUpdate_Entity(t *testing.T) {
	t.Run("TaskUpdate struct fields", func(t *testing.T) {
		taskID := uuid.New()

		update := TaskUpdate{
			TaskID:     taskID,
			UpdateType: TaskUpdateTypeResponse,
		}

		assert.Equal(t, taskID, update.TaskID)
		assert.Equal(t, TaskUpdateTypeResponse, update.UpdateType)
	})

	t.Run("TaskUpdate table name", func(t *testing.T) {
		update := TaskUpdate{}
		assert.Equal(t, "task_updates", update.TableName())
	})

	t.Run("TaskUpdate default values", func(t *testing.T) {
		update := TaskUpdate{
			TaskID: uuid.New(),
		}

		// Default value is set by GORM, not by struct initialization
		// In Go, the zero value for string is ""
		assert.Equal(t, TaskUpdateType(""), update.UpdateType)
	})
}

// TestTaskUpdateType_Constants tests task update type constants
func TestTaskUpdateType_Constants(t *testing.T) {
	t.Run("Task update type constants are correct", func(t *testing.T) {
		assert.Equal(t, TaskUpdateType("response"), TaskUpdateTypeResponse)
		assert.Equal(t, TaskUpdateType("resolution"), TaskUpdateTypeResolution)
		assert.Equal(t, TaskUpdateType("update"), TaskUpdateTypeUpdate)
	})

	t.Run("Task update type constants are unique", func(t *testing.T) {
		updateTypes := []TaskUpdateType{
			TaskUpdateTypeResponse, TaskUpdateTypeResolution, TaskUpdateTypeUpdate,
		}

		seen := make(map[TaskUpdateType]bool)
		for _, updateType := range updateTypes {
			if seen[updateType] {
				t.Errorf("Duplicate update type constant: %s", updateType)
			}
			seen[updateType] = true
		}
	})
}

// TestTaskAttributes_Struct tests TaskAttributes struct
func TestTaskAttributes_Struct(t *testing.T) {
	t.Run("TaskAttributes with all fields", func(t *testing.T) {
		contactID := uuid.New()
		campaignID := uuid.New()
		templateID := uuid.New()
		scheduleAt := time.Now().Add(1 * time.Hour)
		priority := TaskPriorityHigh
		payload := map[string]any{"test": "data"}
		triggeredAt := time.Now()
		respondedAt := time.Now()
		doneAt := time.Now()
		errorMsg := "Test error"

		attrs := TaskAttributes{
			ContactID:   contactID,
			CampaignID:  campaignID,
			TemplateID:  templateID,
			Status:      TaskStatusPending,
			ScheduleAt:  &scheduleAt,
			Priority:    &priority,
			Payload:     payload,
			TriggeredAt: &triggeredAt,
			RespondedAt: &respondedAt,
			DoneAt:      &doneAt,
			Error:       &errorMsg,
		}

		assert.Equal(t, contactID, attrs.ContactID)
		assert.Equal(t, campaignID, attrs.CampaignID)
		assert.Equal(t, templateID, attrs.TemplateID)
		assert.Equal(t, TaskStatusPending, attrs.Status)
		require.NotNil(t, attrs.ScheduleAt)
		assert.Equal(t, scheduleAt, *attrs.ScheduleAt)
		require.NotNil(t, attrs.Priority)
		assert.Equal(t, priority, *attrs.Priority)
		assert.Equal(t, payload, attrs.Payload)
		require.NotNil(t, attrs.TriggeredAt)
		assert.Equal(t, triggeredAt, *attrs.TriggeredAt)
		require.NotNil(t, attrs.RespondedAt)
		assert.Equal(t, respondedAt, *attrs.RespondedAt)
		require.NotNil(t, attrs.DoneAt)
		assert.Equal(t, doneAt, *attrs.DoneAt)
		require.NotNil(t, attrs.Error)
		assert.Equal(t, errorMsg, *attrs.Error)
	})

	t.Run("TaskAttributes with nil optional fields", func(t *testing.T) {
		attrs := TaskAttributes{
			ContactID:  uuid.New(),
			CampaignID: uuid.New(),
			TemplateID: uuid.New(),
			Status:     TaskStatusRunning,
		}

		assert.Equal(t, TaskStatusRunning, attrs.Status)
		assert.Nil(t, attrs.ScheduleAt)
		assert.Nil(t, attrs.Priority)
		assert.Nil(t, attrs.Payload)
		assert.Nil(t, attrs.TriggeredAt)
		assert.Nil(t, attrs.RespondedAt)
		assert.Nil(t, attrs.DoneAt)
		assert.Nil(t, attrs.Error)
	})
}

// TestTask_JSONSerialization tests JSON serialization of task entities
func TestTask_JSONSerialization(t *testing.T) {
	t.Run("Serialize task to JSON", func(t *testing.T) {
		contactID := uuid.New()
		campaignID := uuid.New()
		templateID := uuid.New()
		errorMsg := "Test error"
		payload := map[string]any{"key": "value"}

		task := Task{
			Base:       base.Base{ID: uuid.New()},
			ContactID:  contactID,
			CampaignID: campaignID,
			TemplateID: templateID,
			Status:     TaskStatusFailed,
			Priority:   TaskPriorityCritical,
			Error:      &errorMsg,
			Payload:    base.JSONB{},
		}

		// Set payload properly
		err := task.SetPayload(payload)
		require.NoError(t, err)

		jsonData, err := json.Marshal(task)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		// Verify JSON contains expected fields
		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, contactID.String())
		assert.Contains(t, jsonStr, campaignID.String())
		assert.Contains(t, jsonStr, templateID.String())
		assert.Contains(t, jsonStr, "failed")
		assert.Contains(t, jsonStr, "critical")
		assert.Contains(t, jsonStr, "Test error")
	})

	t.Run("Serialize TaskUpdate to JSON", func(t *testing.T) {
		taskID := uuid.New()
		update := TaskUpdate{
			Base:       base.Base{ID: uuid.New()},
			TaskID:     taskID,
			UpdateType: TaskUpdateTypeResponse,
		}

		jsonData, err := json.Marshal(update)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonData)

		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, taskID.String())
		assert.Contains(t, jsonStr, "response")
	})
}

// TestTask_ComparisonAndEquality tests comparison and equality
func TestTask_ComparisonAndEquality(t *testing.T) {
	t.Run("Same task comparison", func(t *testing.T) {
		contactID := uuid.New()
		campaignID := uuid.New()
		templateID := uuid.New()

		task1 := Task{
			Base:       base.Base{ID: uuid.New()},
			ContactID:  contactID,
			CampaignID: campaignID,
			TemplateID: templateID,
			Status:     TaskStatusPending,
			Priority:   TaskPriorityMedium,
		}
		task2 := Task{
			ContactID:  contactID,
			CampaignID: campaignID,
			TemplateID: templateID,
			Status:     TaskStatusPending,
			Priority:   TaskPriorityMedium,
		}

		assert.Equal(t, task1.ContactID, task2.ContactID)
		assert.Equal(t, task1.CampaignID, task2.CampaignID)
		assert.Equal(t, task1.TemplateID, task2.TemplateID)
		assert.Equal(t, task1.Status, task2.Status)
		assert.Equal(t, task1.Priority, task2.Priority)
		assert.Equal(t, task1.TableName(), task2.TableName())
	})

	t.Run("Different task comparison", func(t *testing.T) {
		task1 := Task{
			Status:   TaskStatusPending,
			Priority: TaskPriorityLow,
		}
		task2 := Task{
			Status:   TaskStatusRunning,
			Priority: TaskPriorityHigh,
		}

		assert.NotEqual(t, task1.Status, task2.Status)
		assert.NotEqual(t, task1.Priority, task2.Priority)
		assert.Equal(t, task1.TableName(), task2.TableName())
	})
}

// TestTask_TimeFieldOperations tests time field manipulations
func TestTask_TimeFieldOperations(t *testing.T) {
	t.Run("Time field updates", func(t *testing.T) {
		task := Task{}

		// Initial state - all nil
		assert.Nil(t, task.ScheduleAt)
		assert.Nil(t, task.TriggeredAt)
		assert.Nil(t, task.RespondedAt)
		assert.Nil(t, task.DoneAt)

		// Set schedule time
		scheduleAt := time.Now().Add(1 * time.Hour)
		task.ScheduleAt = &scheduleAt
		assert.Equal(t, scheduleAt, *task.ScheduleAt)

		// Set triggered time
		triggeredAt := time.Now()
		task.TriggeredAt = &triggeredAt
		assert.Equal(t, triggeredAt, *task.TriggeredAt)

		// Set responded time
		respondedAt := time.Now()
		task.RespondedAt = &respondedAt
		assert.Equal(t, respondedAt, *task.RespondedAt)

		// Set done time
		doneAt := time.Now()
		task.DoneAt = &doneAt
		assert.Equal(t, doneAt, *task.DoneAt)

		// Update schedule time
		newScheduleAt := time.Now().Add(2 * time.Hour)
		task.ScheduleAt = &newScheduleAt
		assert.Equal(t, newScheduleAt, *task.ScheduleAt)
	})

	t.Run("Time sequence validation", func(t *testing.T) {
		baseTime := time.Now()

		respondedAt := baseTime.Add(5 * time.Minute)
		doneAt := baseTime.Add(10 * time.Minute)

		task := Task{
			TriggeredAt: &baseTime,
			RespondedAt: &respondedAt,
			DoneAt:      &doneAt,
		}

		// Verify logical time sequence
		assert.True(t, task.TriggeredAt.Before(*task.RespondedAt))
		assert.True(t, task.RespondedAt.Before(*task.DoneAt))
	})
}

// TestTask_ConstantValues tests constant values
func TestTask_ConstantValues(t *testing.T) {
	t.Run("Default task payload constant", func(t *testing.T) {
		assert.Equal(t, `{"agent":null,"template":null}`, defaultTaskPayload)
		assert.NotEmpty(t, defaultTaskPayload)
		assert.Contains(t, defaultTaskPayload, "agent")
		assert.Contains(t, defaultTaskPayload, "template")
	})
}

// TestTask_BoundaryValues tests boundary conditions
func TestTask_BoundaryValues(t *testing.T) {
	t.Run("UUID boundaries", func(t *testing.T) {
		validUUID := uuid.New()
		zeroUUID := uuid.Nil

		task := Task{
			ContactID:  validUUID,
			CampaignID: validUUID,
			TemplateID: validUUID,
		}

		assert.NotEqual(t, zeroUUID, task.ContactID)
		assert.NotEqual(t, zeroUUID, task.CampaignID)
		assert.NotEqual(t, zeroUUID, task.TemplateID)
	})

	t.Run("Time boundaries", func(t *testing.T) {
		task := Task{}

		// Test with extreme past and future times
		pastTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC) // Unix epoch
		futureTime := time.Date(2100, 12, 31, 23, 59, 59, 999999999, time.UTC)

		task.ScheduleAt = &pastTime
		assert.Equal(t, pastTime, *task.ScheduleAt)

		task.ScheduleAt = &futureTime
		assert.Equal(t, futureTime, *task.ScheduleAt)
	})
}
