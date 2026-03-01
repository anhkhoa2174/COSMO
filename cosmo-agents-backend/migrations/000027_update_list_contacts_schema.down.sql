-- Revert migration 000027 changes on list_contacts table.

BEGIN;

SET search_path TO public;

ALTER TABLE list_contacts
    DROP CONSTRAINT IF EXISTS fk_list_contacts_organization_id;

ALTER TABLE list_contacts
    DROP COLUMN IF EXISTS organization_id;

ALTER TABLE list_contacts
    DROP COLUMN IF EXISTS deleted_at;

ALTER TABLE list_contacts
    DROP COLUMN IF EXISTS is_deleted;

COMMIT;
