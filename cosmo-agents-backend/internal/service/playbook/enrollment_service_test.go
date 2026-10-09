package playbook

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

func TestEnrollContact(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	svc := f.enrollmentService()

	contact := f.seedContact(t, `{}`)
	pb := f.seedPlaybook(t, twoStages())
	empty := f.seedPlaybook(t, nil)
	deleted := f.seedPlaybook(t, twoStages())
	if err := f.playbooks.Delete(ctx, deleted.PlaybookID); err != nil {
		t.Fatalf("delete playbook: %v", err)
	}
	broken := f.seedPlaybook(t, twoStages())
	f.db.MustExec(`UPDATE playbooks SET config = '[1,2]' WHERE playbook_id = $1`, broken.PlaybookID)

	tests := []struct {
		name      string
		contactID uuid.UUID
		playbook  uuid.UUID
		wantErr   string
	}{
		{"missing contact", uuid.New(), pb.PlaybookID, "contact not found"},
		{"missing playbook", contact.ID, uuid.New(), "playbook not found"},
		{"soft-deleted playbook", contact.ID, deleted.PlaybookID, "playbook not found"},
		{"playbook with no stages", contact.ID, empty.PlaybookID, "playbook has no stages"},
		{"unparseable config", contact.ID, broken.PlaybookID, "failed to parse playbook config"},
		{"first enrollment", contact.ID, pb.PlaybookID, ""},
		// The rule worker and a person can both enroll the same contact.
		{"duplicate enrollment", contact.ID, pb.PlaybookID, "already enrolled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.EnrollContact(ctx, tt.contactID, tt.playbook)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("EnrollContact: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}

	if n := f.countEnrollments(t); n != 1 {
		t.Fatalf("enrollments = %d, want exactly the one successful enrollment", n)
	}
	got, err := f.enrollments.GetByContactAndPlaybook(ctx, contact.ID, pb.PlaybookID)
	if err != nil || got == nil {
		t.Fatalf("GetByContactAndPlaybook: %v, %v", got, err)
	}
	if got.EnrollmentStatus != "active" || got.CurrentStageID != "intro" || got.CurrentStageOrder != 1 {
		t.Errorf("enrollment = %s at %s/%d, want active at intro/1", got.EnrollmentStatus, got.CurrentStageID, got.CurrentStageOrder)
	}
	if got.AutomationRuleID != nil {
		t.Errorf("manual enrollment carries rule %v, want none", *got.AutomationRuleID)
	}
	// The service stamps EnrolledAt; the first stage's wait is measured from it.
	if got.EnrolledAt == nil {
		t.Errorf("enrolled_at was not persisted")
	}

	byID, err := svc.GetEnrollmentByID(ctx, got.EnrollmentID)
	if err != nil || byID == nil || byID.ContactID != contact.ID {
		t.Fatalf("GetEnrollmentByID = %v, %v", byID, err)
	}
	if err := svc.UpdateEnrollmentStatus(ctx, got.EnrollmentID, "paused"); err != nil {
		t.Fatalf("UpdateEnrollmentStatus: %v", err)
	}
	byID, _ = svc.GetEnrollmentByID(ctx, got.EnrollmentID)
	if byID.EnrollmentStatus != "paused" {
		t.Errorf("status = %s, want paused", byID.EnrollmentStatus)
	}
	if missing, err := svc.GetEnrollmentByID(ctx, uuid.New()); err != nil || missing != nil {
		t.Errorf("GetEnrollmentByID(unknown) = %v, %v; want nil, nil", missing, err)
	}
}

func TestApproveEnrollmentCreatesEnrollment(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	svc := f.enrollmentService()
	contact := f.seedContact(t, `{}`)
	pb := f.seedPlaybook(t, twoStages())
	req := f.seedRequest(t, contact.ID, pb.PlaybookID)
	reviewer := uuid.New()

	pending, err := svc.ListPendingApprovals(ctx)
	if err != nil || len(pending) != 1 || pending[0].RequestID != req.RequestID {
		t.Fatalf("ListPendingApprovals = %v, %v; want the seeded request", pending, err)
	}

	if err := svc.ApproveEnrollment(ctx, req.RequestID, reviewer); err != nil {
		t.Fatalf("ApproveEnrollment: %v", err)
	}
	if s := f.requestStatus(t, req.RequestID); s != "approved" {
		t.Errorf("request status = %s, want approved", s)
	}
	got, err := f.enrollments.GetByContactAndPlaybook(ctx, contact.ID, pb.PlaybookID)
	if err != nil || got == nil {
		t.Fatalf("approval did not create an enrollment: %v, %v", got, err)
	}
	if got.AutomationRuleID == nil || *got.AutomationRuleID != req.AutomationRuleID {
		t.Errorf("enrollment rule = %v, want the request's rule %v", got.AutomationRuleID, req.AutomationRuleID)
	}
	if got.CurrentStageID != "intro" || got.EnrollmentStatus != "active" {
		t.Errorf("enrollment = %s at %s, want active at intro", got.EnrollmentStatus, got.CurrentStageID)
	}
	if pending, _ := svc.ListPendingApprovals(ctx); len(pending) != 0 {
		t.Errorf("approved request still listed as pending")
	}
}

// An approval that cannot produce an enrollment must not be recorded as
// approved: the request would leave the pending queue, so nobody would see it
// again, and the contact would never be enrolled.
func TestApproveEnrollmentFailuresLeaveRequestPending(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func(t *testing.T, f *fixture) (requestID uuid.UUID)
		wantErr string
	}{
		{
			name: "unknown request",
			setup: func(t *testing.T, f *fixture) uuid.UUID {
				return uuid.New()
			},
			wantErr: "approval request",
		},
		{
			// Deleting a playbook only flips is_active, so its requests survive.
			name: "playbook deleted after the request was raised",
			setup: func(t *testing.T, f *fixture) uuid.UUID {
				pb := f.seedPlaybook(t, twoStages())
				req := f.seedRequest(t, f.seedContact(t, `{}`).ID, pb.PlaybookID)
				if err := f.playbooks.Delete(ctx, pb.PlaybookID); err != nil {
					t.Fatalf("delete playbook: %v", err)
				}
				return req.RequestID
			},
			wantErr: "playbook not found",
		},
		{
			name: "playbook has no stages",
			setup: func(t *testing.T, f *fixture) uuid.UUID {
				pb := f.seedPlaybook(t, []v1schema.PlaybookStage{})
				return f.seedRequest(t, f.seedContact(t, `{}`).ID, pb.PlaybookID).RequestID
			},
			wantErr: "no stages",
		},
		{
			// A person enrolled the contact by hand while the request waited.
			name: "contact already enrolled",
			setup: func(t *testing.T, f *fixture) uuid.UUID {
				pb := f.seedPlaybook(t, twoStages())
				c := f.seedContact(t, `{}`)
				if err := f.enrollmentService().EnrollContact(ctx, c.ID, pb.PlaybookID); err != nil {
					t.Fatalf("EnrollContact: %v", err)
				}
				return f.seedRequest(t, c.ID, pb.PlaybookID).RequestID
			},
			wantErr: "already enrolled",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			id := tt.setup(t, f)

			err := f.enrollmentService().ApproveEnrollment(ctx, id, uuid.New())
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.wantErr)
			}
			pending, _ := f.approvals.ListPending(ctx)
			var ids []uuid.UUID
			for _, p := range pending {
				ids = append(ids, p.RequestID)
			}
			if tt.name != "unknown request" && (len(ids) != 1 || ids[0] != id) {
				t.Errorf("pending requests = %v, want %v still pending", ids, id)
			}
		})
	}
}

// Only a pending request can be decided. Approving twice would try to enroll
// twice; rejecting an approved request would report "rejected" for a contact
// the approval already enrolled.
func TestDecisionsOnlyApplyToPendingRequests(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	svc := f.enrollmentService()
	pb := f.seedPlaybook(t, twoStages())

	approved := f.seedRequest(t, f.seedContact(t, `{}`).ID, pb.PlaybookID)
	if err := svc.ApproveEnrollment(ctx, approved.RequestID, uuid.New()); err != nil {
		t.Fatalf("ApproveEnrollment: %v", err)
	}
	if err := svc.ApproveEnrollment(ctx, approved.RequestID, uuid.New()); err == nil {
		t.Errorf("second approval succeeded, want an error")
	}
	if err := svc.RejectEnrollment(ctx, approved.RequestID, uuid.New()); err == nil {
		t.Errorf("rejecting an approved request succeeded, want an error")
	}
	if s := f.requestStatus(t, approved.RequestID); s != "approved" {
		t.Errorf("status = %s, want approved to stand", s)
	}

	rejected := f.seedRequest(t, f.seedContact(t, `{}`).ID, pb.PlaybookID)
	reviewer := uuid.New()
	if err := svc.RejectEnrollment(ctx, rejected.RequestID, reviewer); err != nil {
		t.Fatalf("RejectEnrollment: %v", err)
	}
	var reviewedBy uuid.UUID
	if err := f.db.Get(&reviewedBy, `SELECT reviewed_by FROM enrollment_approval_requests WHERE request_id = $1`, rejected.RequestID); err != nil {
		t.Fatalf("read reviewed_by: %v", err)
	}
	if reviewedBy != reviewer {
		t.Errorf("reviewed_by = %v, want %v", reviewedBy, reviewer)
	}
	if err := svc.ApproveEnrollment(ctx, rejected.RequestID, uuid.New()); err == nil {
		t.Errorf("approving a rejected request succeeded, want an error")
	}
	if n := f.countEnrollments(t); n != 1 {
		t.Errorf("enrollments = %d, want only the first approval's", n)
	}
	if err := svc.RejectEnrollment(ctx, uuid.New(), reviewer); err == nil {
		t.Errorf("rejecting an unknown request succeeded, want an error")
	}
}
