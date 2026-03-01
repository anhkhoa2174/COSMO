-- Phase 2: Create InteractionLog, OutreachState, and Meeting tables
-- These tables support the Outreach Decision + Draft Engine

-- ============================================
-- 1. InteractionLog Table
-- Stores all messages sent/received and interaction notes
-- ============================================
CREATE TABLE IF NOT EXISTS interaction_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,

    -- Interaction details
    channel VARCHAR(50) NOT NULL DEFAULT 'LinkedIn', -- LinkedIn, Email, Call, Meeting, Note
    direction VARCHAR(20) NOT NULL DEFAULT 'outgoing', -- outgoing, incoming, internal
    content TEXT NOT NULL DEFAULT '',

    -- Metadata
    subject VARCHAR(500),
    url VARCHAR(1000),
    attachments JSONB DEFAULT '[]',

    -- Sentiment analysis (for incoming messages)
    sentiment VARCHAR(20), -- positive, neutral, negative

    -- Timestamps
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for InteractionLog
CREATE INDEX IF NOT EXISTS idx_interaction_logs_contact_id ON interaction_logs(contact_id);
CREATE INDEX IF NOT EXISTS idx_interaction_logs_user_id ON interaction_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_interaction_logs_channel ON interaction_logs(channel);
CREATE INDEX IF NOT EXISTS idx_interaction_logs_direction ON interaction_logs(direction);
CREATE INDEX IF NOT EXISTS idx_interaction_logs_timestamp ON interaction_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_interaction_logs_contact_timestamp ON interaction_logs(contact_id, timestamp DESC);

-- ============================================
-- 2. OutreachState Table
-- Snapshot state for CLI display and flow control
-- ============================================
CREATE TABLE IF NOT EXISTS outreach_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,

    -- Conversation state machine
    conversation_state VARCHAR(30) NOT NULL DEFAULT 'COLD', -- COLD, NO_REPLY, REPLIED, POST_MEETING, DROPPED

    -- Context assessment
    context_level VARCHAR(20) NOT NULL DEFAULT 'LOW', -- LOW, MEDIUM, HIGH

    -- Outreach intent
    outreach_intent VARCHAR(30) NOT NULL DEFAULT 'INTRO', -- INTRO, FOLLOW_UP, RE_ENGAGE, POST_MEETING

    -- Scenario for message generation
    scenario VARCHAR(50) NOT NULL DEFAULT 'role_based', -- role_based, industry_based, no_reply_followup, post_reply, post_meeting, re_engage

    -- Draft message
    message_draft TEXT,

    -- Last outcome tracking
    last_outcome VARCHAR(30) NOT NULL DEFAULT 'none', -- none, sent, no_reply, replied, meeting_booked, meeting_done, dropped

    -- Next step recommendation
    next_step VARCHAR(30) NOT NULL DEFAULT 'SEND', -- SEND, FOLLOW_UP, WAIT, SET_MEETING, DROP

    -- Time tracking
    last_interaction_at TIMESTAMPTZ,
    days_since_last_interaction INTEGER DEFAULT 0,

    -- Follow-up tracking
    followup_count INTEGER NOT NULL DEFAULT 0,
    max_followups INTEGER NOT NULL DEFAULT 3,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Unique constraint: one state per contact per user
    UNIQUE(user_id, contact_id)
);

-- Indexes for OutreachState
CREATE INDEX IF NOT EXISTS idx_outreach_states_contact_id ON outreach_states(contact_id);
CREATE INDEX IF NOT EXISTS idx_outreach_states_user_id ON outreach_states(user_id);
CREATE INDEX IF NOT EXISTS idx_outreach_states_conversation_state ON outreach_states(conversation_state);
CREATE INDEX IF NOT EXISTS idx_outreach_states_next_step ON outreach_states(next_step);
CREATE INDEX IF NOT EXISTS idx_outreach_states_last_outcome ON outreach_states(last_outcome);

-- ============================================
-- 3. Meeting Table
-- Track meetings with contacts
-- ============================================
CREATE TABLE IF NOT EXISTS meetings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,

    -- Meeting details
    title VARCHAR(500),
    time TIMESTAMPTZ NOT NULL,
    duration_minutes INTEGER DEFAULT 30,
    channel VARCHAR(50) NOT NULL DEFAULT 'Zoom', -- Zoom, Google Meet, Call, Offline
    location VARCHAR(500),
    meeting_url VARCHAR(1000),

    -- Status tracking
    status VARCHAR(30) NOT NULL DEFAULT 'scheduled', -- scheduled, completed, cancelled, no_show

    -- Notes and outcomes
    note TEXT,
    outcome TEXT,
    next_steps TEXT,

    -- Participants (JSON array of emails/names)
    participants JSONB DEFAULT '[]',

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for Meeting
CREATE INDEX IF NOT EXISTS idx_meetings_contact_id ON meetings(contact_id);
CREATE INDEX IF NOT EXISTS idx_meetings_user_id ON meetings(user_id);
CREATE INDEX IF NOT EXISTS idx_meetings_status ON meetings(status);
CREATE INDEX IF NOT EXISTS idx_meetings_time ON meetings(time);
CREATE INDEX IF NOT EXISTS idx_meetings_contact_time ON meetings(contact_id, time DESC);

-- ============================================
-- 4. Add last_interaction_at to contacts table
-- ============================================
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS last_interaction_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_contacts_last_interaction ON contacts(last_interaction_at);

-- ============================================
-- 5. Update contacts lifecycle_stage enum values
-- ============================================
-- Add 'meeting' and 'dropped' stages if not exists
UPDATE contacts SET lifecycle_stage = 'new' WHERE lifecycle_stage IS NULL OR lifecycle_stage = '';
