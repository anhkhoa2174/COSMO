CREATE TABLE IF NOT EXISTS draft_templates (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    intent TEXT NOT NULL,
    campaign_id UUID NOT NULL,
    template_id UUID NOT NULL,
    CONSTRAINT fk_draft_templates_campaign_id FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    CONSTRAINT fk_draft_templates_template_id FOREIGN KEY (template_id) REFERENCES templates(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_draft_templates_campaign_intent ON draft_templates (campaign_id, intent);
