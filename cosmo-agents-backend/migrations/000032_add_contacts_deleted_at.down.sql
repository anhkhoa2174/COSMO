-- Remove deleted_at column from contacts table
DROP INDEX IF EXISTS idx_contacts_deleted_at;
ALTER TABLE contacts DROP COLUMN IF EXISTS deleted_at;
