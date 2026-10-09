-- ============================================================
-- Migration: Create outcome_metrics table
-- Feature: 006-agentic-daily-actions
-- ============================================================

CREATE TABLE IF NOT EXISTS outcome_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    period VARCHAR(10) NOT NULL DEFAULT '30d',
    total_sent INTEGER NOT NULL DEFAULT 0,
    total_replied INTEGER NOT NULL DEFAULT 0,
    total_meetings INTEGER NOT NULL DEFAULT 0,
    reply_rate_overall FLOAT NOT NULL DEFAULT 0,
    reply_rate_by_channel JSONB DEFAULT '{}',
    reply_rate_by_strategy JSONB DEFAULT '{}',
    reply_rate_by_industry JSONB DEFAULT '{}',
    reply_rate_by_time_of_day JSONB DEFAULT '{}',
    avg_messages_to_meeting FLOAT DEFAULT 0,
    top_performing_strategies JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_outcome_metrics_user_period ON outcome_metrics(user_id, period);
CREATE INDEX idx_outcome_metrics_user ON outcome_metrics(user_id);
