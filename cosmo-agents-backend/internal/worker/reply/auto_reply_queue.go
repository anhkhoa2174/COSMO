package reply

import (
	"context"

	"github.com/hibiken/asynq"

	"github.com/rockship/cosmo-agents-go/internal/service/autoreply"
	workerpayloads "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	queueworker "github.com/rockship/cosmo-agents-go/pkg/worker"
)

// taskEnqueuer is the subset of the queue client this adapter needs.
type taskEnqueuer interface {
	EnqueueTask(ctx context.Context, taskType string, payload interface{},
		opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// QueueSender hands an auto-reply to the same task a human "approve and send"
// uses.
//
// The adapter exists so the autoreply package stays free of the queue and the
// payload types, and so there is exactly one send path. Auto-replies inherit
// the mail worker's retries, rate limiting, threading and logging rather than
// getting a parallel implementation that would drift from it.
type QueueSender struct {
	client taskEnqueuer
}

func NewQueueSender(client taskEnqueuer) *QueueSender {
	return &QueueSender{client: client}
}

func (q *QueueSender) EnqueueSend(ctx context.Context, in autoreply.SendRequest) error {
	_, err := q.client.EnqueueTask(ctx, queueworker.TypeSendEmail,
		workerpayloads.SendEmailPayload{
			AgentID:    in.AgentID,
			ContactID:  in.ContactID,
			CampaignID: in.CampaignID,
			To:         in.To,
			Subject:    in.Subject,
			Body:       in.Body,
			IsHTML:     true,
			InReplyTo:  in.InReplyTo,
		})
	return err
}
