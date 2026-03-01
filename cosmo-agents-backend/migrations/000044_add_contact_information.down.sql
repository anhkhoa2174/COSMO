-- Remove contact_information column from contacts table
DROP INDEX IF EXISTS idx_contacts_contact_information;
ALTER TABLE contacts DROP COLUMN IF EXISTS contact_information;
