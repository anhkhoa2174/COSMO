-- Revert outreach context fields from contacts table

DROP INDEX IF EXISTS idx_contacts_lifecycle;

ALTER TABLE contacts DROP COLUMN IF EXISTS industry;
ALTER TABLE contacts DROP COLUMN IF EXISTS contact_channel;
ALTER TABLE contacts DROP COLUMN IF EXISTS lifecycle_stage;
ALTER TABLE contacts DROP COLUMN IF EXISTS context_level;
ALTER TABLE contacts DROP COLUMN IF EXISTS outreach_decision;
ALTER TABLE contacts DROP COLUMN IF EXISTS scenario;
ALTER TABLE contacts DROP COLUMN IF EXISTS message_draft;
ALTER TABLE contacts DROP COLUMN IF EXISTS last_outcome;
ALTER TABLE contacts DROP COLUMN IF EXISTS next_step;
ALTER TABLE contacts DROP COLUMN IF EXISTS meeting;
