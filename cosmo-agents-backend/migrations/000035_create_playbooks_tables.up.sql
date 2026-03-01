-- Create playbooks table
CREATE TABLE IF NOT EXISTS playbooks (
    playbook_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    description TEXT,
    playbook_type VARCHAR(50) NOT NULL, -- 'nurture', 'outreach', 're_engagement', 'upsell'

    -- Playbook Configuration (stages, messaging_strategy, timing_rules, channel_sequence)
    config JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Performance Tracking (contacts_enrolled, total_sent, total_replies, reply_rate, etc.)
    performance JSONB NOT NULL DEFAULT '{"contacts_enrolled": 0, "total_sent": 0, "total_replies": 0, "total_meetings": 0, "reply_rate": 0.0, "meeting_rate": 0.0}'::jsonb,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Ensure updated_at trigger function exists
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS trigger AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create automation_rules table
CREATE TABLE IF NOT EXISTS automation_rules (
    automation_rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    segment_id UUID NOT NULL REFERENCES segmentations(id) ON DELETE CASCADE,
    playbook_id UUID NOT NULL REFERENCES playbooks(playbook_id) ON DELETE CASCADE,

    -- Enrollment Criteria (fit_score_threshold, engagement_score_threshold, require_human_approval)
    enrollment_criteria JSONB NOT NULL DEFAULT '{}'::jsonb,

    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create contact_enrollments table
CREATE TABLE IF NOT EXISTS contact_enrollments (
    enrollment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    playbook_id UUID NOT NULL REFERENCES playbooks(playbook_id) ON DELETE CASCADE,
    automation_rule_id UUID REFERENCES automation_rules(automation_rule_id) ON DELETE SET NULL,

    enrollment_status VARCHAR(50) NOT NULL DEFAULT 'pending_approval', -- 'pending_approval', 'active', 'paused', 'completed'
    current_stage_order INTEGER NOT NULL DEFAULT 0,
    current_stage_id VARCHAR(100),

    enrolled_at TIMESTAMP,
    completed_at TIMESTAMP,

    -- Execution Log (stages_completed, events)
    execution_log JSONB NOT NULL DEFAULT '{"stages_completed": [], "events": []}'::jsonb,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(contact_id, playbook_id)
);

-- Create enrollment_approval_requests table
CREATE TABLE IF NOT EXISTS enrollment_approval_requests (
    request_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    playbook_id UUID NOT NULL REFERENCES playbooks(playbook_id) ON DELETE CASCADE,
    automation_rule_id UUID NOT NULL REFERENCES automation_rules(automation_rule_id) ON DELETE CASCADE,

    reason TEXT NOT NULL,
    fit_score INTEGER NOT NULL,
    engagement_score INTEGER NOT NULL,

    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'approved', 'rejected'
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_playbooks_type ON playbooks(playbook_type);
CREATE INDEX IF NOT EXISTS idx_playbooks_active ON playbooks(is_active);

CREATE INDEX IF NOT EXISTS idx_automation_rules_segment ON automation_rules(segment_id);
CREATE INDEX IF NOT EXISTS idx_automation_rules_playbook ON automation_rules(playbook_id);
CREATE INDEX IF NOT EXISTS idx_automation_rules_active ON automation_rules(is_active);

CREATE INDEX IF NOT EXISTS idx_contact_enrollments_contact ON contact_enrollments(contact_id);
CREATE INDEX IF NOT EXISTS idx_contact_enrollments_playbook ON contact_enrollments(playbook_id);
CREATE INDEX IF NOT EXISTS idx_contact_enrollments_status ON contact_enrollments(enrollment_status);

CREATE INDEX IF NOT EXISTS idx_enrollment_requests_contact ON enrollment_approval_requests(contact_id);
CREATE INDEX IF NOT EXISTS idx_enrollment_requests_status ON enrollment_approval_requests(status);
CREATE INDEX IF NOT EXISTS idx_enrollment_requests_created ON enrollment_approval_requests(created_at DESC);

-- Create GIN indexes for JSONB fields
CREATE INDEX IF NOT EXISTS idx_playbooks_config ON playbooks USING GIN (config);
CREATE INDEX IF NOT EXISTS idx_playbooks_performance ON playbooks USING GIN (performance);
CREATE INDEX IF NOT EXISTS idx_automation_rules_criteria ON automation_rules USING GIN (enrollment_criteria);
CREATE INDEX IF NOT EXISTS idx_contact_enrollments_log ON contact_enrollments USING GIN (execution_log);

-- Add updated_at triggers
CREATE TRIGGER update_playbooks_updated_at BEFORE UPDATE ON playbooks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_automation_rules_updated_at BEFORE UPDATE ON automation_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_contact_enrollments_updated_at BEFORE UPDATE ON contact_enrollments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
