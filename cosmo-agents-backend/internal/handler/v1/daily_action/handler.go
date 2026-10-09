package daily_action

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	"github.com/rockship/cosmo-agents-go/internal/handler"
	"github.com/rockship/cosmo-agents-go/internal/mapper"
	"github.com/rockship/cosmo-agents-go/internal/middleware"
	dailyActionRepo "github.com/rockship/cosmo-agents-go/internal/repository/daily_action"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	"github.com/rockship/cosmo-agents-go/internal/schema"
	v1 "github.com/rockship/cosmo-agents-go/internal/schema/v1"
	dailyActionSvc "github.com/rockship/cosmo-agents-go/internal/service/daily_action"
	"github.com/rockship/cosmo-agents-go/internal/worker/dto"
	pkgAuth "github.com/rockship/cosmo-agents-go/pkg/auth"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"github.com/rockship/cosmo-agents-go/pkg/worker"
)

// Handler handles daily action HTTP requests.
type Handler struct {
	generationRepo    *dailyActionRepo.GenerationRepository
	actionRepo        *dailyActionRepo.ActionRepository
	snoozeRepo        *dailyActionRepo.SnoozeRepository
	completionLogRepo *dailyActionRepo.CompletionLogRepository
	sseEventRepo      *dailyActionRepo.SSEEventRepository
	chatMessageRepo   *dailyActionRepo.ChatMessageRepository
	service           *dailyActionSvc.Service
	sseManager        *dailyActionSvc.SSEManager
	workerClient      *worker.Client
	jwtManager        *pkgAuth.JWTManager
	authHelper        *middleware.AuthHelper
	responseHelper    *handler.ResponseHelper
}

// NewHandler creates a new daily action handler.
func NewHandler(
	generationRepo *dailyActionRepo.GenerationRepository,
	actionRepo *dailyActionRepo.ActionRepository,
	snoozeRepo *dailyActionRepo.SnoozeRepository,
	completionLogRepo *dailyActionRepo.CompletionLogRepository,
	sseEventRepo *dailyActionRepo.SSEEventRepository,
	chatMessageRepo *dailyActionRepo.ChatMessageRepository,
	userRepo *userRepo.UserRepository,
	workerClient *worker.Client,
	service *dailyActionSvc.Service,
	sseManager *dailyActionSvc.SSEManager,
	jwtManager *pkgAuth.JWTManager,
) *Handler {
	return &Handler{
		generationRepo:    generationRepo,
		actionRepo:        actionRepo,
		snoozeRepo:        snoozeRepo,
		completionLogRepo: completionLogRepo,
		sseEventRepo:      sseEventRepo,
		chatMessageRepo:   chatMessageRepo,
		service:           service,
		sseManager:        sseManager,
		workerClient:      workerClient,
		jwtManager:        jwtManager,
		authHelper:        middleware.NewAuthHelper(userRepo, nil),
		responseHelper:    handler.NewResponseHelper(),
	}
}

// Generate triggers daily actions generation.
// @Summary Generate daily actions
// @Description Triggers async generation of daily BD actions for the authenticated user. Returns immediately with a generation_id while the pipeline runs in the background.
// @Tags Daily Actions
// @Accept json
// @Produce json
// @Param body body v1.GenerateRequest false "Generation parameters"
// @Success 202 {object} schema.APIResponse[v1.GenerateResponseData] "Generation started or cached"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 429 {object} schema.APIResponse[any] "Generation already in progress"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/daily-actions/generate [post]
func (h *Handler) Generate(c fiber.Ctx) error {
	start := time.Now()
	defer func() {
		dailyActionSvc.HTTPRequestDuration.WithLabelValues("generate").Observe(time.Since(start).Seconds())
	}()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req v1.GenerateRequest
	if err := c.Bind().JSON(&req); err != nil {
		// Request body is optional — use defaults
		req = v1.GenerateRequest{Language: "vi"}
	}
	if req.Language == "" {
		req.Language = "vi"
	}

	logger.Logger.Info().
		Str("user_id", userID.String()).
		Str("language", req.Language).
		Bool("force_refresh", req.ForceRefresh).
		Msg("Generate daily actions requested")

	ctx := c.Context()
	today := time.Now().Format("2006-01-02")

	// Check if generation is already in progress
	active, err := h.generationRepo.FindActiveByUserAndDate(ctx, userID, today)
	if err != nil {
		logger.Error(err).Str("user_id", userID.String()).Msg("Failed to check generation status")
		return h.responseHelper.InternalServerError(c, "Failed to check generation status", err)
	}
	if active != nil && !req.ForceRefresh {
		logger.Logger.Info().Str("user_id", userID.String()).Str("generation_id", active.ID.String()).Msg("Generation already in progress")
		return c.Status(fiber.StatusTooManyRequests).JSON(schema.ErrorResponse(
			fiber.StatusTooManyRequests,
			"Generation already in progress",
			active.ID.String(),
		))
	}

	// Check for cached (ready) generation if not force_refresh
	if !req.ForceRefresh {
		existing, err := h.generationRepo.FindLatestByUserAndDate(ctx, userID, today)
		if err != nil {
			return h.responseHelper.InternalServerError(c, "Failed to check existing generation", err)
		}
		if existing != nil && existing.Status == domain.GenerationStatusReady {
			return c.Status(fiber.StatusAccepted).JSON(schema.SuccessResponse(v1.GenerateResponseData{
				GenerationID:     &existing.ID,
				GenerationStatus: string(existing.Status),
			}))
		}
	}

	// Enqueue Asynq task for async generation
	payload := dto.GenerateActionsPayload{
		UserID:       userID,
		Language:     req.Language,
		ForceRefresh: req.ForceRefresh,
	}
	_, err = h.workerClient.EnqueueTask(ctx, worker.TypeDailyActionGenerate, payload)
	if err != nil {
		logger.Error(err).Str("user_id", userID.String()).Msg("Failed to enqueue generation task")
		return h.responseHelper.InternalServerError(c, "Failed to enqueue generation task", err)
	}

	logger.Logger.Info().Str("user_id", userID.String()).Msg("Daily action generation task enqueued")

	return c.Status(fiber.StatusAccepted).JSON(schema.SuccessResponse(v1.GenerateResponseData{
		GenerationStatus: "started",
	}))
}

// GetDailyActions returns today's daily actions briefing.
// @Summary Get daily actions briefing
// @Description Returns the full DailyActionsBriefing with categories, agent briefing, progress, and pipeline summary for the authenticated user.
// @Tags Daily Actions
// @Produce json
// @Param language query string false "Response language (vi or en)" default(vi)
// @Param include_completed query bool false "Include completed/skipped actions" default(false)
// @Success 200 {object} schema.APIResponse[v1.DailyActionsBriefing] "Daily actions briefing"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/daily-actions [get]
func (h *Handler) GetDailyActions(c fiber.Ctx) error {
	start := time.Now()
	defer func() {
		dailyActionSvc.HTTPRequestDuration.WithLabelValues("get_daily_actions").Observe(time.Since(start).Seconds())
	}()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	ctx := c.Context()
	today := time.Now().Format("2006-01-02")
	language := c.Query("language", "vi")
	includeCompleted := c.Query("include_completed", "false") == "true"

	logger.Logger.Debug().
		Str("user_id", userID.String()).
		Str("language", language).
		Bool("include_completed", includeCompleted).
		Msg("Get daily actions requested")

	// Find latest generation for today
	gen, err := h.generationRepo.FindLatestByUserAndDate(ctx, userID, today)
	if err != nil {
		logger.Error(err).Str("user_id", userID.String()).Msg("Failed to find generation")
		return h.responseHelper.InternalServerError(c, "Failed to find generation", err)
	}

	// No generation yet
	if gen == nil {
		return h.responseHelper.Success(c, v1.DailyActionsBriefing{
			GenerationStatus: "not_generated",
			Date:             today,
			Language:         language,
			Categories:       []v1.ActionCategoryResponse{},
		})
	}

	// Check staleness
	if gen.Status == domain.GenerationStatusReady && h.service != nil {
		stale, _ := h.service.CheckStaleness(ctx, userID, gen)
		if stale {
			gen.Status = domain.GenerationStatusStale
		}
	}

	// If still generating, return status without actions
	if gen.Status == domain.GenerationStatusStarted || gen.Status == domain.GenerationStatusGenerating {
		return h.responseHelper.Success(c, v1.DailyActionsBriefing{
			GenerationID:     &gen.ID,
			GenerationStatus: string(gen.Status),
			Date:             today,
			Language:         language,
			Categories:       []v1.ActionCategoryResponse{},
		})
	}

	// Build categories with first 5 actions per category
	categoryOrder := []domain.CategoryID{
		domain.CategoryReplied,
		domain.CategoryFollowup,
		domain.CategoryNewOutreach,
		domain.CategoryMeetingPrep,
		domain.CategoryEnrichment,
	}
	type categoryMeta struct {
		label       string
		icon        string
		color       string
		description string
	}
	categoryMetadata := map[domain.CategoryID]categoryMeta{
		domain.CategoryReplied:     {"Replied", "💬", "green", "Prospects who replied to your outreach"},
		domain.CategoryFollowup:    {"Follow-up", "⏰", "orange", "Follow-ups due based on your cadence"},
		domain.CategoryNewOutreach: {"New Outreach", "📤", "blue", "New contacts ready for first outreach"},
		domain.CategoryMeetingPrep: {"Meeting Prep", "📅", "purple", "Upcoming meetings needing preparation"},
		domain.CategoryEnrichment:  {"Enrichment", "🔍", "red", "Contacts with missing data to enrich"},
	}

	const perCategory = 5
	var categories []v1.ActionCategoryResponse

	for _, catID := range categoryOrder {
		actions, total, err := h.actionRepo.FindByCategoryWithPagination(
			ctx, gen.ID, string(catID), 0, perCategory, includeCompleted,
		)
		if err != nil {
			continue
		}
		if total == 0 {
			continue
		}

		mapped := mapper.DailyActionsToResponse(actions)
		nextOffset := perCategory
		meta := categoryMetadata[catID]
		categories = append(categories, v1.ActionCategoryResponse{
			ID:          string(catID),
			Label:       meta.label,
			Icon:        meta.icon,
			Color:       meta.color,
			Description: meta.description,
			TotalCount:  int(total),
			Actions:     mapped,
			HasMore:     total > int64(perCategory),
			NextOffset:  &nextOffset,
		})
	}

	if categories == nil {
		categories = []v1.ActionCategoryResponse{}
	}

	// Build progress
	statusCounts, _ := h.actionRepo.CountByGenerationAndStatus(ctx, gen.ID)
	total := 0
	completed := 0
	skipped := 0
	snoozed := 0
	for st, cnt := range statusCounts {
		total += cnt
		switch st {
		case domain.ActionStatusCompleted:
			completed += cnt
		case domain.ActionStatusSkipped:
			skipped += cnt
		case domain.ActionStatusSnoozed:
			snoozed += cnt
		}
	}

	// Build agent briefing from generation JSONB
	var briefing *v1.AgentBriefingResponse
	var agentBriefingData domain.AgentBriefingData
	if err := gen.AgentBriefing.Unmarshal(&agentBriefingData); err == nil && agentBriefingData.Greeting != "" {
		briefing = mapper.AgentBriefingToResponse(&agentBriefingData)
	}

	// Build pipeline summary from generation JSONB
	var pipelineSummary *v1.PipelineSummaryResponse
	var summaryData dailyActionSvc.FrontendPipelineSummary
	if err := gen.PipelineSummary.Unmarshal(&summaryData); err == nil && summaryData.TotalActiveContacts > 0 {
		pipelineSummary = &v1.PipelineSummaryResponse{
			TotalActiveContacts:  summaryData.TotalActiveContacts,
			ContactsByStage:      summaryData.ContactsByStage,
			ContactsByLifecycle:  summaryData.ContactsByLifecycle,
			ResponseRate7d:       summaryData.ResponseRate7d,
			MeetingsBooked7d:     summaryData.MeetingsBooked7d,
			AvgResponseTimeHours: summaryData.AvgResponseTimeHours,
		}
	}

	// Compute completion rate
	var completionRate float64
	if total > 0 {
		completionRate = float64(completed) / float64(total)
	}

	return h.responseHelper.Success(c, v1.DailyActionsBriefing{
		GenerationID:     &gen.ID,
		GenerationStatus: string(gen.Status),
		GeneratedAt:      gen.GeneratedAt,
		Date:             gen.Date,
		Language:         language,
		AgentBriefing:    briefing,
		Categories:       categories,
		PipelineSummary:  pipelineSummary,
		Progress: &v1.ActionProgressResponse{
			Total:          total,
			Completed:      completed,
			Skipped:        skipped,
			Snoozed:        snoozed,
			Remaining:      total - completed - skipped,
			CompletionRate: completionRate,
		},
	})
}

// UpdateAction updates an action's state.
// @Summary Update action state
// @Description Executes a state transition on a daily action (mark_sent, skip, snooze, snooze_custom, mark_completed, reopen). Returns the updated action, optional contact state change, and progress.
// @Tags Daily Actions
// @Accept json
// @Produce json
// @Param action_id path string true "Action UUID"
// @Param body body v1.UpdateActionRequest true "Transition request"
// @Success 200 {object} schema.APIResponse[v1.UpdateActionResponseData] "Action updated"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 404 {object} schema.APIResponse[any] "Action not found"
// @Failure 409 {object} schema.APIResponse[any] "Invalid state transition"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/daily-actions/{action_id} [patch]
func (h *Handler) UpdateAction(c fiber.Ctx) error {
	start := time.Now()
	defer func() {
		dailyActionSvc.HTTPRequestDuration.WithLabelValues("update_action").Observe(time.Since(start).Seconds())
	}()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	actionIDStr := c.Params("action_id")
	actionID, err := parseUUID(actionIDStr)
	if err != nil {
		return h.responseHelper.BadRequest(c, "Invalid action_id", err)
	}

	var req v1.UpdateActionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	logger.Logger.Info().
		Str("user_id", userID.String()).
		Str("action_id", actionIDStr).
		Str("transition", req.Transition).
		Msg("Update action requested")

	ctx := c.Context()

	// Find action and verify ownership
	action, err := h.service.FindActionByIDAndUser(ctx, userID, actionID)
	if err != nil {
		logger.Error(err).Str("user_id", userID.String()).Str("action_id", actionIDStr).Msg("Failed to find action")
		return h.responseHelper.InternalServerError(c, "Failed to find action", err)
	}
	if action == nil {
		logger.Logger.Warn().Str("user_id", userID.String()).Str("action_id", actionIDStr).Msg("Action not found")
		return h.responseHelper.NotFound(c, "Action not found", fmt.Errorf("action %s not found for user", actionIDStr))
	}

	// Validate and execute transition
	transition := domain.Transition(req.Transition)
	if !domain.IsValidTransition(action.Status, transition) {
		logger.Logger.Warn().
			Str("user_id", userID.String()).
			Str("action_id", actionIDStr).
			Str("current_status", string(action.Status)).
			Str("transition", req.Transition).
			Msg("Invalid state transition")
		return c.Status(fiber.StatusConflict).JSON(schema.ErrorResponse(
			fiber.StatusConflict,
			"Invalid state transition",
			map[string]string{
				"current_status": string(action.Status),
				"transition":     req.Transition,
			},
		))
	}

	result, err := h.service.ExecuteTransition(
		ctx, action, transition,
		req.Content, req.Channel, req.SkipReason, req.SnoozeUntil, req.FeedbackAction,
	)
	if err != nil {
		logger.Error(err).Str("user_id", userID.String()).Str("action_id", actionIDStr).Str("transition", req.Transition).Msg("Failed to execute transition")
		return h.responseHelper.InternalServerError(c, "Failed to execute transition", err)
	}

	logger.Logger.Info().
		Str("user_id", userID.String()).
		Str("action_id", actionIDStr).
		Str("transition", req.Transition).
		Str("new_status", string(result.Action.Status)).
		Msg("Action transition executed")

	// Build response
	actionResp := mapper.DailyActionToResponse(result.Action)

	// Get updated progress
	var progress *v1.ActionProgressResponse
	{
		statusCounts, _ := h.actionRepo.CountByGenerationAndStatus(ctx, action.GenerationID)
		total, completed, skipped, snoozed := 0, 0, 0, 0
		for st, cnt := range statusCounts {
			total += cnt
			switch st {
			case domain.ActionStatusCompleted:
				completed += cnt
			case domain.ActionStatusSkipped:
				skipped += cnt
			case domain.ActionStatusSnoozed:
				snoozed += cnt
			}
		}
		progress = &v1.ActionProgressResponse{
			Total:     total,
			Completed: completed,
			Skipped:   skipped,
			Snoozed:   snoozed,
			Remaining: total - completed - skipped,
		}
	}

	// Publish action_updated SSE event
	if h.sseManager != nil {
		eventData, _ := json.Marshal(v1.SSEActionUpdatedEvent{
			EventType: "action_updated",
			Timestamp: time.Now(),
			ActionID:  action.ID,
			NewStatus: string(result.Action.Status),
		})
		_ = h.sseManager.PublishEvent(ctx, userID, &dailyActionSvc.SSEEvent{
			EventType: "action_updated",
			Data:      eventData,
		})
	}

	return h.responseHelper.Success(c, v1.UpdateActionResponseData{
		Action:             actionResp,
		ContactStateChange: result.ContactChange,
		Progress:           progress,
	})
}

// SSEStream opens an SSE event stream.
// @Summary Open SSE event stream
// @Description Opens a Server-Sent Events stream for real-time daily action updates (generation_complete, action_updated, prospect_replied, meeting_approaching, snooze_expired).
// @Tags Daily Actions
// @Produce text/event-stream
// @Param Last-Event-ID header string false "Last event ID for reconnection replay"
// @Success 200 {string} string "SSE event stream"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/daily-actions/events [get]
func (h *Handler) SSEStream(c fiber.Ctx) error {
	dailyActionSvc.HTTPRequestDuration.WithLabelValues("sse_stream").Observe(0) // SSE is long-lived, track connection count instead

	// Try normal auth first (when running behind middleware)
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		// Fallback: parse JWT from query param token (SSE runs before middleware to avoid buffering)
		if token := c.Query("token"); token != "" && h.jwtManager != nil {
			claims, jwtErr := h.jwtManager.ValidateToken(token)
			if jwtErr == nil {
				userID = claims.UserID
				err = nil
			}
		}
	}
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	if h.sseManager == nil {
		logger.Logger.Error().Str("user_id", userID.String()).Msg("SSE manager not configured")
		return h.responseHelper.InternalServerError(c, "SSE not configured", fmt.Errorf("SSE manager is nil"))
	}

	logger.Logger.Info().Str("user_id", userID.String()).Msg("SSE stream opened")

	// Read Last-Event-ID header BEFORE entering stream writer (c is not safe inside callback)
	lastEventID := c.Get("Last-Event-ID")

	ctx := c.Context()

	// Register for events
	eventCh := h.sseManager.Register(ctx, userID)

	// Set SSE headers via raw fasthttp (avoids Fiber middleware buffering)
	fctx := c.RequestCtx()
	fctx.Response.Header.Set("Content-Type", "text/event-stream")
	fctx.Response.Header.Set("Cache-Control", "no-cache")
	fctx.Response.Header.Set("Connection", "keep-alive")
	fctx.Response.Header.Set("X-Accel-Buffering", "no")
	fctx.Response.Header.Set("Access-Control-Allow-Origin", "*")

	// Use raw fasthttp SetBodyStreamWriter (bypasses Fiber buffering)
	fctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		dailyActionSvc.SSEConnectionsActive.Inc()
		defer dailyActionSvc.SSEConnectionsActive.Dec()
		defer h.sseManager.Unregister(userID, eventCh)

		// Send initial retry interval
		fmt.Fprintf(w, "retry: 3000\n\n")
		w.Flush()

		// Replay missed events if Last-Event-ID is provided
		if lastEventID != "" {
			if sinceID, err := uuid.Parse(lastEventID); err == nil {
				missedEvents, err := h.sseEventRepo.FindByUserSince(ctx, userID, sinceID)
				if err != nil {
					logger.Error(err).Str("last_event_id", lastEventID).Msg("failed to replay SSE events")
				} else {
					for _, event := range missedEvents {
						fmt.Fprintf(w, "id: %s\n", event.ID)
						fmt.Fprintf(w, "event: %s\n", event.EventType)
						fmt.Fprintf(w, "data: %s\n\n", string(event.Payload))
						if err := w.Flush(); err != nil {
							return
						}
					}
					if len(missedEvents) > 0 {
						logger.Logger.Info().
							Str("user_id", userID.String()).
							Str("last_event_id", lastEventID).
							Int("replayed", len(missedEvents)).
							Msg("SSE events replayed on reconnect")
					}
				}
			} else {
				logger.Logger.Warn().Str("last_event_id", lastEventID).Msg("invalid Last-Event-ID format, skipping replay")
			}
		}

		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case event, ok := <-eventCh:
				if !ok {
					return
				}
				fmt.Fprintf(w, "id: %s\n", event.ID)
				fmt.Fprintf(w, "event: %s\n", event.EventType)
				fmt.Fprintf(w, "data: %s\n\n", string(event.Data))
				if err := w.Flush(); err != nil {
					return
				}
			case <-heartbeat.C:
				fmt.Fprintf(w, ": heartbeat\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
	return nil
}

// GetSummary returns the daily progress summary.
// @Summary Get daily summary
// @Description Returns completion stats, breakdown by action type, outcomes, carry-over items, and an AI-generated summary narrative.
// @Tags Daily Actions
// @Produce json
// @Param language query string false "Summary language (vi or en)" default(vi)
// @Success 200 {object} schema.APIResponse[v1.DailySummaryResponseData] "Daily summary"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/daily-actions/summary [get]
func (h *Handler) GetSummary(c fiber.Ctx) error {
	start := time.Now()
	defer func() {
		dailyActionSvc.HTTPRequestDuration.WithLabelValues("get_summary").Observe(time.Since(start).Seconds())
	}()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	ctx := c.Context()
	today := time.Now().Format("2006-01-02")

	logger.Logger.Debug().Str("user_id", userID.String()).Msg("Get daily summary requested")

	// Find latest generation for today
	gen, err := h.generationRepo.FindLatestByUserAndDate(ctx, userID, today)
	if err != nil {
		logger.Error(err).Str("user_id", userID.String()).Msg("Failed to find generation for summary")
		return h.responseHelper.InternalServerError(c, "Failed to find generation", err)
	}

	// Build progress from action status counts
	var progress v1.ActionProgressResponse
	if gen != nil {
		statusCounts, _ := h.actionRepo.CountByGenerationAndStatus(ctx, gen.ID)
		total, completed, skipped, snoozed := 0, 0, 0, 0
		for st, cnt := range statusCounts {
			total += cnt
			switch st {
			case domain.ActionStatusCompleted:
				completed += cnt
			case domain.ActionStatusSkipped:
				skipped += cnt
			case domain.ActionStatusSnoozed:
				snoozed += cnt
			}
		}
		remaining := total - completed - skipped
		var completionRate float64
		if total > 0 {
			completionRate = float64(completed) / float64(total) * 100
		}
		progress = v1.ActionProgressResponse{
			Total:          total,
			Completed:      completed,
			Skipped:        skipped,
			Snoozed:        snoozed,
			Remaining:      remaining,
			CompletionRate: completionRate,
		}
	}

	// Build breakdown from completion logs by action type (mark_sent transitions)
	var breakdown v1.SummaryBreakdown
	markSentByType, _ := h.completionLogRepo.CountByUserDateAndActionType(ctx, userID, today, string(domain.TransitionMarkSent))
	breakdown.OutreachSent = markSentByType[string(domain.ActionTypeOutreach)]
	breakdown.FollowupsSent = markSentByType[string(domain.ActionTypeFollowup)]
	breakdown.RepliesHandled = markSentByType[string(domain.ActionTypeRespond)]
	breakdown.MeetingsPrepped = markSentByType[string(domain.ActionTypeMeetingPrep)]
	breakdown.ContactsEnriched = markSentByType[string(domain.ActionTypeEnrich)]

	// Also count mark_completed transitions
	markCompletedByType, _ := h.completionLogRepo.CountByUserDateAndActionType(ctx, userID, today, string(domain.TransitionMarkCompleted))
	breakdown.MeetingsPrepped += markCompletedByType[string(domain.ActionTypeMeetingPrep)]
	breakdown.ContactsEnriched += markCompletedByType[string(domain.ActionTypeEnrich)]

	// Build carry-over: snoozed actions
	var carryOver []v1.CarryOverItem
	if gen != nil {
		snoozedActions, _, _ := h.actionRepo.FindByCategoryWithPagination(ctx, gen.ID, "", 0, 20, true)
		for _, a := range snoozedActions {
			if a.Status == domain.ActionStatusSnoozed {
				var snapshot domain.ContactSnapshot
				_ = a.ContactSnapshot.Unmarshal(&snapshot)
				carryOver = append(carryOver, v1.CarryOverItem{
					ActionID:    a.ID,
					Type:        string(a.Type),
					ContactName: snapshot.Name,
					Reason:      "snoozed",
				})
			}
		}
	}

	return h.responseHelper.Success(c, v1.DailySummaryResponseData{
		Date:         today,
		AgentSummary: buildSummaryNarrative(progress, breakdown),
		Progress:     progress,
		Breakdown:    breakdown,
		Outcomes:     v1.SummaryOutcomes{},
		CarryOver:    carryOver,
	})
}

// buildSummaryNarrative generates a simple text summary of the day's progress.
func buildSummaryNarrative(progress v1.ActionProgressResponse, breakdown v1.SummaryBreakdown) string {
	if progress.Total == 0 {
		return "No actions generated for today yet."
	}
	return fmt.Sprintf(
		"Completed %d of %d actions (%.0f%%). Sent %d outreach, %d follow-ups, handled %d replies.",
		progress.Completed, progress.Total, progress.CompletionRate,
		breakdown.OutreachSent, breakdown.FollowupsSent, breakdown.RepliesHandled,
	)
}

// LoadMoreActions returns paginated actions for a category.
// @Summary Load more actions by category
// @Description Returns paginated actions within a category sorted by priority. Supports offset-based pagination.
// @Tags Daily Actions
// @Produce json
// @Param category_id path string true "Category ID (replied, followup, new_outreach, meeting_prep, enrichment)"
// @Param offset query int false "Pagination offset" default(0)
// @Param limit query int false "Pagination limit (max 50)" default(5)
// @Param include_completed query bool false "Include completed/skipped actions" default(false)
// @Success 200 {object} schema.APIResponse[v1.LoadMoreActionsResponseData] "Paginated actions"
// @Failure 400 {object} schema.APIResponse[any] "Invalid category_id"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/daily-actions/categories/{category_id}/actions [get]
func (h *Handler) LoadMoreActions(c fiber.Ctx) error {
	start := time.Now()
	defer func() {
		dailyActionSvc.HTTPRequestDuration.WithLabelValues("load_more_actions").Observe(time.Since(start).Seconds())
	}()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	ctx := c.Context()
	categoryID := c.Params("category_id")

	logger.Logger.Debug().
		Str("user_id", userID.String()).
		Str("category_id", categoryID).
		Msg("Load more actions requested")

	// Validate category_id
	validCategories := map[string]bool{
		string(domain.CategoryReplied):     true,
		string(domain.CategoryFollowup):    true,
		string(domain.CategoryNewOutreach): true,
		string(domain.CategoryMeetingPrep): true,
		string(domain.CategoryEnrichment):  true,
	}
	if !validCategories[categoryID] {
		return h.responseHelper.BadRequest(c, "Invalid category_id", fmt.Errorf("unknown category: %s", categoryID))
	}

	// Parse pagination params
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "5"))
	if limit <= 0 || limit > 50 {
		limit = 5
	}
	includeCompleted := c.Query("include_completed", "false") == "true"

	today := time.Now().Format("2006-01-02")

	// Find latest generation
	gen, err := h.generationRepo.FindLatestByUserAndDate(ctx, userID, today)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to find generation", err)
	}
	if gen == nil {
		return h.responseHelper.Success(c, v1.LoadMoreActionsResponseData{
			Actions: []v1.DailyActionResponse{},
			HasMore: false,
		})
	}

	// Fetch actions with pagination
	actions, total, err := h.actionRepo.FindByCategoryWithPagination(
		ctx, gen.ID, categoryID, offset, limit, includeCompleted,
	)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to load actions", err)
	}

	mapped := mapper.DailyActionsToResponse(actions)
	hasMore := int64(offset+limit) < total
	var nextOffset *int
	if hasMore {
		next := offset + limit
		nextOffset = &next
	}

	return h.responseHelper.Success(c, v1.LoadMoreActionsResponseData{
		Actions:    mapped,
		HasMore:    hasMore,
		NextOffset: nextOffset,
	})
}

// Chat handles conversational interaction with the BD Agent.
// @Summary Chat with BD Agent
// @Description Sends messages to the BD Agent and receives streamed context-aware responses following the Vercel AI SDK data stream protocol.
// @Tags Daily Actions
// @Accept json
// @Produce plain
// @Param body body v1.ChatRequest true "Chat messages (Vercel AI SDK format)"
// @Success 200 {string} string "Vercel AI SDK data stream"
// @Failure 400 {object} schema.APIResponse[any] "Invalid request"
// @Failure 401 {object} schema.APIResponse[any] "Unauthorized"
// @Failure 500 {object} schema.APIResponse[any] "Internal server error"
// @Security BearerAuth
// @Router /v1/daily-actions/chat [post]
func (h *Handler) Chat(c fiber.Ctx) error {
	start := time.Now()
	defer func() {
		dailyActionSvc.HTTPRequestDuration.WithLabelValues("chat").Observe(time.Since(start).Seconds())
	}()
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req v1.ChatRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	logger.Logger.Info().
		Str("user_id", userID.String()).
		Int("message_count", len(req.Messages)).
		Msg("Chat request received")

	ctx := c.Context()

	// Find current generation for context
	today := time.Now().Format("2006-01-02")
	gen, _ := h.generationRepo.FindLatestByUserAndDate(ctx, userID, today)

	var generationID *uuid.UUID
	if gen != nil {
		generationID = &gen.ID
	}

	// Save user message(s) to chat_messages table
	for _, msg := range req.Messages {
		if msg.Role == "user" {
			chatMsg := &domain.ChatMessage{
				UserID:       userID,
				Role:         msg.Role,
				Content:      msg.Content,
				GenerationID: generationID,
			}
			if _, err := h.chatMessageRepo.Create(ctx, chatMsg); err != nil {
				logger.Error(err).Msg("failed to save chat message")
			}
		}
	}

	// Build context-aware response
	lastUserMsg := ""
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			lastUserMsg = req.Messages[i].Content
			break
		}
	}

	// Generate a context-aware response
	responseContent := h.service.GenerateChatResponse(ctx, userID, lastUserMsg, gen)
	dailyActionSvc.ChatTTFT.Observe(time.Since(start).Seconds())

	// Save assistant message
	assistantMsg := &domain.ChatMessage{
		UserID:       userID,
		Role:         "assistant",
		Content:      responseContent,
		GenerationID: generationID,
	}
	if _, err := h.chatMessageRepo.Create(ctx, assistantMsg); err != nil {
		logger.Error(err).Msg("failed to save assistant message")
	}

	// Stream response using Vercel AI SDK data stream protocol
	messageID := uuid.New().String()

	c.Set("Content-Type", "text/plain; charset=utf-8")
	c.Set("X-Vercel-AI-Data-Stream", "v1")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	c.RequestCtx().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Start frame
		fmt.Fprintf(w, "f:{\"messageId\":\"%s\"}\n", messageID)
		w.Flush()

		// Stream the response content in chunks
		// Split into word-level chunks for natural streaming
		chunks := splitIntoChunks(responseContent, 4)
		for _, chunk := range chunks {
			jsonChunk, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "0:%s\n", jsonChunk)
			w.Flush()
		}

		// Finish step and message frames
		fmt.Fprintf(w, "e:{\"finishReason\":\"stop\",\"isContinued\":false}\n")
		fmt.Fprintf(w, "d:{\"finishReason\":\"stop\"}\n")
		w.Flush()
	})

	return nil
}

// splitIntoChunks splits text into chunks of approximately n words each.
func splitIntoChunks(text string, wordsPerChunk int) []string {
	words := splitWords(text)
	if len(words) == 0 {
		return []string{text}
	}

	var chunks []string
	for i := 0; i < len(words); i += wordsPerChunk {
		end := i + wordsPerChunk
		if end > len(words) {
			end = len(words)
		}
		chunk := ""
		for j := i; j < end; j++ {
			if j > i {
				chunk += " "
			}
			chunk += words[j]
		}
		// Add trailing space between chunks (except last)
		if end < len(words) {
			chunk += " "
		}
		chunks = append(chunks, chunk)
	}
	return chunks
}

// splitWords splits text into words preserving punctuation.
func splitWords(text string) []string {
	var words []string
	current := ""
	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\t' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
			// Preserve newlines as separate tokens
			if r == '\n' {
				words = append(words, "\n")
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}

// parseUUID parses a string into a UUID.
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// ============================================
// Agentic Daily Actions (006-agentic-daily-actions)
// ============================================

// GetOutcomeMetrics returns pre-computed outcome metrics for the authenticated user.
// GET /v1/daily-actions/outcome-metrics?period=30d
func (h *Handler) GetOutcomeMetrics(c fiber.Ctx) error {
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	period := c.Query("period", "30d")

	metrics, err := h.service.GetOutcomeMetrics(c.Context(), userID, period)
	if err != nil {
		// Zeroes here are not facts — the daily worker simply has not run yet.
		// `message` lets the client say "not computed" instead of drawing a
		// confident 0 over real activity.
		return h.responseHelper.Success(c, map[string]interface{}{
			"user_id":            userID,
			"period":             period,
			"total_sent":         0,
			"total_replied":      0,
			"total_meetings":     0,
			"reply_rate_overall": 0,
			"message":            "Metrics have not been computed for this period yet",
		})
	}

	return h.responseHelper.Success(c, metrics)
}

// CreateFromAgent accepts AI-generated recommendations and creates DailyAction records.
// POST /v1/daily-actions/create-from-agent
func (h *Handler) CreateFromAgent(c fiber.Ctx) error {
	userID, err := h.authHelper.GetUserID(c)
	if err != nil {
		return h.responseHelper.HandleAuthError(c, err)
	}

	var req domain.CreateFromAgentRequest
	if err := c.Bind().JSON(&req); err != nil {
		return h.responseHelper.BadRequest(c, "Invalid request body", err)
	}

	if len(req.Recommendations) == 0 {
		return h.responseHelper.BadRequest(c, "No recommendations provided", nil)
	}

	generation, err := h.service.CreateFromAgentRecommendations(c.Context(), userID, &req)
	if err != nil {
		return h.responseHelper.InternalServerError(c, "Failed to create actions", err)
	}

	return h.responseHelper.Created(c, map[string]interface{}{
		"generation_id": generation.ID,
		"action_count":  generation.ActionCount,
		"status":        generation.Status,
	})
}
