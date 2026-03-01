-- Migration 000034: Create user_feedback table for feedback capture

CREATE TABLE IF NOT EXISTS user_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    user_id UUID NOT NULL,
    entity_type TEXT NOT NULL, -- contact/segmentation/etc.
    entity_id UUID NOT NULL,
    feedback_type TEXT NOT NULL, -- score_adjustment/insight_validation/custom_fact
    feedback_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    applied BOOLEAN NOT NULL DEFAULT false,
    applied_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_user_feedback_user_id ON user_feedback(user_id);
CREATE INDEX IF NOT EXISTS idx_user_feedback_entity ON user_feedback(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_user_feedback_type ON user_feedback(feedback_type);
