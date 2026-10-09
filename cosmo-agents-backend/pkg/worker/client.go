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

// aiTaskMaxRetry caps retries for task types that call a paid LLM (or an
// embedding endpoint) on every attempt. Asynq's default of 25 turns a task
// that fails deterministically — a malformed prompt, an oversized email, a
// model the account cannot access — into 25 billed calls that all fail the
// same way. Three attempts still absorbs a transient rate-limit or timeout.
var aiTaskMaxRetry = map[string]int{
	"email:handle_reply":               3,
	"campaign:generate_campaign_email": 3,
	"campaign:generate_reply":          3,
	"contact:enrich":                   3,
	"knowledge:summarize":              3,
	"knowledge:indexing":               3,
	"email:summarize":                  3,
	"conversation:summarize":           3,
	"email:indexing":                   3,
}

// hasOption reports whether the caller already supplied an option of this kind,
// so an explicit choice is never overridden by a default.
func hasOption(opts []asynq.Option, t asynq.OptionType) bool {
	for _, o := range opts {
		if o.Type() == t {
			return true
		}
	}
	return false
}

// EnqueueTask enqueues a task with the given type, payload, and options.
func (c *Client) EnqueueTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	if retry, ok := aiTaskMaxRetry[taskType]; ok && !hasOption(opts, asynq.MaxRetryOpt) {
		opts = append(opts, asynq.MaxRetry(retry))
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
