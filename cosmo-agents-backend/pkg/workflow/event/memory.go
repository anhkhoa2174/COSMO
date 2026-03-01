package event

import (
	"context"
	"fmt"
	"sync"
)

// MemoryHandler implements an in-memory event handler for testing and development.
type MemoryHandler struct {
	mu       sync.RWMutex
	queues   map[string][]Event
	handlers map[string][]func(key string, value interface{}) error
	jobs     map[string]*ScheduledJob
}

// Event represents an in-memory event.
type Event struct {
	Key   string
	Value interface{}
}

// ScheduledJob represents a scheduled task.
type ScheduledJob struct {
	JobID   string
	Trigger string
	Fn      func() error
}

// NewMemoryHandler creates a new in-memory event handler.
func NewMemoryHandler() *MemoryHandler {
	return &MemoryHandler{
		queues:   make(map[string][]Event),
		handlers: make(map[string][]func(key string, value interface{}) error),
		jobs:     make(map[string]*ScheduledJob),
	}
}

// ProduceEvent sends an event to an in-memory queue.
func (h *MemoryHandler) ProduceEvent(ctx context.Context, topic string, key string, value interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	event := Event{
		Key:   key,
		Value: value,
	}

	h.queues[topic] = append(h.queues[topic], event)

	// Trigger any registered handlers
	if handlers, exists := h.handlers[topic]; exists {
		for _, handler := range handlers {
			// Run handler asynchronously
			go func(fn func(key string, value interface{}) error) {
				_ = fn(key, value)
			}(handler)
		}
	}

	return nil
}

// ConsumeEvent registers a handler for events on a topic.
func (h *MemoryHandler) ConsumeEvent(ctx context.Context, topic string, handler func(key string, value interface{}) error) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.handlers[topic] = append(h.handlers[topic], handler)

	// Process any existing events in the queue
	if events, exists := h.queues[topic]; exists {
		for _, event := range events {
			go func(e Event) {
				_ = handler(e.Key, e.Value)
			}(event)
		}
		// Clear processed events
		h.queues[topic] = nil
	}

	return nil
}

// ScheduleEvent schedules a function to run.
func (h *MemoryHandler) ScheduleEvent(ctx context.Context, jobID string, trigger string, fn func() error) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.jobs[jobID] = &ScheduledJob{
		JobID:   jobID,
		Trigger: trigger,
		Fn:      fn,
	}

	return nil
}

// Close closes the memory handler.
func (h *MemoryHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.queues = make(map[string][]Event)
	h.handlers = make(map[string][]func(key string, value interface{}) error)
	h.jobs = make(map[string]*ScheduledJob)

	return nil
}

// GetQueue returns all events in a queue (for testing).
func (h *MemoryHandler) GetQueue(topic string) []Event {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.queues[topic]
}

// GetJob returns a scheduled job by ID (for testing).
func (h *MemoryHandler) GetJob(jobID string) (*ScheduledJob, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	job, exists := h.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}

	return job, nil
}
