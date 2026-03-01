package gmailnotification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"

	"github.com/rockship/cosmo-agents-go/internal/domain"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	conversationRepo "github.com/rockship/cosmo-agents-go/internal/repository/conversation"
	emailRepo "github.com/rockship/cosmo-agents-go/internal/repository/email"
	taskRepo "github.com/rockship/cosmo-agents-go/internal/repository/task"
	workerpayloads "github.com/rockship/cosmo-agents-go/internal/worker/dto"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	googleoauth "github.com/rockship/cosmo-agents-go/pkg/oauth2/google"
	queueworker "github.com/rockship/cosmo-agents-go/pkg/worker"
)

// Worker processes Pub/Sub notifications coming from Gmail.
// Today it simply keeps agent history IDs in sync so that downstream sync jobs
// know where to resume.
type Worker struct {
	agentRepo        agentHistoryUpdater
	conversationRepo *conversationRepo.ConversationRepository
	emailRepo        *emailRepo.Repository
	contactRepo      *contactRepo.ContactRepository
	taskRepo         *taskRepo.TaskRepository
	oauthClient      *googleoauth.Client
	workerClient     lowPriorityEnqueuer
}

// New creates a new Gmail notification worker instance.
func New(
	agentRepo agentHistoryUpdater,
	conversationRepo *conversationRepo.ConversationRepository,
	emailRepo *emailRepo.Repository,
	contactRepo *contactRepo.ContactRepository,
	taskRepo *taskRepo.TaskRepository,
	oauthClient *googleoauth.Client,
	workerClient lowPriorityEnqueuer,
) *Worker {
	return &Worker{
		agentRepo:        agentRepo,
		conversationRepo: conversationRepo,
		emailRepo:        emailRepo,
		contactRepo:      contactRepo,
		taskRepo:         taskRepo,
		oauthClient:      oauthClient,
		workerClient:     workerClient,
	}
}

type agentHistoryUpdater interface {
	FindByEmail(ctx context.Context, email string) ([]domain.Agent, error)
	UpdateLastHistoryID(ctx context.Context, agentID uuid.UUID, historyID string) error
}

type lowPriorityEnqueuer interface {
	EnqueueLowPriorityTask(ctx context.Context, taskType string, payload interface{}, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

// GmailNotificationPayload represents the payload pushed by /v2/google/gmail/notifications.
type GmailNotificationPayload struct {
	EmailAddress string `json:"email_address"`
	HistoryID    string `json:"history_id"`
}

// UnmarshalJSON supports both string and numeric history IDs when decoding payloads.
func (p *GmailNotificationPayload) UnmarshalJSON(data []byte) error {
	type alias struct {
		EmailAddress string          `json:"email_address"`
		HistoryID    json.RawMessage `json:"history_id"`
	}

	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	p.EmailAddress = raw.EmailAddress
	if len(raw.HistoryID) == 0 || string(raw.HistoryID) == "null" {
		p.HistoryID = ""
		return nil
	}

	// Handle string history IDs
	if raw.HistoryID[0] == '"' {
		if err := json.Unmarshal(raw.HistoryID, &p.HistoryID); err != nil {
			return err
		}
		return nil
	}

	// Handle numeric history IDs without converting to float64 to avoid precision loss.
	var number json.Number
	if err := json.Unmarshal(raw.HistoryID, &number); err != nil {
		return fmt.Errorf("unsupported history_id type: %w", err)
	}

	p.HistoryID = number.String()

	return nil
}

// ProcessTask updates agent history IDs when Gmail notifies us about mailbox changes.
func (w *Worker) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload GmailNotificationPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal Gmail notification payload: %w", err)
	}

	email := strings.TrimSpace(strings.ToLower(payload.EmailAddress))
	if email == "" {
		logger.Logger.Warn().Msg("gmail notification ignored: email address missing")
		return nil
	}
	if payload.HistoryID == "" {
		logger.Logger.Warn().
			Str("email", email).
			Msg("gmail notification ignored: history ID missing")
		return nil
	}

	agents, err := w.agentRepo.FindByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to load agents for email %s: %w", email, err)
	}
	if len(agents) == 0 {
		logger.Logger.Warn().
			Str("email", email).
			Msg("gmail notification skipped: no agents with matching email")
		return nil
	}

	for i := range agents {
		agent := agents[i]
		if !shouldUpdateHistory(agent.LastHistoryID, payload.HistoryID) {
			continue
		}

		previousHistoryID := agent.LastHistoryID

		if err := w.agentRepo.UpdateLastHistoryID(ctx, agent.ID, payload.HistoryID); err != nil {
			return fmt.Errorf("failed to update agent %s history id: %w", agent.ID, err)
		}
		agent.LastHistoryID = payload.HistoryID

		logger.Logger.Info().
			Str("agent_id", agent.ID.String()).
			Str("email", email).
			Str("history_id", payload.HistoryID).
			Msg("updated agent history id from Gmail notification")

		// Kick off a sync job so we immediately pull new messages for this history_id.
		if w.workerClient != nil {
			syncPayload := workerpayloads.SyncGmailHistoryPayload{
				AgentID:        agent.ID,
				StartHistoryID: previousHistoryID,
			}
			taskID := fmt.Sprintf("sync-history-%s", agent.ID.String())
			if _, err := w.workerClient.EnqueueLowPriorityTask(
				ctx,
				queueworker.TypeSyncGmailHistory,
				syncPayload,
				asynq.TaskID(taskID),
			); err != nil {
				if errors.Is(err, asynq.ErrTaskIDConflict) {
					// Another sync is already queued/running for this agent; that's fine.
					logger.Logger.Debug().
						Str("agent_id", agent.ID.String()).
						Msg("Skipped enqueue of duplicate sync_history task")
					continue
				}
				logger.Logger.Warn().
					Err(err).
					Str("agent_id", agent.ID.String()).
					Msg("Failed to enqueue sync_history after Gmail notification")
				return fmt.Errorf("failed to enqueue gmail sync task: %w", err)
			}
		}
	}

	return nil
}

func shouldUpdateHistory(current, incoming string) bool {
	if incoming == "" {
		return false
	}
	if current == "" {
		return true
	}

	newID, err := strconv.ParseUint(incoming, 10, 64)
	if err != nil {
		logger.Logger.Warn().
			Err(err).
			Str("incoming_history_id", incoming).
			Msg("gmail notification: invalid incoming history id")
		return false
	}

	oldID, err := strconv.ParseUint(current, 10, 64)
	if err != nil {
		// If we can't parse the stored value, overwrite it with the new one.
		logger.Logger.Warn().
			Err(err).
			Str("current_history_id", current).
			Msg("gmail notification: invalid stored history id, overwriting")
		return true
	}

	return newID > oldID
}
