package playbook

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rockship/cosmo-agents-go/internal/domain/playbook"
)

// AutomationRuleRepository handles automation rule database operations
type AutomationRuleRepository struct {
	db *sqlx.DB
}

// NewAutomationRuleRepository creates a new automation rule repository
func NewAutomationRuleRepository(db *sqlx.DB) *AutomationRuleRepository {
	return &AutomationRuleRepository{db: db}
}

// Create inserts a new automation rule
func (r *AutomationRuleRepository) Create(ctx context.Context, rule *playbook.AutomationRule) error {
	query := `
		INSERT INTO automation_rules (name, segment_id, playbook_id, enrollment_criteria, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING automation_rule_id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		rule.Name, rule.SegmentID, rule.PlaybookID, rule.EnrollmentCriteria, rule.IsActive,
	).Scan(&rule.AutomationRuleID, &rule.CreatedAt, &rule.UpdatedAt)
}

// GetByID fetches an automation rule by ID
func (r *AutomationRuleRepository) GetByID(ctx context.Context, ruleID uuid.UUID) (*playbook.AutomationRule, error) {
	query := `SELECT * FROM automation_rules WHERE automation_rule_id = $1`

	var rule playbook.AutomationRule
	if err := r.db.GetContext(ctx, &rule, query, ruleID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &rule, nil
}

// List fetches all automation rules
func (r *AutomationRuleRepository) List(ctx context.Context) ([]*playbook.AutomationRule, error) {
	query := `SELECT * FROM automation_rules ORDER BY created_at DESC`

	var rules []*playbook.AutomationRule
	if err := r.db.SelectContext(ctx, &rules, query); err != nil {
		return nil, err
	}

	return rules, nil
}

// ListActive fetches all active automation rules
func (r *AutomationRuleRepository) ListActive(ctx context.Context) ([]*playbook.AutomationRule, error) {
	query := `SELECT * FROM automation_rules WHERE is_active = true ORDER BY created_at DESC`

	var rules []*playbook.AutomationRule
	if err := r.db.SelectContext(ctx, &rules, query); err != nil {
		return nil, err
	}

	return rules, nil
}

// Toggle toggles an automation rule's active status
func (r *AutomationRuleRepository) Toggle(ctx context.Context, ruleID uuid.UUID) error {
	query := `UPDATE automation_rules SET is_active = NOT is_active WHERE automation_rule_id = $1`

	_, err := r.db.ExecContext(ctx, query, ruleID)
	return err
}

// UpdateStatus updates the active status to a specific value
func (r *AutomationRuleRepository) UpdateStatus(ctx context.Context, ruleID uuid.UUID, isActive bool) error {
	query := `UPDATE automation_rules SET is_active = $1 WHERE automation_rule_id = $2`

	_, err := r.db.ExecContext(ctx, query, isActive, ruleID)
	return err
}

// Delete soft deletes an automation rule by setting is_active to false
func (r *AutomationRuleRepository) Delete(ctx context.Context, ruleID uuid.UUID) error {
	query := `DELETE FROM automation_rules WHERE automation_rule_id = $1`

	_, err := r.db.ExecContext(ctx, query, ruleID)
	return err
}
