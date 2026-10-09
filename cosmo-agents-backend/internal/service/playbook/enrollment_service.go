package playbook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rockship/cosmo-agents-go/internal/domain/base"
	"github.com/rockship/cosmo-agents-go/internal/domain/playbook"
	contactRepo "github.com/rockship/cosmo-agents-go/internal/repository/contact"
	playbookRepo "github.com/rockship/cosmo-agents-go/internal/repository/playbook"
	v1schema "github.com/rockship/cosmo-agents-go/internal/schema/v1"
)

// EnrollmentService handles contact enrollment business logic
type EnrollmentService struct {
	enrollmentRepo *playbookRepo.EnrollmentRepository
	approvalRepo   *playbookRepo.ApprovalRequestRepository
	playbookRepo   *playbookRepo.Repository
	contactRepo    *contactRepo.ContactRepository
}

// NewEnrollmentService creates a new enrollment service
func NewEnrollmentService(
	enrollmentRepo *playbookRepo.EnrollmentRepository,
	approvalRepo *playbookRepo.ApprovalRequestRepository,
	playbookRepo *playbookRepo.Repository,
	contactRepo *contactRepo.ContactRepository,
) *EnrollmentService {
	return &EnrollmentService{
		enrollmentRepo: enrollmentRepo,
		approvalRepo:   approvalRepo,
		playbookRepo:   playbookRepo,
		contactRepo:    contactRepo,
	}
}

// EnrollContact enrolls a contact into a playbook
func (s *EnrollmentService) EnrollContact(ctx context.Context, contactID, playbookID uuid.UUID) error {
	// Validate contact exists
	contact, err := s.contactRepo.GetByID(ctx, contactID)
	if err != nil || contact == nil {
		return fmt.Errorf("contact not found")
	}

	// Validate playbook exists
	pb, err := s.playbookRepo.GetByID(ctx, playbookID)
	if err != nil || pb == nil {
		return fmt.Errorf("playbook not found")
	}

	// Check if already enrolled
	existing, err := s.enrollmentRepo.GetByContactAndPlaybook(ctx, contactID, playbookID)
	if err != nil {
		return fmt.Errorf("failed to check existing enrollment: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("contact already enrolled in this playbook")
	}

	// Parse playbook config to get first stage
	var config v1schema.PlaybookConfig
	if err := json.Unmarshal(pb.Config, &config); err != nil {
		return fmt.Errorf("failed to parse playbook config: %w", err)
	}

	if len(config.Stages) == 0 {
		return fmt.Errorf("playbook has no stages")
	}

	firstStage := config.Stages[0]

	// Initialize execution log
	executionLog := []map[string]interface{}{}
	logBytes, _ := json.Marshal(executionLog)

	// Create enrollment
	now := time.Now()
	enrollment := &playbook.ContactEnrollment{
		ContactID:         contactID,
		PlaybookID:        playbookID,
		AutomationRuleID:  nil, // Manual enrollment
		EnrollmentStatus:  "active",
		CurrentStageOrder: firstStage.Order,
		CurrentStageID:    firstStage.ID,
		EnrolledAt:        &now,
		ExecutionLog:      base.JSONB(logBytes),
	}

	if err := s.enrollmentRepo.Create(ctx, enrollment); err != nil {
		return fmt.Errorf("failed to create enrollment: %w", err)
	}

	return nil
}

// ListPendingApprovals lists all pending approval requests
func (s *EnrollmentService) ListPendingApprovals(ctx context.Context) ([]*playbook.EnrollmentApprovalRequest, error) {
	return s.approvalRepo.ListPending(ctx)
}

// ApproveEnrollment approves an enrollment request and creates actual enrollment
func (s *EnrollmentService) ApproveEnrollment(ctx context.Context, requestID, reviewedBy uuid.UUID) error {
	// Read the request BEFORE approving it. Approving flips its status out of
	// 'pending', so looking it up afterwards through ListPending found nothing
	// and the enrollment below was silently never created.
	approvalReq, err := s.pendingRequest(ctx, requestID)
	if err != nil {
		return err
	}

	// Everything that can stop the enrollment is checked before the request is
	// marked approved: an approved request leaves the pending queue, so one
	// whose enrollment then failed was never shown to anyone again.
	pb, err := s.playbookRepo.GetByID(ctx, approvalReq.PlaybookID)
	if err != nil {
		return fmt.Errorf("failed to get playbook: %w", err)
	}
	if pb == nil {
		return fmt.Errorf("playbook not found")
	}

	var config v1schema.PlaybookConfig
	if err := json.Unmarshal(pb.Config, &config); err != nil {
		return fmt.Errorf("failed to parse playbook config: %w", err)
	}
	if len(config.Stages) == 0 {
		return fmt.Errorf("playbook has no stages")
	}

	existing, err := s.enrollmentRepo.GetByContactAndPlaybook(ctx, approvalReq.ContactID, approvalReq.PlaybookID)
	if err != nil {
		return fmt.Errorf("failed to check existing enrollment: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("%w: contact already enrolled in this playbook", ErrAlreadyDecided)
	}

	if err := s.approvalRepo.Approve(ctx, requestID, reviewedBy); err != nil {
		if errors.Is(err, playbookRepo.ErrRequestNotPending) {
			return fmt.Errorf("%w: decided by someone else just now", ErrAlreadyDecided)
		}
		return fmt.Errorf("failed to approve request: %w", err)
	}

	firstStage := config.Stages[0]
	executionLog := []map[string]interface{}{}
	logBytes, _ := json.Marshal(executionLog)

	now := time.Now()
	enrollment := &playbook.ContactEnrollment{
		ContactID:         approvalReq.ContactID,
		PlaybookID:        approvalReq.PlaybookID,
		AutomationRuleID:  &approvalReq.AutomationRuleID,
		EnrollmentStatus:  "active",
		CurrentStageOrder: firstStage.Order,
		CurrentStageID:    firstStage.ID,
		EnrolledAt:        &now,
		ExecutionLog:      base.JSONB(logBytes),
	}

	if err := s.enrollmentRepo.Create(ctx, enrollment); err != nil {
		return fmt.Errorf("failed to create enrollment: %w", err)
	}

	return nil
}

// ErrAlreadyDecided marks a request that can no longer be approved or
// rejected; the handler answers 409 rather than 500.
var ErrAlreadyDecided = errors.New("approval request already decided")

// RejectEnrollment rejects an enrollment request
func (s *EnrollmentService) RejectEnrollment(ctx context.Context, requestID, reviewedBy uuid.UUID) error {
	if _, err := s.pendingRequest(ctx, requestID); err != nil {
		return err
	}
	if err := s.approvalRepo.Reject(ctx, requestID, reviewedBy); err != nil {
		if errors.Is(err, playbookRepo.ErrRequestNotPending) {
			return fmt.Errorf("%w: decided by someone else just now", ErrAlreadyDecided)
		}
		return err
	}
	return nil
}

// pendingRequest loads a request that is still awaiting a decision. A decided
// request is not decided again: approving twice would enroll twice, and
// rejecting an approved request would report "rejected" for a contact the
// approval already enrolled.
func (s *EnrollmentService) pendingRequest(ctx context.Context, requestID uuid.UUID) (*playbook.EnrollmentApprovalRequest, error) {
	req, err := s.approvalRepo.GetByID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to load approval request: %w", err)
	}
	if req.Status != "pending" {
		return nil, fmt.Errorf("%w: it is %s", ErrAlreadyDecided, req.Status)
	}
	return req, nil
}

// GetEnrollmentByID gets an enrollment by ID
func (s *EnrollmentService) GetEnrollmentByID(ctx context.Context, enrollmentID uuid.UUID) (*playbook.ContactEnrollment, error) {
	return s.enrollmentRepo.GetByID(ctx, enrollmentID)
}

// UpdateEnrollmentStatus updates enrollment status (pause, resume, complete)
func (s *EnrollmentService) UpdateEnrollmentStatus(ctx context.Context, enrollmentID uuid.UUID, status string) error {
	return s.enrollmentRepo.UpdateStatus(ctx, enrollmentID, status)
}
