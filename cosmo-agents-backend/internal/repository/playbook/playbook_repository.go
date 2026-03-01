package playbook

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rockship/cosmo-agents-go/internal/domain/playbook"
)

// Repository handles playbook database operations
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new playbook repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new playbook
func (r *Repository) Create(ctx context.Context, p *playbook.Playbook) error {
	query := `
		INSERT INTO playbooks (name, description, playbook_type, config, performance, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING playbook_id, created_at, updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		p.Name, p.Description, p.PlaybookType, p.Config, p.Performance, p.IsActive,
	).Scan(&p.PlaybookID, &p.CreatedAt, &p.UpdatedAt)
}

// GetByID fetches a playbook by ID
func (r *Repository) GetByID(ctx context.Context, playbookID uuid.UUID) (*playbook.Playbook, error) {
	query := `SELECT * FROM playbooks WHERE playbook_id = $1 AND is_active = true`

	var p playbook.Playbook
	if err := r.db.GetContext(ctx, &p, query, playbookID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &p, nil
}

// List fetches all active playbooks
func (r *Repository) List(ctx context.Context) ([]*playbook.Playbook, error) {
	query := `SELECT * FROM playbooks WHERE is_active = true ORDER BY created_at DESC`

	var playbooks []*playbook.Playbook
	if err := r.db.SelectContext(ctx, &playbooks, query); err != nil {
		return nil, err
	}

	return playbooks, nil
}

// Update updates a playbook
func (r *Repository) Update(ctx context.Context, p *playbook.Playbook) error {
	query := `
		UPDATE playbooks
		SET name = $1, description = $2, playbook_type = $3, config = $4, performance = $5, is_active = $6
		WHERE playbook_id = $7
		RETURNING updated_at
	`

	return r.db.QueryRowContext(
		ctx, query,
		p.Name, p.Description, p.PlaybookType, p.Config, p.Performance, p.IsActive, p.PlaybookID,
	).Scan(&p.UpdatedAt)
}

// Delete soft deletes a playbook
func (r *Repository) Delete(ctx context.Context, playbookID uuid.UUID) error {
	query := `UPDATE playbooks SET is_active = false WHERE playbook_id = $1`

	result, err := r.db.ExecContext(ctx, query, playbookID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("playbook not found")
	}

	return nil
}

// UpdatePerformance updates playbook performance metrics
func (r *Repository) UpdatePerformance(ctx context.Context, playbookID uuid.UUID, performance []byte) error {
	query := `UPDATE playbooks SET performance = $1 WHERE playbook_id = $2`

	_, err := r.db.ExecContext(ctx, query, performance, playbookID)
	return err
}
