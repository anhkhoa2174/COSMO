package daily_action

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	contact "github.com/rockship/cosmo-agents-go/internal/domain/contact"
	domain "github.com/rockship/cosmo-agents-go/internal/domain/daily_action"
	outreach "github.com/rockship/cosmo-agents-go/internal/domain/outreach"
	dailyActionRepo "github.com/rockship/cosmo-agents-go/internal/repository/daily_action"
	outreachRepo "github.com/rockship/cosmo-agents-go/internal/repository/outreach"
	userRepo "github.com/rockship/cosmo-agents-go/internal/repository/user"
	outreachSvc "github.com/rockship/cosmo-agents-go/internal/service/outreach"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
	"gorm.io/gorm"
)

// Service orchestrates daily action generation and management.
type Service struct {
	outreachSvc        *outreachSvc.Service
	generationRepo     *dailyActionRepo.GenerationRepository
	actionRepo         *dailyActionRepo.ActionRepository
	snoozeRepo         *dailyActionRepo.SnoozeRepository
	completionLogRepo  *dailyActionRepo.CompletionLogRepository
	meetingRepo        *outreachRepo.MeetingRepository
	interactionLogRepo *outreachRepo.InteractionLogRepository
	userRepo           *userRepo.UserRepository

	// aiClient powers LLM-based action ranking. Optional: when nil the
	// heuristic score from ComputePriority is the final order.
	aiClient chatClient
}

// WithAIPrioritizer enables LLM-based ranking of the day's actions. Left unset,
// the service falls back to the deterministic priority formula.
func (s *Service) WithAIPrioritizer(client chatClient) *Service {
	s.aiClient = client
	return s
}

// NewService creates a new daily action service.
func NewService(
	outreachSvc *outreachSvc.Service,
	generationRepo *dailyActionRepo.GenerationRepository,
	actionRepo *dailyActionRepo.ActionRepository,
	snoozeRepo *dailyActionRepo.SnoozeRepository,
	completionLogRepo *dailyActionRepo.CompletionLogRepository,
	meetingRepo *outreachRepo.MeetingRepository,
	interactionLogRepo *outreachRepo.InteractionLogRepository,
	userRepo *userRepo.UserRepository,
) *Service {
	return &Service{
		outreachSvc:        outreachSvc,
		generationRepo:     generationRepo,
		actionRepo:         actionRepo,
		snoozeRepo:         snoozeRepo,
		completionLogRepo:  completionLogRepo,
		meetingRepo:        meetingRepo,
		interactionLogRepo: interactionLogRepo,
		userRepo:           userRepo,
	}
}

// GenerateActions runs the full generation pipeline for a user.
// It creates a DailyActionGeneration record, fetches contacts via the outreach
// service, determines conversation states, computes priorities, generates
// type-specific data, and batch-creates the daily action records.
func (s *Service) GenerateActions(ctx context.Context, userID uuid.UUID, language string, forceRefresh bool) (*domain.DailyActionGeneration, error) {
	startTime := time.Now()
	defer func() {
		GenerationDuration.Observe(time.Since(startTime).Seconds())
	}()

	today := time.Now().Format("2006-01-02")

	logger.Logger.Info().
		Str("user_id", userID.String()).
		Str("language", language).
		Bool("force_refresh", forceRefresh).
		Str("date", today).
		Msg("Starting daily action generation pipeline")

	// Check for an existing active (in-progress) generation
	active, err := s.generationRepo.FindActiveByUserAndDate(ctx, userID, today)
	if err != nil {
		return nil, fmt.Errorf("check active generation: %w", err)
	}
	if active != nil && !forceRefresh {
		return active, nil
	}

	// The new generation's ID is fixed up front so the one it replaces can be
	// linked to it before being soft-deleted: MarkReplaced only touches live
	// rows, so linking after the delete silently did nothing.
	newGenID := uuid.New()

	// If force_refresh, soft-delete previous generation first to free unique constraint
	if forceRefresh {
		prev, err := s.generationRepo.FindLatestByUserAndDate(ctx, userID, today)
		if err != nil {
			return nil, fmt.Errorf("find previous generation: %w", err)
		}
		if prev != nil {
			// Update replaced_by_id on the old generation for traceability
			if err := s.generationRepo.MarkReplaced(ctx, prev.ID, newGenID); err != nil {
				logger.Error(err).Msg("failed to mark previous generation as replaced")
			}
			// Soft-delete old generation to free unique constraint (user_id, date) WHERE is_deleted = FALSE AND replaced_by_id IS NULL
			if err := s.generationRepo.SoftDelete(ctx, prev.ID); err != nil {
				logger.Error(err).Msg("failed to soft-delete previous generation")
			}
			// Soft-delete old actions
			if err := s.actionRepo.DeleteByGenerationID(ctx, prev.ID); err != nil {
				logger.Error(err).Msg("failed to delete old actions")
			}
		}
	}

	// Create new generation record
	gen := domain.DailyActionGeneration{
		UserID:   userID,
		Date:     today,
		Status:   domain.GenerationStatusStarted,
		Language: language,
	}
	gen.ID = newGenID
	created, err := s.generationRepo.Create(ctx, &gen)
	if err != nil {
		return nil, fmt.Errorf("create generation: %w", err)
	}
	gen = *created

	// A failed run is retired rather than left started/generating: those
	// statuses mean "already under way" to FindActiveByUserAndDate, so a
	// leftover row would be returned as success to every later request
	// (including the queue's retry) for the rest of the day.
	abandon := func() { _ = s.generationRepo.SoftDelete(ctx, gen.ID) }

	// Update status to generating
	if err := s.generationRepo.UpdateStatus(ctx, gen.ID, domain.GenerationStatusGenerating, nil); err != nil {
		abandon()
		return nil, fmt.Errorf("update generation status: %w", err)
	}

	// Fetch user info for draft generation
	user, err := s.userRepo.GetDetailByID(ctx, userID)
	if err != nil || user == nil {
		abandon()
		if err == nil {
			err = fmt.Errorf("user %s not found", userID)
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	// Fetch org info for SuggestContacts
	// A member who neither administers nor created an organisation has no
	// "main" one; that is not a failure (suggestions are scoped to the user).
	org, err := s.userRepo.FindUserMainOrganization(ctx, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		abandon()
		return nil, fmt.Errorf("get user org: %w", err)
	}
	orgID := uuid.Nil
	if org != nil {
		orgID = org.ID
	}

	// Fetch contact suggestions via outreach service
	// Limit to 10 per batch to avoid OpenAI rate limits on draft generation.
	// User can regenerate after completing a batch to get the next set.
	suggestions, err := s.outreachSvc.SuggestContacts(ctx, userID, orgID, false, "mixed", 10)
	if err != nil {
		logger.Error(err).Str("user_id", userID.String()).Str("generation_id", gen.ID.String()).Msg("Failed to fetch contact suggestions")
		abandon()
		return nil, fmt.Errorf("suggest contacts: %w", err)
	}

	logger.Logger.Info().
		Str("user_id", userID.String()).
		Str("generation_id", gen.ID.String()).
		Int("suggestion_count", len(suggestions.Suggestions)).
		Msg("Contact suggestions fetched")

	// Build daily actions from suggestions
	var actions []domain.DailyAction
	for _, sc := range suggestions.Suggestions {
		action, err := s.buildAction(ctx, gen.ID, userID, sc, user.Name, language)
		if err != nil {
			logger.Error(err).Str("contact_id", sc.Contact.ID.String()).Msg("failed to build action for contact")
			continue
		}
		actions = append(actions, *action)
	}

	// Let the model rank the batch; falls back to the heuristic order on any
	// failure, so this can only reorder a generation, never block one.
	s.rerankActions(ctx, actions)

	// Batch create actions
	if err := s.actionRepo.BatchCreate(ctx, actions); err != nil {
		abandon()
		return nil, fmt.Errorf("batch create actions: %w", err)
	}

	// Build agent briefing
	agentBriefing := GenerateAgentBriefing(actions, language)
	briefingJSON, _ := json.Marshal(agentBriefing)

	// Build pipeline summary (frontend-compatible format)
	pipelineSummary := s.BuildPipelineSummary(ctx, userID, actions)
	summaryJSON, _ := json.Marshal(pipelineSummary)

	// Mark generation as ready
	now := time.Now()
	if err := s.generationRepo.UpdateStatus(ctx, gen.ID, domain.GenerationStatusReady, map[string]interface{}{
		"action_count":     len(actions),
		"generated_at":     now,
		"agent_briefing":   base.JSONB(briefingJSON),
		"pipeline_summary": base.JSONB(summaryJSON),
	}); err != nil {
		// Left in "generating", the row would be returned as today's batch
		// for the rest of the day.
		abandon()
		return nil, fmt.Errorf("finalize generation: %w", err)
	}

	GenerationActionCount.Observe(float64(len(actions)))

	logger.Logger.Info().
		Str("user_id", userID.String()).
		Str("generation_id", gen.ID.String()).
		Int("action_count", len(actions)).
		Float64("duration_seconds", time.Since(startTime).Seconds()).
		Msg("Daily action generation completed")

	gen.Status = domain.GenerationStatusReady
	gen.ActionCount = len(actions)
	gen.GeneratedAt = &now
	gen.AgentBriefing = base.JSONB(briefingJSON)
	gen.PipelineSummary = base.JSONB(summaryJSON)

	return &gen, nil
}

// buildAction creates a single DailyAction from a SuggestContact.
func (s *Service) buildAction(
	ctx context.Context,
	generationID, userID uuid.UUID,
	sc *outreachSvc.SuggestContact,
	userName, language string,
) (*domain.DailyAction, error) {
	c := sc.Contact
	state := sc.State

	// Determine action type and category from outreach state
	actionType, categoryID := mapActionTypeAndCategory(sc, state)

	// Find the nearest upcoming meeting for this contact (for priority scoring
	// and meeting_prep). The repository returns newest-first, so keep scanning.
	var meeting *outreach.Meeting
	meetings, _ := s.meetingRepo.FindByContactIDAndUserID(ctx, c.ID, userID)
	for _, m := range meetings {
		if m.Status == "scheduled" && m.Time.After(time.Now()) &&
			(meeting == nil || m.Time.Before(meeting.Time)) {
			meeting = m
		}
	}

	// Override to meeting_prep if there's an imminent meeting (< 48h)
	if meeting != nil && time.Until(meeting.Time).Hours() < 48 {
		if actionType != domain.ActionTypeRespond {
			actionType = domain.ActionTypeMeetingPrep
			categoryID = domain.CategoryMeetingPrep
		}
	}

	// Compute priority
	priority, factors := ComputePriority(c, state, meeting)
	factorsJSON, _ := json.Marshal(factors)

	// Build contact snapshot
	snapshot := buildContactSnapshot(c)
	snapshotJSON, _ := json.Marshal(snapshot)

	// Build reasoning
	reasoning := buildReasoning(actionType, sc, state)

	action := &domain.DailyAction{
		UserID:          userID,
		GenerationID:    generationID,
		ContactID:       c.ID,
		Type:            actionType,
		CategoryID:      categoryID,
		Priority:        priority,
		PriorityFactors: base.JSONB(factorsJSON),
		Reasoning:       reasoning,
		Status:          domain.ActionStatusSuggested,
		ContactSnapshot: base.JSONB(snapshotJSON),
	}

	// Generate type-specific data
	switch actionType {
	case domain.ActionTypeOutreach, domain.ActionTypeFollowup:
		data := s.buildOutreachData(ctx, userID, c.ID, sc, state, userName, language)
		dataJSON, _ := json.Marshal(data)
		action.OutreachData = base.JSONB(dataJSON)

	case domain.ActionTypeRespond:
		data := buildRespondData(sc, state)
		dataJSON, _ := json.Marshal(data)
		action.RespondData = base.JSONB(dataJSON)

	case domain.ActionTypeMeetingPrep:
		if meeting != nil {
			data := buildMeetingData(meeting)
			dataJSON, _ := json.Marshal(data)
			action.MeetingData = base.JSONB(dataJSON)
		}

	case domain.ActionTypeEnrich:
		data := buildEnrichmentData(c)
		dataJSON, _ := json.Marshal(data)
		action.EnrichmentData = base.JSONB(dataJSON)
	}

	return action, nil
}

// mapActionTypeAndCategory maps outreach state to action type and category.
func mapActionTypeAndCategory(sc *outreachSvc.SuggestContact, state *outreach.OutreachState) (domain.ActionType, domain.CategoryID) {
	if state == nil {
		// No state → check contact status for enrichment
		if sc.Contact.Status == "pending" {
			return domain.ActionTypeEnrich, domain.CategoryEnrichment
		}
		return domain.ActionTypeOutreach, domain.CategoryNewOutreach
	}

	switch state.ConversationState {
	case "REPLIED":
		return domain.ActionTypeRespond, domain.CategoryReplied
	case "NO_REPLY":
		switch state.NextStep {
		case "FOLLOW_UP_1", "FOLLOW_UP_2", "FOLLOW_UP":
			return domain.ActionTypeFollowup, domain.CategoryFollowup
		case "DROP":
			return domain.ActionTypeFollowup, domain.CategoryFollowup
		default:
			return domain.ActionTypeFollowup, domain.CategoryFollowup
		}
	case "COLD":
		if state.NextStep == "SEND" {
			return domain.ActionTypeOutreach, domain.CategoryNewOutreach
		}
		return domain.ActionTypeOutreach, domain.CategoryNewOutreach
	case "POST_MEETING":
		return domain.ActionTypeFollowup, domain.CategoryFollowup
	default:
		if sc.Contact.Status == "pending" {
			return domain.ActionTypeEnrich, domain.CategoryEnrichment
		}
		return domain.ActionTypeOutreach, domain.CategoryNewOutreach
	}
}

// buildContactSnapshot creates a ContactSnapshot from a Contact entity.
func buildContactSnapshot(c *contact.Contact) domain.ContactSnapshot {
	return domain.ContactSnapshot{
		ID:             c.ID,
		Name:           c.Name,
		Email:          c.ContactInformation,
		Company:        c.Company,
		JobTitle:       c.JobTitle,
		Source:         c.Source,
		Status:         c.Status,
		OutreachStage:  c.OutreachStage,
		LifecycleStage: c.BusinessStage,
	}
}

// buildReasoning generates a human-readable reasoning string for an action.
func buildReasoning(actionType domain.ActionType, sc *outreachSvc.SuggestContact, state *outreach.OutreachState) string {
	switch actionType {
	case domain.ActionTypeRespond:
		return fmt.Sprintf("%s replied — review and respond", sc.Contact.Name)
	case domain.ActionTypeFollowup:
		days := 0
		if state != nil {
			days = state.DaysSinceLastInteraction
		}
		return fmt.Sprintf("Follow up with %s (%d days since last interaction)", sc.Contact.Name, days)
	case domain.ActionTypeOutreach:
		return fmt.Sprintf("Initial outreach to %s at %s", sc.Contact.Name, sc.Contact.Company)
	case domain.ActionTypeMeetingPrep:
		return fmt.Sprintf("Prepare for upcoming meeting with %s", sc.Contact.Name)
	case domain.ActionTypeEnrich:
		missing := sc.Contact.GetMissingFields()
		return fmt.Sprintf("Enrich profile for %s — %d fields missing", sc.Contact.Name, len(missing))
	default:
		return fmt.Sprintf("Action for %s", sc.Contact.Name)
	}
}

// buildOutreachData generates OutreachActionData for outreach/followup actions.
func (s *Service) buildOutreachData(
	ctx context.Context,
	userID, contactID uuid.UUID,
	sc *outreachSvc.SuggestContact,
	state *outreach.OutreachState,
	userName, language string,
) domain.OutreachActionData {
	data := domain.OutreachActionData{
		DraftMessage: sc.MessageDraft,
		Scenario:     sc.Contact.Scenario,
		ContextLevel: sc.Contact.ContextLevel,
	}

	if state != nil {
		data.DaysSinceLastInteraction = state.DaysSinceLastInteraction
		data.OutreachState = &domain.OutreachStateSnapshot{
			ConversationState: state.ConversationState,
			NextStep:          state.NextStep,
			FollowupCount:     state.FollowupCount,
			MaxFollowups:      state.MaxFollowups,
		}
		// Derive followup number from next_step (which follow-up to send next)
		switch state.NextStep {
		case "FOLLOW_UP_1":
			fu := 1
			data.FollowupNumber = &fu
		case "FOLLOW_UP_2":
			fu := 2
			data.FollowupNumber = &fu
		case "FOLLOW_UP":
			fu := state.FollowupCount + 1
			data.FollowupNumber = &fu
		default:
			if state.FollowupCount > 0 {
				fu := state.FollowupCount
				data.FollowupNumber = &fu
			}
		}
		data.IsFinalFollowup = state.FollowupCount >= state.MaxFollowups-1
		if state.LastInteractionAt != nil {
			t := state.LastInteractionAt.Format("2006-01-02")
			data.LastSentDate = &t
		}
	}

	// Reuse the draft already persisted on the outreach state before paying for
	// a new one. SuggestContact.MessageDraft is never populated by the suggest
	// queries, so without this every generation regenerates a draft per contact
	// — one LLM call each — even when an unchanged draft is sitting in the DB.
	// A draft is only reused when it was written after the contact's last
	// interaction; anything older was written for a conversation that has since
	// moved on and must be regenerated.
	if data.DraftMessage == "" && state != nil && state.MessageDraft != nil && *state.MessageDraft != "" {
		fresh := state.LastInteractionAt == nil || !state.UpdatedAt.Before(*state.LastInteractionAt)
		if fresh {
			data.DraftMessage = *state.MessageDraft
		}
	}

	// If no draft from suggestions or state, generate one
	if data.DraftMessage == "" {
		draft, err := s.outreachSvc.GenerateDraftForHandlerWithLanguage(ctx, userID, contactID, language, userName)
		if err != nil {
			logger.Error(err).Str("contact_id", contactID.String()).Msg("failed to generate draft")
		} else if draft != nil {
			data.DraftMessage = draft.Draft
			data.Scenario = string(draft.Scenario)
			data.ContextLevel = string(draft.ContextLevel)
		}
	}

	return data
}

// buildRespondData generates RespondActionData for respond (replied) actions.
func buildRespondData(sc *outreachSvc.SuggestContact, state *outreach.OutreachState) domain.RespondActionData {
	data := domain.RespondActionData{
		ReplyChannel:      sc.Contact.ContactChannel,
		IntentAssessment:  "needs_review",
		RecommendedAction: "reply",
	}

	if state != nil && state.LastInteractionAt != nil {
		data.ReplyTimestamp = state.LastInteractionAt.Format(time.RFC3339)
	}

	return data
}

// buildMeetingData generates MeetingActionData from a meeting entity.
func buildMeetingData(meeting *outreach.Meeting) domain.MeetingActionData {
	data := domain.MeetingActionData{
		MeetingID:              meeting.ID.String(),
		MeetingTime:            meeting.Time.Format(time.RFC3339),
		MeetingDurationMinutes: meeting.DurationMinutes,
		MeetingChannel:         meeting.Channel,
		HoursUntilMeeting:      time.Until(meeting.Time).Hours(),
	}
	if meeting.Title != nil {
		data.MeetingTitle = *meeting.Title
	}
	return data
}

// buildEnrichmentData generates EnrichmentActionData from contact missing fields.
func buildEnrichmentData(c *contact.Contact) domain.EnrichmentActionData {
	missing := c.GetMissingFields()
	impact := "low"
	if len(missing) >= 4 {
		impact = "high"
	} else if len(missing) >= 2 {
		impact = "medium"
	}

	data := domain.EnrichmentActionData{
		MissingFields: missing,
		QualityImpact: impact,
		ContactStatus: c.Status,
	}

	// Add suggested sources for common missing fields
	for _, field := range missing {
		switch field {
		case "contact_information":
			data.SuggestedSources = append(data.SuggestedSources, domain.SuggestedSource{
				Field:  field,
				Source: "LinkedIn",
			})
		case "company":
			data.SuggestedSources = append(data.SuggestedSources, domain.SuggestedSource{
				Field:  field,
				Source: "Apollo.io",
			})
		case "job_title":
			data.SuggestedSources = append(data.SuggestedSources, domain.SuggestedSource{
				Field:  field,
				Source: "LinkedIn",
			})
		}
	}

	return data
}

// countByCategory counts actions per category.
func countByCategory(actions []domain.DailyAction) map[domain.CategoryID]int {
	counts := make(map[domain.CategoryID]int)
	for _, a := range actions {
		counts[a.CategoryID]++
	}
	return counts
}

// FrontendPipelineSummary matches the frontend PipelineSummary TypeScript type.
type FrontendPipelineSummary struct {
	TotalActiveContacts  int            `json:"total_active_contacts"`
	ContactsByStage      map[string]int `json:"contacts_by_stage"`
	ContactsByLifecycle  map[string]int `json:"contacts_by_lifecycle"`
	ResponseRate7d       float64        `json:"response_rate_7d"`
	MeetingsBooked7d     int            `json:"meetings_booked_7d"`
	AvgResponseTimeHours float64        `json:"avg_response_time_hours"`
}

// BuildPipelineSummary computes pipeline health metrics for the frontend.
// It derives contact stage counts from generated actions and queries 7-day metrics from repos.
func (s *Service) BuildPipelineSummary(ctx context.Context, userID uuid.UUID, actions []domain.DailyAction) FrontendPipelineSummary {
	summary := FrontendPipelineSummary{
		TotalActiveContacts: len(actions),
		ContactsByStage:     make(map[string]int),
		ContactsByLifecycle: make(map[string]int),
	}

	// Derive stage counts from contact snapshots in the generated actions
	for _, a := range actions {
		var snapshot domain.ContactSnapshot
		if err := a.ContactSnapshot.Unmarshal(&snapshot); err == nil {
			if snapshot.OutreachStage != "" {
				summary.ContactsByStage[snapshot.OutreachStage]++
			}
			if snapshot.LifecycleStage != "" {
				summary.ContactsByLifecycle[snapshot.LifecycleStage]++
			}
		}
	}

	// Compute 7-day metrics from interaction logs and meetings
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	outgoing, err := s.interactionLogRepo.CountByUserDirectionSince(ctx, userID, "outgoing", sevenDaysAgo)
	if err != nil {
		logger.Error(err).Msg("failed to count outgoing interactions for pipeline summary")
	}
	incoming, err := s.interactionLogRepo.CountByUserDirectionSince(ctx, userID, "incoming", sevenDaysAgo)
	if err != nil {
		logger.Error(err).Msg("failed to count incoming interactions for pipeline summary")
	}
	if outgoing > 0 {
		summary.ResponseRate7d = float64(incoming) / float64(outgoing)
	}

	meetingsCount, err := s.meetingRepo.CountByUserSince(ctx, userID, sevenDaysAgo)
	if err != nil {
		logger.Error(err).Msg("failed to count meetings for pipeline summary")
	}
	summary.MeetingsBooked7d = int(meetingsCount)

	// Estimate avg response time: if we have incoming replies, approximate from days
	if incoming > 0 && outgoing > 0 {
		// Rough estimate: 7 days worth of interactions, avg spread
		summary.AvgResponseTimeHours = (7.0 * 24.0) / float64(incoming)
		if summary.AvgResponseTimeHours > 168 { // cap at 7 days
			summary.AvgResponseTimeHours = 168
		}
	}

	return summary
}

// CheckStaleness checks if a generation's results have become stale.
// Returns true if new events (incoming interactions, meeting changes) have occurred
// since the generation was created.
func (s *Service) CheckStaleness(ctx context.Context, userID uuid.UUID, gen *domain.DailyActionGeneration) (bool, error) {
	if gen == nil || gen.GeneratedAt == nil {
		return false, nil
	}
	if gen.Status == domain.GenerationStatusStale {
		return true, nil
	}

	generatedAt := *gen.GeneratedAt

	// Check for new incoming interactions since generation
	hasNew, err := s.interactionLogRepo.HasIncomingByUserAfter(ctx, userID, generatedAt)
	if err != nil {
		return false, fmt.Errorf("check incoming interactions: %w", err)
	}
	if hasNew {
		logger.Logger.Info().
			Str("user_id", userID.String()).
			Str("generation_id", gen.ID.String()).
			Msg("Generation marked as stale due to new interactions")
		_ = s.generationRepo.UpdateStatus(ctx, gen.ID, domain.GenerationStatusStale, nil)
		return true, nil
	}

	return false, nil
}

// ============================================
// Agentic Daily Actions (006-agentic-daily-actions)
// ============================================

// GetOutcomeMetrics returns pre-computed outcome metrics for a user.
func (s *Service) GetOutcomeMetrics(ctx context.Context, userID uuid.UUID, period string) (*domain.OutcomeMetrics, error) {
	var metrics domain.OutcomeMetrics
	err := s.generationRepo.GetDB().WithContext(ctx).
		Where("user_id = ? AND period = ?", userID, period).
		First(&metrics).Error
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}

// CreateFromAgentRecommendations creates DailyAction records from AI-generated recommendations.
func (s *Service) CreateFromAgentRecommendations(ctx context.Context, userID uuid.UUID, req *domain.CreateFromAgentRequest) (*domain.DailyActionGeneration, error) {
	today := time.Now().Format("2006-01-02")

	// Create generation record
	gen := &domain.DailyActionGeneration{
		UserID:      userID,
		Date:        today,
		Status:      domain.GenerationStatusReady,
		Language:    req.Language,
		ActionCount: len(req.Recommendations),
	}
	now := time.Now()
	gen.GeneratedAt = &now

	// Build agent briefing
	briefingData := domain.AgentBriefingData{
		Greeting:           req.StrategicPlan,
		StrategicReasoning: req.StrategicPlan,
	}
	_ = gen.AgentBriefing.Marshal(briefingData)

	createdGen, err := s.generationRepo.Create(ctx, gen)
	if err != nil {
		return nil, fmt.Errorf("create generation: %w", err)
	}
	gen = createdGen

	// Create actions from recommendations
	actions := make([]domain.DailyAction, 0, len(req.Recommendations))
	for _, rec := range req.Recommendations {
		action := domain.DailyAction{
			UserID:       userID,
			GenerationID: gen.ID,
			ContactID:    rec.ContactID,
			Type:         domain.ActionType(rec.ActionType),
			CategoryID:   domain.CategoryID(rec.CategoryID),
			Priority:     rec.PriorityRank,
			Reasoning:    rec.StrategicReasoning,
			Status:       domain.ActionStatusSuggested,
		}

		// Build outreach data with agentic fields
		outreachData := map[string]interface{}{
			"draft_message":        rec.DraftMessage,
			"scenario":             rec.MessagingStrategy,
			"messaging_strategy":   rec.MessagingStrategy,
			"strategic_reasoning":  rec.StrategicReasoning,
			"recommended_channel":  rec.RecommendedChannel,
			"referenced_knowledge": rec.ReferencedKnowledge,
			"confidence_level":     rec.ConfidenceLevel,
			"outcome_pattern_cited": rec.OutcomePatternCited,
		}
		_ = action.OutreachData.Marshal(outreachData)

		// Build contact snapshot (minimal — we don't have full contact data here)
		snapshot := map[string]interface{}{
			"id": rec.ContactID,
		}
		_ = action.ContactSnapshot.Marshal(snapshot)

		actions = append(actions, action)
	}

	if err := s.actionRepo.BatchCreate(ctx, actions); err != nil {
		return nil, fmt.Errorf("batch create actions: %w", err)
	}

	return gen, nil
}
