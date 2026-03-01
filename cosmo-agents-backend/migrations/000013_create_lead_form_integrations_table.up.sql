CREATE TABLE IF NOT EXISTS lead_form_integrations (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    campaign_id UUID NOT NULL,
    integration_type TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT fk_lead_form_integrations_campaign_id FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS lead_field_mappings (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    campaign_id UUID NOT NULL,
    form_integration_id UUID NOT NULL,
    custom_field_id UUID,
    external_field_name TEXT NOT NULL,
    mapping_type TEXT NOT NULL,
    contact_field_name TEXT,
    CONSTRAINT fk_lead_field_mappings_campaign_id FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    CONSTRAINT fk_lead_field_mappings_form FOREIGN KEY (form_integration_id) REFERENCES lead_form_integrations(id) ON DELETE CASCADE,
    CONSTRAINT fk_lead_field_mappings_custom_field FOREIGN KEY (custom_field_id) REFERENCES custom_fields(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_integration_external_field ON lead_field_mappings (form_integration_id, external_field_name);
CREATE INDEX IF NOT EXISTS idx_lead_field_mappings_integration_id ON lead_field_mappings (form_integration_id);
