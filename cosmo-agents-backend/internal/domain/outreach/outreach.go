package outreach

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"gorm.io/gorm"
)

// ============================================
// Enums and Constants
// ============================================

// InteractionChannel defines the channel of interaction
type InteractionChannel string

const (
	ChannelLinkedIn InteractionChannel = "LinkedIn"
	ChannelEmail    InteractionChannel = "Email"
	ChannelCall     InteractionChannel = "Call"
	ChannelMeeting  InteractionChannel = "Meeting"
	ChannelNote     InteractionChannel = "Note"
)

// InteractionDirection defines the direction of interaction
type InteractionDirection string

const (
	DirectionOutgoing InteractionDirection = "outgoing"
	DirectionIncoming InteractionDirection = "incoming"
	DirectionInternal InteractionDirection = "internal"
)

// Sentiment defines the sentiment of an interaction
type Sentiment string

const (
	SentimentPositive Sentiment = "positive"
	SentimentNeutral  Sentiment = "neutral"
	SentimentNegative Sentiment = "negative"
)

// ConversationState defines the state of the conversation with a contact
type ConversationState string

const (
	StateCold        ConversationState = "COLD"
	StateNoReply     ConversationState = "NO_REPLY"
	StateReplied     ConversationState = "REPLIED"
	StatePostMeeting ConversationState = "POST_MEETING"
	StateDropped     ConversationState = "DROPPED"
)

// ContextLevel defines how much context we have
type ContextLevel string

const (
	ContextLow    ContextLevel = "LOW"
	ContextMedium ContextLevel = "MEDIUM"
	ContextHigh   ContextLevel = "HIGH"
)

// OutreachIntent defines the intent of outreach
type OutreachIntent string

const (
	IntentIntro       OutreachIntent = "INTRO"
	IntentFollowUp    OutreachIntent = "FOLLOW_UP"
	IntentReEngage    OutreachIntent = "RE_ENGAGE"
	IntentPostMeeting OutreachIntent = "POST_MEETING"
)

// Scenario defines the message scenario
type Scenario string

const (
	ScenarioRoleBased           Scenario = "role_based"
	ScenarioIndustryBased       Scenario = "industry_based"
	ScenarioNoReplyFollowup     Scenario = "no_reply_followup"
	ScenarioPostReply           Scenario = "post_reply"
	ScenarioMeetingConfirmation Scenario = "meeting_confirmation" // For confirming meeting time
	ScenarioPostMeeting         Scenario = "post_meeting"         // For follow-up after meeting done
	ScenarioReEngage            Scenario = "re_engage"
)

// LastOutcome defines the outcome of the last interaction
type LastOutcome string

const (
	OutcomeNone          LastOutcome = "none"
	OutcomeSent          LastOutcome = "sent"
	OutcomeNoReply       LastOutcome = "no_reply"
	OutcomeReplied       LastOutcome = "replied"
	OutcomeMeetingBooked LastOutcome = "meeting_booked"
	OutcomeMeetingDone   LastOutcome = "meeting_done"
	OutcomeDropped       LastOutcome = "dropped"
)

// NextStepAction defines the recommended next action
type NextStepAction string

const (
	// Pre-sales / Outreach
	NextStepSend             NextStepAction = "SEND"                // Send initial message
	NextStepFollowUp1        NextStepAction = "FOLLOW_UP_1"         // Follow-up #1 (Day 4-5)
	NextStepFollowUp2        NextStepAction = "FOLLOW_UP_2"         // Follow-up #2 (Day 9-12)
	NextStepSetMeeting       NextStepAction = "SET_MEETING"         // Propose meeting
	NextStepFollowUpMeeting1 NextStepAction = "FOLLOW_UP_MEETING_1" // Meeting confirmation follow-up #1
	NextStepFollowUpMeeting2 NextStepAction = "FOLLOW_UP_MEETING_2" // Meeting confirmation follow-up #2
	NextStepPrepareMeeting   NextStepAction = "PREPARE_MEETING"     // Prepare meeting materials
	NextStepWait             NextStepAction = "WAIT"                // Wait for response / wait for meeting day

	// Sales / Post-meeting
	NextStepFollowUp NextStepAction = "FOLLOW_UP" // Follow-up deal / proposal

	// End states
	NextStepDrop NextStepAction = "DROP" // Drop contact
)

// MeetingChannel defines the meeting channel
type MeetingChannel string

const (
	MeetingChannelZoom       MeetingChannel = "Zoom"
	MeetingChannelGoogleMeet MeetingChannel = "Google Meet"
	MeetingChannelCall       MeetingChannel = "Call"
	MeetingChannelOffline    MeetingChannel = "Offline"
)

// MeetingStatus defines the status of a meeting
type MeetingStatus string

const (
	MeetingScheduled MeetingStatus = "scheduled"
	MeetingCompleted MeetingStatus = "completed"
	MeetingCancelled MeetingStatus = "cancelled"
	MeetingNoShow    MeetingStatus = "no_show"
)

// ============================================
// InteractionLog Model
// ============================================

// InteractionLog stores all messages sent/received and interaction notes
type InteractionLog struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_interaction_logs_user_id" json:"user_id"`
	ContactID uuid.UUID `gorm:"type:uuid;not null;index:idx_interaction_logs_contact_id" json:"contact_id"`

	// Interaction details
	Channel   string `gorm:"type:varchar(50);not null;default:'LinkedIn'" json:"channel"`
	Direction string `gorm:"type:varchar(20);not null;default:'outgoing'" json:"direction"`
	Content   string `gorm:"type:text;not null;default:''" json:"content"`

	// Metadata
	Subject     *string    `gorm:"type:varchar(500)" json:"subject,omitempty"`
	URL         *string    `gorm:"type:varchar(1000)" json:"url,omitempty"`
	Attachments base.JSONB `gorm:"type:jsonb;default:'[]'" json:"attachments"`

	// Sentiment analysis
	Sentiment *string `gorm:"type:varchar(20)" json:"sentiment,omitempty"`

	// Timestamps
	Timestamp time.Time `gorm:"not null;default:now();index:idx_interaction_logs_timestamp" json:"timestamp"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name
func (InteractionLog) TableName() string {
	return "interaction_logs"
}

// BeforeCreate hook to set UUID before creating record
func (i *InteractionLog) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	if i.Timestamp.IsZero() {
		i.Timestamp = time.Now()
	}
	return nil
}

// ============================================
// OutreachState Model
// ============================================

// OutreachState is the snapshot state for CLI display and flow control
type OutreachState struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_outreach_states_user_id" json:"user_id"`
	ContactID uuid.UUID `gorm:"type:uuid;not null;index:idx_outreach_states_contact_id;uniqueIndex:idx_outreach_state_unique" json:"contact_id"`

	// Conversation state machine
	ConversationState string `gorm:"type:varchar(30);not null;default:'COLD';index" json:"conversation_state"`

	// Context assessment
	ContextLevel string `gorm:"type:varchar(20);not null;default:'LOW'" json:"context_level"`

	// Outreach intent
	OutreachIntent string `gorm:"type:varchar(30);not null;default:'INTRO'" json:"outreach_intent"`

	// Scenario for message generation
	Scenario string `gorm:"type:varchar(50);not null;default:'role_based'" json:"scenario"`

	// Draft message
	MessageDraft *string `gorm:"type:text" json:"message_draft,omitempty"`

	// Last outcome tracking
	LastOutcome string `gorm:"type:varchar(30);not null;default:'none';index" json:"last_outcome"`

	// Next step recommendation
	NextStep string `gorm:"type:varchar(30);not null;default:'SEND';index" json:"next_step"`

	// Time tracking
	LastInteractionAt        *time.Time `json:"last_interaction_at,omitempty"`
	DaysSinceLastInteraction int        `gorm:"default:0" json:"days_since_last_interaction"`

	// Follow-up tracking
	FollowupCount int `gorm:"not null;default:0" json:"followup_count"`
	MaxFollowups  int `gorm:"not null;default:2" json:"max_followups"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name
func (OutreachState) TableName() string {
	return "outreach_states"
}

// BeforeCreate hook to set UUID before creating record
func (o *OutreachState) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

// CalculateDaysSinceLastInteraction calculates days since last interaction
func (o *OutreachState) CalculateDaysSinceLastInteraction() {
	if o.LastInteractionAt == nil {
		o.DaysSinceLastInteraction = 0
		return
	}
	duration := time.Since(*o.LastInteractionAt)
	o.DaysSinceLastInteraction = int(duration.Hours() / 24)
}

// ShouldDrop checks if contact should be dropped based on followup count
func (o *OutreachState) ShouldDrop() bool {
	return o.FollowupCount >= o.MaxFollowups && o.LastOutcome == string(OutcomeNoReply)
}

// ============================================
// Meeting Model
// ============================================

// Meeting tracks meetings with contacts
type Meeting struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_meetings_user_id" json:"user_id"`
	ContactID uuid.UUID `gorm:"type:uuid;not null;index:idx_meetings_contact_id" json:"contact_id"`

	// Meeting details
	Title           *string   `gorm:"type:varchar(500)" json:"title,omitempty"`
	Time            time.Time `gorm:"not null;index:idx_meetings_time" json:"time"`
	DurationMinutes int       `gorm:"default:30" json:"duration_minutes"`
	Channel         string    `gorm:"type:varchar(50);not null;default:'Zoom'" json:"channel"`
	Location        *string   `gorm:"type:varchar(500)" json:"location,omitempty"`
	MeetingURL      *string   `gorm:"type:varchar(1000)" json:"meeting_url,omitempty"`

	// Status tracking
	Status string `gorm:"type:varchar(30);not null;default:'scheduled';index" json:"status"`

	// Notes and outcomes
	Note      *string `gorm:"type:text" json:"note,omitempty"`
	Outcome   *string `gorm:"type:text" json:"outcome,omitempty"`
	NextSteps *string `gorm:"type:text" json:"next_steps,omitempty"`

	// Meeting content (transcript, notes, etc.)
	MeetingContent *string `gorm:"type:text" json:"meeting_content,omitempty"`

	// AI-generated meeting prep (talking points, discovery questions - generated BEFORE meeting)
	MeetingPrep *string `gorm:"type:text" json:"meeting_prep,omitempty"`

	// Participants
	Participants base.JSONB `gorm:"type:jsonb;default:'[]'" json:"participants"`

	// Timestamps
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name
func (Meeting) TableName() string {
	return "meetings"
}

// BeforeCreate hook to set UUID before creating record
func (m *Meeting) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// IsCompleted checks if meeting is completed
func (m *Meeting) IsCompleted() bool {
	return m.Status == string(MeetingCompleted)
}

// IsPending checks if meeting is still scheduled
func (m *Meeting) IsPending() bool {
	return m.Status == string(MeetingScheduled)
}

// ============================================
// Feedback Types for Task 8
// ============================================

// FeedbackAction defines what BD actually did with the draft
type FeedbackAction string

const (
	ActionUsedDraft     FeedbackAction = "used_draft"     // BD used the draft as-is
	ActionModifiedDraft FeedbackAction = "modified_draft" // BD modified the draft
	ActionWroteOwn      FeedbackAction = "wrote_own"      // BD wrote their own message
	ActionSkipped       FeedbackAction = "skipped"        // BD skipped this contact
)

// FeedbackOutcome defines the outcome of the outreach
type FeedbackOutcome string

const (
	FeedbackOutcomeNoAction      FeedbackOutcome = "no_action"
	FeedbackOutcomeSent          FeedbackOutcome = "sent"
	FeedbackOutcomeReplied       FeedbackOutcome = "replied"
	FeedbackOutcomeMeetingBooked FeedbackOutcome = "meeting_booked"
	FeedbackOutcomeMeetingDone   FeedbackOutcome = "meeting_done"
	FeedbackOutcomeDropped       FeedbackOutcome = "dropped"
)

// ============================================
// OutreachFeedback Model
// ============================================

// OutreachFeedback tracks AI suggestions vs BD actions and outcomes
type OutreachFeedback struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index:idx_outreach_feedback_user" json:"user_id"`
	ContactID uuid.UUID `gorm:"type:uuid;not null;index:idx_outreach_feedback_contact" json:"contact_id"`

	// What AI suggested
	SuggestedScenario string  `gorm:"type:varchar(50);not null" json:"suggested_scenario"`
	SuggestedIntent   string  `gorm:"type:varchar(50);not null" json:"suggested_intent"`
	SuggestedDraft    *string `gorm:"type:text" json:"suggested_draft,omitempty"`
	SuggestedNextStep *string `gorm:"type:varchar(50)" json:"suggested_next_step,omitempty"`

	// What BD actually did
	ActualAction  *string `gorm:"type:varchar(50)" json:"actual_action,omitempty"`
	ActualContent *string `gorm:"type:text" json:"actual_content,omitempty"`

	// Outcome tracking
	Outcome        *string `gorm:"type:varchar(50);index:idx_outreach_feedback_outcome" json:"outcome,omitempty"`
	ReplySentiment *string `gorm:"type:varchar(20)" json:"reply_sentiment,omitempty"`
	DaysToReply    *int    `json:"days_to_reply,omitempty"`

	// Metadata
	ContextLevel      string `gorm:"type:varchar(20);not null" json:"context_level"`
	ConversationState string `gorm:"type:varchar(50);not null" json:"conversation_state"`

	// Timestamps
	CreatedAt        time.Time  `gorm:"autoCreateTime;index:idx_outreach_feedback_created" json:"created_at"`
	OutcomeUpdatedAt *time.Time `json:"outcome_updated_at,omitempty"`
}

// TableName specifies the table name
func (OutreachFeedback) TableName() string {
	return "outreach_feedback"
}

// BeforeCreate hook to set UUID before creating record
func (f *OutreachFeedback) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}

// ============================================
// Scenario Statistics (for analytics)
// ============================================

// ScenarioStats holds statistics for a scenario
type ScenarioStats struct {
	Scenario       string  `json:"scenario"`
	ContextLevel   string  `json:"context_level"`
	TotalSuggested int     `json:"total_suggested"`
	DraftsUsed     int     `json:"drafts_used"`
	DraftsModified int     `json:"drafts_modified"`
	WroteOwn       int     `json:"wrote_own"`
	Skipped        int     `json:"skipped"`
	TotalSent      int     `json:"total_sent"`
	TotalReplied   int     `json:"total_replied"`
	TotalMeetings  int     `json:"total_meetings"`
	ReplyRate      float64 `json:"reply_rate"`       // replied/sent * 100
	MeetingRate    float64 `json:"meeting_rate"`     // meetings/replied * 100
	DraftUsageRate float64 `json:"draft_usage_rate"` // (used+modified)/total * 100
	AvgDaysToReply float64 `json:"avg_days_to_reply"`
}
