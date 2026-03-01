-- Migration 000033: Extend contacts for facts/insights/scores and add segmentation & interaction tables

-- Extend contacts with AI/facts fields and scores JSONB
ALTER TABLE contacts
    ADD COLUMN IF NOT EXISTS confirmed_facts JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS ai_insights JSONB NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS insight_validation JSONB NOT NULL DEFAULT '{"confirmed_insights": [], "rejected_insights": [], "pending_validation": []}'::jsonb,
    ADD COLUMN IF NOT EXISTS scores JSONB NOT NULL DEFAULT '{"fit": 0, "engagement": 0, "priority": 0}'::jsonb;

-- Backfill nulls to defaults (defensive)
UPDATE contacts SET confirmed_facts = '{}'::jsonb WHERE confirmed_facts IS NULL;
UPDATE contacts SET ai_insights = '{}'::jsonb WHERE ai_insights IS NULL;
UPDATE contacts SET insight_validation = '{"confirmed_insights": [], "rejected_insights": [], "pending_validation": []}'::jsonb WHERE insight_validation IS NULL;
UPDATE contacts SET scores = '{"fit": 0, "engagement": 0, "priority": 0}'::jsonb WHERE scores IS NULL;

-- Helpful JSONB indexes for querying
CREATE INDEX IF NOT EXISTS idx_contacts_confirmed_facts ON contacts USING GIN (confirmed_facts);
CREATE INDEX IF NOT EXISTS idx_contacts_ai_insights ON contacts USING GIN (ai_insights);
CREATE INDEX IF NOT EXISTS idx_contacts_scores ON contacts USING GIN (scores);

-- Segmentation definitions
CREATE TABLE IF NOT EXISTS segmentations (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    priority INTEGER NOT NULL DEFAULT 5,
    criteria JSONB NOT NULL DEFAULT '{"filters": [], "scoring_rules": [], "exclusions": []}'::jsonb,
    icp_definition JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    CONSTRAINT fk_segmentations_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT ck_segmentations_priority CHECK (priority >= 1 AND priority <= 10)
);

CREATE INDEX IF NOT EXISTS idx_segmentations_user_id ON segmentations (user_id);
CREATE INDEX IF NOT EXISTS idx_segmentations_priority ON segmentations (priority DESC);
CREATE INDEX IF NOT EXISTS idx_segmentations_active ON segmentations (is_active);
CREATE INDEX IF NOT EXISTS idx_segmentations_criteria ON segmentations USING GIN (criteria);

-- Contact ↔ Segment scores (dynamic membership)
CREATE TABLE IF NOT EXISTS contact_segment_scores (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    contact_id UUID NOT NULL,
    segmentation_id UUID NOT NULL,
    fit_score INTEGER NOT NULL DEFAULT 0,
    score_breakdown JSONB NOT NULL DEFAULT '{}'::jsonb,
    score_type TEXT NOT NULL DEFAULT 'auto', -- auto | manual
    status TEXT NOT NULL DEFAULT 'qualified', -- qualified | active | paused | excluded
    passes_filters BOOLEAN NOT NULL DEFAULT TRUE,
    enrolled_in_campaign BOOLEAN NOT NULL DEFAULT FALSE,
    current_campaign_id UUID,
    CONSTRAINT fk_contact_segment_contact FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE,
    CONSTRAINT fk_contact_segment_segmentation FOREIGN KEY (segmentation_id) REFERENCES segmentations(id) ON DELETE CASCADE,
    CONSTRAINT ck_contact_segment_fit_score CHECK (fit_score >= 0 AND fit_score <= 100),
    CONSTRAINT uq_contact_segment UNIQUE (contact_id, segmentation_id)
);

CREATE INDEX IF NOT EXISTS idx_contact_segment_contact ON contact_segment_scores (contact_id);
CREATE INDEX IF NOT EXISTS idx_contact_segment_segmentation ON contact_segment_scores (segmentation_id);
CREATE INDEX IF NOT EXISTS idx_contact_segment_fit_score ON contact_segment_scores (fit_score DESC);
CREATE INDEX IF NOT EXISTS idx_contact_segment_status ON contact_segment_scores (status);

-- Interaction log (email/linkedin/call/etc.)
CREATE TABLE IF NOT EXISTS interactions (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    contact_id UUID NOT NULL,
    campaign_id UUID,
    segmentation_id UUID,
    interaction_type TEXT NOT NULL,
    channel TEXT,
    direction TEXT,
    content JSONB NOT NULL DEFAULT '{}'::jsonb,
    ai_analysis JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    CONSTRAINT fk_interactions_contact_id FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE,
    CONSTRAINT fk_interactions_campaign_id FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE SET NULL,
    CONSTRAINT fk_interactions_segmentation_id FOREIGN KEY (segmentation_id) REFERENCES segmentations(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_interactions_contact ON interactions (contact_id);
CREATE INDEX IF NOT EXISTS idx_interactions_campaign ON interactions (campaign_id);
CREATE INDEX IF NOT EXISTS idx_interactions_type ON interactions (interaction_type);
CREATE INDEX IF NOT EXISTS idx_interactions_occurred ON interactions (occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_interactions_ai_analysis ON interactions USING GIN (ai_analysis);
