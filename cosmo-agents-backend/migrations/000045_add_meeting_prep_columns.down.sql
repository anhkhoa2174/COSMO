-- Rollback: Remove meeting_content and meeting_prep columns from meetings table

ALTER TABLE meetings DROP COLUMN IF EXISTS meeting_content;
ALTER TABLE meetings DROP COLUMN IF EXISTS meeting_prep;
