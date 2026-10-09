-- Drop lifecycle_stage column and its index from contacts table
-- This field is replaced by outreach_stage and next_step
DROP INDEX IF EXISTS idx_contacts_lifecycle;
ALTER TABLE contacts DROP COLUMN IF EXISTS lifecycle_stage;
