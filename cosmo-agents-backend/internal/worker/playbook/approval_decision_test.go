package playbook

import (
	"testing"

	playbookDomain "github.com/rockship/cosmo-agents-go/internal/domain/playbook"
)

func TestShouldRequestApproval(t *testing.T) {
	req := func(status string, fit int) *playbookDomain.EnrollmentApprovalRequest {
		return &playbookDomain.EnrollmentApprovalRequest{Status: status, FitScore: fit}
	}

	tests := []struct {
		name     string
		latest   *playbookDomain.EnrollmentApprovalRequest
		fitScore int
		want     bool
	}{
		{"never proposed", nil, 70, true},
		{"still pending is not duplicated", req("pending", 70), 90, false},
		// The rule re-runs every few minutes; a rejection must survive the next run.
		{"rejected with the same score stays rejected", req("rejected", 70), 70, false},
		{"rejected with a lower score stays rejected", req("rejected", 70), 60, false},
		{"rejected but the score has since risen is reconsidered", req("rejected", 70), 71, true},
		{"approved defers to the enrollment check", req("approved", 70), 70, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRequestApproval(tt.latest, tt.fitScore); got != tt.want {
				t.Errorf("shouldRequestApproval(%v, %d) = %v, want %v", tt.latest, tt.fitScore, got, tt.want)
			}
		})
	}
}
