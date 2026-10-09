package daily_action

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// DailyActionGeneration tracks each generation run.
// Created by POST /v1/daily-actions/generate, referenced by GET /v1/daily-actions.
type DailyActionGeneration struct {
	base.Base
	base.TimestampMixin
	base.SoftDeleteMixin

	UserID       uuid.UUID        `gorm:"type:uuid;not null;index:idx_gen_user_date" json:"user_id"`
	Date         string           `gorm:"type:date;not null;index:idx_gen_user_date" json:"date"`
	Status       GenerationStatus `gorm:"type:varchar(20);not null;default:'started'" json:"status"`
	Language     string           `gorm:"type:varchar(10);default:'vi'" json:"language"`
	ActionCount  int              `gorm:"default:0" json:"action_count"`
	GeneratedAt  *time.Time       `json:"generated_at"`
	ReplacedByID *uuid.UUID       `gorm:"type:uuid" json:"replaced_by_id"`

	AgentBriefing   base.JSONB `gorm:"type:jsonb;default:'{}'" json:"agent_briefing"`
	PipelineSummary base.JSONB `gorm:"type:jsonb;default:'{}'" json:"pipeline_summary"`
}

func (DailyActionGeneration) TableName() string {
	return "daily_action_generations"
}

// DailyAction is a persisted action suggestion with UUID.
// Created during generation, updated via PATCH /v1/daily-actions/{action_id}.
type DailyAction struct {
	base.Base
	base.TimestampMixin
	base.SoftDeleteMixin

	UserID       uuid.UUID  `gorm:"type:uuid;not null;index:idx_action_user_gen" json:"user_id"`
	GenerationID uuid.UUID  `gorm:"type:uuid;not null;index:idx_action_user_gen" json:"generation_id"`
	ContactID    uuid.UUID  `gorm:"type:uuid;not null;index:idx_action_contact" json:"contact_id"`
	Type         ActionType `gorm:"type:varchar(20);not null;index:idx_action_category" json:"type"`
	CategoryID   CategoryID `gorm:"type:varchar(20);not null;index:idx_action_category" json:"category_id"`

	Priority        int        `gorm:"not null;index:idx_action_priority" json:"priority"`
	PriorityFactors base.JSONB `gorm:"type:jsonb;default:'[]'" json:"priority_factors"`

	Reasoning string `gorm:"type:text" json:"reasoning"`

	Status          ActionStatus `gorm:"type:varchar(20);not null;default:'suggested'" json:"status"`
	StatusChangedAt *time.Time   `json:"status_changed_at"`
	SnoozeUntil     *time.Time   `json:"snooze_until"`

	OutreachData   base.JSONB `gorm:"type:jsonb" json:"outreach_data,omitempty"`
	MeetingData    base.JSONB `gorm:"type:jsonb" json:"meeting_data,omitempty"`
	EnrichmentData base.JSONB `gorm:"type:jsonb" json:"enrichment_data,omitempty"`
	RespondData    base.JSONB `gorm:"type:jsonb" json:"respond_data,omitempty"`

	ContactSnapshot base.JSONB `gorm:"type:jsonb;not null;column:contact" json:"contact"`
}

func (DailyAction) TableName() string {
	return "daily_actions"
}

// ActionSnooze tracks snooze records for actions.
type ActionSnooze struct {
	base.Base
	base.TimestampMixin

	UserID      uuid.UUID  `gorm:"type:uuid;not null;index:idx_snooze_user" json:"user_id"`
	ActionID    uuid.UUID  `gorm:"type:uuid;not null;index:idx_snooze_action" json:"action_id"`
	SnoozeUntil time.Time  `gorm:"not null" json:"snooze_until"`
	ClearedAt   *time.Time `json:"cleared_at"`
}

func (ActionSnooze) TableName() string {
	return "action_snoozes"
}

// ActionCompletionLog is an append-only audit log of action state changes.
type ActionCompletionLog struct {
	base.Base
	base.TimestampMixin

	UserID      uuid.UUID  `gorm:"type:uuid;not null;index:idx_log_user_date" json:"user_id"`
	ActionID    uuid.UUID  `gorm:"type:uuid;not null" json:"action_id"`
	ActionType  ActionType `gorm:"type:varchar(20);not null" json:"action_type"`
	Transition  string     `gorm:"type:varchar(20);not null" json:"transition"`
	ContactID   uuid.UUID  `gorm:"type:uuid;not null" json:"contact_id"`
	ContactName string     `gorm:"type:varchar(255)" json:"contact_name"`
	Content     *string    `gorm:"type:text" json:"content,omitempty"`
	Channel     *string    `gorm:"type:varchar(50)" json:"channel,omitempty"`
	SkipReason  *string    `gorm:"type:text" json:"skip_reason,omitempty"`
	Feedback    *string    `gorm:"type:varchar(50)" json:"feedback,omitempty"`
	Date        string     `gorm:"type:date;not null;index:idx_log_user_date" json:"date"`
}

func (ActionCompletionLog) TableName() string {
	return "action_completion_logs"
}

// SSEEvent is a transient event record for Last-Event-ID replay support.
type SSEEvent struct {
	base.Base
	base.TimestampMixin

	UserID    uuid.UUID  `gorm:"type:uuid;not null;index:idx_sse_user_time" json:"user_id"`
	EventType string     `gorm:"type:varchar(30);not null" json:"event_type"`
	Payload   base.JSONB `gorm:"type:jsonb;not null" json:"payload"`
}

func (SSEEvent) TableName() string {
	return "sse_events"
}

// ChatSession groups an Ask COSMO conversation.
//
// The title is derived from the first question rather than asked for: a user
// opening a chat widget wants to type a question, not name a thread.
type ChatSession struct {
	base.Base
	base.TimestampMixin

	UserID uuid.UUID `gorm:"type:uuid;not null;index:idx_chat_sessions_user" json:"user_id"`
	Title  string    `gorm:"type:text;not null;default:''" json:"title"`
}

func (ChatSession) TableName() string {
	return "chat_sessions"
}

// ChatMessage stores conversational request/response pairs for the BD Agent chat.
type ChatMessage struct {
	base.Base
	base.TimestampMixin

	UserID           uuid.UUID  `gorm:"type:uuid;not null;index:idx_chat_user" json:"user_id"`
	Role             string     `gorm:"type:varchar(20);not null" json:"role"`
	Content          string     `gorm:"type:text;not null" json:"content"`
	ClassifiedIntent *string    `gorm:"type:varchar(50)" json:"classified_intent,omitempty"`
	GenerationID     *uuid.UUID `gorm:"type:uuid" json:"generation_id,omitempty"`

	// Nil for the messages written before sessions existed.
	SessionID *uuid.UUID `gorm:"type:uuid;index:idx_chat_messages_session_id" json:"session_id,omitempty"`
}

func (ChatMessage) TableName() string {
	return "chat_messages"
}

// ContactSnapshot represents the denormalized contact data embedded in each action.
type ContactSnapshot struct {
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

// PriorityFactor represents a single factor contributing to the priority score.
type PriorityFactor struct {
	Factor      string `json:"factor"`
	Value       int    `json:"value"`
	Description string `json:"description"`
}

// AgentBriefingData represents the AI-generated briefing stored as JSONB.
type AgentBriefingData struct {
	Greeting           string            `json:"greeting"`
	StrategicReasoning string            `json:"strategic_reasoning"`
	MemoryReferences   []MemoryReference `json:"memory_references"`
	CategoryCounts     []CategoryCount   `json:"category_counts"`
}

// MemoryReference is a conversation memory citation used in the agent briefing.
type MemoryReference struct {
	ContactID      string `json:"contact_id"`
	ContactName    string `json:"contact_name"`
	EventSummary   string `json:"event_summary"`
	EventTimestamp string `json:"event_timestamp"`
	Relevance      string `json:"relevance"`
}

// CategoryCount represents a category summary badge in the briefing header.
type CategoryCount struct {
	Category string `json:"category"`
	Label    string `json:"label"`
	Count    int    `json:"count"`
	Icon     string `json:"icon"`
	Color    string `json:"color"`
}

// OutreachActionData holds type-specific data for outreach/followup actions.
type OutreachActionData struct {
	DraftMessage             string                 `json:"draft_message"`
	Scenario                 string                 `json:"scenario"`
	ContextLevel             string                 `json:"context_level"`
	CompanyContext           string                 `json:"company_context,omitempty"`
	FollowupNumber           *int                   `json:"followup_number,omitempty"`
	DaysSinceLastInteraction int                    `json:"days_since_last_interaction"`
	PreviousMessagesCount    int                    `json:"previous_messages_count"`
	LastSentDate             *string                `json:"last_sent_date,omitempty"`
	IsFinalFollowup          bool                   `json:"is_final_followup"`
	OutreachState            *OutreachStateSnapshot `json:"outreach_state,omitempty"`
}

// OutreachStateSnapshot captures the contact's outreach state at generation time.
type OutreachStateSnapshot struct {
	ConversationState string `json:"conversation_state"`
	NextStep          string `json:"next_step"`
	FollowupCount     int    `json:"followup_count"`
	MaxFollowups      int    `json:"max_followups"`
}

// RespondActionData holds type-specific data for respond (replied) actions.
type RespondActionData struct {
	ReplyPreview        string               `json:"reply_preview"`
	ReplyTimestamp      string               `json:"reply_timestamp"`
	ReplyChannel        string               `json:"reply_channel"`
	IntentAssessment    string               `json:"intent_assessment"`
	IntentReasoning     string               `json:"intent_reasoning"`
	RecommendedAction   string               `json:"recommended_action"`
	DraftResponse       string               `json:"draft_response"`
	ConversationContext *ConversationContext `json:"conversation_context,omitempty"`
}

// ConversationContext provides context about the conversation history.
type ConversationContext struct {
	TotalInteractions          int      `json:"total_interactions"`
	DaysInConversation         int      `json:"days_in_conversation"`
	LastOutgoingMessagePreview string   `json:"last_outgoing_message_preview,omitempty"`
	KeyTopicsDiscussed         []string `json:"key_topics_discussed,omitempty"`
}

// MeetingActionData holds type-specific data for meeting_prep actions.
type MeetingActionData struct {
	MeetingID              string           `json:"meeting_id"`
	MeetingTitle           string           `json:"meeting_title"`
	MeetingTime            string           `json:"meeting_time"`
	MeetingDurationMinutes int              `json:"meeting_duration_minutes"`
	MeetingChannel         string           `json:"meeting_channel,omitempty"`
	HoursUntilMeeting      float64          `json:"hours_until_meeting"`
	Briefing               *MeetingBriefing `json:"briefing,omitempty"`
}

// MeetingBriefing holds the pre-generated meeting preparation document.
type MeetingBriefing struct {
	ProspectProfileSummary string               `json:"prospect_profile_summary"`
	ConversationSummary    *ConversationSummary `json:"conversation_summary,omitempty"`
	PainPoints             []PainPoint          `json:"pain_points,omitempty"`
	SuggestedAgenda        []AgendaItem         `json:"suggested_agenda,omitempty"`
	DiscoveryQuestions     []string             `json:"discovery_questions,omitempty"`
	RecommendedNextSteps   []string             `json:"recommended_next_steps,omitempty"`
	RiskFlags              []string             `json:"risk_flags,omitempty"`
}

// ConversationSummary summarizes previous conversation touchpoints.
type ConversationSummary struct {
	TouchpointCount int      `json:"touchpoint_count"`
	DurationDays    int      `json:"duration_days"`
	ToneAssessment  string   `json:"tone_assessment"`
	KeyTopics       []string `json:"key_topics,omitempty"`
}

// PainPoint represents an identified pain point with confidence.
type PainPoint struct {
	PainPoint  string   `json:"pain_point"`
	Confidence float64  `json:"confidence"`
	Evidence   []string `json:"evidence,omitempty"`
}

// AgendaItem represents a suggested meeting agenda topic.
type AgendaItem struct {
	Topic           string `json:"topic"`
	DurationMinutes int    `json:"duration_minutes"`
	Notes           string `json:"notes,omitempty"`
}

// EnrichmentActionData holds type-specific data for enrich actions.
type EnrichmentActionData struct {
	MissingFields    []string          `json:"missing_fields"`
	QualityImpact    string            `json:"quality_impact"`
	ContactStatus    string            `json:"contact_status"`
	SuggestedSources []SuggestedSource `json:"suggested_sources,omitempty"`
}

// SuggestedSource suggests where to find missing contact data.
type SuggestedSource struct {
	Field  string `json:"field"`
	Source string `json:"source"`
	URL    string `json:"url,omitempty"`
}
