-- Rollback for migration 000026

BEGIN;

SET search_path TO public;

-- Drop indexes
DROP INDEX IF EXISTS ix_campaigns_user_id;
DROP INDEX IF EXISTS ix_campaigns_organization_id;

-- Drop foreign keys that were added
ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS fk_campaigns_agent_id;
ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS fk_campaigns_list_contact_id;
ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS fk_campaigns_organization_id;
ALTER TABLE campaigns DROP CONSTRAINT IF EXISTS fk_campaigns_user_id;

-- Drop columns introduced by the migration
ALTER TABLE campaigns DROP COLUMN IF EXISTS _sale_rep_node_states;
ALTER TABLE campaigns DROP COLUMN IF EXISTS is_deleted;
ALTER TABLE campaigns DROP COLUMN IF EXISTS playbook;
ALTER TABLE campaigns DROP COLUMN IF EXISTS cmetadata;
ALTER TABLE campaigns DROP COLUMN IF EXISTS agent_id;
ALTER TABLE campaigns DROP COLUMN IF EXISTS status;
ALTER TABLE campaigns DROP COLUMN IF EXISTS schedule;
ALTER TABLE campaigns DROP COLUMN IF EXISTS organization_id;
ALTER TABLE campaigns DROP COLUMN IF EXISTS updated_at;
ALTER TABLE campaigns DROP COLUMN IF EXISTS created_at;
ALTER TABLE campaigns DROP COLUMN IF EXISTS list_contact_id;
ALTER TABLE campaigns DROP COLUMN IF EXISTS user_id;
ALTER TABLE campaigns DROP COLUMN IF EXISTS name;

COMMIT;
