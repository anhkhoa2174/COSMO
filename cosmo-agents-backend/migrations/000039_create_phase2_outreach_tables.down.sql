-- Phase 2: Rollback - Drop InteractionLog, OutreachState, and Meeting tables

-- Drop indexes first
DROP INDEX IF EXISTS idx_contacts_last_interaction;

-- Drop tables in reverse order (due to foreign key constraints)
DROP TABLE IF EXISTS meetings;
DROP TABLE IF EXISTS outreach_states;
DROP TABLE IF EXISTS interaction_logs;

-- Remove added column from contacts
ALTER TABLE contacts DROP COLUMN IF EXISTS last_interaction_at;
