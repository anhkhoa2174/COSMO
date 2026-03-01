-- Remove outreach_stage and followup_count columns from contacts table

-- Drop index first
DROP INDEX IF EXISTS idx_contacts_outreach_stage;

-- Drop columns
ALTER TABLE contacts DROP COLUMN IF EXISTS outreach_stage;
ALTER TABLE contacts DROP COLUMN IF EXISTS followup_count;
