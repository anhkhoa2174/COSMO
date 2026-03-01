package playbook

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rockship/cosmo-agents-go/internal/domain/playbook"
)

// EnrollmentRepository handles contact enrollment database operations
type EnrollmentRepository struct {
	db *sqlx.DB
}

// NewEnrollmentRepository creates a new enrollment repository
func NewEnrollmentRepository(db *sqlx.DB) *EnrollmentRepository {
	return &EnrollmentRepository{db: db}
}

// Create inserts a new contact enrollment
func (r *EnrollmentRepository) Create(ctx context.Context, enrollment *playbook.ContactEnrollment) error {
	query := `
		INSERT INTO contact_enrollments
		(contact_id, playbook_id, automation_rule_id, enrollment_status, current_stage_order, current_stage_id, execution_log)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING enrollment_id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		enrollment.ContactID, enrollment.PlaybookID, enrollment.AutomationRuleID,
		enrollment.EnrollmentStatus, enrollment.CurrentStageOrder, enrollment.CurrentStageID,
		enrollment.ExecutionLog,
	).Scan(&enrollment.EnrollmentID, &enrollment.CreatedAt, &enrollment.UpdatedAt)
}

// GetByID fetches an enrollment by ID
func (r *EnrollmentRepository) GetByID(ctx context.Context, enrollmentID uuid.UUID) (*playbook.ContactEnrollment, error) {
	query := `SELECT * FROM contact_enrollments WHERE enrollment_id = $1`

	var enrollment playbook.ContactEnrollment
	if err := r.db.GetContext(ctx, &enrollment, query, enrollmentID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &enrollment, nil
}

// GetByContactAndPlaybook fetches enrollment by contact and playbook
func (r *EnrollmentRepository) GetByContactAndPlaybook(ctx context.Context, contactID, playbookID uuid.UUID) (*playbook.ContactEnrollment, error) {
	query := `SELECT * FROM contact_enrollments WHERE contact_id = $1 AND playbook_id = $2`

	var enrollment playbook.ContactEnrollment
	if err := r.db.GetContext(ctx, &enrollment, query, contactID, playbookID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &enrollment, nil
}

// UpdateStatus updates enrollment status
func (r *EnrollmentRepository) UpdateStatus(ctx context.Context, enrollmentID uuid.UUID, status string) error {
	query := `UPDATE contact_enrollments SET enrollment_status = $1 WHERE enrollment_id = $2`

	_, err := r.db.ExecContext(ctx, query, status, enrollmentID)
	return err
}

// UpdateStage updates current stage
func (r *EnrollmentRepository) UpdateStage(ctx context.Context, enrollmentID uuid.UUID, stageOrder int, stageID string) error {
	query := `UPDATE contact_enrollments SET current_stage_order = $1, current_stage_id = $2 WHERE enrollment_id = $3`

	_, err := r.db.ExecContext(ctx, query, stageOrder, stageID, enrollmentID)
	return err
}

// UpdateExecutionLog updates execution log for an enrollment.
func (r *EnrollmentRepository) UpdateExecutionLog(ctx context.Context, enrollmentID uuid.UUID, log []byte) error {
	query := `UPDATE contact_enrollments SET execution_log = $1 WHERE enrollment_id = $2`
	_, err := r.db.ExecContext(ctx, query, log, enrollmentID)
	return err
}

// UpdateCompletedAt updates completed timestamp.
func (r *EnrollmentRepository) UpdateCompletedAt(ctx context.Context, enrollmentID uuid.UUID) error {
	query := `UPDATE contact_enrollments SET completed_at = NOW() WHERE enrollment_id = $1`
	_, err := r.db.ExecContext(ctx, query, enrollmentID)
	return err
}

// ListByStatus fetches enrollments by status.
func (r *EnrollmentRepository) ListByStatus(ctx context.Context, status string) ([]*playbook.ContactEnrollment, error) {
	query := `SELECT * FROM contact_enrollments WHERE enrollment_status = $1 ORDER BY updated_at ASC`

	var enrollments []*playbook.ContactEnrollment
	if err := r.db.SelectContext(ctx, &enrollments, query, status); err != nil {
		return nil, err
	}

	return enrollments, nil
}

// ApprovalRequestRepository handles approval request database operations
type ApprovalRequestRepository struct {
	db *sqlx.DB
}

// NewApprovalRequestRepository creates a new approval request repository
func NewApprovalRequestRepository(db *sqlx.DB) *ApprovalRequestRepository {
	return &ApprovalRequestRepository{db: db}
}

// Create inserts a new approval request
func (r *ApprovalRequestRepository) Create(ctx context.Context, req *playbook.EnrollmentApprovalRequest) error {
	query := `
		INSERT INTO enrollment_approval_requests
		(contact_id, playbook_id, automation_rule_id, reason, fit_score, engagement_score, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING request_id, created_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		req.ContactID, req.PlaybookID, req.AutomationRuleID,
		req.Reason, req.FitScore, req.EngagementScore, req.Status,
	).Scan(&req.RequestID, &req.CreatedAt)
}

// ListPending fetches all pending approval requests
func (r *ApprovalRequestRepository) ListPending(ctx context.Context) ([]*playbook.EnrollmentApprovalRequest, error) {
	query := `SELECT * FROM enrollment_approval_requests WHERE status = 'pending' ORDER BY created_at DESC`

	var requests []*playbook.EnrollmentApprovalRequest
	if err := r.db.SelectContext(ctx, &requests, query); err != nil {
		return nil, err
	}

	return requests, nil
}

// GetPendingByContactRule fetches a pending request for contact/playbook/rule.
func (r *ApprovalRequestRepository) GetPendingByContactRule(ctx context.Context, contactID, playbookID, ruleID uuid.UUID) (*playbook.EnrollmentApprovalRequest, error) {
	query := `
		SELECT * FROM enrollment_approval_requests
		WHERE status = 'pending'
		  AND contact_id = $1
		  AND playbook_id = $2
		  AND automation_rule_id = $3
		LIMIT 1
	`

	var req playbook.EnrollmentApprovalRequest
	if err := r.db.GetContext(ctx, &req, query, contactID, playbookID, ruleID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &req, nil
}

// Approve marks a request as approved
func (r *ApprovalRequestRepository) Approve(ctx context.Context, requestID, reviewedBy uuid.UUID) error {
	query := `
		UPDATE enrollment_approval_requests
		SET status = 'approved', reviewed_by = $1, reviewed_at = NOW()
		WHERE request_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, reviewedBy, requestID)
	return err
}

// Reject marks a request as rejected
func (r *ApprovalRequestRepository) Reject(ctx context.Context, requestID, reviewedBy uuid.UUID) error {
	query := `
		UPDATE enrollment_approval_requests
		SET status = 'rejected', reviewed_by = $1, reviewed_at = NOW()
		WHERE request_id = $2
	`

	_, err := r.db.ExecContext(ctx, query, reviewedBy, requestID)
	return err
}
