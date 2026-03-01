package dto

import "github.com/google/uuid"

// EvaluateRulesPayload triggers evaluation for all rules or a specific rule.
type EvaluateRulesPayload struct {
	RuleID *uuid.UUID `json:"rule_id,omitempty"`
}

// ProcessEnrollmentsPayload triggers processing for all enrollments or a specific enrollment.
type ProcessEnrollmentsPayload struct {
	EnrollmentID *uuid.UUID `json:"enrollment_id,omitempty"`
}
