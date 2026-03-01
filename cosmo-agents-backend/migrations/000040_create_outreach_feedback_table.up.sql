-- Enable UUID extension (required for uuid_generate_v4)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Outreach Feedback table for tracking AI suggestions vs BD actions
CREATE TABLE IF NOT EXISTS outreach_feedback (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,

    -- What AI suggested
    suggested_scenario VARCHAR(50) NOT NULL,
    suggested_intent VARCHAR(50) NOT NULL,
    suggested_draft TEXT,
    suggested_next_step VARCHAR(50),

    -- What BD actually did
    actual_action VARCHAR(50), -- 'used_draft', 'modified_draft', 'wrote_own', 'skipped'
    actual_content TEXT,       -- The actual message sent (if different from draft)

    -- Outcome tracking
    outcome VARCHAR(50),       -- 'no_action', 'sent', 'replied', 'meeting_booked', 'meeting_done', 'dropped'
    reply_sentiment VARCHAR(20), -- 'positive', 'neutral', 'negative'
    days_to_reply INT,         -- How many days until reply (null if no reply)

    -- Metadata
    context_level VARCHAR(20) NOT NULL, -- 'LOW', 'MEDIUM', 'HIGH'
    conversation_state VARCHAR(50) NOT NULL,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    outcome_updated_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT fk_feedback_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_feedback_contact FOREIGN KEY (contact_id) REFERENCES contacts(id)
);

-- Index for analytics queries
CREATE INDEX idx_outreach_feedback_user ON outreach_feedback(user_id);
CREATE INDEX idx_outreach_feedback_scenario ON outreach_feedback(suggested_scenario);
CREATE INDEX idx_outreach_feedback_outcome ON outreach_feedback(outcome);
CREATE INDEX idx_outreach_feedback_created ON outreach_feedback(created_at);

-- Scenario statistics view for quick analytics
CREATE OR REPLACE VIEW outreach_scenario_stats AS
SELECT
    user_id,
    suggested_scenario,
    context_level,
    COUNT(*) as total_suggestions,
    COUNT(CASE WHEN actual_action = 'used_draft' THEN 1 END) as drafts_used,
    COUNT(CASE WHEN actual_action = 'modified_draft' THEN 1 END) as drafts_modified,
    COUNT(CASE WHEN actual_action = 'wrote_own' THEN 1 END) as wrote_own,
    COUNT(CASE WHEN actual_action = 'skipped' THEN 1 END) as skipped,
    COUNT(CASE WHEN outcome = 'sent' THEN 1 END) as total_sent,
    COUNT(CASE WHEN outcome = 'replied' THEN 1 END) as total_replied,
    COUNT(CASE WHEN outcome = 'meeting_booked' THEN 1 END) as total_meetings,
    ROUND(
        COUNT(CASE WHEN outcome = 'replied' THEN 1 END)::DECIMAL /
        NULLIF(COUNT(CASE WHEN outcome = 'sent' THEN 1 END), 0) * 100, 2
    ) as reply_rate,
    ROUND(
        COUNT(CASE WHEN outcome = 'meeting_booked' THEN 1 END)::DECIMAL /
        NULLIF(COUNT(CASE WHEN outcome = 'replied' THEN 1 END), 0) * 100, 2
    ) as meeting_rate,
    AVG(days_to_reply) as avg_days_to_reply
FROM outreach_feedback
WHERE outcome IS NOT NULL
GROUP BY user_id, suggested_scenario, context_level;
