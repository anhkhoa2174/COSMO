-- Migration 000027: Align list_contacts schema with Go domain expectations
-- Adds soft delete columns and organization linkage required by handlers.

BEGIN;

SET search_path TO public;

-- Add organization reference and soft-delete columns if missing.
ALTER TABLE list_contacts
    ADD COLUMN IF NOT EXISTS organization_id UUID;

ALTER TABLE list_contacts
    ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE list_contacts
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Ensure existing rows respect the default.
UPDATE list_contacts
SET is_deleted = FALSE
WHERE is_deleted IS NULL;

-- Foreign key to organizations (idempotent).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_list_contacts_organization_id'
    ) THEN
        ALTER TABLE list_contacts
            ADD CONSTRAINT fk_list_contacts_organization_id
            FOREIGN KEY (organization_id)
            REFERENCES organizations(id)
            ON DELETE CASCADE;
    END IF;
END
$$;

-- Helpful indexes (created previously but guarded for completeness).
CREATE UNIQUE INDEX IF NOT EXISTS idx_list_contacts_source_id_source_user_id
    ON list_contacts (source_id, source, user_id);

CREATE INDEX IF NOT EXISTS ix_list_contacts_user_id
    ON list_contacts (user_id);

COMMIT;
