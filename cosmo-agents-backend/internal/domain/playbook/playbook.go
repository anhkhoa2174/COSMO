package playbook

import (
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
)

// Playbook domain model
type Playbook struct {
	PlaybookID   uuid.UUID  `db:"playbook_id"`
	Name         string     `db:"name"`
	Description  string     `db:"description"`
	PlaybookType string     `db:"playbook_type"` // "nurture", "outreach", "re_engagement", "upsell"
	Config       base.JSONB `db:"config"`        // PlaybookConfig JSON
	Performance  base.JSONB `db:"performance"`   // PlaybookPerformance JSON
	IsActive     bool       `db:"is_active"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

// AutomationRule domain model
type AutomationRule struct {
	AutomationRuleID   uuid.UUID  `db:"automation_rule_id"`
	Name               string     `db:"name"`
	SegmentID          uuid.UUID  `db:"segment_id"`
	PlaybookID         uuid.UUID  `db:"playbook_id"`
	EnrollmentCriteria base.JSONB `db:"enrollment_criteria"` // EnrollmentCriteria JSON
	IsActive           bool       `db:"is_active"`
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
}

// ContactEnrollment tracks contact progress through playbook
type ContactEnrollment struct {
	EnrollmentID      uuid.UUID  `db:"enrollment_id"`
	ContactID         uuid.UUID  `db:"contact_id"`
	PlaybookID        uuid.UUID  `db:"playbook_id"`
	AutomationRuleID  *uuid.UUID `db:"automation_rule_id"`
	EnrollmentStatus  string     `db:"enrollment_status"` // "pending_approval", "active", "paused", "completed"
	CurrentStageOrder int        `db:"current_stage_order"`
	CurrentStageID    string     `db:"current_stage_id"`
	EnrolledAt        *time.Time `db:"enrolled_at"`
	CompletedAt       *time.Time `db:"completed_at"`
	ExecutionLog      base.JSONB `db:"execution_log"` // Stage execution history
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
}

// EnrollmentApprovalRequest for human-in-the-loop approval
type EnrollmentApprovalRequest struct {
	RequestID        uuid.UUID  `db:"request_id"`
	ContactID        uuid.UUID  `db:"contact_id"`
	PlaybookID       uuid.UUID  `db:"playbook_id"`
	AutomationRuleID uuid.UUID  `db:"automation_rule_id"`
	Reason           string     `db:"reason"`
	FitScore         int        `db:"fit_score"`
	EngagementScore  int        `db:"engagement_score"`
	Status           string     `db:"status"` // "pending", "approved", "rejected"
	ReviewedBy       *uuid.UUID `db:"reviewed_by"`
	ReviewedAt       *time.Time `db:"reviewed_at"`
	CreatedAt        time.Time  `db:"created_at"`
}

// TableName returns the table name for Playbook
func (Playbook) TableName() string {
	return "playbooks"
}

// TableName returns the table name for AutomationRule
func (AutomationRule) TableName() string {
	return "automation_rules"
}

// TableName returns the table name for ContactEnrollment
func (ContactEnrollment) TableName() string {
	return "contact_enrollments"
}

// TableName returns the table name for EnrollmentApprovalRequest
func (EnrollmentApprovalRequest) TableName() string {
	return "enrollment_approval_requests"
}
