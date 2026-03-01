-- Add outreach_stage and followup_count columns to contacts table
-- These are used by the outreach state machine to track conversation states

-- Add outreach_stage column (tracks: COLD, NO_REPLY, REPLIED, POST_MEETING, DROPPED)
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS outreach_stage VARCHAR(50) DEFAULT 'COLD';

-- Add followup_count column (tracks number of follow-up messages sent)
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS followup_count INTEGER DEFAULT 0;

-- Create index for faster filtering by outreach_stage
CREATE INDEX IF NOT EXISTS idx_contacts_outreach_stage ON contacts(outreach_stage);

-- Update existing contacts: set outreach_stage based on existing data
UPDATE contacts
SET outreach_stage = CASE
    WHEN lifecycle_stage = 'replied' THEN 'REPLIED'
    WHEN lifecycle_stage = 'contacted' AND next_step = 'WAIT' THEN 'NO_REPLY'
    WHEN lifecycle_stage = 'contacted' THEN 'NO_REPLY'
    WHEN lifecycle_stage = 'meeting_scheduled' THEN 'POST_MEETING'
    WHEN do_not_contact = true THEN 'DROPPED'
    ELSE 'COLD'
END
WHERE outreach_stage IS NULL OR outreach_stage = 'COLD';
