package daily_action

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	v1 "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	dailyActionSvc "github.com/rockship/cosmo-agents-go/internal/service/daily_action"
	"github.com/rockship/cosmo-agents-go/internal/worker/dto"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// GenerationWorker handles the daily_action:generate Asynq task.
type GenerationWorker struct {
	service    *dailyActionSvc.Service
	sseManager *dailyActionSvc.SSEManager
}

// NewGenerationWorker creates a new generation worker.
func NewGenerationWorker(service *dailyActionSvc.Service, sseManager ...*dailyActionSvc.SSEManager) *GenerationWorker {
	w := &GenerationWorker{service: service}
	if len(sseManager) > 0 {
		w.sseManager = sseManager[0]
	}
	return w
}

// HandleGenerateActions processes a daily_action:generate task.
func (w *GenerationWorker) HandleGenerateActions(ctx context.Context, task *asynq.Task) error {
	var payload dto.GenerateActionsPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	logger.Logger.Info().
		Str("task_type", worker.TypeDailyActionGenerate).
		Str("user_id", payload.UserID.String()).
		Str("language", payload.Language).
		Bool("force_refresh", payload.ForceRefresh).
		Msg("Processing daily action generation")

	gen, err := w.service.GenerateActions(ctx, payload.UserID, payload.Language, payload.ForceRefresh)
	if err != nil {
		logger.Error(err).
			Str("user_id", payload.UserID.String()).
			Msg("daily action generation failed")
		return fmt.Errorf("generate actions: %w", err)
	}

	logger.Logger.Info().
		Str("user_id", payload.UserID.String()).
		Str("generation_id", gen.ID.String()).
		Int("action_count", gen.ActionCount).
		Msg("Daily action generation completed")

	// Publish generation_complete SSE event
	if w.sseManager != nil {
		eventData, _ := json.Marshal(v1.SSEGenerationCompleteEvent{
			EventType:    "generation_complete",
			Timestamp:    time.Now(),
			GenerationID: gen.ID,
			ActionCount:  gen.ActionCount,
		})
		_ = w.sseManager.PublishEvent(ctx, payload.UserID, &dailyActionSvc.SSEEvent{
			EventType: "generation_complete",
			Data:      eventData,
		})
	}

	return nil
}
