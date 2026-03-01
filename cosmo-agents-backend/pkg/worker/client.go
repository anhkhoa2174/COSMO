package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

// Client wraps Asynq client for task enqueueing.
type Client struct {
	client *asynq.Client
}

// NewClient creates a new worker client.
func NewClient(cfg Config) *Client {
	return &Client{
		client: asynq.NewClient(cfg.ClientConfig()),
	}
}

// Close closes the client connection.
func (c *Client) Close() error {
	return c.client.Close()
}

// EnqueueTask enqueues a task with the given type, payload, and options.
func (c *Client) EnqueueTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	task := asynq.NewTask(taskType, payloadBytes)
	info, err := c.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue task: %w", err)
	}

	return info, nil
}

// EnqueueTaskIn enqueues a task to be processed after the given delay.
func (c *Client) EnqueueTaskIn(ctx context.Context, taskType string, payload interface{}, delay time.Duration, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	opts = append(opts, asynq.ProcessIn(delay))
	return c.EnqueueTask(ctx, taskType, payload, opts...)
}

// EnqueueTaskAt enqueues a task to be processed at the given time.
func (c *Client) EnqueueTaskAt(ctx context.Context, taskType string, payload interface{}, processAt time.Time, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	opts = append(opts, asynq.ProcessAt(processAt))
	return c.EnqueueTask(ctx, taskType, payload, opts...)
}

// EnqueueTaskWithRetry enqueues a task with custom retry settings.
func (c *Client) EnqueueTaskWithRetry(ctx context.Context, taskType string, payload interface{}, maxRetry int, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	opts = append(opts, asynq.MaxRetry(maxRetry))
	return c.EnqueueTask(ctx, taskType, payload, opts...)
}

// EnqueueCriticalTask enqueues a task in the critical priority queue.
func (c *Client) EnqueueCriticalTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	opts = append(opts, asynq.Queue(QueueCritical))
	return c.EnqueueTask(ctx, taskType, payload, opts...)
}

// EnqueueLowPriorityTask enqueues a task in the low priority queue.
func (c *Client) EnqueueLowPriorityTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	opts = append(opts, asynq.Queue(QueueLow))
	return c.EnqueueTask(ctx, taskType, payload, opts...)
}
