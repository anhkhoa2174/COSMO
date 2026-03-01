-- Rollback migration 000028: templates soft delete and contacts schema

-- Part 1: Rollback contacts schema updates
DROP INDEX IF EXISTS idx_contacts_user_filtering_sort;
DROP INDEX IF EXISTS idx_contacts_profile_gin;
DROP INDEX IF EXISTS uq_contacts_source_id_source_user_id;

-- Recreate old index
CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_source_id_source 
ON contacts (source_id, source);

-- Drop foreign key constraint
ALTER TABLE contacts 
DROP CONSTRAINT IF EXISTS fk_contacts_organization_id;

-- Drop columns from contacts
ALTER TABLE contacts DROP COLUMN IF EXISTS tags;
ALTER TABLE contacts DROP COLUMN IF EXISTS organization_id;
ALTER TABLE contacts DROP COLUMN IF EXISTS do_not_contact;
ALTER TABLE contacts DROP COLUMN IF EXISTS profile;
ALTER TABLE contacts DROP COLUMN IF EXISTS is_deleted;

-- Part 2: Rollback templates soft delete
ALTER TABLE templates
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS is_deleted;
