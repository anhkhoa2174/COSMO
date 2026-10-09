package outreach

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/contact"
	"github.com/rockship/cosmo-agents-go/internal/domain/outreach"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	outreachRepo "github.com/rockship/cosmo-agents-go/internal/repository/outreach"
	"github.com/rockship/cosmo-agents-go/pkg/ai"
	"github.com/rockship/cosmo-agents-go/pkg/logger"
)

// ErrUnknownEvent is returned for an outreach event the state machine does not
// know; it is the caller's mistake, so the handler answers 400.
var ErrUnknownEvent = errors.New("unknown outreach event")

// Config holds outreach service configuration
// The json tags matter: this type is serialised as the `effective` and
// `defaults` objects of the outreach settings endpoint, and the settings page
// reads it by the same snake_case keys it posts back in OutreachSettings.
// Without them Go emits the Go field names, every lookup on the page returns
// undefined, and each timing input reads NaN — which the form then reports as
// "enter a whole number" on fields nobody had touched, blocking saves that had
// nothing to do with timing.
type Config struct {
	// Follow-up timing (days after last sent message)
	FollowUp1MinDays int `json:"follow_up1_min_days"` // Minimum days before FU #1 is suggested (default: 4)
	FollowUp1MaxDays int `json:"follow_up1_max_days"` // Maximum days for FU #1 window (default: 5)
	FollowUp2MinDays int `json:"follow_up2_min_days"` // Minimum days before FU #2 is suggested (default: 4)
	FollowUp2MaxDays int `json:"follow_up2_max_days"` // Maximum days for FU #2 window (default: 5)

	// Auto no-reply transition
	NoReplyHours int `json:"no_reply_hours"` // Hours after sending before auto-transitioning to NO_REPLY (default: 8)

	// Meeting confirmation follow-up timing
	MeetingConfirmMinDays int `json:"meeting_confirm_min_days"` // Days before meeting confirmation follow-up (default: 1)

	ReEngageThresholdDays int `json:"re_engage_threshold_days"` // Days before re-engage (default: 60)
	MaxFollowups          int `json:"max_followups"`            // Maximum follow-up attempts (default: 2)
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		FollowUp1MinDays:      4,  // FU #1 after 4 days (Day 4-5)
		FollowUp1MaxDays:      5,  // FU #1 until 5 days
		FollowUp2MinDays:      4,  // FU #2 after 4 days from FU #1 (Day 9-12)
		FollowUp2MaxDays:      5,  // FU #2 until 5 days from FU #1
		NoReplyHours:          8,  // Auto-transition to NO_REPLY after 8 hours
		MeetingConfirmMinDays: 1,  // Meeting confirm FU after 1 day
		ReEngageThresholdDays: 60, // Re-engage after 60 days
		MaxFollowups:          2,  // Max 2 follow-ups before drop
	}
}

// Service handles outreach operations
type Service struct {
	interactionRepo *outreachRepo.InteractionLogRepository
	stateRepo       *outreachRepo.OutreachStateRepository
	meetingRepo     *outreachRepo.MeetingRepository
	feedbackRepo    *outreachRepo.FeedbackRepository
	contactRepo     *contactRepo.ContactRepository
	openAIClient    *ai.OpenAIClient
	config          Config

	// Resolves the per-organisation cadence. Unset, every account keeps the
	// built-in defaults, which is how this service behaved before settings
	// existed.
	orgConfig *orgConfigLoader
}

// WithOrgSettings makes the cadence admin-configurable per organisation.
// `load` returns the raw settings JSON for a user's organisation.
func (s *Service) WithOrgSettings(
	load func(ctx context.Context, userID uuid.UUID) ([]byte, error),
) *Service {
	s.orgConfig = newOrgConfigLoader(load)
	return s
}

// InvalidateOrgConfig drops the cached cadence for a user so a just-saved
// change applies on the next evaluation instead of after the cache expires.
func (s *Service) InvalidateOrgConfig(userID uuid.UUID) {
	s.orgConfig.Invalidate(userID)
}

// configFor returns the cadence in force for this user.
func (s *Service) configFor(ctx context.Context, userID uuid.UUID) Config {
	return s.orgConfig.Get(ctx, userID, s.config)
}

// ConfigFor returns the effective cadence for a user's organisation.
func (s *Service) ConfigFor(ctx context.Context, userID uuid.UUID) Config {
	return s.configFor(ctx, userID)
}

// OutreachService is an alias for Service for handler compatibility
type OutreachService = Service

// NewService creates a new outreach service
func NewService(
	interactionRepo *outreachRepo.InteractionLogRepository,
	stateRepo *outreachRepo.OutreachStateRepository,
	meetingRepo *outreachRepo.MeetingRepository,
	feedbackRepo *outreachRepo.FeedbackRepository,
	contactRepo *contactRepo.ContactRepository,
	config *Config,
) *Service {
	cfg := DefaultConfig()
	if config != nil {
		cfg = *config
	}

	return &Service{
		interactionRepo: interactionRepo,
		stateRepo:       stateRepo,
		meetingRepo:     meetingRepo,
		feedbackRepo:    feedbackRepo,
		contactRepo:     contactRepo,
		config:          cfg,
	}
}

// NewServiceWithAI creates a new outreach service with AI support
func NewServiceWithAI(
	interactionRepo *outreachRepo.InteractionLogRepository,
	stateRepo *outreachRepo.OutreachStateRepository,
	meetingRepo *outreachRepo.MeetingRepository,
	feedbackRepo *outreachRepo.FeedbackRepository,
	contactRepo *contactRepo.ContactRepository,
	openAIClient *ai.OpenAIClient,
	config *Config,
) *Service {
	cfg := DefaultConfig()
	if config != nil {
		cfg = *config
	}

	return &Service{
		interactionRepo: interactionRepo,
		stateRepo:       stateRepo,
		meetingRepo:     meetingRepo,
		feedbackRepo:    feedbackRepo,
		contactRepo:     contactRepo,
		openAIClient:    openAIClient,
		config:          cfg,
	}
}

// SetOpenAIClient sets the OpenAI client for AI-powered draft generation
func (s *Service) SetOpenAIClient(client *ai.OpenAIClient) {
	s.openAIClient = client
}

// ============================================
// Conversation State Machine
// ============================================

// ConversationStateResult holds the result of state determination
type ConversationStateResult struct {
	State                    outreach.ConversationState
	OutreachIntent           outreach.OutreachIntent
	Scenario                 outreach.Scenario
	NextStep                 outreach.NextStepAction
	DaysSinceLastInteraction int
	FollowupCount            int
	LastOutgoing             *outreach.InteractionLog
	LastIncoming             *outreach.InteractionLog
}

// DetermineConversationState determines the conversation state for a contact
// This is the core state machine logic
func (s *Service) DetermineConversationState(ctx context.Context, contactEntity *contact.Contact, userID uuid.UUID) (*ConversationStateResult, error) {
	cfg := s.configFor(ctx, userID)
	now := time.Now()

	// 0. Respect stored DROPPED state (manual drop or auto-drop via events)
	if contactEntity.OutreachStage == string(outreach.StateDropped) {
		return &ConversationStateResult{
			State:          outreach.StateDropped,
			NextStep:       outreach.NextStepDrop,
			OutreachIntent: outreach.IntentIntro,
			Scenario:       outreach.ScenarioRoleBased,
		}, nil
	}

	// Get interaction logs
	logs, err := s.interactionRepo.FindByContactIDAndUserID(ctx, contactEntity.ID, userID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get interaction logs: %w", err)
	}

	// DEBUG: Log interaction logs count
	logger.Logger.Debug().
		Str("contact_id", contactEntity.ID.String()).
		Str("user_id", userID.String()).
		Int("logs_count", len(logs)).
		Msg("outreach: DetermineConversationState - logs retrieved")

	// Check for scheduled (not completed) meeting — meeting confirmation flow is event-driven
	// and cannot be reconstructed from interaction logs, so respect the stored next_step
	hasScheduledMeeting, _ := s.meetingRepo.HasScheduledMeeting(ctx, contactEntity.ID)
	if hasScheduledMeeting {
		storedStep := outreach.NextStepAction(contactEntity.NextStep)
		switch storedStep {
		case outreach.NextStepFollowUpMeeting1, outreach.NextStepFollowUpMeeting2, outreach.NextStepPrepareMeeting, outreach.NextStepWait:
			return &ConversationStateResult{
				State:          outreach.StateReplied,
				NextStep:       storedStep,
				OutreachIntent: outreach.IntentFollowUp,
				Scenario:       outreach.ScenarioMeetingConfirmation,
			}, nil
		}
	}

	// Get meetings
	hasMeeting, err := s.meetingRepo.HasCompletedMeeting(ctx, contactEntity.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check meetings: %w", err)
	}

	// Get last outgoing and incoming (filter by userID to match logs query)
	lastOutgoing, _ := s.interactionRepo.GetLastOutgoingByUserID(ctx, contactEntity.ID, userID)
	lastIncoming, _ := s.interactionRepo.GetLastIncomingByUserID(ctx, contactEntity.ID, userID)

	// Calculate time since last interaction (in days)
	var daysSinceLastInteraction int
	var lastInteractionTime *time.Time

	if len(logs) > 0 {
		lastInteractionTime = &logs[0].Timestamp
		// Calculate days since last interaction
		daysSinceLastInteraction = int(now.Sub(*lastInteractionTime).Hours() / 24)

		// DEBUG: Log timing calculation
		logger.Logger.Debug().
			Str("contact_id", contactEntity.ID.String()).
			Time("last_interaction_time", *lastInteractionTime).
			Time("now", now).
			Int("days_since", daysSinceLastInteraction).
			Int("config_followup1_min", cfg.FollowUp1MinDays).
			Bool("should_transition_to_fu1", daysSinceLastInteraction >= cfg.FollowUp1MinDays).
			Msg("outreach: DetermineConversationState - timing calculation")
	} else {
		logger.Logger.Debug().
			Str("contact_id", contactEntity.ID.String()).
			Msg("outreach: DetermineConversationState - NO LOGS FOUND")
	}

	// Count followups (outgoing messages without reply)
	// followupCount = number of FOLLOW-UPs sent (NOT counting initial message)
	// countOutgoing = 1 (only initial) → followupCount = 0 → suggest FU#1
	// countOutgoing = 2 (initial + FU#1) → followupCount = 1 → suggest FU#2
	// countOutgoing = 3 (initial + FU#1 + FU#2) → followupCount = 2 → suggest DROP
	followupCount := 0
	if lastOutgoing != nil && lastIncoming == nil {
		count, _ := s.interactionRepo.CountOutgoingByUserID(ctx, contactEntity.ID, userID)
		// Subtract 1 to exclude initial message
		if count > 0 {
			followupCount = count - 1
		}
	} else if lastOutgoing != nil && lastIncoming != nil {
		if lastOutgoing.Timestamp.After(lastIncoming.Timestamp) {
			// Count outgoing messages after last incoming
			followupCount = 1 // At least one follow-up sent after reply
		}
	}

	result := &ConversationStateResult{
		DaysSinceLastInteraction: daysSinceLastInteraction,
		FollowupCount:            followupCount,
		LastOutgoing:             lastOutgoing,
		LastIncoming:             lastIncoming,
	}

	// State priority: DROPPED > POST_MEETING > REPLIED > NO_REPLY > COLD

	// 1. Check DROPPED
	if followupCount >= cfg.MaxFollowups && lastIncoming == nil {
		result.State = outreach.StateDropped
		result.NextStep = outreach.NextStepDrop
		result.OutreachIntent = outreach.IntentIntro // No intent for dropped
		result.Scenario = outreach.ScenarioRoleBased
		return result, nil
	}

	// 2. Check POST_MEETING
	if hasMeeting {
		result.State = outreach.StatePostMeeting
		result.OutreachIntent = outreach.IntentPostMeeting
		result.Scenario = outreach.ScenarioPostMeeting
		result.NextStep = outreach.NextStepFollowUp
		return result, nil
	}

	// 3. Check REPLIED
	if lastIncoming != nil {
		if lastOutgoing == nil || lastIncoming.Timestamp.After(lastOutgoing.Timestamp) {
			// Last action was incoming (reply) - customer has responded
			result.State = outreach.StateReplied
			result.OutreachIntent = outreach.IntentFollowUp
			result.Scenario = outreach.ScenarioPostReply
			result.NextStep = outreach.NextStepSetMeeting // Default: propose meeting

			// Check sentiment for next step
			if lastIncoming.Sentiment != nil {
				switch *lastIncoming.Sentiment {
				case string(outreach.SentimentPositive):
					result.NextStep = outreach.NextStepSetMeeting
				case string(outreach.SentimentNegative):
					result.NextStep = outreach.NextStepWait
				}
			}
			return result, nil
		}
	}

	// 4. Check NO_REPLY - need follow-up based on sent count and timing
	if lastOutgoing != nil && lastIncoming == nil {
		// Check 8-hour threshold before transitioning to next step
		hoursSinceLastOutgoing := now.Sub(lastOutgoing.Timestamp).Hours()
		if hoursSinceLastOutgoing < float64(cfg.NoReplyHours) {
			// Less than 8 hours since last outgoing — still waiting for reply
			if followupCount == 0 {
				result.State = outreach.StateCold
				result.OutreachIntent = outreach.IntentIntro
				result.Scenario = outreach.ScenarioRoleBased
			} else {
				result.State = outreach.StateNoReply
				result.OutreachIntent = outreach.IntentFollowUp
				result.Scenario = outreach.ScenarioNoReplyFollowup
			}
			result.NextStep = outreach.NextStepWait
			return result, nil
		}

		// Past 8 hours → transition to next step
		result.State = outreach.StateNoReply
		result.OutreachIntent = outreach.IntentFollowUp
		result.Scenario = outreach.ScenarioNoReplyFollowup

		// Determine next step based on followup count
		if followupCount == 0 {
			// No FU sent yet - past 8 hours → ready for FU #1
			result.NextStep = outreach.NextStepFollowUp1
		} else if followupCount == 1 {
			// FU #1 sent - past 8 hours → ready for FU #2
			result.NextStep = outreach.NextStepFollowUp2
		} else {
			// FU #2 sent - past 8 hours → DROP
			result.NextStep = outreach.NextStepDrop
			result.State = outreach.StateDropped
		}
		return result, nil
	}

	// 5. Check RE_ENGAGE (cold contact not contacted for a long time)
	if daysSinceLastInteraction >= cfg.ReEngageThresholdDays && lastOutgoing != nil {
		result.State = outreach.StateCold // Re-engage is treated as cold
		result.OutreachIntent = outreach.IntentReEngage
		result.Scenario = outreach.ScenarioReEngage
		result.NextStep = outreach.NextStepSend
		return result, nil
	}

	// 5. Default: COLD
	result.State = outreach.StateCold
	result.OutreachIntent = outreach.IntentIntro
	result.NextStep = outreach.NextStepSend

	// Determine scenario based on context
	if contactEntity.Industry != "" {
		result.Scenario = outreach.ScenarioIndustryBased
	} else {
		result.Scenario = outreach.ScenarioRoleBased
	}

	return result, nil
}

// ============================================
// Suggest Logic
// ============================================

// SuggestContact represents a contact suggestion for outreach
type SuggestContact struct {
	Contact      *contact.Contact        `json:"contact"`
	State        *outreach.OutreachState `json:"state"`
	Type         string                  `json:"type"` // "cold" or "followup"
	NextStep     string                  `json:"next_step"`
	DaysSince    int                     `json:"days_since"`
	MessageDraft string                  `json:"message_draft"`
}

// SuggestResult wraps suggestions with total counts
type SuggestResult struct {
	Suggestions []*SuggestContact `json:"suggestions"`
	Total       int               `json:"total"`
}

// SuggestContacts suggests contacts for outreach
// If isAdmin is true, it fetches all contacts in the organization; otherwise, only the user's own contacts
// PipelineStalledAfter is how long a contact still in play may go without an
// interaction before the dashboard calls it stalled.
const PipelineStalledAfter = 5 * 24 * time.Hour

// PipelineSummary counts the caller's contacts by outreach stage, follow-up
// depth and staleness, across all of them rather than the suggestion list.
func (s *Service) PipelineSummary(ctx context.Context, userID uuid.UUID) (*contactRepo.PipelineSummary, error) {
	return s.contactRepo.PipelineSummary(ctx, userID, time.Now().Add(-PipelineStalledAfter))
}

func (s *Service) SuggestContacts(ctx context.Context, userID uuid.UUID, organizationID uuid.UUID, isAdmin bool, suggestType string, limit int) (*SuggestResult, error) {
	if limit <= 0 {
		limit = 50
	}

	// Determine orgIDs based on role
	// Admin sees all contacts in organization, member sees only their own
	var orgIDs []uuid.UUID
	if isAdmin {
		orgIDs = []uuid.UUID{organizationID}
	}
	// If not admin, orgIDs remains nil and SearchWithFilter will filter by userID

	var suggestions []*SuggestContact
	var total int

	switch suggestType {
	case "cold":
		suggestions, total, _ = s.suggestColdContacts(ctx, userID, orgIDs, limit)
	case "followup":
		suggestions, total, _ = s.suggestFollowupContacts(ctx, userID, orgIDs, limit)
	case "mixed":
		suggestions, total, _ = s.suggestMixedContacts(ctx, userID, orgIDs, limit)
	default:
		return nil, fmt.Errorf("invalid suggest type: %s", suggestType)
	}

	return &SuggestResult{
		Suggestions: suggestions,
		Total:       total,
	}, nil
}

// heldByNextStep is true when the next-step engine has put the contact on
// hold — waiting until a stated date, nurtured until a revisit date, or
// closed — so suggesting it today would contradict that decision.
func heldByNextStep(c *contact.Contact, now time.Time) bool {
	if c.NextAction != nil {
		switch *c.NextAction {
		case "SUPPRESS", "DISQUALIFY":
			return true
		}
	}
	return c.NextActionDueAt != nil && c.NextActionDueAt.After(now)
}

func (s *Service) suggestColdContacts(ctx context.Context, userID uuid.UUID, orgIDs []uuid.UUID, limit int) ([]*SuggestContact, int, error) {
	// Find READY contacts needing initial outreach (next_step = SEND)
	contacts, total, err := s.contactRepo.SearchWithFilter(ctx, userID, orgIDs, map[string]interface{}{
		"status":         "ready",
		"next_step":      "SEND",
		"do_not_contact": false, // opted-out contacts are never due outreach
	}, nil)
	if err != nil {
		return nil, 0, err
	}

	suggestions := make([]*SuggestContact, 0, limit)
	for _, c := range contacts {
		if len(suggestions) >= limit {
			break
		}
		if heldByNextStep(c, time.Now()) {
			continue
		}
		suggestions = append(suggestions, &SuggestContact{
			Contact:  c,
			Type:     "cold",
			NextStep: c.NextStep,
		})
	}

	return suggestions, total, nil
}

func (s *Service) suggestFollowupContacts(ctx context.Context, userID uuid.UUID, orgIDs []uuid.UUID, limit int) ([]*SuggestContact, int, error) {
	// Find contacts needing follow-up action (not SEND, WAIT, or DROP)
	contacts, total, err := s.contactRepo.SearchWithFilter(ctx, userID, orgIDs, map[string]interface{}{
		"status":         "ready",
		"next_step":      []string{"FOLLOW_UP_1", "FOLLOW_UP_2", "SET_MEETING", "FOLLOW_UP_MEETING_1", "FOLLOW_UP_MEETING_2", "PREPARE_MEETING", "FOLLOW_UP"},
		"do_not_contact": false, // opted-out contacts are never due outreach
	}, nil)
	if err != nil {
		return nil, 0, err
	}

	suggestions := make([]*SuggestContact, 0, limit)
	repliedSuggestions := make([]*SuggestContact, 0)
	noReplySuggestions := make([]*SuggestContact, 0)
	postMeetingSuggestions := make([]*SuggestContact, 0)

	for _, c := range contacts {
		if heldByNextStep(c, time.Now()) {
			continue
		}
		// Load outreach state so mapActionTypeAndCategory can determine category
		state, _ := s.stateRepo.FindByContactID(ctx, c.ID, userID)

		suggestion := &SuggestContact{
			Contact:  c,
			State:    state,
			Type:     "followup",
			NextStep: c.NextStep,
		}

		switch c.OutreachStage {
		case string(outreach.StateReplied):
			repliedSuggestions = append(repliedSuggestions, suggestion)
		case string(outreach.StatePostMeeting):
			postMeetingSuggestions = append(postMeetingSuggestions, suggestion)
		case string(outreach.StateNoReply):
			noReplySuggestions = append(noReplySuggestions, suggestion)
		default:
			noReplySuggestions = append(noReplySuggestions, suggestion)
		}
	}

	// Priority: REPLIED > POST_MEETING > NO_REPLY
	suggestions = append(suggestions, repliedSuggestions...)
	suggestions = append(suggestions, postMeetingSuggestions...)
	suggestions = append(suggestions, noReplySuggestions...)

	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions, total, nil
}

func (s *Service) suggestMixedContacts(ctx context.Context, userID uuid.UUID, orgIDs []uuid.UUID, limit int) ([]*SuggestContact, int, error) {
	// 60% cold, 40% follow-up
	coldLimit := int(float64(limit) * 0.6)
	followupLimit := limit - coldLimit

	coldSuggestions, coldTotal, _ := s.suggestColdContacts(ctx, userID, orgIDs, coldLimit)
	followupSuggestions, followupTotal, _ := s.suggestFollowupContacts(ctx, userID, orgIDs, followupLimit)

	// Fill up if one type is short
	totalFound := len(coldSuggestions) + len(followupSuggestions)
	if totalFound < limit {
		remaining := limit - totalFound
		if len(coldSuggestions) < coldLimit {
			moreFollowup, _, _ := s.suggestFollowupContacts(ctx, userID, orgIDs, remaining+followupLimit)
			followupSuggestions = moreFollowup
		} else if len(followupSuggestions) < followupLimit {
			moreCold, _, _ := s.suggestColdContacts(ctx, userID, orgIDs, remaining+coldLimit)
			coldSuggestions = moreCold
		}
	}

	result := make([]*SuggestContact, 0, len(coldSuggestions)+len(followupSuggestions))
	result = append(result, coldSuggestions...)
	result = append(result, followupSuggestions...)

	return result, coldTotal + followupTotal, nil
}

// ============================================
// Draft Logic
// ============================================

// DraftContext holds context for message generation
type DraftContext struct {
	Contact      *contact.Contact
	State        *ConversationStateResult
	LastOutgoing *outreach.InteractionLog
	LastIncoming *outreach.InteractionLog
	Notes        []*outreach.InteractionLog // Team notes for context
	UserName     string                     // BD user's display name for message signature
}

// GenerateDraft generates a message draft for a contact
// Always uses AI if OpenAI client is configured, with full conversation history
func (s *Service) GenerateDraft(ctx context.Context, contactEntity *contact.Contact, userID uuid.UUID) (string, error) {
	stateResult, err := s.DetermineConversationState(ctx, contactEntity, userID)
	if err != nil {
		return "", err
	}

	// Cannot draft for dropped contacts
	if stateResult.State == outreach.StateDropped {
		return "", fmt.Errorf("contact dropped - no draft")
	}

	// Fetch full conversation history (not just notes)
	conversationHistory, histErr := s.interactionRepo.FindByContactIDAndUserID(ctx, contactEntity.ID, userID, 20)
	if histErr != nil {
		fmt.Printf("[GenerateDraft] Error fetching conversation history: %v\n", histErr)
	}

	// Fetch team notes separately
	notes, notesErr := s.interactionRepo.FindNotes(ctx, contactEntity.ID, userID, 10)
	if notesErr != nil {
		fmt.Printf("[GenerateDraft] Error fetching notes: %v\n", notesErr)
	}

	fmt.Printf("[GenerateDraft] Contact: %s, OpenAI configured: %v, History count: %d, Notes count: %d\n",
		contactEntity.ID, s.openAIClient != nil, len(conversationHistory), len(notes))

	draftCtx := &DraftContext{
		Contact:      contactEntity,
		State:        stateResult,
		LastOutgoing: stateResult.LastOutgoing,
		LastIncoming: stateResult.LastIncoming,
		Notes:        notes,
	}

	// Always use AI draft generation if OpenAI client is configured
	if s.openAIClient != nil {
		fmt.Printf("[GenerateDraft] Using AI draft generation\n")
		aiDraft, aiErr := s.generateDraftWithFullContext(ctx, draftCtx, conversationHistory)
		if aiErr != nil {
			fmt.Printf("[GenerateDraft] AI generation failed: %v, falling back to template\n", aiErr)
		} else if aiDraft != "" {
			fmt.Printf("[GenerateDraft] AI draft generated successfully\n")
			return aiDraft, nil
		}
		// Fallback to template-based draft if AI fails
	} else {
		fmt.Printf("[GenerateDraft] OpenAI not configured, using template\n")
	}

	return s.generateDraftByScenario(draftCtx), nil
}

// GenerateDraftWithLanguage generates a message draft for a contact with specified language
func (s *Service) GenerateDraftWithLanguage(ctx context.Context, contactEntity *contact.Contact, userID uuid.UUID, language string) (string, error) {
	stateResult, err := s.DetermineConversationState(ctx, contactEntity, userID)
	if err != nil {
		return "", err
	}

	// Cannot draft for dropped contacts
	if stateResult.State == outreach.StateDropped {
		return "", fmt.Errorf("contact dropped - no draft")
	}

	// Fetch full conversation history
	conversationHistory, histErr := s.interactionRepo.FindByContactIDAndUserID(ctx, contactEntity.ID, userID, 20)
	if histErr != nil {
		fmt.Printf("[GenerateDraftWithLanguage] Error fetching conversation history: %v\n", histErr)
	}

	// Fetch team notes separately
	notes, notesErr := s.interactionRepo.FindNotes(ctx, contactEntity.ID, userID, 10)
	if notesErr != nil {
		fmt.Printf("[GenerateDraftWithLanguage] Error fetching notes: %v\n", notesErr)
	}

	fmt.Printf("[GenerateDraftWithLanguage] Contact: %s, Language: %s, OpenAI configured: %v\n",
		contactEntity.ID, language, s.openAIClient != nil)

	draftCtx := &DraftContext{
		Contact:      contactEntity,
		State:        stateResult,
		LastOutgoing: stateResult.LastOutgoing,
		LastIncoming: stateResult.LastIncoming,
		Notes:        notes,
	}

	// Always use AI draft generation if OpenAI client is configured
	if s.openAIClient != nil {
		aiDraft, aiErr := s.generateDraftWithFullContextAndLanguage(ctx, draftCtx, conversationHistory, language)
		if aiErr != nil {
			fmt.Printf("[GenerateDraftWithLanguage] AI generation failed: %v, falling back to template\n", aiErr)
		} else if aiDraft != "" {
			return aiDraft, nil
		}
	}

	return s.generateDraftByScenario(draftCtx), nil
}

// GenerateDraftWithLanguageAndUserName generates a message draft with specified language and user name.
// If language is empty, AI will auto-detect the best language based on the contact's name.
func (s *Service) GenerateDraftWithLanguageAndUserName(ctx context.Context, contactEntity *contact.Contact, userID uuid.UUID, language string, userName string) (string, error) {
	stateResult, err := s.DetermineConversationState(ctx, contactEntity, userID)
	if err != nil {
		return "", err
	}

	// Cannot draft for dropped contacts
	if stateResult.State == outreach.StateDropped {
		return "", fmt.Errorf("contact dropped - no draft")
	}

	// Fetch full conversation history
	conversationHistory, histErr := s.interactionRepo.FindByContactIDAndUserID(ctx, contactEntity.ID, userID, 20)
	if histErr != nil {
		fmt.Printf("[GenerateDraftWithLanguageAndUserName] Error fetching conversation history: %v\n", histErr)
	}

	// Fetch team notes separately
	notes, notesErr := s.interactionRepo.FindNotes(ctx, contactEntity.ID, userID, 10)
	if notesErr != nil {
		fmt.Printf("[GenerateDraftWithLanguageAndUserName] Error fetching notes: %v\n", notesErr)
	}

	fmt.Printf("[GenerateDraftWithLanguageAndUserName] Contact: %s, Language: %s, UserName: %s, OpenAI configured: %v\n",
		contactEntity.ID, language, userName, s.openAIClient != nil)

	draftCtx := &DraftContext{
		Contact:      contactEntity,
		State:        stateResult,
		LastOutgoing: stateResult.LastOutgoing,
		LastIncoming: stateResult.LastIncoming,
		Notes:        notes,
		UserName:     userName,
	}

	// Always use AI draft generation if OpenAI client is configured
	if s.openAIClient != nil {
		aiDraft, aiErr := s.generateDraftWithFullContextAndLanguage(ctx, draftCtx, conversationHistory, language)
		if aiErr != nil {
			fmt.Printf("[GenerateDraftWithLanguageAndUserName] AI generation failed: %v, falling back to template\n", aiErr)
		} else if aiDraft != "" {
			return aiDraft, nil
		}
	}

	return s.generateDraftByScenario(draftCtx), nil
}

// GetDraftContext returns the full context for draft generation (for AI use)
func (s *Service) GetDraftContext(ctx context.Context, contactEntity *contact.Contact, userID uuid.UUID) (*DraftContext, error) {
	stateResult, err := s.DetermineConversationState(ctx, contactEntity, userID)
	if err != nil {
		return nil, err
	}

	// Fetch notes for context
	notes, _ := s.interactionRepo.FindNotes(ctx, contactEntity.ID, userID, 10)

	return &DraftContext{
		Contact:      contactEntity,
		State:        stateResult,
		LastOutgoing: stateResult.LastOutgoing,
		LastIncoming: stateResult.LastIncoming,
		Notes:        notes,
	}, nil
}

func (s *Service) generateDraftByScenario(ctx *DraftContext) string {
	switch ctx.State.Scenario {
	case outreach.ScenarioRoleBased:
		return s.generateColdRoleBasedDraft(ctx)
	case outreach.ScenarioIndustryBased:
		return s.generateColdIndustryBasedDraft(ctx)
	case outreach.ScenarioNoReplyFollowup:
		return s.generateNoReplyFollowupDraft(ctx)
	case outreach.ScenarioPostReply:
		return s.generatePostReplyDraft(ctx)
	case outreach.ScenarioMeetingConfirmation:
		return s.generateMeetingConfirmationDraft(ctx)
	case outreach.ScenarioPostMeeting:
		return s.generatePostMeetingDraft(ctx)
	case outreach.ScenarioReEngage:
		return s.generateReEngageDraft(ctx)
	default:
		return s.generateColdRoleBasedDraft(ctx)
	}
}

func (s *Service) generateColdRoleBasedDraft(ctx *DraftContext) string {
	name := s.getContactName(ctx.Contact)
	jobTitle := ctx.Contact.JobTitle
	company := ctx.Contact.Company

	// Customize value proposition based on job title
	valueProposition := s.getValuePropositionForRole(jobTitle)

	return fmt.Sprintf(`Chào %s,

Em thấy anh/chị đang làm %s tại %s - rất ấn tượng với những gì team đang xây dựng!

%s

Em có thể chia sẻ một vài insights từ các team tương tự mà em đã hỗ trợ. Anh/chị có tiện 15 phút tuần này để em demo nhanh không? Hoàn toàn không ràng buộc - em chỉ muốn xem có thể giúp gì cho anh/chị.

Cảm ơn anh/chị!`, name, jobTitle, company, valueProposition)
}

func (s *Service) generateColdIndustryBasedDraft(ctx *DraftContext) string {
	name := s.getContactName(ctx.Contact)
	jobTitle := ctx.Contact.JobTitle
	company := ctx.Contact.Company
	industry := ctx.Contact.Industry

	// Get industry-specific insight
	industryInsight := s.getIndustryInsight(industry)

	return fmt.Sprintf(`Chào %s,

Em thấy anh/chị đang làm %s tại %s - một trong những công ty ấn tượng trong ngành %s!

%s

Em vừa giúp một team tương tự tăng 40%% response rate trong 2 tuần. Anh/chị có muốn em chia sẻ cụ thể cách họ làm trong 15 phút không?

Nếu không phù hợp cũng không sao - em chỉ muốn connect và xem có thể hỗ trợ gì cho anh/chị thôi!`, name, jobTitle, company, industry, industryInsight)
}

func (s *Service) generateNoReplyFollowupDraft(ctx *DraftContext) string {
	name := s.getContactName(ctx.Contact)
	daysSince := ctx.State.DaysSinceLastInteraction
	followupCount := ctx.State.FollowupCount

	var timeRef string
	if daysSince == 1 {
		timeRef = "hôm qua"
	} else if daysSince < 7 {
		timeRef = fmt.Sprintf("%d ngày trước", daysSince)
	} else {
		timeRef = "tuần trước"
	}

	// Different follow-up messages based on count
	if followupCount == 0 {
		return fmt.Sprintf(`Chào %s,

Em quay lại về message %s - không biết anh/chị có cơ hội xem chưa?

Em hiểu anh/chị rất bận, nên em chỉ cần 10 phút để show một vài điều em nghĩ có thể giúp ích cho team. Nếu không phù hợp, anh/chị cứ nói thẳng nhé - em hoàn toàn tôn trọng!

Anh/chị thường rảnh khung nào trong tuần?`, name, timeRef)
	} else if followupCount == 1 {
		return fmt.Sprintf(`Chào %s,

Em follow-up lần cuối về tin nhắn %s. Em không muốn làm phiền anh/chị.

Nếu timing chưa phù hợp, em có thể:
• Gửi tài liệu để anh/chị đọc khi rảnh
• Kết nối lại sau 1-2 tháng
• Hoặc connect với người phù hợp hơn trong team

Anh/chị thấy option nào OK nhất?`, name, timeRef)
	}

	return fmt.Sprintf(`Chào %s,

Em hiểu timing có thể chưa phù hợp. Em sẽ không spam thêm nữa.

Nếu sau này có nhu cầu về outreach/sales automation, anh/chị cứ ping em nhé. Chúc anh/chị và team %s thành công!`, name, ctx.Contact.Company)
}

func (s *Service) generatePostReplyDraft(ctx *DraftContext) string {
	name := s.getContactName(ctx.Contact)
	company := ctx.Contact.Company

	// Extract key points from last incoming message
	lastReplyContent := ""
	if ctx.LastIncoming != nil {
		lastReplyContent = ctx.LastIncoming.Content
	}

	// Check sentiment
	sentiment := "neutral"
	if ctx.LastIncoming != nil && ctx.LastIncoming.Sentiment != nil {
		sentiment = *ctx.LastIncoming.Sentiment
	}

	var response string
	switch sentiment {
	case string(outreach.SentimentPositive):
		response = fmt.Sprintf(`Tuyệt vời! Rất vui khi anh/chị quan tâm.

Để giúp anh/chị hiểu rõ hơn cách em có thể hỗ trợ %s, em đề xuất một cuộc gọi ngắn 20 phút:
• Demo nhanh product với case study cụ thể
• Q&A để giải đáp mọi thắc mắc
• Nếu phù hợp, em có thể setup trial cho team

Anh/chị có thể chọn slot tại đây: [calendar link] hoặc cho em biết khung giờ tiện nhé!`, company)
	case string(outreach.SentimentNegative):
		response = `Cảm ơn anh/chị đã thẳng thắn chia sẻ. Em hoàn toàn tôn trọng quyết định của anh/chị.

Nếu sau này có nhu cầu hoặc muốn tham khảo ý kiến về outreach strategy, anh/chị cứ ping em nhé - em luôn sẵn sàng hỗ trợ không ràng buộc.

Chúc anh/chị và team thành công!`
	default:
		// Check for common phrases in reply
		replyLower := strings.ToLower(lastReplyContent)
		if strings.Contains(replyLower, "tuần sau") || strings.Contains(replyLower, "next week") {
			response = `Perfect! Vậy tuần sau em sẽ follow-up để book lịch nhé.

Trong lúc đó, anh/chị có muốn em gửi trước một case study của team tương tự để tham khảo không? Sẽ giúp anh/chị có cái nhìn tổng quan trước khi em nói chuyện.`
		} else if strings.Contains(replyLower, "bận") || strings.Contains(replyLower, "busy") {
			response = `Em hiểu anh/chị đang bận. Không sao cả!

Em có thể:
1. Gửi video demo 3 phút để anh/chị xem khi rảnh
2. Book lịch trước cho 2 tuần sau
3. Kết nối với ai đó trong team có thể evaluate trước

Anh/chị thấy option nào tiện nhất?`
		} else {
			response = fmt.Sprintf(`Cảm ơn anh/chị đã phản hồi!

Để em có thể đề xuất giải pháp phù hợp nhất cho %s, anh/chị có thể chia sẻ thêm:
• Team sales hiện có bao nhiêu người?
• Đang dùng tool gì để outreach (LinkedIn/Email/CRM)?
• Mục tiêu chính là tăng response rate hay scale volume?

Từ đó em sẽ chuẩn bị demo custom cho anh/chị!`, company)
		}
	}

	return fmt.Sprintf(`Chào %s,

%s`, name, response)
}

func (s *Service) generateMeetingConfirmationDraft(ctx *DraftContext) string {
	name := s.getContactName(ctx.Contact)

	return fmt.Sprintf(`Chào %s,

Tuyệt vời! Em confirm lịch meeting nhé:

📅 **Thời gian:** [Thời gian đã chốt]
📍 **Hình thức:** [Zoom/Google Meet/Call]

Trước buổi gặp, em sẽ gửi thêm:
• Link meeting + agenda
• Một số tài liệu tham khảo về giải pháp

Anh/chị có cần em chuẩn bị thêm gì không? Hẹn gặp anh/chị! 🙌`, name)
}

func (s *Service) generatePostMeetingDraft(ctx *DraftContext) string {
	name := s.getContactName(ctx.Contact)
	company := ctx.Contact.Company

	return fmt.Sprintf(`Chào %s,

Cảm ơn anh/chị đã dành thời gian trao đổi! Rất vui được hiểu thêm về %s và những thách thức team đang đối mặt.

Như đã thảo luận, đây là summary và next steps:

📋 **Những gì em đã thống nhất:**
• [Điểm chính 1 từ cuộc họp]
• [Điểm chính 2 từ cuộc họp]

🎯 **Bước tiếp theo:**
• Em sẽ gửi proposal/tài liệu chi tiết trong [X ngày]
• [Action item từ phía anh/chị]
• Book follow-up call vào [ngày] để review

Anh/chị confirm giúp em timeline này nhé. Nếu có gì thay đổi hoặc cần thêm thông tin, cứ ping em bất cứ lúc nào!

Cảm ơn anh/chị! 🙏`, name, company)
}

func (s *Service) generateReEngageDraft(ctx *DraftContext) string {
	name := s.getContactName(ctx.Contact)
	company := ctx.Contact.Company
	daysSince := ctx.State.DaysSinceLastInteraction

	var timeContext string
	if daysSince > 90 {
		timeContext = "vài tháng"
	} else if daysSince > 30 {
		timeContext = "hơn 1 tháng"
	} else {
		timeContext = "một thời gian"
	}

	return fmt.Sprintf(`Chào %s,

Lâu rồi em chưa liên lạc (%s rồi nhỉ!). Hy vọng anh/chị và team %s vẫn khỏe và đang có nhiều thành tựu mới!

Em quay lại vì có một vài updates em nghĩ anh/chị sẽ quan tâm:

🚀 **Những gì mới:**
• Tính năng AI giúp personalize message tự động (tăng 50%% response rate)
• Integration mới với [CRM/tool phổ biến]
• Case study từ team tương tự với %s

Không biết context của anh/chị giờ thế nào - nếu outreach vẫn là priority, em rất vui được catch up 15 phút và chia sẻ những gì mới.

Nếu không phù hợp lúc này cũng không sao - em chỉ muốn reconnect thôi! 😊`, name, timeContext, company, company)
}

func (s *Service) getContactName(c *contact.Contact) string {
	if c.Name != "" && c.Name != "N/A" {
		// Return just the first part of the name (first name)
		parts := strings.Fields(c.Name)
		if len(parts) > 0 {
			return parts[0]
		}
		return c.Name
	}
	return "anh/chị"
}

// getValuePropositionForRole returns a customized value proposition based on job title
func (s *Service) getValuePropositionForRole(jobTitle string) string {
	jobTitleLower := strings.ToLower(jobTitle)

	switch {
	case strings.Contains(jobTitleLower, "ceo") || strings.Contains(jobTitleLower, "founder") || strings.Contains(jobTitleLower, "director"):
		return "Em hiểu anh/chị cần tối ưu hiệu quả của team sales với nguồn lực có hạn. Em có thể giúp tự động hóa quy trình outreach, giúp team tập trung vào những deals quan trọng nhất."
	case strings.Contains(jobTitleLower, "sales") || strings.Contains(jobTitleLower, "bd") || strings.Contains(jobTitleLower, "business development"):
		return "Em biết việc tiếp cận đúng người, đúng thời điểm là thách thức lớn nhất. Em có thể giúp anh/chị xác định leads tiềm năng và tối ưu message để tăng tỷ lệ phản hồi."
	case strings.Contains(jobTitleLower, "marketing"):
		return "Em hiểu marketing cần align chặt với sales để tạo pipeline chất lượng. Em có thể giúp team track được ROI từ từng chiến dịch outreach và cải thiện conversion."
	case strings.Contains(jobTitleLower, "hr") || strings.Contains(jobTitleLower, "talent") || strings.Contains(jobTitleLower, "recruiting"):
		return "Em biết việc tiếp cận ứng viên tốt ngày càng khó. Em có thể giúp anh/chị personalize outreach đến từng ứng viên và theo dõi hiệu quả của từng message."
	case strings.Contains(jobTitleLower, "product") || strings.Contains(jobTitleLower, "tech") || strings.Contains(jobTitleLower, "engineer"):
		return "Em hiểu team tech cần tools hiệu quả, dễ integrate. Em có API và automation giúp anh/chị connect với CRM hiện tại và tự động hóa workflow."
	default:
		return "Em có thể giúp team của anh/chị tiếp cận khách hàng tiềm năng một cách hiệu quả hơn, với message được personalize cho từng người và theo dõi kết quả real-time."
	}
}

// getIndustryInsight returns industry-specific insights and challenges
func (s *Service) getIndustryInsight(industry string) string {
	industryLower := strings.ToLower(industry)

	switch {
	case strings.Contains(industryLower, "finance") || strings.Contains(industryLower, "banking") || strings.Contains(industryLower, "fintech"):
		return "Ngành tài chính đòi hỏi sự tin tưởng cao từ khách hàng. Em có thể giúp anh/chị xây dựng relationship một cách có hệ thống, với follow-up đúng thời điểm và nội dung phù hợp."
	case strings.Contains(industryLower, "education") || strings.Contains(industryLower, "edtech"):
		return "Ngành education có cycle dài và nhiều stakeholders. Em có thể giúp anh/chị track conversation với từng người và coordinate outreach một cách thông minh."
	// Generic "tech" goes after fintech/edtech, which also contain "tech".
	case strings.Contains(industryLower, "tech") || strings.Contains(industryLower, "software") || strings.Contains(industryLower, "saas"):
		return "Em hiểu ngành tech cạnh tranh khốc liệt - khách hàng nhận hàng trăm cold outreach mỗi tuần. Em có thể giúp anh/chị nổi bật với message cá nhân hóa dựa trên hành vi và nhu cầu thực sự của từng prospect."
	case strings.Contains(industryLower, "healthcare") || strings.Contains(industryLower, "medical"):
		return "Ngành healthcare cần approach chuyên nghiệp và compliance cao. Em có thể giúp anh/chị personalize message cho từng role (bác sĩ, admin, procurement) một cách hiệu quả."
	case strings.Contains(industryLower, "retail") || strings.Contains(industryLower, "ecommerce"):
		return "Retail/E-commerce cần tiếp cận nhanh và scale lớn. Em có thể giúp anh/chị automate outreach mà vẫn giữ được sự cá nhân hóa cho từng brand."
	default:
		return "Em đã làm việc với nhiều team trong ngành tương tự và hiểu những thách thức về outreach. Em có thể chia sẻ best practices và giúp anh/chị áp dụng ngay."
	}
}

// ============================================
// AI Draft Generation
// ============================================

// generateDraftWithFullContext generates a personalized message using AI with FULL conversation history
// This is the primary method - it considers ALL context to create the perfect next message
func (s *Service) generateDraftWithFullContext(ctx context.Context, draftCtx *DraftContext, conversationHistory []*outreach.InteractionLog) (string, error) {
	if s.openAIClient == nil {
		return "", fmt.Errorf("OpenAI client not configured")
	}

	// Build comprehensive prompt with full conversation history
	prompt := s.buildFullContextPrompt(draftCtx, conversationHistory)

	// System prompt focused on understanding customer and creating engaging messages
	systemPrompt := `Bạn là một chuyên gia tư vấn B2B - mục tiêu của bạn là THỰC SỰ GIÚP ĐỠ khách hàng, không phải bán hàng.

🎯 MỤC TIÊU CHÍNH:
Dựa vào TOÀN BỘ lịch sử hội thoại và thông tin khách hàng, viết tin nhắn tiếp theo sao cho khách hàng:
1. Cảm thấy được LẮNG NGHE và THẤU HIỂU
2. Nhận ra bạn đang cố gắng GIÚP HỌ, không phải bán hàng
3. HỨNG THÚ muốn tiếp tục cuộc hội thoại

📝 PHÂN TÍCH TRƯỚC KHI VIẾT:
- Khách hàng này đang ở giai đoạn nào? (Mới biết / Đã quan tâm / Đang cân nhắc / Đã từ chối)
- Họ đã nói gì? Quan tâm điều gì? Lo ngại điều gì?
- Tin nhắn trước của em có hiệu quả không? Cần điều chỉnh gì?
- Điều gì sẽ khiến họ MUỐN reply?

💬 NGUYÊN TẮC VIẾT:
1. REFERENCE cụ thể những gì họ đã nói/chia sẻ trước đó
2. ACKNOWLEDGE cảm xúc và quan điểm của họ
3. PROVIDE VALUE trước khi ask anything (insight, tip, resource)
4. CTA phải NATURAL và LOW-PRESSURE

🚫 TUYỆT ĐỐI TRÁNH:
- Tin nhắn generic có thể gửi cho bất kỳ ai
- Push bán hàng khi họ chưa sẵn sàng
- Bỏ qua những gì họ đã nói trước đó
- Dùng template cứng nhắc

✅ NÊN LÀM:
- Personalize dựa trên conversation history
- Tiếp nối tự nhiên từ tin nhắn trước
- Show empathy nếu họ có concerns
- Offer something helpful (không đòi hỏi gì)

📏 FORMAT:
- 3-5 câu ngắn gọn
- Tiếng Việt tự nhiên, thân thiện
- Có thể dùng 1-2 emoji phù hợp
- Nếu là tin đầu tiên: mở đầu với observation về họ
- Nếu là follow-up: reference tin nhắn trước
- KHÔNG ký tên ở cuối tin nhắn (không có "Anh Khoa", "Trân trọng", v.v.)

⚠️ QUY TẮC XƯNG HÔ (BẮT BUỘC - QUAN TRỌNG NHẤT):
- LUÔN LUÔN xưng "em" khi nói về bản thân. VD: "em muốn chia sẻ", "em thấy rằng"
- LUÔN LUÔN gọi đối phương là "anh/chị" (CẢ HAI, KHÔNG ĐƯỢC CHỌN MỘT). VD: "Chào anh/chị [Tên]", "anh/chị có thể"
- KHÔNG BAO GIỜ chỉ dùng "anh" hoặc chỉ dùng "chị" riêng lẻ. PHẢI LUÔN viết "anh/chị" đầy đủ
- KHÔNG BAO GIỜ dùng: "tôi", "mình", "bạn", "quý khách"

Output: CHỈ TRẢ VỀ NỘI DUNG TIN NHẮN. Không giải thích.`

	// Generate using OpenAI
	response, err := s.openAIClient.ChatCompletion(ctx, ai.ChatCompletionRequest{
		SystemPrompt: systemPrompt,
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.85,
		MaxTokens:   600,
	})

	if err != nil {
		return "", fmt.Errorf("AI draft generation failed: %w", err)
	}

	// Post-process: replace any name placeholders with actual user name
	result := strings.TrimSpace(response)
	result = replaceNamePlaceholders(result, draftCtx.UserName)
	result = normalizeVietnamesePronouns(result)

	return result, nil
}

// generateDraftWithFullContextAndLanguage generates a personalized message using AI with language support
func (s *Service) generateDraftWithFullContextAndLanguage(ctx context.Context, draftCtx *DraftContext, conversationHistory []*outreach.InteractionLog, language string) (string, error) {
	if s.openAIClient == nil {
		return "", fmt.Errorf("OpenAI client not configured")
	}

	// Build comprehensive prompt with full conversation history
	prompt := s.buildFullContextPromptWithLanguage(draftCtx, conversationHistory, language)

	// Get system prompt based on language
	systemPrompt := s.getSystemPromptForLanguage(language)

	// Generate using OpenAI
	response, err := s.openAIClient.ChatCompletion(ctx, ai.ChatCompletionRequest{
		SystemPrompt: systemPrompt,
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.85,
		MaxTokens:   600,
	})

	if err != nil {
		return "", fmt.Errorf("AI draft generation failed: %w", err)
	}

	// Post-process: replace any name placeholders with actual user name
	result := strings.TrimSpace(response)
	result = replaceNamePlaceholders(result, draftCtx.UserName)
	result = normalizeVietnamesePronouns(result)

	return result, nil
}

// replaceNamePlaceholders replaces common AI-generated name placeholders with the actual user name
func replaceNamePlaceholders(text string, userName string) string {
	placeholders := []string{
		"[Tên bạn]",
		"[tên bạn]",
		"[Tên của bạn]",
		"[tên của bạn]",
		"[Your Name]",
		"[your name]",
		"[YOUR NAME]",
		"[Tên Bạn]",
	}
	for _, placeholder := range placeholders {
		text = strings.ReplaceAll(text, placeholder, userName)
	}
	return text
}

// Compiled regexes for Vietnamese pronoun normalization
var (
	// Match standalone "anh" not preceded by a letter and not followed by a letter or "/"
	reStandaloneAnh = regexp.MustCompile(`(^|[^\p{L}])([Aa]nh)([^\p{L}/]|$)`)
	// Match standalone "chị" not preceded by a letter or "/" and not followed by a letter
	reStandaloneChi = regexp.MustCompile(`(^|[^\p{L}/])([Cc]hị)([^\p{L}]|$)`)
)

// normalizeVietnamesePronouns replaces standalone "anh" or "chị" with "anh/chị"
// to ensure gender-neutral addressing in Vietnamese messages.
// Safe to call on non-Vietnamese text (no Vietnamese words = no replacements).
func normalizeVietnamesePronouns(text string) string {
	// Run 2 passes to handle adjacent matches where boundaries overlap
	for i := 0; i < 2; i++ {
		text = reStandaloneAnh.ReplaceAllString(text, "${1}anh/chị${3}")
		text = reStandaloneChi.ReplaceAllString(text, "${1}anh/chị${3}")
	}
	// Fix capitalization at start of text
	if strings.HasPrefix(text, "anh/chị") {
		text = "Anh/chị" + text[len("anh/chị"):]
	}
	return text
}

// getSystemPromptForLanguage returns the appropriate system prompt based on language.
// If language is empty, returns a prompt that tells AI to auto-detect from contact name.
func (s *Service) getSystemPromptForLanguage(language string) string {
	if language == "" {
		return `You are a B2B consultant expert. Your goal is to GENUINELY HELP the customer, not to sell.

🌐 LANGUAGE DETECTION:
You MUST detect the contact's nationality from their name, company, and location, then write the message in their NATIVE LANGUAGE.

Use your world knowledge of name origins, phonetics, etymology, and cultural patterns. Examples:
- "Nguyen Van Minh", "Tran Thi Lan" → Vietnamese → write in Vietnamese
- "Hiroaki Yoshino", "Tanaka Yui" → Japanese → write in Japanese
- "김민수", "Park Jimin" → Korean → write in Korean
- "王伟", "Zhang Wei", "李华" → Chinese → write in Chinese

⚠️ IMPORTANT: For ALL OTHER name origins, ALWAYS write in ENGLISH:
- Indian/South Asian: "Naba Fatima", "Rajesh Kumar", "Priya Sharma", "Muhammad Ali" → ENGLISH
- Arabic/Middle Eastern: "Ahmed Hassan", "Fatima Al-Said" → ENGLISH
- Southeast Asian (Thai, Filipino, etc.): "Somchai", "Maria Santos" → ENGLISH
- European: "Pierre Dubois", "Müller", "Marco Rossi", "García" → ENGLISH
- African: "Kwame Asante", "Amara Diallo" → ENGLISH
- Any other origin: → ENGLISH

Rules:
1. ONLY write in Vietnamese, Japanese, Korean, or Chinese if the name CLEARLY originates from those cultures.
2. For ALL other nationalities (Indian, Arabic, European, African, etc.), ALWAYS write in ENGLISH. B2B communication with these contacts should be in English.
3. Cross-reference with company name, city, and country as secondary signals.
4. If truly ambiguous, default to ENGLISH.
5. ALWAYS use polite/professional business tone appropriate for that language's culture.

LANGUAGE-SPECIFIC TONE:
- Vietnamese: LUÔN xưng "em", LUÔN gọi đối phương là "anh/chị" (CẢ HAI, KHÔNG chọn riêng "anh" hay "chị"). VD: "Chào anh/chị [Tên], em là...". KHÔNG dùng "tôi", "mình", "bạn".
- Japanese: です/ます form, address as 〇〇さん
- Korean: 존댓말, address as 〇〇님
- All languages: 3-5 short sentences, natural tone, 1-2 emojis OK, DO NOT sign your name at the end

CORE PRINCIPLES:
1. REFERENCE specifically what they've said/shared before
2. ACKNOWLEDGE their emotions and viewpoints
3. PROVIDE VALUE before asking anything
4. CTA must be NATURAL and LOW-PRESSURE

ABSOLUTELY AVOID:
- Generic messages that could be sent to anyone
- Pushing sales when they're not ready
- Ignoring what they've said before
- Claiming to work at the contact's company. Their employer is where THEY work,
  never yours: "At <their company>, we've been enhancing our offerings" reads as an
  obvious template error and destroys credibility on the first line.

Output: ONLY RETURN THE MESSAGE CONTENT. No explanations. No language labels.`
	}

	if language == "en" {
		return `You are a B2B consultant expert - your goal is to GENUINELY HELP the customer, not to sell.

🎯 MAIN OBJECTIVES:
Based on the FULL conversation history and customer information, write the next message so that the customer:
1. Feels HEARD and UNDERSTOOD
2. Recognizes you're trying to HELP THEM, not sell
3. Is EXCITED to continue the conversation

📝 ANALYZE BEFORE WRITING:
- What stage is this customer at? (New / Interested / Considering / Declined)
- What have they said? What are they interested in? What concerns do they have?
- Was my previous message effective? What adjustments are needed?
- What would make them WANT to reply?

💬 WRITING PRINCIPLES:
1. REFERENCE specifically what they've said/shared before
2. ACKNOWLEDGE their emotions and viewpoints
3. PROVIDE VALUE before asking anything (insight, tip, resource)
4. CTA must be NATURAL and LOW-PRESSURE

🚫 ABSOLUTELY AVOID:
- Generic messages that could be sent to anyone
- Pushing sales when they're not ready
- Ignoring what they've said before
- Using rigid templates
- Claiming to work at the contact's company. Their employer is where THEY work,
  never yours: "At <their company>, we've been enhancing our offerings" reads as an
  obvious template error and destroys credibility on the first line.

✅ DO:
- Personalize based on conversation history
- Continue naturally from the previous message
- Show empathy if they have concerns
- Offer something helpful (without demanding anything)

📝 MESSAGE GUIDELINES BY NEXT_STEP:

📤 COLD OUTREACH:
- SEND: Focus on insight/observation about them + offer specific value

📭 NO REPLY (NO_REPLY):
- FOLLOW_UP_1: MUST start with "Following up on my message from [X days] ago" or "Not sure if you had a chance to see my previous message". Be gentle, respectful, add new value or different angle
- FOLLOW_UP_2: MUST mention this is "my last follow-up" or "I don't want to bother you". Offer alternatives: (1) send materials, (2) reconnect later, (3) introduce to someone else

💬 REPLIED:
- SET_MEETING: Thank them for replying, respond directly to their message, PROPOSE SPECIFIC TIMES to meet (e.g., "Would Tuesday 10am or Thursday 2pm work for you?")

📅 WAITING FOR MEETING CONFIRMATION:
- FOLLOW_UP_MEETING_1: MUST reference the proposed time "Following up on the meeting time I suggested". Ask if the time works, offer alternatives if needed
- FOLLOW_UP_MEETING_2: MUST say "this is my last reminder about the meeting". Offer options: (1) confirm original time, (2) suggest different time, (3) reconnect when they're less busy

✅ MEETING CONFIRMED:
- PREPARE_MEETING: Confirm meeting time, SEND MEETING LINK (Zoom/Google Meet), send brief agenda, ask if they need anything else prepared

🤝 POST-MEETING:
- FOLLOW_UP: MUST start with "Thank you for taking the time to meet [yesterday/last week]". Recap key discussion points, list specific action items, propose clear next steps

🔄 RE-ENGAGE:
- RE_ENGAGE: Warm, friendly, mention something new/relevant since last contact

📏 FORMAT:
- 3-5 short sentences
- Natural, friendly English
- Can use 1-2 appropriate emojis
- If first message: start with an observation about them
- If follow-up: reference the previous message
- DO NOT sign your name at the end of the message (no "Best regards, Name", etc.)

Output: ONLY RETURN THE MESSAGE CONTENT. No explanations.`
	}

	if language == "ja" {
		return `あなたはB2Bコンサルタントの専門家です。目標は顧客を本当に助けることであり、売り込みではありません。

🎯 主な目標:
会話履歴と顧客情報に基づいて、次のメッセージを作成してください：
1. 顧客が「聞いてもらえている」「理解してもらえている」と感じること
2. あなたが「助けようとしている」と認識してもらうこと
3. 会話を続けたいと思ってもらうこと

📝 書く前に分析:
- この顧客はどの段階にいるか？（新規 / 関心あり / 検討中 / 辞退）
- 何を言ったか？何に興味があるか？何を懸念しているか？
- 前回のメッセージは効果的だったか？調整が必要か？
- 何が返信したいと思わせるか？

💬 ライティング原則:
1. 相手が以前言ったことを具体的に参照する
2. 相手の感情や視点を認める
3. 何かを求める前に価値を提供する（インサイト、ヒント、リソース）
4. CTAは自然で低圧力であること

🚫 絶対に避けること:
- 誰にでも送れるような汎用的なメッセージ
- 準備ができていない時にセールスを押し付けること
- 相手が以前言ったことを無視すること
- 硬直したテンプレートの使用

📏 フォーマット:
- 3〜5文の短い文章
- 自然でプロフェッショナルな日本語（です/ます体）
- 相手を「〇〇さん」と呼ぶ
- 適切な絵文字を1〜2個使用可能
- 最初のメッセージ：相手に関する観察から始める
- フォローアップ：前回のメッセージを参照する
- メッセージの最後に署名しないこと（「よろしくお願いします、〇〇」などは不要）

Output: メッセージの内容のみを返してください。説明は不要です。`
	}

	// Default: Vietnamese
	return `Bạn là một chuyên gia tư vấn B2B - mục tiêu của bạn là THỰC SỰ GIÚP ĐỠ khách hàng, không phải bán hàng.

🎯 MỤC TIÊU CHÍNH:
Dựa vào TOÀN BỘ lịch sử hội thoại và thông tin khách hàng, viết tin nhắn tiếp theo sao cho khách hàng:
1. Cảm thấy được LẮNG NGHE và THẤU HIỂU
2. Nhận ra bạn đang cố gắng GIÚP HỌ, không phải bán hàng
3. HỨNG THÚ muốn tiếp tục cuộc hội thoại

📝 PHÂN TÍCH TRƯỚC KHI VIẾT:
- Khách hàng này đang ở giai đoạn nào? (Mới biết / Đã quan tâm / Đang cân nhắc / Đã từ chối)
- Họ đã nói gì? Quan tâm điều gì? Lo ngại điều gì?
- Tin nhắn trước của em có hiệu quả không? Cần điều chỉnh gì?
- Điều gì sẽ khiến họ MUỐN reply?

💬 NGUYÊN TẮC VIẾT:
1. REFERENCE cụ thể những gì họ đã nói/chia sẻ trước đó
2. ACKNOWLEDGE cảm xúc và quan điểm của họ
3. PROVIDE VALUE trước khi ask anything (insight, tip, resource)
4. CTA phải NATURAL và LOW-PRESSURE

🚫 TUYỆT ĐỐI TRÁNH:
- Tin nhắn generic có thể gửi cho bất kỳ ai
- Push bán hàng khi họ chưa sẵn sàng
- Bỏ qua những gì họ đã nói trước đó
- Dùng template cứng nhắc

✅ NÊN LÀM:
- Personalize dựa trên conversation history
- Tiếp nối tự nhiên từ tin nhắn trước
- Show empathy nếu họ có concerns
- Offer something helpful (không đòi hỏi gì)

📏 FORMAT:
- 3-5 câu ngắn gọn
- Tiếng Việt tự nhiên, thân thiện
- Có thể dùng 1-2 emoji phù hợp
- Nếu là tin đầu tiên: mở đầu với observation về họ
- Nếu là follow-up: reference tin nhắn trước
- KHÔNG ký tên ở cuối tin nhắn (không có "Anh Khoa", "Trân trọng", v.v.)

⚠️ QUY TẮC XƯNG HÔ (BẮT BUỘC - QUAN TRỌNG NHẤT):
- LUÔN LUÔN xưng "em" khi nói về bản thân. VD: "em muốn chia sẻ", "em thấy rằng"
- LUÔN LUÔN gọi đối phương là "anh/chị" (CẢ HAI, KHÔNG ĐƯỢC CHỌN MỘT). VD: "Chào anh/chị [Tên]", "anh/chị có thể"
- KHÔNG BAO GIỜ chỉ dùng "anh" hoặc chỉ dùng "chị" riêng lẻ. PHẢI LUÔN viết "anh/chị" đầy đủ
- KHÔNG BAO GIỜ dùng: "tôi", "mình", "bạn", "quý khách"

Output: CHỈ TRẢ VỀ NỘI DUNG TIN NHẮN. Không giải thích.`
}

// buildFullContextPromptWithLanguage builds a comprehensive prompt with language support
func (s *Service) buildFullContextPromptWithLanguage(draftCtx *DraftContext, conversationHistory []*outreach.InteractionLog, language string) string {
	// For auto-detect (empty language), use English format for the context prompt
	// The AI will decide the output language based on the contact name
	if language == "" {
		language = "en"
	}

	var sb strings.Builder

	if language == "en" {
		// English version
		// Add sender (BD) information first
		if draftCtx.UserName != "" {
			sb.WriteString("═══════════════════════════════════════\n")
			sb.WriteString("🙋 YOUR INFORMATION (Sender)\n")
			sb.WriteString("═══════════════════════════════════════\n")
			sb.WriteString(fmt.Sprintf("Your name: %s\n", draftCtx.UserName))
			sb.WriteString("(This is the sender's name - DO NOT sign with this name at the end of the message)\n\n")
		}

		sb.WriteString("═══════════════════════════════════════\n")
		sb.WriteString("👤 CUSTOMER INFORMATION\n")
		sb.WriteString("═══════════════════════════════════════\n")
		sb.WriteString(fmt.Sprintf("Name: %q\n", sanitizeField(draftCtx.Contact.Name)))
		if draftCtx.Contact.JobTitle != "" {
			sb.WriteString(fmt.Sprintf("Title: %q\n", sanitizeField(draftCtx.Contact.JobTitle)))
		}
		if draftCtx.Contact.Company != "" {
			// Spelled out because this is the only company name in the prompt.
			// Left as a bare "Company:" the model assumed it was its own and
			// wrote "At <their company>, we've been enhancing our offerings".
			sb.WriteString(fmt.Sprintf("Company where THEY work: %q\n", sanitizeField(draftCtx.Contact.Company)))
			sb.WriteString(fmt.Sprintf("(You do NOT work at %s. Never write \"At %s, we...\" or imply you are their colleague.)\n",
				sanitizeField(draftCtx.Contact.Company), sanitizeField(draftCtx.Contact.Company)))
		}
		if draftCtx.Contact.Industry != "" {
			sb.WriteString(fmt.Sprintf("Their industry: %q\n", sanitizeField(draftCtx.Contact.Industry)))
		}

		sb.WriteString(buildContactFactsBlock(draftCtx.Contact, "en"))

		sb.WriteString("\n═══════════════════════════════════════\n")
		sb.WriteString("📊 CURRENT STATUS\n")
		sb.WriteString("═══════════════════════════════════════\n")
		sb.WriteString(fmt.Sprintf("Stage: %s\n", s.translateStateToEnglish(draftCtx.State.State)))
		sb.WriteString(fmt.Sprintf("Scenario: %s\n", s.translateScenarioToEnglish(draftCtx.State.Scenario)))
		sb.WriteString(fmt.Sprintf("Days since last contact: %d\n", draftCtx.State.DaysSinceLastInteraction))
		sb.WriteString(fmt.Sprintf("Follow-up count: %d\n", draftCtx.State.FollowupCount))
		sb.WriteString(fmt.Sprintf("🎯 NEXT ACTION: %s\n", s.translateNextStepToEnglish(draftCtx.State.NextStep, draftCtx.State.FollowupCount)))

		sb.WriteString("\n═══════════════════════════════════════\n")
		sb.WriteString("💬 CONVERSATION HISTORY (chronological)\n")
		sb.WriteString("═══════════════════════════════════════\n")

		if len(conversationHistory) == 0 {
			sb.WriteString("(No conversation history - this is the first message)\n")
		} else {
			for i := len(conversationHistory) - 1; i >= 0; i-- {
				interaction := conversationHistory[i]
				timeStr := interaction.Timestamp.Format("02/01/2006 15:04")

				var roleLabel string
				switch interaction.Direction {
				case string(outreach.DirectionOutgoing):
					roleLabel = "🔵 SENT"
				case string(outreach.DirectionIncoming):
					roleLabel = "🟢 CUSTOMER REPLIED"
				case string(outreach.DirectionInternal):
					continue
				default:
					roleLabel = "📝 NOTE"
				}

				sb.WriteString(fmt.Sprintf("\n[%s] %s via %s\n", timeStr, roleLabel, interaction.Channel))
				sb.WriteString(fmt.Sprintf("Content: %s\n", interaction.Content))
				if interaction.Sentiment != nil && *interaction.Sentiment != "" {
					sb.WriteString(fmt.Sprintf("Sentiment: %s\n", *interaction.Sentiment))
				}
			}
		}

		if len(draftCtx.Notes) > 0 {
			sb.WriteString("\n═══════════════════════════════════════\n")
			sb.WriteString("📋 TEAM NOTES\n")
			sb.WriteString("═══════════════════════════════════════\n")
			for _, note := range draftCtx.Notes {
				sb.WriteString(fmt.Sprintf("- %s\n", note.Content))
			}
		}

		sb.WriteString("\n═══════════════════════════════════════\n")
		sb.WriteString("✍️ INSTRUCTION\n")
		sb.WriteString("═══════════════════════════════════════\n")

		// Add specific requirement based on next_step (English)
		switch draftCtx.State.NextStep {
		case outreach.NextStepFollowUp1:
			sb.WriteString("⚠️ THIS IS FOLLOW-UP #1: MUST start by referencing the previous message (e.g., 'Following up on my message from X days ago...')\n")
		case outreach.NextStepFollowUp2:
			sb.WriteString("⚠️ THIS IS FOLLOW-UP #2 (FINAL): MUST mention this is the last follow-up and offer alternatives\n")
		case outreach.NextStepSetMeeting:
			sb.WriteString("⚠️ CUSTOMER REPLIED: Respond directly to what they said and PROPOSE SPECIFIC MEETING TIMES\n")
		case outreach.NextStepFollowUpMeeting1:
			sb.WriteString("⚠️ MEETING CONFIRMATION FOLLOW-UP #1: MUST reference the proposed meeting time\n")
		case outreach.NextStepFollowUpMeeting2:
			sb.WriteString("⚠️ MEETING CONFIRMATION FOLLOW-UP #2 (FINAL): MUST mention this is the last reminder about the meeting\n")
		case outreach.NextStepPrepareMeeting:
			sb.WriteString("⚠️ PREPARE MEETING: Confirm time, SEND MEETING LINK, send brief agenda\n")
		case outreach.NextStepFollowUp:
			sb.WriteString("⚠️ POST-MEETING FOLLOW-UP: MUST start by thanking them for the meeting, recap key points and action items\n")
		}

		sb.WriteString("Based on the conversation history above, write the next message that naturally continues the conversation and addresses customer needs.\n")
	} else {
		// Vietnamese version (default)
		// Add sender (BD) information first
		if draftCtx.UserName != "" {
			sb.WriteString("═══════════════════════════════════════\n")
			sb.WriteString("🙋 THÔNG TIN NGƯỜI GỬI (Bạn)\n")
			sb.WriteString("═══════════════════════════════════════\n")
			sb.WriteString(fmt.Sprintf("Tên của bạn: %s\n", draftCtx.UserName))
			sb.WriteString("(Đây là tên người gửi - KHÔNG ký tên này ở cuối tin nhắn)\n\n")
		}

		sb.WriteString("═══════════════════════════════════════\n")
		sb.WriteString("👤 THÔNG TIN KHÁCH HÀNG\n")
		sb.WriteString("═══════════════════════════════════════\n")
		sb.WriteString(fmt.Sprintf("Tên: %q\n", sanitizeField(draftCtx.Contact.Name)))
		if draftCtx.Contact.JobTitle != "" {
			sb.WriteString(fmt.Sprintf("Chức vụ: %q\n", sanitizeField(draftCtx.Contact.JobTitle)))
		}
		if draftCtx.Contact.Company != "" {
			sb.WriteString(fmt.Sprintf("Công ty NƠI HỌ LÀM VIỆC: %q\n", sanitizeField(draftCtx.Contact.Company)))
			sb.WriteString(fmt.Sprintf("(Bạn KHÔNG làm ở %s. Tuyệt đối không viết \"Tại %s, chúng tôi...\" hay ám chỉ mình là đồng nghiệp của họ.)\n",
				sanitizeField(draftCtx.Contact.Company), sanitizeField(draftCtx.Contact.Company)))
		}
		if draftCtx.Contact.Industry != "" {
			sb.WriteString(fmt.Sprintf("Ngành của họ: %q\n", sanitizeField(draftCtx.Contact.Industry)))
		}

		sb.WriteString(buildContactFactsBlock(draftCtx.Contact, "vi"))

		sb.WriteString("\n═══════════════════════════════════════\n")
		sb.WriteString("📊 TRẠNG THÁI HIỆN TẠI\n")
		sb.WriteString("═══════════════════════════════════════\n")
		sb.WriteString(fmt.Sprintf("Giai đoạn: %s\n", s.translateState(draftCtx.State.State)))
		sb.WriteString(fmt.Sprintf("Tình huống: %s\n", s.translateScenarioToVietnamese(draftCtx.State.Scenario)))
		sb.WriteString(fmt.Sprintf("Số ngày từ lần liên hệ cuối: %d\n", draftCtx.State.DaysSinceLastInteraction))
		sb.WriteString(fmt.Sprintf("Số lần follow-up: %d\n", draftCtx.State.FollowupCount))
		sb.WriteString(fmt.Sprintf("🎯 HÀNH ĐỘNG TIẾP THEO: %s\n", s.translateNextStepToVietnamese(draftCtx.State.NextStep, draftCtx.State.FollowupCount)))

		sb.WriteString("\n═══════════════════════════════════════\n")
		sb.WriteString("💬 LỊCH SỬ HỘI THOẠI (theo thời gian)\n")
		sb.WriteString("═══════════════════════════════════════\n")

		if len(conversationHistory) == 0 {
			sb.WriteString("(Chưa có lịch sử hội thoại - đây là tin nhắn đầu tiên)\n")
		} else {
			for i := len(conversationHistory) - 1; i >= 0; i-- {
				interaction := conversationHistory[i]
				timeStr := interaction.Timestamp.Format("02/01/2006 15:04")

				var roleLabel string
				switch interaction.Direction {
				case string(outreach.DirectionOutgoing):
					roleLabel = "🔵 MÌNH GỬI"
				case string(outreach.DirectionIncoming):
					roleLabel = "🟢 KHÁCH TRẢ LỜI"
				case string(outreach.DirectionInternal):
					continue
				default:
					roleLabel = "📝 GHI CHÚ"
				}

				sb.WriteString(fmt.Sprintf("\n[%s] %s qua %s\n", timeStr, roleLabel, interaction.Channel))
				sb.WriteString(fmt.Sprintf("Nội dung: %s\n", interaction.Content))
				if interaction.Sentiment != nil && *interaction.Sentiment != "" {
					sb.WriteString(fmt.Sprintf("Cảm xúc: %s\n", *interaction.Sentiment))
				}
			}
		}

		if len(draftCtx.Notes) > 0 {
			sb.WriteString("\n═══════════════════════════════════════\n")
			sb.WriteString("📋 GHI CHÚ CỦA TEAM\n")
			sb.WriteString("═══════════════════════════════════════\n")
			for _, note := range draftCtx.Notes {
				sb.WriteString(fmt.Sprintf("- %s\n", note.Content))
			}
		}

		sb.WriteString("\n═══════════════════════════════════════\n")
		sb.WriteString("✍️ HƯỚNG DẪN\n")
		sb.WriteString("═══════════════════════════════════════\n")

		// Add specific requirement based on next_step (Vietnamese)
		switch draftCtx.State.NextStep {
		case outreach.NextStepFollowUp1:
			sb.WriteString("⚠️ ĐÂY LÀ FOLLOW-UP LẦN 1: BẮT BUỘC mở đầu bằng việc nhắc lại tin nhắn trước (VD: 'Em quay lại về tin nhắn X ngày trước...')\n")
		case outreach.NextStepFollowUp2:
			sb.WriteString("⚠️ ĐÂY LÀ FOLLOW-UP LẦN 2 (CUỐI): BẮT BUỘC nói đây là lần cuối follow-up và offer alternatives\n")
		case outreach.NextStepSetMeeting:
			sb.WriteString("⚠️ KHÁCH ĐÃ REPLY: Respond trực tiếp đến họ nói gì và ĐỀ XUẤT THỜI GIAN CỤ THỂ để gặp\n")
		case outreach.NextStepFollowUpMeeting1:
			sb.WriteString("⚠️ FOLLOW-UP XÁC NHẬN LỊCH LẦN 1: BẮT BUỘC nhắc lại thời gian đã đề xuất\n")
		case outreach.NextStepFollowUpMeeting2:
			sb.WriteString("⚠️ FOLLOW-UP XÁC NHẬN LỊCH LẦN 2 (CUỐI): BẮT BUỘC nói đây là lần cuối nhắc về lịch meeting\n")
		case outreach.NextStepPrepareMeeting:
			sb.WriteString("⚠️ CHUẨN BỊ MEETING: Xác nhận lại thời gian, GỬI LINK MEETING, gửi agenda ngắn gọn\n")
		case outreach.NextStepFollowUp:
			sb.WriteString("⚠️ FOLLOW-UP SAU MEETING: BẮT BUỘC mở đầu bằng cảm ơn đã dành thời gian gặp, recap key points và action items\n")
		}

		sb.WriteString("Dựa trên lịch sử hội thoại ở trên, hãy viết tin nhắn tiếp theo sao cho tự nhiên và đáp ứng nhu cầu khách hàng.\n")
	}

	return sb.String()
}

// translateStateToEnglish translates conversation state to English
func (s *Service) translateStateToEnglish(state outreach.ConversationState) string {
	switch state {
	case outreach.StateCold:
		return "Cold (Never contacted)"
	case outreach.StateNoReply:
		return "No Reply (Waiting for response)"
	case outreach.StateReplied:
		return "Replied (Customer responded)"
	case outreach.StatePostMeeting:
		return "Post-Meeting"
	case outreach.StateDropped:
		return "Dropped"
	default:
		return string(state)
	}
}

// translateNextStepToEnglish translates next step action to English with follow-up count
func (s *Service) translateNextStepToEnglish(nextStep outreach.NextStepAction, followupCount int) string {
	switch nextStep {
	case outreach.NextStepSend:
		return "Send first message (introduce yourself and provide value)"
	case outreach.NextStepFollowUp1:
		return "Follow-up #1 (Day 4-5: gentle reminder, add new value)"
	case outreach.NextStepFollowUp2:
		return "Follow-up #2 (Day 9-12: last attempt, try different angle)"
	case outreach.NextStepSetMeeting:
		return "Schedule meeting (customer replied, propose specific times)"
	case outreach.NextStepFollowUpMeeting1:
		return "Follow-up meeting confirmation #1 (remind about proposed time)"
	case outreach.NextStepFollowUpMeeting2:
		return "Follow-up meeting confirmation #2 (last attempt to confirm)"
	case outreach.NextStepPrepareMeeting:
		return "Prepare meeting (send agenda, materials, meeting link)"
	case outreach.NextStepWait:
		return "Wait (waiting for response or meeting day)"
	case outreach.NextStepFollowUp:
		return "Follow-up deal (send recap, proposal, continue sales cadence)"
	case outreach.NextStepDrop:
		return "Drop (no further outreach recommended)"
	default:
		return string(nextStep)
	}
}

// translateNextStepToVietnamese translates next step action to Vietnamese with follow-up count
func (s *Service) translateNextStepToVietnamese(nextStep outreach.NextStepAction, followupCount int) string {
	switch nextStep {
	case outreach.NextStepSend:
		return "Gửi tin nhắn đầu tiên (giới thiệu bản thân và cung cấp giá trị)"
	case outreach.NextStepFollowUp1:
		return "Follow-up lần 1 (Day 4-5: nhắc nhở nhẹ nhàng, thêm giá trị mới)"
	case outreach.NextStepFollowUp2:
		return "Follow-up lần 2 (Day 9-12: lần cuối, thử góc tiếp cận khác)"
	case outreach.NextStepSetMeeting:
		return "Đề xuất lịch meeting (khách đã phản hồi, đề xuất thời gian cụ thể)"
	case outreach.NextStepFollowUpMeeting1:
		return "Follow-up xác nhận lịch lần 1 (nhắc về thời gian đề xuất)"
	case outreach.NextStepFollowUpMeeting2:
		return "Follow-up xác nhận lịch lần 2 (lần cuối để confirm)"
	case outreach.NextStepPrepareMeeting:
		return "Chuẩn bị meeting (gửi agenda, tài liệu, link meeting)"
	case outreach.NextStepWait:
		return "Chờ đợi (chờ phản hồi hoặc đến ngày meeting)"
	case outreach.NextStepFollowUp:
		return "Follow-up deal (gửi recap, proposal, tiếp tục sales cadence)"
	case outreach.NextStepDrop:
		return "Dừng outreach (không nên liên hệ thêm)"
	default:
		return string(nextStep)
	}
}

// translateScenarioToEnglish translates scenario to English with clear instructions
func (s *Service) translateScenarioToEnglish(scenario outreach.Scenario) string {
	switch scenario {
	case outreach.ScenarioRoleBased:
		return "Cold outreach - Focus on role-specific value proposition"
	case outreach.ScenarioIndustryBased:
		return "Cold outreach - Focus on industry-specific insights"
	case outreach.ScenarioNoReplyFollowup:
		return "No reply follow-up - Gentle reminder, offer alternative value"
	case outreach.ScenarioPostReply:
		return "Post-reply - Respond directly to what they said, propose clear next step"
	case outreach.ScenarioMeetingConfirmation:
		return "MEETING CONFIRMATION - Customer agreed to meet! Confirm the time, promise to send meeting link/agenda/materials before the meeting"
	case outreach.ScenarioPostMeeting:
		return "Post-meeting follow-up - Recap key points discussed, list action items, express enthusiasm"
	case outreach.ScenarioReEngage:
		return "Re-engage - Warm reconnection, mention something new/relevant"
	default:
		return string(scenario)
	}
}

// translateScenarioToVietnamese translates scenario to Vietnamese with clear instructions
func (s *Service) translateScenarioToVietnamese(scenario outreach.Scenario) string {
	switch scenario {
	case outreach.ScenarioRoleBased:
		return "Cold outreach - Tập trung vào giá trị theo role"
	case outreach.ScenarioIndustryBased:
		return "Cold outreach - Tập trung vào insight theo ngành"
	case outreach.ScenarioNoReplyFollowup:
		return "Follow-up sau không reply - Nhắc nhở nhẹ nhàng, đề xuất giá trị khác"
	case outreach.ScenarioPostReply:
		return "Sau khi khách trả lời - Respond trực tiếp, đề xuất next step rõ ràng"
	case outreach.ScenarioMeetingConfirmation:
		return "XÁC NHẬN MEETING - Khách đã đồng ý gặp! Confirm thời gian, hứa gửi link meeting/agenda/tài liệu trước buổi gặp"
	case outreach.ScenarioPostMeeting:
		return "Follow-up sau meeting - Recap nội dung đã thảo luận, liệt kê action items, thể hiện enthusiasm"
	case outreach.ScenarioReEngage:
		return "Re-engage - Kết nối lại warm, mention điều mới/relevant"
	default:
		return string(scenario)
	}
}

// buildFullContextPrompt builds a comprehensive prompt with full conversation history
func (s *Service) buildFullContextPrompt(draftCtx *DraftContext, conversationHistory []*outreach.InteractionLog) string {
	var sb strings.Builder

	// Contact info
	sb.WriteString("═══════════════════════════════════════\n")
	sb.WriteString("👤 THÔNG TIN KHÁCH HÀNG\n")
	sb.WriteString("═══════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("Tên: %q\n", sanitizeField(draftCtx.Contact.Name)))
	if draftCtx.Contact.JobTitle != "" {
		sb.WriteString(fmt.Sprintf("Chức vụ: %q\n", sanitizeField(draftCtx.Contact.JobTitle)))
	}
	if draftCtx.Contact.Company != "" {
		sb.WriteString(fmt.Sprintf("Công ty: %q\n", sanitizeField(draftCtx.Contact.Company)))
	}
	if draftCtx.Contact.Industry != "" {
		sb.WriteString(fmt.Sprintf("Ngành: %q\n", sanitizeField(draftCtx.Contact.Industry)))
	}

	// Current state
	sb.WriteString("\n═══════════════════════════════════════\n")
	sb.WriteString("📊 TRẠNG THÁI HIỆN TẠI\n")
	sb.WriteString("═══════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("Giai đoạn: %s\n", s.translateState(draftCtx.State.State)))
	sb.WriteString(fmt.Sprintf("Số ngày từ lần liên hệ cuối: %d\n", draftCtx.State.DaysSinceLastInteraction))
	sb.WriteString(fmt.Sprintf("Số lần follow-up: %d\n", draftCtx.State.FollowupCount))

	// Full conversation history (chronological order)
	sb.WriteString("\n═══════════════════════════════════════\n")
	sb.WriteString("💬 LỊCH SỬ HỘI THOẠI (theo thời gian)\n")
	sb.WriteString("═══════════════════════════════════════\n")

	if len(conversationHistory) == 0 {
		sb.WriteString("(Chưa có lịch sử hội thoại - đây là tin nhắn đầu tiên)\n")
	} else {
		// Reverse to show chronological order (oldest first)
		for i := len(conversationHistory) - 1; i >= 0; i-- {
			interaction := conversationHistory[i]
			timeStr := interaction.Timestamp.Format("02/01/2006 15:04")

			var roleLabel string
			switch interaction.Direction {
			case string(outreach.DirectionOutgoing):
				roleLabel = "🔵 MÌNH GỬI"
			case string(outreach.DirectionIncoming):
				roleLabel = "🟢 KHÁCH TRẢ LỜI"
			case string(outreach.DirectionInternal):
				continue // Skip internal notes in conversation history
			default:
				roleLabel = "📝 " + interaction.Direction
			}

			sb.WriteString(fmt.Sprintf("\n[%s - %s via %s]\n", timeStr, roleLabel, interaction.Channel))
			sb.WriteString(fmt.Sprintf("%s\n", interaction.Content))

			if interaction.Sentiment != nil && *interaction.Sentiment != "" {
				sb.WriteString(fmt.Sprintf("(Sentiment: %s)\n", *interaction.Sentiment))
			}
		}
	}

	// Team notes (internal context)
	if len(draftCtx.Notes) > 0 {
		sb.WriteString("\n═══════════════════════════════════════\n")
		sb.WriteString("📋 GHI CHÚ NỘI BỘ CỦA TEAM\n")
		sb.WriteString("═══════════════════════════════════════\n")
		for _, note := range draftCtx.Notes {
			timeStr := note.Timestamp.Format("02/01/2006")
			sb.WriteString(fmt.Sprintf("[%s] %s\n", timeStr, note.Content))
		}
	}

	// Instructions
	sb.WriteString("\n═══════════════════════════════════════\n")
	sb.WriteString("🎯 YÊU CẦU\n")
	sb.WriteString("═══════════════════════════════════════\n")
	sb.WriteString("Dựa vào TOÀN BỘ thông tin trên, viết tin nhắn tiếp theo sao cho:\n")
	sb.WriteString("- Tiếp nối TỰ NHIÊN từ cuộc hội thoại trước (nếu có)\n")
	sb.WriteString("- Khách hàng cảm thấy được LẮNG NGHE và QUAN TÂM\n")
	sb.WriteString("- Mang lại GIÁ TRỊ cụ thể cho họ\n")
	sb.WriteString("- Khiến họ MUỐN trả lời\n")

	return sb.String()
}

// translateState converts state code to Vietnamese
func (s *Service) translateState(state outreach.ConversationState) string {
	switch state {
	case outreach.StateCold:
		return "MỚI - Chưa liên hệ"
	case outreach.StateNoReply:
		return "ĐÃ GỬI - Chờ phản hồi"
	case outreach.StateReplied:
		return "ĐÃ TRẢ LỜI - Đang trao đổi"
	case outreach.StatePostMeeting:
		return "SAU MEETING - Follow-up"
	case outreach.StateDropped:
		return "ĐÃ DỪNG"
	default:
		return string(state)
	}
}

// generateDraftWithAI generates a personalized message draft using AI based on notes and contact context
// (Legacy method - kept for backward compatibility)
func (s *Service) generateDraftWithAI(ctx context.Context, draftCtx *DraftContext) (string, error) {
	if s.openAIClient == nil {
		return "", fmt.Errorf("OpenAI client not configured")
	}

	// Build prompt with context
	prompt := s.buildAIDraftPrompt(draftCtx)

	// System prompt for outreach message generation
	systemPrompt := `Bạn là một chuyên gia sales B2B với phong cách tư vấn - tập trung vào việc GIÚP ĐỠ khách hàng, không phải bán hàng.

Nhiệm vụ: Viết tin nhắn LinkedIn mang tính GIÁ TRỊ và CHÂN THÀNH, dựa trên context cụ thể của từng contact.

NGUYÊN TẮC VÀNG:
1. LUÔN mở đầu bằng một observation cụ thể về contact (công ty, role, achievement)
2. FOCUS vào giá trị bạn có thể MANG LẠI, không phải sản phẩm bạn muốn bán
3. Đặt em vào vị trí NGƯỜI GIÚP ĐỠ, không phải người bán hàng
4. CTA phải low-commitment (hỏi ý kiến, share insight, 10 phút call)
5. Tham khảo NOTES của team sales - đây là thông tin quan trọng nhất

QUY TẮC VIẾT:
- Ngắn gọn: 4-6 câu tối đa
- Tự nhiên như đang nhắn tin với đồng nghiệp
- Tiếng Việt thân thiện, có thể dùng emoji vừa phải (1-2 emoji)
- TRÁNH: "Em muốn giới thiệu...", "Sản phẩm của em...", "Công ty em..."
- NÊN: "Em có thể giúp...", "Em vừa thấy một cách...", "Team tương tự đã..."

⚠️ QUY TẮC XƯNG HÔ (BẮT BUỘC - QUAN TRỌNG NHẤT):
- LUÔN LUÔN xưng "em" khi nói về bản thân. VD: "em muốn chia sẻ", "em thấy rằng"
- LUÔN LUÔN gọi đối phương là "anh/chị" (CẢ HAI, KHÔNG ĐƯỢC CHỌN MỘT). VD: "Chào anh/chị [Tên]", "anh/chị có thể"
- KHÔNG BAO GIỜ chỉ dùng "anh" hoặc chỉ dùng "chị" riêng lẻ. PHẢI LUÔN viết "anh/chị" đầy đủ
- KHÔNG BAO GIỜ dùng: "tôi", "mình", "bạn", "quý khách"

THEO CONTEXT VÀ NEXT_STEP:

📤 COLD OUTREACH:
- SEND: Focus vào insight/observation về họ + offer value cụ thể

📭 KHÔNG CÓ PHẢN HỒI (NO_REPLY):
- FOLLOW_UP_1: BẮT BUỘC mở đầu bằng "Em quay lại về tin nhắn [X ngày] trước" hoặc "Không biết anh/chị đã xem tin nhắn em gửi chưa". Nhẹ nhàng, tôn trọng, thêm giá trị mới hoặc góc nhìn khác
- FOLLOW_UP_2: BẮT BUỘC nói đây là "lần cuối em follow-up" hoặc "em không muốn làm phiền". Offer alternatives: (1) gửi tài liệu, (2) kết nối lại sau, (3) giới thiệu người khác

💬 ĐÃ PHẢN HỒI (REPLIED):
- SET_MEETING: Cảm ơn họ đã reply, respond trực tiếp đến content họ nói, ĐỀ XUẤT THỜI GIAN CỤ THỂ để gặp (VD: "Anh/chị có thể gặp 10h sáng thứ 3 hoặc 2h chiều thứ 5 không?")

📅 CHỜ XÁC NHẬN LỊCH MEETING:
- FOLLOW_UP_MEETING_1: BẮT BUỘC nhắc lại thời gian đã đề xuất "Em quay lại về lịch meeting [thời gian] em đề xuất". Hỏi xem thời gian đó có phù hợp không, offer thời gian khác nếu cần
- FOLLOW_UP_MEETING_2: BẮT BUỘC nói "đây là lần cuối em nhắc về lịch meeting". Offer options: (1) confirm thời gian cũ, (2) đề xuất thời gian khác, (3) kết nối lại sau khi anh/chị rảnh hơn

✅ ĐÃ CONFIRM LỊCH:
- PREPARE_MEETING: Xác nhận lại thời gian meeting, GỬI LINK MEETING (Zoom/Google Meet), gửi agenda ngắn gọn, hỏi có cần chuẩn bị gì thêm không

🤝 SAU MEETING:
- FOLLOW_UP (post-meeting): BẮT BUỘC mở đầu bằng "Cảm ơn anh/chị đã dành thời gian gặp [hôm qua/tuần trước]". Recap key points đã thảo luận, list action items cụ thể, đề xuất next steps rõ ràng

🔄 RE-ENGAGE:
- RE_ENGAGE: Warm, friendly, mention something new/relevant kể từ lần cuối liên hệ

KHÔNG ký tên ở cuối tin nhắn (không có "Anh Khoa", "Trân trọng", v.v.)

Output: CHỈ TRẢ VỀ NỘI DUNG TIN NHẮN, không giải thích. Bắt đầu ngay bằng câu mở.`

	// Generate using OpenAI
	response, err := s.openAIClient.ChatCompletion(ctx, ai.ChatCompletionRequest{
		SystemPrompt: systemPrompt,
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.8,
		MaxTokens:   500,
	})

	if err != nil {
		return "", fmt.Errorf("AI draft generation failed: %w", err)
	}

	// Post-process: replace any name placeholders with actual user name
	result := strings.TrimSpace(response)
	result = replaceNamePlaceholders(result, draftCtx.UserName)
	result = normalizeVietnamesePronouns(result)

	return result, nil
}

// buildAIDraftPrompt builds the prompt for AI draft generation
func (s *Service) buildAIDraftPrompt(draftCtx *DraftContext) string {
	var sb strings.Builder

	// Contact info
	sb.WriteString("=== THÔNG TIN CONTACT ===\n")
	sb.WriteString(fmt.Sprintf("Tên: %q\n", sanitizeField(draftCtx.Contact.Name)))
	if draftCtx.Contact.JobTitle != "" {
		sb.WriteString(fmt.Sprintf("Chức vụ: %q\n", sanitizeField(draftCtx.Contact.JobTitle)))
	}
	if draftCtx.Contact.Company != "" {
		sb.WriteString(fmt.Sprintf("Công ty: %q\n", sanitizeField(draftCtx.Contact.Company)))
	}
	if draftCtx.Contact.Industry != "" {
		sb.WriteString(fmt.Sprintf("Ngành: %q\n", sanitizeField(draftCtx.Contact.Industry)))
	}

	// Conversation state
	sb.WriteString("\n=== TRẠNG THÁI ===\n")
	sb.WriteString(fmt.Sprintf("Conversation State: %s\n", draftCtx.State.State))
	sb.WriteString(fmt.Sprintf("Outreach Intent: %s\n", draftCtx.State.OutreachIntent))
	sb.WriteString(fmt.Sprintf("Scenario: %s\n", draftCtx.State.Scenario))
	sb.WriteString(fmt.Sprintf("Next Step: %s\n", draftCtx.State.NextStep))
	if draftCtx.State.DaysSinceLastInteraction > 0 {
		sb.WriteString(fmt.Sprintf("Số ngày từ lần liên hệ cuối: %d ngày\n", draftCtx.State.DaysSinceLastInteraction))
	}
	if draftCtx.State.FollowupCount > 0 {
		sb.WriteString(fmt.Sprintf("Số lần đã follow-up: %d (đây là follow-up lần %d)\n", draftCtx.State.FollowupCount, draftCtx.State.FollowupCount+1))
	} else if draftCtx.State.State == outreach.StateNoReply {
		sb.WriteString("Số lần đã follow-up: 0 (đây là follow-up lần 1 - LẦN ĐẦU follow-up sau tin initial)\n")
	}

	// Last outgoing message
	if draftCtx.LastOutgoing != nil {
		sb.WriteString("\n=== TIN NHẮN GỬI GẦN NHẤT ===\n")
		sb.WriteString(fmt.Sprintf("Nội dung: %s\n", draftCtx.LastOutgoing.Content))
	}

	// Last incoming message
	if draftCtx.LastIncoming != nil {
		sb.WriteString("\n=== PHẢN HỒI GẦN NHẤT ===\n")
		sb.WriteString(fmt.Sprintf("Nội dung: %s\n", draftCtx.LastIncoming.Content))
		if draftCtx.LastIncoming.Sentiment != nil {
			sb.WriteString(fmt.Sprintf("Sentiment: %s\n", *draftCtx.LastIncoming.Sentiment))
		}
	}

	// Team notes - THIS IS THE KEY CONTEXT
	if len(draftCtx.Notes) > 0 {
		sb.WriteString("\n=== GHI CHÚ CỦA ĐỘI SALES ===\n")
		sb.WriteString("(Đây là các ghi chú quan trọng cần được phản ánh trong message)\n")
		for i, note := range draftCtx.Notes {
			sb.WriteString(fmt.Sprintf("\n[Ghi chú %d - %s]\n", i+1, note.Timestamp.Format("02/01/2006 15:04")))
			sb.WriteString(note.Content)
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n=== YÊU CẦU ===\n")

	// Add specific requirement based on next_step
	switch draftCtx.State.NextStep {
	case outreach.NextStepFollowUp1:
		sb.WriteString("⚠️ ĐÂY LÀ FOLLOW-UP LẦN 1: BẮT BUỘC mở đầu bằng việc nhắc lại tin nhắn trước (VD: 'Em quay lại về tin nhắn X ngày trước...')\n")
	case outreach.NextStepFollowUp2:
		sb.WriteString("⚠️ ĐÂY LÀ FOLLOW-UP LẦN 2 (CUỐI): BẮT BUỘC nói đây là lần cuối follow-up và offer alternatives\n")
	case outreach.NextStepSetMeeting:
		sb.WriteString("⚠️ KHÁCH ĐÃ REPLY: Respond trực tiếp đến họ nói gì và ĐỀ XUẤT THỜI GIAN CỤ THỂ để gặp\n")
	case outreach.NextStepFollowUpMeeting1:
		sb.WriteString("⚠️ FOLLOW-UP XÁC NHẬN LỊCH LẦN 1: BẮT BUỘC nhắc lại thời gian đã đề xuất\n")
	case outreach.NextStepFollowUpMeeting2:
		sb.WriteString("⚠️ FOLLOW-UP XÁC NHẬN LỊCH LẦN 2 (CUỐI): BẮT BUỘC nói đây là lần cuối nhắc về lịch meeting\n")
	case outreach.NextStepPrepareMeeting:
		sb.WriteString("⚠️ CHUẨN BỊ MEETING: Xác nhận lại thời gian, GỬI LINK MEETING, gửi agenda ngắn gọn\n")
	case outreach.NextStepFollowUp:
		sb.WriteString("⚠️ FOLLOW-UP SAU MEETING: BẮT BUỘC mở đầu bằng cảm ơn đã dành thời gian gặp, recap key points và action items\n")
	}

	sb.WriteString("Viết tin nhắn LinkedIn phù hợp với context trên. ")
	sb.WriteString("QUAN TRỌNG: Phải tham khảo và phản ánh nội dung ghi chú của đội sales trong message.")

	return sb.String()
}

// ============================================
// Update Logic
// ============================================

// UpdateEvent represents an outreach event
type UpdateEvent string

const (
	EventSent             UpdateEvent = "sent"              // BD sends message (initial or follow-up)
	EventReplied          UpdateEvent = "replied"           // Customer replied
	EventNoReply          UpdateEvent = "no_reply"          // No reply after waiting period
	EventMeetingBooked    UpdateEvent = "meeting_booked"    // Meeting booked (proposed schedule)
	EventMeetingConfirmed UpdateEvent = "meeting_confirmed" // Customer confirmed meeting schedule
	EventNoConfirmation   UpdateEvent = "no_confirmation"   // No meeting schedule confirmation
	EventMeetingDone      UpdateEvent = "meeting_done"      // Meeting completed
	EventDrop             UpdateEvent = "drop"              // Drop contact
)

// UpdateOutreachInput holds input for updating outreach state
type UpdateOutreachInput struct {
	ContactID uuid.UUID
	UserID    uuid.UUID
	Event     UpdateEvent
	Content   string // Message content (for sent/replied)
	Channel   outreach.InteractionChannel
	Sentiment *outreach.Sentiment // For replied events
}

// UpdateOutreach updates the outreach state based on an event
func (s *Service) UpdateOutreach(ctx context.Context, input *UpdateOutreachInput) error {
	// Get or create outreach state
	state, err := s.stateRepo.FindByContactID(ctx, input.ContactID, input.UserID)
	if err != nil {
		return err
	}

	cfg := s.configFor(ctx, input.UserID)
	if state == nil {
		state = &outreach.OutreachState{
			UserID:       input.UserID,
			ContactID:    input.ContactID,
			MaxFollowups: cfg.MaxFollowups,
		}
	}

	now := time.Now()

	switch input.Event {
	case EventSent:
		// Create outgoing interaction log (always create to track timing)
		logContent := input.Content
		if logContent == "" {
			logContent = "[Message sent]" // Placeholder for tracking timing
		}
		log := &outreach.InteractionLog{
			UserID:    input.UserID,
			ContactID: input.ContactID,
			Channel:   string(input.Channel),
			Direction: string(outreach.DirectionOutgoing),
			Content:   logContent,
			Timestamp: now,
		}
		if err := s.interactionRepo.Create(ctx, log); err != nil {
			return err
		}

		// DEBUG: Log interaction log creation
		logger.Logger.Info().
			Str("contact_id", input.ContactID.String()).
			Str("user_id", input.UserID.String()).
			Str("direction", string(outreach.DirectionOutgoing)).
			Time("timestamp", now).
			Msg("outreach: interaction_log created for EventSent")

		state.LastOutcome = string(outreach.OutcomeSent)
		state.LastInteractionAt = &now

		// Preserve conversation state based on current state (don't reset to COLD)
		switch outreach.ConversationState(state.ConversationState) {
		case outreach.StateNoReply:
			// Sending a follow-up: stay NO_REPLY, increment followup count, wait for response
			state.FollowupCount++
			state.NextStep = string(outreach.NextStepWait)
		case outreach.StateReplied:
			// Responding to a reply: stay REPLIED, wait for next response
			state.NextStep = string(outreach.NextStepWait)
		case outreach.StatePostMeeting:
			// Follow-up on deal: stay POST_MEETING, wait
			state.NextStep = string(outreach.NextStepWait)
		default:
			// Initial cold outreach: transition to NO_REPLY, wait for response
			state.ConversationState = string(outreach.StateNoReply)
			state.NextStep = string(outreach.NextStepWait)
		}

		// Update contact fields including next_step and outreach_stage
		updatedContact, updateErr := s.contactRepo.UpdateFields(ctx, input.ContactID, map[string]interface{}{
			"last_interaction_at": now,
			"next_step":           string(outreach.NextStepWait),
			"outreach_stage":      state.ConversationState,
			"last_outcome":        string(outreach.OutcomeSent),
		}, input.UserID, uuid.Nil)
		if updateErr != nil {
			logger.Logger.Error().Err(updateErr).
				Str("contact_id", input.ContactID.String()).
				Str("user_id", input.UserID.String()).
				Msg("outreach: failed to update contact fields on EventSent")
		} else {
			logger.Logger.Info().
				Str("contact_id", input.ContactID.String()).
				Str("next_step", updatedContact.NextStep).
				Msg("outreach: contact updated on EventSent")
		}

	case EventReplied:
		// Create incoming interaction log
		sentimentStr := ""
		if input.Sentiment != nil {
			sentimentStr = string(*input.Sentiment)
		}

		log := &outreach.InteractionLog{
			UserID:    input.UserID,
			ContactID: input.ContactID,
			Channel:   string(input.Channel),
			Direction: string(outreach.DirectionIncoming),
			Content:   input.Content,
			Timestamp: now,
			Sentiment: &sentimentStr,
		}
		if err := s.interactionRepo.Create(ctx, log); err != nil {
			return err
		}

		state.LastOutcome = string(outreach.OutcomeReplied)
		state.LastInteractionAt = &now
		state.ConversationState = string(outreach.StateReplied)
		state.FollowupCount = 0                              // Reset followup count when customer replies
		state.NextStep = string(outreach.NextStepSetMeeting) // Propose meeting schedule

		// Update contact fields including next_step and outreach_stage
		s.contactRepo.UpdateFields(ctx, input.ContactID, map[string]interface{}{
			"last_interaction_at": now,
			"next_step":           string(outreach.NextStepSetMeeting),
			"outreach_stage":      string(outreach.StateReplied),
			"last_outcome":        string(outreach.OutcomeReplied),
			"followup_count":      0,
		}, input.UserID, uuid.Nil)

	case EventNoReply:
		// No reply after sending message - need follow-up
		state.LastOutcome = string(outreach.OutcomeNoReply)
		state.FollowupCount++
		state.ConversationState = string(outreach.StateNoReply)

		// Determine next step based on followup count
		// followup_count = 1 (initial sent, no reply) → FOLLOW_UP_1
		// followup_count = 2 (FU1 sent, no reply) → FOLLOW_UP_2
		// followup_count = 3+ (FU2 sent, no reply) → DROP
		// The organisation's cap, as in DetermineConversationState — not the
		// value frozen into the row when it was first created.
		if state.FollowupCount >= cfg.MaxFollowups+1 {
			state.NextStep = string(outreach.NextStepDrop)
			state.ConversationState = string(outreach.StateDropped)
		} else if state.FollowupCount == 1 {
			state.NextStep = string(outreach.NextStepFollowUp1) // Follow-up #1
		} else {
			state.NextStep = string(outreach.NextStepFollowUp2) // Follow-up #2
		}

		// Update contact fields including next_step and outreach_stage
		updateFields := map[string]interface{}{
			"next_step":      state.NextStep,
			"outreach_stage": state.ConversationState,
			"last_outcome":   string(outreach.OutcomeNoReply),
			"followup_count": state.FollowupCount,
		}
		s.contactRepo.UpdateFields(ctx, input.ContactID, updateFields, input.UserID, uuid.Nil)

	case EventMeetingBooked:
		// Meeting proposed, waiting for customer confirmation
		state.LastOutcome = string(outreach.OutcomeMeetingBooked)
		state.LastInteractionAt = &now
		state.ConversationState = string(outreach.StateReplied) // Still REPLIED since not confirmed yet
		state.NextStep = string(outreach.NextStepWait)          // Wait for customer confirmation
		state.Scenario = string(outreach.ScenarioMeetingConfirmation)
		state.FollowupCount = 0 // Reset meeting followup count

		// Update contact fields including next_step and outreach_stage
		s.contactRepo.UpdateFields(ctx, input.ContactID, map[string]interface{}{
			"last_interaction_at": now,
			"next_step":           string(outreach.NextStepWait),
			"outreach_stage":      string(outreach.StateReplied),
			"scenario":            string(outreach.ScenarioMeetingConfirmation),
			"last_outcome":        string(outreach.OutcomeMeetingBooked),
			"followup_count":      0,
		}, input.UserID, uuid.Nil)

	case EventNoConfirmation:
		// No meeting schedule confirmation - need meeting follow-up
		state.FollowupCount++

		if state.FollowupCount >= 3 {
			// Already followed up meeting 2 times but still no confirmation → DROP
			state.NextStep = string(outreach.NextStepDrop)
			state.ConversationState = string(outreach.StateDropped)
		} else if state.FollowupCount == 1 {
			state.NextStep = string(outreach.NextStepFollowUpMeeting1) // Meeting confirmation follow-up #1
		} else {
			state.NextStep = string(outreach.NextStepFollowUpMeeting2) // Meeting confirmation follow-up #2
		}

		// Update contact fields including next_step and outreach_stage
		noConfirmFields := map[string]interface{}{
			"next_step":      state.NextStep,
			"outreach_stage": state.ConversationState,
			"followup_count": state.FollowupCount,
		}
		s.contactRepo.UpdateFields(ctx, input.ContactID, noConfirmFields, input.UserID, uuid.Nil)

	case EventMeetingConfirmed:
		// Customer confirmed meeting schedule
		state.LastInteractionAt = &now
		state.ConversationState = string(outreach.StateReplied)
		state.NextStep = string(outreach.NextStepPrepareMeeting) // Prepare meeting materials
		state.FollowupCount = 0

		// Update contact fields including next_step and business_stage
		s.contactRepo.UpdateFields(ctx, input.ContactID, map[string]interface{}{
			"business_stage":      string(contact.BusinessStageSales),
			"last_interaction_at": now,
			"next_step":           string(outreach.NextStepPrepareMeeting),
			"outreach_stage":      string(outreach.StateReplied),
			"followup_count":      0,
		}, input.UserID, uuid.Nil)

	case EventMeetingDone:
		// Meeting completed
		state.LastOutcome = string(outreach.OutcomeMeetingDone)
		state.LastInteractionAt = &now
		state.ConversationState = string(outreach.StatePostMeeting)
		state.Scenario = string(outreach.ScenarioPostMeeting)
		state.NextStep = string(outreach.NextStepFollowUp) // Follow-up deal / proposal
		state.FollowupCount = 0

		// Update contact fields including next_step and outreach_stage
		s.contactRepo.UpdateFields(ctx, input.ContactID, map[string]interface{}{
			"last_interaction_at": now,
			"next_step":           string(outreach.NextStepFollowUp),
			"outreach_stage":      string(outreach.StatePostMeeting),
			"scenario":            string(outreach.ScenarioPostMeeting),
			"last_outcome":        string(outreach.OutcomeMeetingDone),
			"followup_count":      0,
		}, input.UserID, uuid.Nil)

	case EventDrop:
		state.LastOutcome = string(outreach.OutcomeDropped)
		state.ConversationState = string(outreach.StateDropped)
		state.NextStep = string(outreach.NextStepDrop)

		// Update contact fields including next_step and outreach_stage
		s.contactRepo.UpdateFields(ctx, input.ContactID, map[string]interface{}{
			"next_step":      string(outreach.NextStepDrop),
			"outreach_stage": string(outreach.StateDropped),
			"last_outcome":   string(outreach.OutcomeDropped),
		}, input.UserID, uuid.Nil)

	default:
		return fmt.Errorf("%w: %q", ErrUnknownEvent, input.Event)
	}

	// Recalculate days since last interaction
	state.CalculateDaysSinceLastInteraction()

	// Save state
	return s.stateRepo.Upsert(ctx, state)
}

// ============================================
// Meeting Operations
// ============================================

// CreateMeetingInput holds input for creating a meeting
type CreateMeetingInput struct {
	ContactID       uuid.UUID
	UserID          uuid.UUID
	Title           string
	Time            time.Time
	DurationMinutes int
	Channel         outreach.MeetingChannel
	Location        string
	MeetingURL      string
	Note            string
}

// CreateMeeting creates a new meeting
func (s *Service) CreateMeeting(ctx context.Context, input *CreateMeetingInput) (*outreach.Meeting, error) {
	meeting := &outreach.Meeting{
		UserID:          input.UserID,
		ContactID:       input.ContactID,
		Time:            input.Time,
		DurationMinutes: input.DurationMinutes,
		Channel:         string(input.Channel),
		Status:          string(outreach.MeetingScheduled),
	}

	if input.Title != "" {
		meeting.Title = &input.Title
	}
	if input.Location != "" {
		meeting.Location = &input.Location
	}
	if input.MeetingURL != "" {
		meeting.MeetingURL = &input.MeetingURL
	}
	if input.Note != "" {
		meeting.Note = &input.Note
	}

	if err := s.meetingRepo.Create(ctx, meeting); err != nil {
		return nil, err
	}

	// Update outreach state
	s.UpdateOutreach(ctx, &UpdateOutreachInput{
		ContactID: input.ContactID,
		UserID:    input.UserID,
		Event:     EventMeetingBooked,
	})

	return meeting, nil
}

// UpdateMeetingInput holds input for updating a meeting
type UpdateMeetingInput struct {
	MeetingID      uuid.UUID
	UserID         uuid.UUID
	Status         outreach.MeetingStatus
	Note           string
	Outcome        string
	NextSteps      string
	MeetingContent string
}

// UpdateMeeting updates a meeting
func (s *Service) UpdateMeeting(ctx context.Context, input *UpdateMeetingInput) (*outreach.Meeting, error) {
	meeting, err := s.meetingRepo.FindByID(ctx, input.MeetingID)
	if err != nil {
		return nil, err
	}
	if meeting == nil || meeting.UserID != input.UserID {
		return nil, fmt.Errorf("meeting not found")
	}

	meeting.Status = string(input.Status)
	if input.Note != "" {
		meeting.Note = &input.Note
	}
	if input.Outcome != "" {
		meeting.Outcome = &input.Outcome
	}
	if input.NextSteps != "" {
		meeting.NextSteps = &input.NextSteps
	}
	if input.MeetingContent != "" {
		meeting.MeetingContent = &input.MeetingContent
	}

	if err := s.meetingRepo.Update(ctx, meeting); err != nil {
		return nil, err
	}

	// If meeting completed, update outreach state
	if input.Status == outreach.MeetingCompleted {
		s.UpdateOutreach(ctx, &UpdateOutreachInput{
			ContactID: meeting.ContactID,
			UserID:    input.UserID,
			Event:     EventMeetingDone,
		})
	}

	return meeting, nil
}

// GetMeetings gets meetings for a contact
func (s *Service) GetMeetings(ctx context.Context, contactID, userID uuid.UUID) ([]*outreach.Meeting, error) {
	return s.meetingRepo.FindByContactIDAndUserID(ctx, contactID, userID)
}

// ============================================
// Handler-Compatible Methods
// ============================================

// DraftResult represents the result of generating a draft
type DraftResult struct {
	ContactID    uuid.UUID                  `json:"contact_id"`
	Draft        string                     `json:"draft"`
	Scenario     outreach.Scenario          `json:"scenario"`
	ContextLevel outreach.ContextLevel      `json:"context_level"`
	State        *outreach.OutreachState    `json:"state"`
	Notes        []*outreach.InteractionLog `json:"notes,omitempty"` // Team notes for context

	// Contact info for display
	ContactName        string `json:"contact_name,omitempty"`
	ContactCompany     string `json:"contact_company,omitempty"`
	ContactJobTitle    string `json:"contact_job_title,omitempty"`
	ContactInformation string `json:"contact_information,omitempty"` // LinkedIn URL or email
	ContactChannel     string `json:"contact_channel,omitempty"`     // LinkedIn, Email, etc.
}

// GenerateDraftForHandler generates a draft for a contact (handler-compatible signature)
func (s *Service) GenerateDraftForHandler(ctx context.Context, userID, contactID uuid.UUID) (*DraftResult, error) {
	contactEntity, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to find contact: %w", err)
	}
	if contactEntity == nil {
		return nil, fmt.Errorf("contact not found")
	}

	stateResult, err := s.DetermineConversationState(ctx, contactEntity, userID)
	if err != nil {
		return nil, err
	}

	draft, err := s.GenerateDraft(ctx, contactEntity, userID)
	if err != nil {
		return nil, err
	}

	// Get or create outreach state
	state, _ := s.stateRepo.FindByContactID(ctx, contactID, userID)
	if state == nil {
		state = &outreach.OutreachState{
			UserID:            userID,
			ContactID:         contactID,
			ConversationState: string(stateResult.State),
			OutreachIntent:    string(stateResult.OutreachIntent),
			Scenario:          string(stateResult.Scenario),
			NextStep:          string(stateResult.NextStep),
			MessageDraft:      &draft,
		}
		s.stateRepo.Upsert(ctx, state)
	} else {
		state.MessageDraft = &draft
		s.stateRepo.Update(ctx, state)
	}

	// Determine context level
	contextLevel := outreach.ContextLow
	if contactEntity.Industry != "" && contactEntity.JobTitle != "" {
		contextLevel = outreach.ContextMedium
	}
	if contactEntity.Industry != "" && contactEntity.JobTitle != "" && contactEntity.Company != "" {
		contextLevel = outreach.ContextHigh
	}

	// Fetch notes for context
	notes, _ := s.interactionRepo.FindNotes(ctx, contactID, userID, 10)

	return &DraftResult{
		ContactID:    contactID,
		Draft:        draft,
		Scenario:     stateResult.Scenario,
		ContextLevel: contextLevel,
		State:        state,
		Notes:        notes,
	}, nil
}

// GenerateDraftForHandlerWithLanguage generates a draft for a contact with language support
// userName is the display name of the BD user for personalizing the message signature
func (s *Service) GenerateDraftForHandlerWithLanguage(ctx context.Context, userID, contactID uuid.UUID, language string, userName string) (*DraftResult, error) {
	contactEntity, err := s.contactRepo.FindByID(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to find contact: %w", err)
	}
	if contactEntity == nil {
		return nil, fmt.Errorf("contact not found")
	}

	stateResult, err := s.DetermineConversationState(ctx, contactEntity, userID)
	if err != nil {
		return nil, err
	}

	draft, err := s.GenerateDraftWithLanguageAndUserName(ctx, contactEntity, userID, language, userName)
	if err != nil {
		return nil, err
	}

	// Get or create outreach state
	state, _ := s.stateRepo.FindByContactID(ctx, contactID, userID)
	if state == nil {
		state = &outreach.OutreachState{
			UserID:            userID,
			ContactID:         contactID,
			ConversationState: string(stateResult.State),
			OutreachIntent:    string(stateResult.OutreachIntent),
			Scenario:          string(stateResult.Scenario),
			NextStep:          string(stateResult.NextStep),
			MessageDraft:      &draft,
		}
		s.stateRepo.Upsert(ctx, state)
	} else {
		state.MessageDraft = &draft
		s.stateRepo.Update(ctx, state)
	}

	// Determine context level
	contextLevel := outreach.ContextLow
	if contactEntity.Industry != "" && contactEntity.JobTitle != "" {
		contextLevel = outreach.ContextMedium
	}
	if contactEntity.Industry != "" && contactEntity.JobTitle != "" && contactEntity.Company != "" {
		contextLevel = outreach.ContextHigh
	}

	// Fetch notes for context
	notes, _ := s.interactionRepo.FindNotes(ctx, contactID, userID, 10)

	return &DraftResult{
		ContactID:    contactID,
		Draft:        draft,
		Scenario:     stateResult.Scenario,
		ContextLevel: contextLevel,
		State:        state,
		Notes:        notes,

		ContactName:        contactEntity.Name,
		ContactCompany:     contactEntity.Company,
		ContactJobTitle:    contactEntity.JobTitle,
		ContactInformation: contactEntity.ContactInformation,
		ContactChannel:     contactEntity.ContactChannel,
	}, nil
}

// UpdateOutreachResult represents the result of updating outreach
type UpdateOutreachResult struct {
	ContactID     uuid.UUID                  `json:"contact_id"`
	PreviousState outreach.ConversationState `json:"previous_state"`
	NewState      outreach.ConversationState `json:"new_state"`
	NextStep      outreach.NextStepAction    `json:"next_step"`
	State         *outreach.OutreachState    `json:"state"`
}

// UpdateOutreachForHandler updates outreach state (handler-compatible signature)
func (s *Service) UpdateOutreachForHandler(ctx context.Context, userID, contactID uuid.UUID, event, content, channel, sentiment string) (*UpdateOutreachResult, error) {
	// Get current state for comparison
	currentState, _ := s.stateRepo.FindByContactID(ctx, contactID, userID)
	previousState := outreach.StateCold
	if currentState != nil {
		previousState = outreach.ConversationState(currentState.ConversationState)
	}

	// Parse channel
	channelType := outreach.ChannelLinkedIn
	if channel != "" {
		channelType = outreach.InteractionChannel(channel)
	}

	// Parse sentiment
	var sentimentPtr *outreach.Sentiment
	if sentiment != "" {
		s := outreach.Sentiment(sentiment)
		sentimentPtr = &s
	}

	// Update outreach
	input := &UpdateOutreachInput{
		ContactID: contactID,
		UserID:    userID,
		Event:     UpdateEvent(event),
		Content:   content,
		Channel:   channelType,
		Sentiment: sentimentPtr,
	}

	if err := s.UpdateOutreach(ctx, input); err != nil {
		return nil, err
	}

	// Get updated state
	newState, err := s.stateRepo.FindByContactID(ctx, contactID, userID)
	if err != nil {
		return nil, err
	}

	return &UpdateOutreachResult{
		ContactID:     contactID,
		PreviousState: previousState,
		NewState:      outreach.ConversationState(newState.ConversationState),
		NextStep:      outreach.NextStepAction(newState.NextStep),
		State:         newState,
	}, nil
}

// GetOutreachState gets the outreach state for a contact (handler-compatible)
// Always recalculates from DetermineConversationState and syncs both outreach_states and contacts tables
func (s *Service) GetOutreachState(ctx context.Context, userID, contactID uuid.UUID) (*outreach.OutreachState, error) {
	contactEntity, err := s.contactRepo.FindByIDAndUserID(ctx, userID, contactID)
	if err != nil {
		return nil, fmt.Errorf("contact not found: %w", err)
	}

	// Always recalculate real-time state
	stateResult, err := s.DetermineConversationState(ctx, contactEntity, userID)
	if err != nil {
		return nil, err
	}

	// Determine context level based on contact data
	contextLevel := string(outreach.ContextLow)
	if contactEntity.Company != "" && contactEntity.JobTitle != "" {
		contextLevel = string(outreach.ContextMedium)
	}
	// Both columns default to '{}', so "non-empty bytes" is true for everyone.
	if jsonHasContent(contactEntity.AIInsights) || jsonHasContent(contactEntity.ConfirmedFacts) {
		contextLevel = string(outreach.ContextHigh)
	}

	// Try to find existing state to preserve fields like LastOutcome
	existingState, _ := s.stateRepo.FindByContactID(ctx, contactID, userID)
	lastOutcome := string(outreach.OutcomeNone)
	if existingState != nil {
		lastOutcome = existingState.LastOutcome
	}

	// Build state with real-time values
	state := &outreach.OutreachState{
		UserID:            userID,
		ContactID:         contactID,
		ConversationState: string(stateResult.State),
		ContextLevel:      contextLevel,
		OutreachIntent:    string(stateResult.OutreachIntent),
		Scenario:          string(stateResult.Scenario),
		LastOutcome:       lastOutcome,
		NextStep:          string(stateResult.NextStep),
		FollowupCount:     stateResult.FollowupCount,
		MaxFollowups:      s.config.MaxFollowups,
	}

	if stateResult.LastOutgoing != nil {
		state.LastInteractionAt = &stateResult.LastOutgoing.Timestamp
		state.DaysSinceLastInteraction = stateResult.DaysSinceLastInteraction
	}

	// Save to outreach_states table
	if err := s.stateRepo.Upsert(ctx, state); err != nil {
		return nil, fmt.Errorf("failed to save outreach state: %w", err)
	}

	// Sync contacts table so left panel matches right panel
	newNextStep := string(stateResult.NextStep)
	newOutreachStage := string(stateResult.State)
	if contactEntity.NextStep != newNextStep || contactEntity.OutreachStage != newOutreachStage {
		s.contactRepo.UpdateFields(ctx, contactID, map[string]interface{}{
			"next_step":      newNextStep,
			"outreach_stage": newOutreachStage,
		}, userID, uuid.Nil)
	}

	return state, nil
}

// jsonHasContent reports whether a JSONB value holds anything beyond an empty
// object, empty array or null.
func jsonHasContent(raw []byte) bool {
	switch strings.TrimSpace(string(raw)) {
	case "", "{}", "[]", "null":
		return false
	}
	return true
}

// GetInteractionHistory gets interaction history for a contact (handler-compatible)
func (s *Service) GetInteractionHistory(ctx context.Context, userID, contactID uuid.UUID, limit int) ([]*outreach.InteractionLog, error) {
	return s.interactionRepo.FindByContactIDAndUserID(ctx, contactID, userID, limit)
}

// CreateMeetingForHandler creates a meeting (handler-compatible signature)
func (s *Service) CreateMeetingForHandler(ctx context.Context, userID, contactID uuid.UUID, title, timeStr string, durationMinutes int, channel, location, meetingURL, note string) (*outreach.Meeting, error) {
	meetingTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid time format: %w", err)
	}

	if durationMinutes <= 0 {
		durationMinutes = 30
	}

	channelType := outreach.MeetingChannelZoom
	if channel != "" {
		channelType = outreach.MeetingChannel(channel)
	}

	input := &CreateMeetingInput{
		ContactID:       contactID,
		UserID:          userID,
		Title:           title,
		Time:            meetingTime,
		DurationMinutes: durationMinutes,
		Channel:         channelType,
		Location:        location,
		MeetingURL:      meetingURL,
		Note:            note,
	}

	return s.CreateMeeting(ctx, input)
}

// UpdateMeetingForHandler updates a meeting (handler-compatible signature)
func (s *Service) UpdateMeetingForHandler(ctx context.Context, userID, meetingID uuid.UUID, status, note, outcome, nextSteps, meetingContent string) (*outreach.Meeting, error) {
	statusType := outreach.MeetingScheduled
	if status != "" {
		statusType = outreach.MeetingStatus(status)
	}

	input := &UpdateMeetingInput{
		MeetingID:      meetingID,
		UserID:         userID,
		Status:         statusType,
		Note:           note,
		Outcome:        outcome,
		NextSteps:      nextSteps,
		MeetingContent: meetingContent,
	}

	return s.UpdateMeeting(ctx, input)
}

// GetMeetingsForHandler gets meetings for a contact (handler-compatible)
func (s *Service) GetMeetingsForHandler(ctx context.Context, userID, contactID uuid.UUID) ([]*outreach.Meeting, error) {
	return s.GetMeetings(ctx, contactID, userID)
}

// GetUpcomingMeetingsForHandler gets all upcoming scheduled meetings for a user (handler-compatible)
func (s *Service) GetUpcomingMeetingsForHandler(ctx context.Context, userID uuid.UUID) ([]*outreach.Meeting, error) {
	meetings, _, err := s.meetingRepo.FindUpcoming(ctx, userID, nil)
	return meetings, err
}

// DeleteMeetingForHandler deletes a meeting (handler-compatible)
func (s *Service) DeleteMeetingForHandler(ctx context.Context, userID, meetingID uuid.UUID) error {
	meeting, err := s.meetingRepo.FindByID(ctx, meetingID)
	if err != nil {
		return err
	}
	if meeting == nil {
		return fmt.Errorf("meeting not found")
	}
	// Check that the user owns this meeting
	if meeting.UserID != userID {
		return fmt.Errorf("unauthorized to delete this meeting")
	}
	return s.meetingRepo.Delete(ctx, meetingID)
}

// GenerateMeetingPrep generates meeting preparation document (talking points, discovery questions)
// This is generated BEFORE the meeting based on conversation history and contact info
// language: "vi" for Vietnamese, "en" for English
func (s *Service) GenerateMeetingPrep(ctx context.Context, userID, meetingID uuid.UUID, language string) (*outreach.Meeting, error) {
	meeting, err := s.meetingRepo.FindByID(ctx, meetingID)
	if err != nil {
		return nil, err
	}
	if meeting == nil || meeting.UserID != userID {
		return nil, fmt.Errorf("meeting not found")
	}

	// Meeting Prep is generated BEFORE the meeting (when scheduled)
	if meeting.Status != string(outreach.MeetingScheduled) {
		return nil, fmt.Errorf("meeting prep can only be generated for scheduled meetings")
	}

	// Get contact info for context
	contact, err := s.contactRepo.FindByID(ctx, meeting.ContactID)
	if err != nil {
		return nil, fmt.Errorf("failed to find contact: %w", err)
	}

	// Get conversation history (interaction logs)
	interactions, err := s.interactionRepo.FindByContactIDAndUserID(ctx, meeting.ContactID, userID, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}

	// Get team notes for additional context
	notes, _ := s.interactionRepo.FindNotes(ctx, meeting.ContactID, userID, 10)

	// Build prompt for AI based on language
	prompt := s.buildMeetingPrepPrompt(meeting, contact, interactions, notes, language)

	// Generate prep using AI
	prep, err := s.generateMeetingPrepWithAI(ctx, prompt, language)
	if err != nil {
		return nil, fmt.Errorf("failed to generate meeting prep: %w", err)
	}

	// Save the prep to the meeting
	meeting.MeetingPrep = &prep
	if err := s.meetingRepo.Update(ctx, meeting); err != nil {
		return nil, fmt.Errorf("failed to save meeting prep: %w", err)
	}

	return meeting, nil
}

// buildMeetingPrepPrompt builds the prompt for generating meeting preparation document
// language: "vi" for Vietnamese, "en" for English
func (s *Service) buildMeetingPrepPrompt(meeting *outreach.Meeting, contactEntity *contact.Contact, interactions []*outreach.InteractionLog, notes []*outreach.InteractionLog, language string) string {
	var sb strings.Builder

	if language == "en" {
		return s.buildMeetingPrepPromptEnglish(meeting, contactEntity, interactions, notes)
	}

	// Vietnamese version (default)
	sb.WriteString("Bạn là trợ lý AI chuyên chuẩn bị tài liệu cho cuộc họp kinh doanh.\n\n")
	sb.WriteString("Dựa trên thông tin khách hàng và lịch sử trao đổi dưới đây, hãy tạo tài liệu chuẩn bị cuộc họp (Meeting Prep) với các câu hỏi discovery cực kỳ chuẩn xác và các talking points phù hợp.\n\n")

	// Contact info
	sb.WriteString("=== THÔNG TIN KHÁCH HÀNG ===\n")
	if contactEntity != nil {
		if contactEntity.Name != "" && contactEntity.Name != "N/A" {
			sb.WriteString(fmt.Sprintf("Tên: %q\n", sanitizeField(contactEntity.Name)))
		}
		if contactEntity.JobTitle != "" && contactEntity.JobTitle != "N/A" {
			sb.WriteString(fmt.Sprintf("Chức vụ: %q\n", sanitizeField(contactEntity.JobTitle)))
		}
		if contactEntity.Company != "" && contactEntity.Company != "N/A" {
			sb.WriteString(fmt.Sprintf("Công ty: %q\n", sanitizeField(contactEntity.Company)))
		}
		if contactEntity.Industry != "" && contactEntity.Industry != "N/A" {
			sb.WriteString(fmt.Sprintf("Ngành: %q\n", sanitizeField(contactEntity.Industry)))
		}
		if contactEntity.ContactInformation != "" && contactEntity.ContactInformation != "N/A" {
			sb.WriteString(fmt.Sprintf("Thông tin liên hệ: %s\n", contactEntity.ContactInformation))
		}
		if contactEntity.ContextLevel != "" {
			sb.WriteString(fmt.Sprintf("Context Level: %s\n", contactEntity.ContextLevel))
		}
	}

	// Meeting info
	sb.WriteString("\n=== THÔNG TIN CUỘC HỌP ===\n")
	if meeting.Title != nil && *meeting.Title != "" {
		sb.WriteString(fmt.Sprintf("Tiêu đề: %s\n", *meeting.Title))
	}
	sb.WriteString(fmt.Sprintf("Thời gian: %s\n", meeting.Time.Format("02/01/2006 15:04")))
	sb.WriteString(fmt.Sprintf("Thời lượng: %d phút\n", meeting.DurationMinutes))
	sb.WriteString(fmt.Sprintf("Kênh: %s\n", meeting.Channel))

	// Conversation history
	sb.WriteString("\n=== LỊCH SỬ TRAO ĐỔI ===\n")
	if len(interactions) == 0 {
		sb.WriteString("Chưa có lịch sử trao đổi.\n")
	} else {
		for i, interaction := range interactions {
			direction := "→ Gửi đi"
			if interaction.Direction == string(outreach.DirectionIncoming) {
				direction = "← Nhận"
			}
			sb.WriteString(fmt.Sprintf("%d. [%s] %s (%s):\n", i+1, interaction.Timestamp.Format("02/01/2006"), direction, interaction.Channel))
			sb.WriteString(fmt.Sprintf("   %s\n\n", interaction.Content))
		}
	}

	// Team notes
	if len(notes) > 0 {
		sb.WriteString("\n=== GHI CHÚ CỦA TEAM ===\n")
		for _, note := range notes {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", note.Timestamp.Format("02/01/2006"), note.Content))
		}
	}

	// Instructions
	sb.WriteString("\n\n=== YÊU CẦU ===\n")
	sb.WriteString("Hãy tạo Meeting Prep Document với các phần sau:\n\n")
	sb.WriteString("1. **Mục tiêu cuộc họp** (2-3 mục tiêu cụ thể cần đạt được)\n\n")
	sb.WriteString("2. **Talking Points** (5-7 điểm chính cần nói trong cuộc họp, dựa trên context đã có)\n\n")
	sb.WriteString("3. **Discovery Questions** (7-10 câu hỏi cực kỳ chuẩn xác để khám phá nhu cầu, pain points và cơ hội):\n")
	sb.WriteString("   - Câu hỏi về tình trạng hiện tại (Current State)\n")
	sb.WriteString("   - Câu hỏi về thách thức/pain points\n")
	sb.WriteString("   - Câu hỏi về mục tiêu/kỳ vọng\n")
	sb.WriteString("   - Câu hỏi về quy trình ra quyết định\n")
	sb.WriteString("   - Câu hỏi về timeline và budget\n\n")
	sb.WriteString("4. **Potential Objections** (3-5 phản đối có thể gặp và cách xử lý)\n\n")
	sb.WriteString("5. **Next Steps đề xuất** (các bước tiếp theo sau cuộc họp)\n\n")
	sb.WriteString("Lưu ý:\n")
	sb.WriteString("- Các câu hỏi phải DỰA TRÊN thông tin đã có về khách hàng và lịch sử trao đổi\n")
	sb.WriteString("- Tránh hỏi lại những gì đã biết\n")
	sb.WriteString("- Câu hỏi phải cụ thể, không chung chung\n")
	sb.WriteString("- Trả lời bằng tiếng Việt, rõ ràng và dễ sử dụng trong cuộc họp")

	return sb.String()
}

// buildMeetingPrepPromptEnglish builds the English version of meeting prep prompt
func (s *Service) buildMeetingPrepPromptEnglish(meeting *outreach.Meeting, contactEntity *contact.Contact, interactions []*outreach.InteractionLog, notes []*outreach.InteractionLog) string {
	var sb strings.Builder

	sb.WriteString("You are an AI assistant specialized in preparing business meeting documents.\n\n")
	sb.WriteString("Based on the customer information and conversation history below, create a Meeting Prep document with highly accurate discovery questions and appropriate talking points.\n\n")

	// Contact info
	sb.WriteString("=== CUSTOMER INFORMATION ===\n")
	if contactEntity != nil {
		if contactEntity.Name != "" && contactEntity.Name != "N/A" {
			sb.WriteString(fmt.Sprintf("Name: %q\n", sanitizeField(contactEntity.Name)))
		}
		if contactEntity.JobTitle != "" && contactEntity.JobTitle != "N/A" {
			sb.WriteString(fmt.Sprintf("Job Title: %q\n", sanitizeField(contactEntity.JobTitle)))
		}
		if contactEntity.Company != "" && contactEntity.Company != "N/A" {
			sb.WriteString(fmt.Sprintf("Company: %q\n", sanitizeField(contactEntity.Company)))
		}
		if contactEntity.Industry != "" && contactEntity.Industry != "N/A" {
			sb.WriteString(fmt.Sprintf("Industry: %q\n", sanitizeField(contactEntity.Industry)))
		}
		if contactEntity.ContactInformation != "" && contactEntity.ContactInformation != "N/A" {
			sb.WriteString(fmt.Sprintf("Contact Information: %s\n", contactEntity.ContactInformation))
		}
		if contactEntity.ContextLevel != "" {
			sb.WriteString(fmt.Sprintf("Context Level: %s\n", contactEntity.ContextLevel))
		}
	}

	// Meeting info
	sb.WriteString("\n=== MEETING INFORMATION ===\n")
	if meeting.Title != nil && *meeting.Title != "" {
		sb.WriteString(fmt.Sprintf("Title: %s\n", *meeting.Title))
	}
	sb.WriteString(fmt.Sprintf("Time: %s\n", meeting.Time.Format("01/02/2006 15:04")))
	sb.WriteString(fmt.Sprintf("Duration: %d minutes\n", meeting.DurationMinutes))
	sb.WriteString(fmt.Sprintf("Channel: %s\n", meeting.Channel))

	// Conversation history
	sb.WriteString("\n=== CONVERSATION HISTORY ===\n")
	if len(interactions) == 0 {
		sb.WriteString("No conversation history yet.\n")
	} else {
		for i, interaction := range interactions {
			direction := "→ Sent"
			if interaction.Direction == string(outreach.DirectionIncoming) {
				direction = "← Received"
			}
			sb.WriteString(fmt.Sprintf("%d. [%s] %s (%s):\n", i+1, interaction.Timestamp.Format("01/02/2006"), direction, interaction.Channel))
			sb.WriteString(fmt.Sprintf("   %s\n\n", interaction.Content))
		}
	}

	// Team notes
	if len(notes) > 0 {
		sb.WriteString("\n=== TEAM NOTES ===\n")
		for _, note := range notes {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", note.Timestamp.Format("01/02/2006"), note.Content))
		}
	}

	// Instructions
	sb.WriteString("\n\n=== REQUIREMENTS ===\n")
	sb.WriteString("Create a Meeting Prep Document with the following sections:\n\n")
	sb.WriteString("1. **Meeting Objectives** (2-3 specific goals to achieve)\n\n")
	sb.WriteString("2. **Talking Points** (5-7 key points to discuss in the meeting, based on existing context)\n\n")
	sb.WriteString("3. **Discovery Questions** (7-10 highly accurate questions to uncover needs, pain points, and opportunities):\n")
	sb.WriteString("   - Questions about current state\n")
	sb.WriteString("   - Questions about challenges/pain points\n")
	sb.WriteString("   - Questions about goals/expectations\n")
	sb.WriteString("   - Questions about decision-making process\n")
	sb.WriteString("   - Questions about timeline and budget\n\n")
	sb.WriteString("4. **Potential Objections** (3-5 possible objections and how to handle them)\n\n")
	sb.WriteString("5. **Suggested Next Steps** (follow-up actions after the meeting)\n\n")
	sb.WriteString("Notes:\n")
	sb.WriteString("- Questions must be BASED ON the existing information about the customer and conversation history\n")
	sb.WriteString("- Avoid asking about things already known\n")
	sb.WriteString("- Questions should be specific, not generic\n")
	sb.WriteString("- Respond in English, clear and easy to use in the meeting")

	return sb.String()
}

// generateMeetingPrepWithAI generates meeting prep using AI
// language: "vi" for Vietnamese, "en" for English
func (s *Service) generateMeetingPrepWithAI(ctx context.Context, prompt string, language string) (string, error) {
	if s.openAIClient == nil {
		return "", fmt.Errorf("OpenAI client not configured")
	}

	systemPrompt := "Bạn là trợ lý AI chuyên chuẩn bị tài liệu cho cuộc họp kinh doanh. Bạn giỏi trong việc đặt câu hỏi discovery chính xác và tạo talking points hiệu quả. Trả lời bằng tiếng Việt."
	if language == "en" {
		systemPrompt = "You are an AI assistant specialized in preparing business meeting documents. You excel at crafting accurate discovery questions and creating effective talking points. Respond in English."
	}

	response, err := s.openAIClient.ChatCompletion(ctx, ai.ChatCompletionRequest{
		SystemPrompt: systemPrompt,
		Messages: []ai.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   2000,
		Temperature: 0.7,
	})

	if err != nil {
		return "", err
	}

	return response, nil
}

// ============================================
// Feedback Loop Operations (Task 8)
// ============================================

// RecordSuggestion records an AI suggestion for feedback tracking
func (s *Service) RecordSuggestion(ctx context.Context, userID, contactID uuid.UUID, scenario, intent, draft, nextStep, contextLevel, conversationState string) (*outreach.OutreachFeedback, error) {
	feedback := &outreach.OutreachFeedback{
		UserID:            userID,
		ContactID:         contactID,
		SuggestedScenario: scenario,
		SuggestedIntent:   intent,
		ContextLevel:      contextLevel,
		ConversationState: conversationState,
	}

	if draft != "" {
		feedback.SuggestedDraft = &draft
	}
	if nextStep != "" {
		feedback.SuggestedNextStep = &nextStep
	}

	if err := s.feedbackRepo.Create(ctx, feedback); err != nil {
		return nil, fmt.Errorf("failed to record suggestion: %w", err)
	}

	return feedback, nil
}

// RecordAction records what BD actually did with a suggestion
func (s *Service) RecordAction(ctx context.Context, feedbackID uuid.UUID, action, actualContent string) error {
	feedback, err := s.feedbackRepo.FindByID(ctx, feedbackID)
	if err != nil {
		return err
	}
	if feedback == nil {
		return fmt.Errorf("feedback not found")
	}

	feedback.ActualAction = &action
	if actualContent != "" {
		feedback.ActualContent = &actualContent
	}

	return s.feedbackRepo.Update(ctx, feedback)
}

// RecordOutcome records the outcome of an outreach attempt
func (s *Service) RecordOutcome(ctx context.Context, contactID, userID uuid.UUID, outcome string, sentiment *string) error {
	// Find the latest feedback for this contact
	feedback, err := s.feedbackRepo.FindLatestForContact(ctx, contactID, userID)
	if err != nil {
		return err
	}
	if feedback == nil {
		// No feedback to update
		return nil
	}

	// Calculate days to reply if applicable
	var daysToReply *int
	if outcome == string(outreach.FeedbackOutcomeReplied) || outcome == string(outreach.FeedbackOutcomeMeetingBooked) {
		days := int(time.Since(feedback.CreatedAt).Hours() / 24)
		daysToReply = &days
	}

	return s.feedbackRepo.UpdateOutcome(ctx, feedback.ID, outcome, sentiment, daysToReply)
}

// GetScenarioStats gets statistics for all scenarios
func (s *Service) GetScenarioStats(ctx context.Context, userID uuid.UUID) ([]*outreach.ScenarioStats, error) {
	return s.feedbackRepo.GetScenarioStats(ctx, userID)
}

// GetOverallStats gets overall statistics
func (s *Service) GetOverallStats(ctx context.Context, userID uuid.UUID) (*outreach.ScenarioStats, error) {
	return s.feedbackRepo.GetOverallStats(ctx, userID)
}

// FeedbackStatsResponse holds feedback statistics response
type FeedbackStatsResponse struct {
	Overall    *outreach.ScenarioStats   `json:"overall"`
	ByScenario []*outreach.ScenarioStats `json:"by_scenario"`
	Insights   []string                  `json:"insights"`
}

// GetFeedbackStats gets comprehensive feedback statistics with insights
func (s *Service) GetFeedbackStats(ctx context.Context, userID uuid.UUID) (*FeedbackStatsResponse, error) {
	overall, err := s.GetOverallStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	byScenario, err := s.GetScenarioStats(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Generate insights
	insights := s.generateInsights(overall, byScenario)

	return &FeedbackStatsResponse{
		Overall:    overall,
		ByScenario: byScenario,
		Insights:   insights,
	}, nil
}

// generateInsights generates actionable insights from feedback data
func (s *Service) generateInsights(overall *outreach.ScenarioStats, byScenario []*outreach.ScenarioStats) []string {
	var insights []string

	if overall == nil || overall.TotalSuggested == 0 {
		return []string{"Not enough data yet. Keep using the system to generate insights."}
	}

	// Draft usage insight
	if overall.DraftUsageRate > 70 {
		insights = append(insights, fmt.Sprintf("Great! You're using %.1f%% of AI drafts. The suggestions are working well.", overall.DraftUsageRate))
	} else if overall.DraftUsageRate < 30 {
		insights = append(insights, fmt.Sprintf("You're only using %.1f%% of AI drafts. Consider providing feedback to improve suggestions.", overall.DraftUsageRate))
	}

	// Reply rate insight
	if overall.ReplyRate > 20 {
		insights = append(insights, fmt.Sprintf("Strong reply rate of %.1f%%. Your outreach is resonating.", overall.ReplyRate))
	} else if overall.ReplyRate < 10 && overall.TotalSent > 10 {
		insights = append(insights, fmt.Sprintf("Reply rate is %.1f%%. Consider adjusting your approach or targeting.", overall.ReplyRate))
	}

	// Meeting rate insight
	if overall.MeetingRate > 30 {
		insights = append(insights, fmt.Sprintf("Excellent conversion! %.1f%% of replies turn into meetings.", overall.MeetingRate))
	}

	// Find best performing scenario
	var bestScenario *outreach.ScenarioStats
	for _, s := range byScenario {
		if s.TotalSent >= 5 { // Need at least 5 sends for meaningful comparison
			if bestScenario == nil || s.ReplyRate > bestScenario.ReplyRate {
				bestScenario = s
			}
		}
	}

	if bestScenario != nil && bestScenario.ReplyRate > 0 {
		insights = append(insights, fmt.Sprintf("Best performing: '%s' scenario with %.1f%% reply rate.", bestScenario.Scenario, bestScenario.ReplyRate))
	}

	// Find worst performing scenario
	var worstScenario *outreach.ScenarioStats
	for _, s := range byScenario {
		if s.TotalSent >= 5 {
			if worstScenario == nil || s.ReplyRate < worstScenario.ReplyRate {
				worstScenario = s
			}
		}
	}

	if worstScenario != nil && bestScenario != nil && worstScenario.Scenario != bestScenario.Scenario {
		insights = append(insights, fmt.Sprintf("Consider improving: '%s' scenario has only %.1f%% reply rate.", worstScenario.Scenario, worstScenario.ReplyRate))
	}

	if len(insights) == 0 {
		insights = append(insights, "Keep going! More data will help generate better insights.")
	}

	return insights
}

// RecordSuggestionFromDraft records a suggestion when draft is generated
func (s *Service) RecordSuggestionFromDraft(ctx context.Context, result *DraftResult, userID uuid.UUID) (*outreach.OutreachFeedback, error) {
	if s.feedbackRepo == nil {
		return nil, nil // Feedback repo not initialized
	}

	return s.RecordSuggestion(
		ctx,
		userID,
		result.ContactID,
		string(result.Scenario),
		string(result.State.OutreachIntent),
		result.Draft,
		result.State.NextStep,
		string(result.ContextLevel),
		result.State.ConversationState,
	)
}

// ============================================
// Contact Notes Operations (Team Conversation History)
// ============================================

// AddNote adds an internal note for a contact
func (s *Service) AddNote(ctx context.Context, userID, contactID uuid.UUID, content string) (*outreach.InteractionLog, error) {
	note := &outreach.InteractionLog{
		UserID:    userID,
		ContactID: contactID,
		Channel:   string(outreach.ChannelNote),
		Direction: string(outreach.DirectionInternal),
		Content:   content,
		Timestamp: time.Now(),
	}

	if err := s.interactionRepo.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	return note, nil
}

// GetNotes gets all internal notes for a contact
func (s *Service) GetNotes(ctx context.Context, userID, contactID uuid.UUID, limit int) ([]*outreach.InteractionLog, error) {
	return s.interactionRepo.FindNotes(ctx, contactID, userID, limit)
}

// AddNoteForHandler adds a note (handler-compatible signature)
func (s *Service) AddNoteForHandler(ctx context.Context, userID, contactID uuid.UUID, content string) (*outreach.InteractionLog, error) {
	return s.AddNote(ctx, userID, contactID, content)
}

// GetNotesForHandler gets notes (handler-compatible signature)
func (s *Service) GetNotesForHandler(ctx context.Context, userID, contactID uuid.UUID, limit int) ([]*outreach.InteractionLog, error) {
	return s.GetNotes(ctx, userID, contactID, limit)
}

// AddInteraction adds a conversation interaction (message sent or received)
// direction: "outgoing" for messages sent by user, "incoming" for messages from client
func (s *Service) AddInteraction(ctx context.Context, userID, contactID uuid.UUID, content, direction, channel, sentiment string) (*outreach.InteractionLog, error) {
	// Validate direction
	if direction != string(outreach.DirectionOutgoing) && direction != string(outreach.DirectionIncoming) {
		direction = string(outreach.DirectionOutgoing) // Default to outgoing
	}

	// Default channel
	if channel == "" {
		channel = string(outreach.ChannelLinkedIn)
	}

	interaction := &outreach.InteractionLog{
		UserID:    userID,
		ContactID: contactID,
		Channel:   channel,
		Direction: direction,
		Content:   content,
		Timestamp: time.Now(),
	}

	// Set sentiment if provided
	if sentiment != "" {
		interaction.Sentiment = &sentiment
	}

	if err := s.interactionRepo.Create(ctx, interaction); err != nil {
		return nil, fmt.Errorf("failed to create interaction: %w", err)
	}

	// Get contact for state recalculation
	contactEntity, err := s.contactRepo.GetByID(ctx, contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	// Recalculate state after adding interaction
	stateResult, err := s.DetermineConversationState(ctx, contactEntity, userID)
	if err != nil {
		// Log error but don't fail the interaction creation
		// The worker will recalculate later
	} else {
		// Update contact with new state
		s.contactRepo.UpdateFields(ctx, contactID, map[string]interface{}{
			"last_interaction_at": interaction.Timestamp,
			"next_step":           string(stateResult.NextStep),
			"outreach_stage":      string(stateResult.State),
			"scenario":            string(stateResult.Scenario),
			"followup_count":      stateResult.FollowupCount,
		}, userID, uuid.Nil)
	}

	return interaction, nil
}

// UpdateNote updates an internal note
func (s *Service) UpdateNote(ctx context.Context, userID, noteID uuid.UUID, content string) (*outreach.InteractionLog, error) {
	note, err := s.interactionRepo.FindNoteByID(ctx, noteID)
	if err != nil {
		return nil, fmt.Errorf("failed to find note: %w", err)
	}
	if note == nil {
		return nil, fmt.Errorf("note not found")
	}
	if note.UserID != userID {
		return nil, fmt.Errorf("unauthorized to update this note")
	}
	if note.Direction != string(outreach.DirectionInternal) {
		return nil, fmt.Errorf("can only update internal notes")
	}

	if err := s.interactionRepo.UpdateNote(ctx, noteID, content); err != nil {
		return nil, fmt.Errorf("failed to update note: %w", err)
	}

	note.Content = content
	return note, nil
}

// UpdateNoteForHandler updates a note (handler-compatible signature)
func (s *Service) UpdateNoteForHandler(ctx context.Context, userID, noteID uuid.UUID, content string) (*outreach.InteractionLog, error) {
	return s.UpdateNote(ctx, userID, noteID, content)
}

// DeleteNote deletes an internal note
func (s *Service) DeleteNote(ctx context.Context, userID, noteID uuid.UUID) error {
	note, err := s.interactionRepo.FindNoteByID(ctx, noteID)
	if err != nil {
		return fmt.Errorf("failed to find note: %w", err)
	}
	if note == nil {
		return fmt.Errorf("note not found")
	}
	if note.UserID != userID {
		return fmt.Errorf("unauthorized to delete this note")
	}
	if note.Direction != string(outreach.DirectionInternal) {
		return fmt.Errorf("can only delete internal notes")
	}

	if err := s.interactionRepo.DeleteNote(ctx, noteID); err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	return nil
}

// DeleteNoteForHandler deletes a note (handler-compatible signature)
func (s *Service) DeleteNoteForHandler(ctx context.Context, userID, noteID uuid.UUID) error {
	return s.DeleteNote(ctx, userID, noteID)
}
