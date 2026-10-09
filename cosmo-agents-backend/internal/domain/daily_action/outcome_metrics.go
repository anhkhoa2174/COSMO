package daily_action

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"gorm.io/gorm"
)

// OutcomeMetrics stores pre-computed daily aggregated performance data.
type OutcomeMetrics struct {
	ID                      uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID                  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_outcome_metrics_user_period" json:"user_id"`
	ComputedAt              time.Time `gorm:"not null;default:now()" json:"computed_at"`
	Period                  string    `gorm:"type:varchar(10);not null;default:'30d';uniqueIndex:idx_outcome_metrics_user_period" json:"period"`
	TotalSent               int       `gorm:"not null;default:0" json:"total_sent"`
	TotalReplied            int       `gorm:"not null;default:0" json:"total_replied"`
	TotalMeetings           int       `gorm:"not null;default:0" json:"total_meetings"`
	ReplyRateOverall        float64   `gorm:"not null;default:0" json:"reply_rate_overall"`
	ReplyRateByChannel      base.JSONB `gorm:"type:jsonb;default:'{}'" json:"reply_rate_by_channel"`
	ReplyRateByStrategy     base.JSONB `gorm:"type:jsonb;default:'{}'" json:"reply_rate_by_strategy"`
	ReplyRateByIndustry     base.JSONB `gorm:"type:jsonb;default:'{}'" json:"reply_rate_by_industry"`
	ReplyRateByTimeOfDay    base.JSONB `gorm:"type:jsonb;default:'{}'" json:"reply_rate_by_time_of_day"`
	AvgMessagesToMeeting    float64   `gorm:"default:0" json:"avg_messages_to_meeting"`
	TopPerformingStrategies base.JSONB `gorm:"type:jsonb;default:'[]'" json:"top_performing_strategies"`
	CreatedAt               time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (OutcomeMetrics) TableName() string { return "outcome_metrics" }

func (m *OutcomeMetrics) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}

// AgentRecommendation represents the AI's output per contact (used in create-from-agent endpoint).
type AgentRecommendation struct {
	ContactID           uuid.UUID `json:"contact_id"`
	PriorityRank        int       `json:"priority_rank"`
	ActionType          string    `json:"action_type"`
	CategoryID          string    `json:"category_id"`
	RecommendedChannel  string    `json:"recommended_channel"`
	MessagingStrategy   string    `json:"messaging_strategy"`
	StrategicReasoning  string    `json:"strategic_reasoning"`
	DraftMessage        string    `json:"draft_message"`
	ReferencedKnowledge []string  `json:"referenced_knowledge"`
	ConfidenceLevel     string    `json:"confidence_level"`
	OutcomePatternCited string    `json:"outcome_pattern_cited"`
}

// CreateFromAgentRequest is the request body for POST /v1/daily-actions/create-from-agent.
type CreateFromAgentRequest struct {
	Language        string                `json:"language"`
	Recommendations []AgentRecommendation `json:"recommendations"`
	StrategicPlan   string                `json:"strategic_plan"`
	FocusAreas      []string              `json:"focus_areas"`
	OutcomeInsights []string              `json:"outcome_insights"`
}
