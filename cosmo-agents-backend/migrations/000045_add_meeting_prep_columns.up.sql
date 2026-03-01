-- Add meeting_content and meeting_prep columns to meetings table
-- meeting_content: stores transcript, notes, etc. after the meeting
-- meeting_prep: AI-generated preparation document (talking points, discovery questions) before the meeting

ALTER TABLE meetings ADD COLUMN IF NOT EXISTS meeting_content TEXT;
ALTER TABLE meetings ADD COLUMN IF NOT EXISTS meeting_prep TEXT;
