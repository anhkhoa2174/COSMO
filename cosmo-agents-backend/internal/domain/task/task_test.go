package task

import (
	"testing"

	"github.com/google/uuid"
)

func TestTaskBeforeCreateDefaults(t *testing.T) {
	task := Task{}

	if err := task.BeforeCreate(nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if task.ID == uuid.Nil {
		t.Fatal("expected task ID to be set")
	}
	if task.Status != TaskStatusPending {
		t.Fatalf("expected default status %q, got %q", TaskStatusPending, task.Status)
	}
	if task.Priority != TaskPriorityMedium {
		t.Fatalf("expected default priority %q, got %q", TaskPriorityMedium, task.Priority)
	}
	if string(task.Payload) != defaultTaskPayload {
		t.Fatalf("expected default payload %s, got %s", defaultTaskPayload, string(task.Payload))
	}
}

func TestTaskSetAndGetPayload(t *testing.T) {
	task := Task{}
	payload := map[string]any{
		"agent":    map[string]any{"id": "agent-1"},
		"template": nil,
	}

	if err := task.SetPayload(payload); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	out, err := task.GetPayload()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out["agent"] == nil {
		t.Fatal("expected agent key to be present")
	}
}

func TestTaskValidateStatusTransition(t *testing.T) {
	task := Task{Status: TaskStatusPending}

	if err := task.ValidateStatusTransition(TaskStatusRunning); err != nil {
		t.Fatalf("expected transition to succeed, got %v", err)
	}

	if err := task.ValidateStatusTransition(TaskStatus("unknown")); err == nil {
		t.Fatal("expected invalid status error")
	}

	task.Status = TaskStatusDone
	if err := task.ValidateStatusTransition(TaskStatusRunning); err == nil {
		t.Fatal("expected transition from done to running to fail")
	}
}
