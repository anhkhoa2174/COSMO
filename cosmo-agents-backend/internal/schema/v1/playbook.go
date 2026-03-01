package v1

import (
	"time"

	"github.com/google/uuid"
)

// Playbook types
type PlaybookType string

const (
	PlaybookTypeNurture      PlaybookType = "nurture"
	PlaybookTypeOutreach     PlaybookType = "outreach"
	PlaybookTypeReEngagement PlaybookType = "re_engagement"
	PlaybookTypeUpsell       PlaybookType = "upsell"
)

// Stage types
type StageType string

const (
	StageTypeEmail       StageType = "email"
	StageTypeLinkedIn    StageType = "linkedin"
	StageTypeCall        StageType = "call"
	StageTypeWait        StageType = "wait"
	StageTypeConditional StageType = "conditional"
)

// Success criteria actions
type SuccessAction string

const (
	SuccessActionAdvance   SuccessAction = "advance"
	SuccessActionComplete  SuccessAction = "complete"
	SuccessActionNextStage SuccessAction = "next_stage"
	SuccessActionPause     SuccessAction = "pause"
)

// PlaybookStage represents a single stage in a playbook
type PlaybookStage struct {
	ID                string            `json:"id"`
	Order             int               `json:"order"`
	Name              string            `json:"name"`
	Type              StageType         `json:"type"`
	TriggerConditions TriggerConditions `json:"trigger_conditions"`
	ContentConfig     *ContentConfig    `json:"content_config,omitempty"`
	SuccessCriteria   SuccessCriteria   `json:"success_criteria"`
}

// TriggerConditions defines when a stage should execute
type TriggerConditions struct {
	WaitDuration int     `json:"wait_duration"` // days
	WaitForEvent *string `json:"wait_for_event,omitempty"`
}

// ContentConfig for email/linkedin stages
type ContentConfig struct {
	Template           string `json:"template,omitempty"`
	AIGenerationPrompt string `json:"ai_generation_prompt,omitempty"`
}

// SuccessCriteria defines what happens on different outcomes
type SuccessCriteria struct {
	OnReply   SuccessAction `json:"on_reply"`
	OnTimeout SuccessAction `json:"on_timeout"`
}

// PlaybookConfig stores the full playbook configuration
type PlaybookConfig struct {
	Stages            []PlaybookStage        `json:"stages"`
	MessagingStrategy map[string]interface{} `json:"messaging_strategy,omitempty"`
	TimingRules       map[string]interface{} `json:"timing_rules,omitempty"`
	ChannelSequence   []string               `json:"channel_sequence,omitempty"`
}

// PlaybookPerformance tracks playbook metrics
type PlaybookPerformance struct {
	ContactsEnrolled int     `json:"contacts_enrolled"`
	TotalSent        int     `json:"total_sent"`
	TotalReplies     int     `json:"total_replies"`
	TotalMeetings    int     `json:"total_meetings"`
	ReplyRate        float64 `json:"reply_rate"`
	MeetingRate      float64 `json:"meeting_rate"`
}

// CreatePlaybookRequest for POST /v1/playbooks/create
type CreatePlaybookRequest struct {
	Name         string          `json:"name" validate:"required"`
	Description  string          `json:"description"`
	PlaybookType PlaybookType    `json:"playbook_type" validate:"required"`
	Stages       []PlaybookStage `json:"stages" validate:"required,min=1"`
}

// PlaybookRead for API responses
type PlaybookRead struct {
	PlaybookID   uuid.UUID           `json:"playbook_id"`
	Name         string              `json:"name"`
	Description  string              `json:"description"`
	PlaybookType PlaybookType        `json:"playbook_type"`
	Config       PlaybookConfig      `json:"config"`
	Performance  PlaybookPerformance `json:"performance"`
	IsActive     bool                `json:"is_active"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

// GenerateContentRequest for POST /v1/playbooks/{id}/stages/{stage_id}/generate-content
type GenerateContentRequest struct {
	ContactID          uuid.UUID `json:"contact_id" validate:"required"`
	AIGenerationPrompt string    `json:"ai_generation_prompt"`
}

// GenerateContentResponse with AI-generated content
type GenerateContentResponse struct {
	Subject             string            `json:"subject"`
	Body                string            `json:"body"`
	PersonalizationData map[string]string `json:"personalization_data"`
}

// MeetingBriefRequest for POST /v1/contacts/{id}/generate-meeting-brief
type MeetingBriefRequest struct {
	// Optional: can add filters like "last N interactions"
}

// MeetingBriefInteraction represents a recent interaction
type MeetingBriefInteraction struct {
	Type      string  `json:"type"`
	Date      string  `json:"date"`
	Summary   string  `json:"summary"`
	Sentiment *string `json:"sentiment,omitempty"`
}

// MeetingBriefFact represents confirmed facts by category
type MeetingBriefFact struct {
	Category string   `json:"category"`
	Facts    []string `json:"facts"`
}

// MeetingBriefSegment represents segment fit
type MeetingBriefSegment struct {
	Name     string `json:"name"`
	FitScore int    `json:"fit_score"`
}

// MeetingBriefResponse with all meeting prep data
type MeetingBriefResponse struct {
	LastInteractions   []MeetingBriefInteraction `json:"last_interactions"`
	ConfirmedFacts     []MeetingBriefFact        `json:"confirmed_facts"`
	Segments           []MeetingBriefSegment     `json:"segments"`
	TalkingPoints      []string                  `json:"talking_points"`
	DiscoveryQuestions []string                  `json:"discovery_questions"`
	RiskFlags          []string                  `json:"risk_flags"`
}

// AutomationRuleRequest for POST /v1/automation-rules/create
type AutomationRuleRequest struct {
	Name               string             `json:"name" validate:"required"`
	SegmentID          uuid.UUID          `json:"segment_id" validate:"required"`
	PlaybookID         uuid.UUID          `json:"playbook_id" validate:"required"`
	EnrollmentCriteria EnrollmentCriteria `json:"enrollment_criteria" validate:"required"`
	IsActive           bool               `json:"is_active"`
}

// EnrollmentCriteria defines when to auto-enroll
type EnrollmentCriteria struct {
	FitScoreThreshold        int  `json:"fit_score_threshold" validate:"min=0,max=100"`
	EngagementScoreThreshold *int `json:"engagement_score_threshold,omitempty"`
	RequireHumanApproval     bool `json:"require_human_approval"`
}

// AutomationRuleRead for API responses
type AutomationRuleRead struct {
	AutomationRuleID   uuid.UUID          `json:"automation_rule_id"`
	Name               string             `json:"name"`
	SegmentID          uuid.UUID          `json:"segment_id"`
	SegmentName        string             `json:"segment_name"`
	PlaybookID         uuid.UUID          `json:"playbook_id"`
	PlaybookName       string             `json:"playbook_name"`
	EnrollmentCriteria EnrollmentCriteria `json:"enrollment_criteria"`
	IsActive           bool               `json:"is_active"`
	Stats              AutomationStats    `json:"stats"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// AutomationStats tracks enrollment metrics
type AutomationStats struct {
	ContactsEnrolled        int `json:"contacts_enrolled"`
	ContactsPendingApproval int `json:"contacts_pending_approval"`
	ContactsInProgress      int `json:"contacts_in_progress"`
	ContactsCompleted       int `json:"contacts_completed"`
}

// EnrollContactRequest for POST /v1/contacts/{id}/enroll
type EnrollContactRequest struct {
	PlaybookID uuid.UUID `json:"playbook_id" validate:"required"`
}

// EnrollmentApprovalRequest for POST /v1/contacts/{id}/enrollments/{enrollment_id}/approve
type EnrollmentApprovalRequest struct {
	// Can add notes, etc.
}

// CampaignAnalyticsResponse for POST /v1/campaigns/{id}/analyze
type CampaignAnalyticsResponse struct {
	Metrics            CampaignMetrics      `json:"metrics"`
	SegmentPerformance []SegmentPerformance `json:"segment_performance"`
	Recommendations    []AIRecommendation   `json:"recommendations"`
}

// CampaignMetrics overall campaign performance
type CampaignMetrics struct {
	CampaignName    string  `json:"campaign_name"`
	TotalContacts   int     `json:"total_contacts"`
	EmailsSent      int     `json:"emails_sent"`
	EmailsDelivered int     `json:"emails_delivered"`
	EmailsOpened    int     `json:"emails_opened"`
	EmailsClicked   int     `json:"emails_clicked"`
	EmailsReplied   int     `json:"emails_replied"`
	MeetingsBooked  int     `json:"meetings_booked"`
	OpenRate        float64 `json:"open_rate"`
	ClickRate       float64 `json:"click_rate"`
	ReplyRate       float64 `json:"reply_rate"`
	MeetingRate     float64 `json:"meeting_rate"`
}

// SegmentPerformance breakdown by segment
type SegmentPerformance struct {
	SegmentName string  `json:"segment_name"`
	Contacts    int     `json:"contacts"`
	OpenRate    float64 `json:"open_rate"`
	ReplyRate   float64 `json:"reply_rate"`
	MeetingRate float64 `json:"meeting_rate"`
	ROIScore    int     `json:"roi_score"`
}

// AIRecommendation from campaign intelligence
type AIRecommendation struct {
	Type           string `json:"type"` // "opportunity" | "warning" | "insight"
	Title          string `json:"title"`
	Description    string `json:"description"`
	ExpectedImpact string `json:"expected_impact"`
	Confidence     int    `json:"confidence"`
}
