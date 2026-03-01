-- Rollback: Remove status and missing_fields columns from contacts table

DROP INDEX IF EXISTS idx_contacts_status;

ALTER TABLE contacts
DROP COLUMN IF EXISTS status,
DROP COLUMN IF EXISTS missing_fields;
