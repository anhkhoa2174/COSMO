package reply

import (
	"testing"

	"github.com/google/uuid"
)

func TestHandleReplyTask(t *testing.T) {
	task, err := NewHandleReplyTask(uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Type() != TypeHandleReply {
		t.Fatalf("expected task type %s, got %s", TypeHandleReply, task.Type())
	}
}
