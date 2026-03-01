-- Remove business_stage column from contacts table
DROP INDEX IF EXISTS idx_contacts_business_stage;
ALTER TABLE contacts DROP COLUMN IF EXISTS business_stage;
