package task

import (
	"testing"

	"github.com/google/uuid"
)

func TestTaskUpdateFields(t *testing.T) {
	taskID := uuid.New()
	update := TaskUpdate{TaskID: taskID}

	if update.TableName() != "task_updates" {
		t.Fatalf("unexpected table")
	}
	if update.TaskID != taskID {
		t.Fatalf("task id mismatch")
	}
}
