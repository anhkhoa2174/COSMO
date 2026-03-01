package worker

import (
	"context"

	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// WorkerClientAdapter adapts pkg/worker.Client to WorkerClient interface
type WorkerClientAdapter struct {
	client *worker.Client
}

// NewWorkerClientAdapter creates a new worker client adapter
func NewWorkerClientAdapter(client *worker.Client) *WorkerClientAdapter {
	return &WorkerClientAdapter{client: client}
}

// EnqueueTask enqueues a background task
func (a *WorkerClientAdapter) EnqueueTask(ctx context.Context, taskType string, payload interface{}) error {
	_, err := a.client.EnqueueTask(ctx, taskType, payload)
	return err
}
