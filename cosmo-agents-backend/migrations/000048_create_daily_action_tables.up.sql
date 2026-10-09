-- Daily Action Generations
CREATE TABLE IF NOT EXISTS daily_action_generations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'started',
    language VARCHAR(10) DEFAULT 'vi',
    action_count INTEGER DEFAULT 0,
    generated_at TIMESTAMPTZ,
    replaced_by_id UUID,
    agent_briefing JSONB DEFAULT '{}',
    pipeline_summary JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE UNIQUE INDEX idx_gen_user_date ON daily_action_generations(user_id, date) WHERE is_deleted = FALSE AND replaced_by_id IS NULL;

-- Daily Actions
CREATE TABLE IF NOT EXISTS daily_actions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    generation_id UUID NOT NULL REFERENCES daily_action_generations(id),
    contact_id UUID NOT NULL REFERENCES contacts(id),
    type VARCHAR(20) NOT NULL,
    category_id VARCHAR(20) NOT NULL,
    priority INTEGER NOT NULL,
    priority_factors JSONB DEFAULT '[]',
    reasoning TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'suggested',
    status_changed_at TIMESTAMPTZ,
    snooze_until TIMESTAMPTZ,
    outreach_data JSONB,
    meeting_data JSONB,
    enrichment_data JSONB,
    respond_data JSONB,
    contact JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX idx_action_user_gen ON daily_actions(user_id, generation_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_action_category ON daily_actions(category_id, priority) WHERE is_deleted = FALSE;
CREATE INDEX idx_action_contact ON daily_actions(contact_id) WHERE is_deleted = FALSE;
CREATE INDEX idx_action_priority ON daily_actions(user_id, priority) WHERE is_deleted = FALSE AND status = 'suggested';

-- Action Snoozes
CREATE TABLE IF NOT EXISTS action_snoozes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    action_id UUID NOT NULL REFERENCES daily_actions(id),
    snooze_until TIMESTAMPTZ NOT NULL,
    cleared_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_snooze_user ON action_snoozes(user_id) WHERE cleared_at IS NULL;
CREATE INDEX idx_snooze_action ON action_snoozes(action_id) WHERE cleared_at IS NULL;

-- Action Completion Logs (append-only)
CREATE TABLE IF NOT EXISTS action_completion_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    action_id UUID NOT NULL REFERENCES daily_actions(id),
    action_type VARCHAR(20) NOT NULL,
    transition VARCHAR(20) NOT NULL,
    contact_id UUID NOT NULL,
    contact_name VARCHAR(255),
    content TEXT,
    channel VARCHAR(50),
    skip_reason TEXT,
    feedback VARCHAR(50),
    date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_log_user_date ON action_completion_logs(user_id, date);

-- SSE Events (transient, for replay support)
CREATE TABLE IF NOT EXISTS sse_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    event_type VARCHAR(30) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sse_user_time ON sse_events(user_id, created_at);

-- Chat Messages
CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    classified_intent VARCHAR(50),
    generation_id UUID REFERENCES daily_action_generations(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_chat_user ON chat_messages(user_id, created_at);
