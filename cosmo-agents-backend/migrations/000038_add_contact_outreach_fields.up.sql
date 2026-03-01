-- Add outreach context fields to contacts table
-- These fields support the Contact View requirements for sales outreach

-- Industry field (e.g., Fintech, SaaS)
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS industry VARCHAR(255) DEFAULT '';

-- Contact channel (e.g., LinkedIn, Email)
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS contact_channel VARCHAR(100) DEFAULT '';

-- Lifecycle stage with index for filtering
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS lifecycle_stage VARCHAR(50) DEFAULT 'new';
CREATE INDEX IF NOT EXISTS idx_contacts_lifecycle ON contacts(lifecycle_stage);

-- Context level (LOW, MEDIUM, HIGH)
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS context_level VARCHAR(20) DEFAULT 'LOW';

-- Outreach decision (INTRO, FOLLOW-UP, NURTURE, HOLD)
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS outreach_decision VARCHAR(50) DEFAULT 'INTRO';

-- Scenario type (Role-based, Post-reply, etc.)
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS scenario VARCHAR(100) DEFAULT '';

-- Message draft for outreach
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS message_draft TEXT DEFAULT '';

-- Last outcome of outreach
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS last_outcome VARCHAR(255) DEFAULT '';

-- Next step (SEND, MEETING, CALL, WAIT, CLOSE)
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS next_step VARCHAR(50) DEFAULT 'SEND';

-- Meeting details if scheduled
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS meeting VARCHAR(500) DEFAULT '';

-- Set default values for existing contacts based on their current state
UPDATE contacts
SET lifecycle_stage = 'new',
    context_level = 'LOW',
    outreach_decision = 'INTRO',
    next_step = 'SEND'
WHERE lifecycle_stage IS NULL OR lifecycle_stage = '';
