package v1

import (
	"time"

	"github.com/google/uuid"
)

// --- Request DTOs ---

// GenerateRequest is the request body for POST /v1/daily-actions/generate.
type GenerateRequest struct {
	Language     string `json:"language" validate:"omitempty,oneof=en vi"`
	ForceRefresh bool   `json:"force_refresh"`
}

// UpdateActionRequest is the request body for PATCH /v1/daily-actions/{action_id}.
type UpdateActionRequest struct {
	Transition     string     `json:"transition" validate:"required,oneof=mark_sent skip snooze snooze_custom mark_completed reopen"`
	Content        *string    `json:"content,omitempty"`
	Channel        *string    `json:"channel,omitempty" validate:"omitempty,oneof=LinkedIn Email Call Meeting"`
	SkipReason     *string    `json:"skip_reason,omitempty"`
	SnoozeUntil    *time.Time `json:"snooze_until,omitempty"`
	FeedbackAction *string    `json:"feedback_action,omitempty" validate:"omitempty,oneof=used_draft modified_draft wrote_own skipped"`
}

// ChatRequest is the request body for POST /v1/daily-actions/chat.
type ChatRequest struct {
	Messages []ChatMessageInput `json:"messages" validate:"required,min=1"`
	Stream   *bool              `json:"stream,omitempty"`
}

// ChatMessageInput represents a single message in the chat request.
type ChatMessageInput struct {
	Role    string `json:"role" validate:"required,oneof=user assistant system"`
	Content string `json:"content" validate:"required"`
}

// --- Response DTOs ---

// GenerateResponseData is the data payload for POST /v1/daily-actions/generate.
type GenerateResponseData struct {
	GenerationID     *uuid.UUID `json:"generation_id,omitempty"`
	GenerationStatus string     `json:"generation_status"`
}

// DailyActionsBriefing is the data payload for GET /v1/daily-actions.
type DailyActionsBriefing struct {
	GenerationID     *uuid.UUID              `json:"generation_id,omitempty"`
	GenerationStatus string                  `json:"generation_status"`
	GeneratedAt      *time.Time              `json:"generated_at,omitempty"`
	Date             string                  `json:"date"`
	Language         string                  `json:"language"`
	AgentBriefing    *AgentBriefingResponse  `json:"agent_briefing,omitempty"`
	Categories       []ActionCategoryResponse `json:"categories"`
	PipelineSummary  *PipelineSummaryResponse `json:"pipeline_summary,omitempty"`
	Progress         *ActionProgressResponse  `json:"progress,omitempty"`
}

// AgentBriefingResponse represents the AI-generated briefing in the response.
type AgentBriefingResponse struct {
	Greeting           string                    `json:"greeting"`
	StrategicReasoning string                    `json:"strategic_reasoning"`
	MemoryReferences   []MemoryReferenceResponse `json:"memory_references"`
	CategoryCounts     []CategoryCountResponse   `json:"category_counts"`
}

// MemoryReferenceResponse is a conversation memory citation.
type MemoryReferenceResponse struct {
	ContactID      string `json:"contact_id"`
	ContactName    string `json:"contact_name"`
	EventSummary   string `json:"event_summary"`
	EventTimestamp string `json:"event_timestamp"`
	Relevance      string `json:"relevance"`
}

// CategoryCountResponse represents a category summary badge.
type CategoryCountResponse struct {
	Category string `json:"category"`
	Label    string `json:"label"`
	Count    int    `json:"count"`
	Icon     string `json:"icon"`
	Color    string `json:"color"`
}

// ActionCategoryResponse represents a group of actions in a category.
type ActionCategoryResponse struct {
	ID          string                `json:"id"`
	Label       string                `json:"label"`
	Icon        string                `json:"icon"`
	Color       string                `json:"color"`
	Description string                `json:"description"`
	Actions     []DailyActionResponse `json:"actions"`
	TotalCount  int                   `json:"total_count"`
	HasMore     bool                  `json:"has_more"`
	NextOffset  *int                  `json:"next_offset,omitempty"`
}

// DailyActionResponse represents a single action card.
type DailyActionResponse struct {
	ID              uuid.UUID                   `json:"id"`
	Type            string                      `json:"type"`
	Contact         ActionContactResponse       `json:"contact"`
	Priority        int                         `json:"priority"`
	PriorityFactors []PriorityFactorResponse    `json:"priority_factors,omitempty"`
	Reasoning       string                      `json:"reasoning"`
	Status          string                      `json:"status"`
	StatusChangedAt *time.Time                  `json:"status_changed_at,omitempty"`
	SnoozeUntil     *time.Time                  `json:"snooze_until,omitempty"`
	OutreachData    *OutreachActionDataResponse `json:"outreach_data,omitempty"`
	MeetingData     *MeetingActionDataResponse  `json:"meeting_data,omitempty"`
	EnrichmentData  *EnrichmentActionDataResponse `json:"enrichment_data,omitempty"`
	RespondData     *RespondActionDataResponse  `json:"respond_data,omitempty"`
	CreatedAt       time.Time                   `json:"created_at"`
	UpdatedAt       time.Time                   `json:"updated_at"`
}

// PriorityFactorResponse represents a priority score breakdown factor.
type PriorityFactorResponse struct {
	Factor      string `json:"factor"`
	Value       int    `json:"value"`
	Description string `json:"description"`
}

// ActionContactResponse is the lightweight contact summary embedded in actions.
type ActionContactResponse struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email,omitempty"`
	Company        string    `json:"company,omitempty"`
	JobTitle       string    `json:"job_title,omitempty"`
	LinkedinURL    string    `json:"linkedin_url,omitempty"`
	Source         string    `json:"source,omitempty"`
	Status         string    `json:"status,omitempty"`
	OutreachStage  string    `json:"outreach_stage,omitempty"`
	LifecycleStage string    `json:"lifecycle_stage,omitempty"`
	AvatarURL      *string   `json:"avatar_url,omitempty"`
}

// PipelineSummaryResponse holds high-level pipeline stats.
type PipelineSummaryResponse struct {
	TotalActiveContacts  int            `json:"total_active_contacts"`
	ContactsByStage      map[string]int `json:"contacts_by_stage"`
	ContactsByLifecycle  map[string]int `json:"contacts_by_lifecycle"`
	ResponseRate7d       float64        `json:"response_rate_7d"`
	MeetingsBooked7d     int            `json:"meetings_booked_7d"`
	AvgResponseTimeHours float64        `json:"avg_response_time_hours"`
}

// ActionProgressResponse holds running tally of action completion.
type ActionProgressResponse struct {
	Total          int     `json:"total"`
	Completed      int     `json:"completed"`
	Skipped        int     `json:"skipped"`
	Snoozed        int     `json:"snoozed"`
	Remaining      int     `json:"remaining"`
	CompletionRate float64 `json:"completion_rate"`
}

// --- Type-specific action data responses ---

// OutreachActionDataResponse holds outreach/followup-specific data.
type OutreachActionDataResponse struct {
	DraftMessage             string                        `json:"draft_message"`
	Scenario                 string                        `json:"scenario"`
	ContextLevel             string                        `json:"context_level"`
	CompanyContext            string                        `json:"company_context,omitempty"`
	FollowupNumber           *int                          `json:"followup_number,omitempty"`
	DaysSinceLastInteraction int                           `json:"days_since_last_interaction"`
	PreviousMessagesCount    int                           `json:"previous_messages_count"`
	LastSentDate             *string                       `json:"last_sent_date,omitempty"`
	IsFinalFollowup          bool                          `json:"is_final_followup"`
	OutreachState            *OutreachStateSnapshotResponse `json:"outreach_state,omitempty"`
}

// OutreachStateSnapshotResponse captures outreach state at generation time.
type OutreachStateSnapshotResponse struct {
	ConversationState string `json:"conversation_state"`
	NextStep          string `json:"next_step"`
	FollowupCount     int    `json:"followup_count"`
	MaxFollowups      int    `json:"max_followups"`
}

// MeetingActionDataResponse holds meeting_prep-specific data.
type MeetingActionDataResponse struct {
	MeetingID              string                   `json:"meeting_id"`
	MeetingTitle           string                   `json:"meeting_title"`
	MeetingTime            string                   `json:"meeting_time"`
	MeetingDurationMinutes int                      `json:"meeting_duration_minutes"`
	MeetingChannel         string                   `json:"meeting_channel,omitempty"`
	HoursUntilMeeting      float64                  `json:"hours_until_meeting"`
	Briefing               *MeetingBriefingResponse `json:"briefing,omitempty"`
}

// MeetingBriefingResponse holds the meeting preparation document.
type MeetingBriefingResponse struct {
	ProspectProfileSummary string                         `json:"prospect_profile_summary"`
	ConversationSummary    *ConversationSummaryResponse   `json:"conversation_summary,omitempty"`
	PainPoints             []PainPointResponse            `json:"pain_points,omitempty"`
	SuggestedAgenda        []AgendaItemResponse           `json:"suggested_agenda,omitempty"`
	DiscoveryQuestions     []string                       `json:"discovery_questions,omitempty"`
	RecommendedNextSteps   []string                       `json:"recommended_next_steps,omitempty"`
	RiskFlags              []string                       `json:"risk_flags,omitempty"`
}

// ConversationSummaryResponse summarizes previous conversation touchpoints.
type ConversationSummaryResponse struct {
	TouchpointCount int      `json:"touchpoint_count"`
	DurationDays    int      `json:"duration_days"`
	ToneAssessment  string   `json:"tone_assessment"`
	KeyTopics       []string `json:"key_topics,omitempty"`
}

// PainPointResponse represents an identified pain point.
type PainPointResponse struct {
	PainPoint  string   `json:"pain_point"`
	Confidence float64  `json:"confidence"`
	Evidence   []string `json:"evidence,omitempty"`
}

// AgendaItemResponse represents a suggested meeting agenda topic.
type AgendaItemResponse struct {
	Topic           string `json:"topic"`
	DurationMinutes int    `json:"duration_minutes"`
	Notes           string `json:"notes,omitempty"`
}

// EnrichmentActionDataResponse holds enrich-specific data.
type EnrichmentActionDataResponse struct {
	MissingFields    []string                  `json:"missing_fields"`
	QualityImpact    string                    `json:"quality_impact"`
	ContactStatus    string                    `json:"contact_status"`
	SuggestedSources []SuggestedSourceResponse `json:"suggested_sources,omitempty"`
}

// SuggestedSourceResponse suggests where to find missing contact data.
type SuggestedSourceResponse struct {
	Field  string `json:"field"`
	Source string `json:"source"`
	URL    string `json:"url,omitempty"`
}

// RespondActionDataResponse holds respond (replied) specific data.
type RespondActionDataResponse struct {
	ReplyPreview        string                          `json:"reply_preview"`
	ReplyTimestamp      string                          `json:"reply_timestamp"`
	ReplyChannel        string                          `json:"reply_channel"`
	IntentAssessment    string                          `json:"intent_assessment"`
	IntentReasoning     string                          `json:"intent_reasoning"`
	RecommendedAction   string                          `json:"recommended_action"`
	DraftResponse       string                          `json:"draft_response"`
	ConversationContext *ConversationContextResponse     `json:"conversation_context,omitempty"`
}

// ConversationContextResponse provides context about conversation history.
type ConversationContextResponse struct {
	TotalInteractions          int      `json:"total_interactions"`
	DaysInConversation         int      `json:"days_in_conversation"`
	LastOutgoingMessagePreview string   `json:"last_outgoing_message_preview,omitempty"`
	KeyTopicsDiscussed         []string `json:"key_topics_discussed,omitempty"`
}

// --- Update action response ---

// UpdateActionResponseData is the data payload for PATCH /v1/daily-actions/{action_id}.
type UpdateActionResponseData struct {
	Action             DailyActionResponse     `json:"action"`
	ContactStateChange *ContactStateChange     `json:"contact_state_change,omitempty"`
	Progress           *ActionProgressResponse `json:"progress,omitempty"`
}

// ContactStateChange indicates the contact's pipeline state changed.
type ContactStateChange struct {
	PreviousState string `json:"previous_state"`
	NewState      string `json:"new_state"`
	NewNextStep   string `json:"new_next_step"`
}

// --- Daily summary response ---

// DailySummaryResponseData is the data payload for GET /v1/daily-actions/summary.
type DailySummaryResponseData struct {
	Date         string                  `json:"date"`
	AgentSummary string                  `json:"agent_summary"`
	Progress     ActionProgressResponse  `json:"progress"`
	Breakdown    SummaryBreakdown        `json:"breakdown"`
	Outcomes     SummaryOutcomes         `json:"outcomes"`
	CarryOver    []CarryOverItem         `json:"carry_over,omitempty"`
}

// SummaryBreakdown holds action completion counts by type.
type SummaryBreakdown struct {
	OutreachSent      int `json:"outreach_sent"`
	FollowupsSent     int `json:"followups_sent"`
	RepliesHandled    int `json:"replies_handled"`
	MeetingsPrepped   int `json:"meetings_prepped"`
	ContactsEnriched  int `json:"contacts_enriched"`
}

// SummaryOutcomes holds outcome metrics for the day.
type SummaryOutcomes struct {
	ResponsesReceivedToday int `json:"responses_received_today"`
	MeetingsBookedToday    int `json:"meetings_booked_today"`
	ContactsDropped        int `json:"contacts_dropped"`
}

// CarryOverItem represents an action that carries over to tomorrow.
type CarryOverItem struct {
	ActionID    uuid.UUID `json:"action_id"`
	Type        string    `json:"type"`
	ContactName string    `json:"contact_name"`
	Reason      string    `json:"reason"`
}

// --- Load more actions response ---

// LoadMoreActionsResponseData is the data payload for GET /v1/daily-actions/categories/{id}/actions.
type LoadMoreActionsResponseData struct {
	Actions    []DailyActionResponse `json:"actions"`
	HasMore    bool                  `json:"has_more"`
	NextOffset *int                  `json:"next_offset,omitempty"`
}

// --- SSE Event schemas ---

// SSEProspectRepliedEvent is sent when a prospect responds to outreach.
type SSEProspectRepliedEvent struct {
	EventType         string                `json:"event_type"`
	EventID           string                `json:"event_id"`
	Timestamp         time.Time             `json:"timestamp"`
	Contact           ActionContactResponse `json:"contact"`
	ReplyPreview      string                `json:"reply_preview"`
	ReplyTimestamp    time.Time             `json:"reply_timestamp"`
	ReplyChannel      string                `json:"reply_channel"`
	IntentAssessment  string                `json:"intent_assessment"`
	AgentReasoning    string                `json:"agent_reasoning"`
	RecommendedAction string                `json:"recommended_action"`
	DraftResponse     string                `json:"draft_response,omitempty"`
	ActionID          *uuid.UUID            `json:"action_id,omitempty"`
}

// SSEMeetingApproachingEvent is sent 2 hours before a scheduled meeting.
type SSEMeetingApproachingEvent struct {
	EventType    string                `json:"event_type"`
	EventID      string                `json:"event_id"`
	Timestamp    time.Time             `json:"timestamp"`
	MeetingID    string                `json:"meeting_id"`
	MeetingTitle string                `json:"meeting_title"`
	MeetingTime  time.Time             `json:"meeting_time"`
	Contact      ActionContactResponse `json:"contact"`
	HoursUntil   float64               `json:"hours_until"`
	HasPrep      bool                  `json:"has_prep"`
	AgentMessage string                `json:"agent_message"`
	ActionID     *uuid.UUID            `json:"action_id,omitempty"`
}

// SSEFollowupDueEvent is sent when a follow-up reaches its cadence deadline.
type SSEFollowupDueEvent struct {
	EventType      string                `json:"event_type"`
	EventID        string                `json:"event_id"`
	Timestamp      time.Time             `json:"timestamp"`
	Contact        ActionContactResponse `json:"contact"`
	FollowupNumber int                   `json:"followup_number"`
	DaysSince      int                   `json:"days_since"`
	IsFinal        bool                  `json:"is_final"`
	AgentMessage   string                `json:"agent_message"`
	ActionID       *uuid.UUID            `json:"action_id,omitempty"`
}

// SSESnoozeExpiredEvent is sent when snoozed actions reappear.
type SSESnoozeExpiredEvent struct {
	EventType    string                `json:"event_type"`
	EventID      string                `json:"event_id"`
	Timestamp    time.Time             `json:"timestamp"`
	Actions      []DailyActionResponse `json:"actions"`
	AgentMessage string                `json:"agent_message"`
}

// SSEGenerationCompleteEvent is sent when async generation finishes.
type SSEGenerationCompleteEvent struct {
	EventType    string    `json:"event_type"`
	EventID      string    `json:"event_id"`
	Timestamp    time.Time `json:"timestamp"`
	GenerationID uuid.UUID `json:"generation_id"`
	ActionCount  int       `json:"action_count"`
}

// SSEActionUpdatedEvent is sent when an action is updated from another session.
type SSEActionUpdatedEvent struct {
	EventType string    `json:"event_type"`
	EventID   string    `json:"event_id"`
	Timestamp time.Time `json:"timestamp"`
	ActionID  uuid.UUID `json:"action_id"`
	NewStatus string    `json:"new_status"`
	UpdatedBy string    `json:"updated_by"`
}
