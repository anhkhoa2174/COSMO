-- Migration 000026: Align campaigns table with expected schema and seed sample data
-- This migration is idempotent: it guards all ALTER/CREATE statements.

BEGIN;

SET search_path TO public;

-- Ensure pgcrypto is available for UUID helpers if needed later.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Align campaigns table columns
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS name VARCHAR;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS user_id UUID;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS list_contact_id UUID;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS organization_id UUID;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS schedule TIMESTAMPTZ;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS status VARCHAR;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS agent_id UUID;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS cmetadata JSONB;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS playbook VARCHAR;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS _sale_rep_node_states JSONB;

-- Enforce desired defaults and constraints on existing columns
UPDATE campaigns SET cmetadata = '{}'::jsonb WHERE cmetadata IS NULL;
ALTER TABLE campaigns ALTER COLUMN cmetadata SET DEFAULT '{}'::jsonb;
ALTER TABLE campaigns ALTER COLUMN cmetadata SET NOT NULL;

ALTER TABLE campaigns ALTER COLUMN is_deleted SET DEFAULT FALSE;
UPDATE campaigns SET is_deleted = FALSE WHERE is_deleted IS NULL;
ALTER TABLE campaigns ALTER COLUMN is_deleted SET NOT NULL;

-- Ensure schedule column uses timestamptz
ALTER TABLE campaigns ALTER COLUMN schedule TYPE TIMESTAMPTZ USING schedule::timestamptz;

-- Foreign key constraints (added only if missing)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_campaigns_user_id'
    ) THEN
        ALTER TABLE campaigns
            ADD CONSTRAINT fk_campaigns_user_id
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_campaigns_list_contact_id'
    ) THEN
        ALTER TABLE campaigns
            ADD CONSTRAINT fk_campaigns_list_contact_id
            FOREIGN KEY (list_contact_id) REFERENCES list_contacts(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_campaigns_organization_id'
    ) THEN
        ALTER TABLE campaigns
            ADD CONSTRAINT fk_campaigns_organization_id
            FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_campaigns_agent_id'
    ) THEN
        ALTER TABLE campaigns
            ADD CONSTRAINT fk_campaigns_agent_id
            FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE SET NULL;
    END IF;
END
$$;

-- Helpful indexes
CREATE INDEX IF NOT EXISTS ix_campaigns_user_id ON campaigns (user_id);
CREATE INDEX IF NOT EXISTS ix_campaigns_organization_id ON campaigns (organization_id);

COMMIT;
