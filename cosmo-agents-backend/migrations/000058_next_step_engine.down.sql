DROP TABLE IF EXISTS next_step_decisions;
DROP INDEX IF EXISTS idx_contacts_next_action_due_at;
ALTER TABLE contacts
    DROP COLUMN IF EXISTS next_action,
    DROP COLUMN IF EXISTS next_action_args,
    DROP COLUMN IF EXISTS next_action_reason,
    DROP COLUMN IF EXISTS next_action_due_at,
    DROP COLUMN IF EXISTS next_action_decision_id;
