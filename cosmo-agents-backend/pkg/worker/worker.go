package worker

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

// ParsePayload parses asynq task payload into the provided struct.
func ParsePayload(task *asynq.Task, payload interface{}) error {
	if err := json.Unmarshal(task.Payload(), payload); err != nil {
		return err
	}
	return nil
}

// NewTask creates a new asynq task with the given type and payload.
func NewTask(taskType string, payload interface{}) (*asynq.Task, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(taskType, payloadBytes), nil
}

// WorkerTaskOptions holds options for creating tasks.
type WorkerTaskOptions struct {
	Queue    string
	MaxRetry int
	Timeout  int
}

// Default queue names
const (
	QueueBackground = "background"
	QueueEmail      = "email"
	QueueAnalytics  = "analytics"
)

// GetQueueName returns the appropriate queue name based on priority.
func GetQueueName(priority string) string {
	switch priority {
	case "email":
		return QueueEmail
	case "analytics":
		return QueueAnalytics
	default:
		return QueueBackground
	}
}
