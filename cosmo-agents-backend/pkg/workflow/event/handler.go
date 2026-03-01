package event

import "context"

// Handler defines the interface for event handling in workflows.
type Handler interface {
	// ProduceEvent sends an event to a topic or queue.
	ProduceEvent(ctx context.Context, topic string, key string, value interface{}) error

	// ConsumeEvent receives events from a topic or queue.
	ConsumeEvent(ctx context.Context, topic string, handler func(key string, value interface{}) error) error

	// ScheduleEvent schedules a function to run at a specific time or interval.
	ScheduleEvent(ctx context.Context, jobID string, trigger string, fn func() error) error

	// Close closes the event handler and releases resources.
	Close() error
}
