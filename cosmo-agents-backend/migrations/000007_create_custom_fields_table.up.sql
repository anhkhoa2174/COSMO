CREATE TABLE IF NOT EXISTS custom_fields (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id UUID NOT NULL,
    organization_id UUID,
    name TEXT NOT NULL,
    normalized_name TEXT NOT NULL,
    data_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    options TEXT[],
    sample_data TEXT,
    fallback_value TEXT,
    CONSTRAINT fk_custom_fields_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_custom_fields_organization_id FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT chk_custom_fields_data_type CHECK (data_type IN ('text','number','email','select','date','url')),
    CONSTRAINT chk_custom_fields_entity_type CHECK (entity_type IN ('contact','company'))
);

CREATE INDEX IF NOT EXISTS idx_custom_fields_user_id ON custom_fields (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_custom_fields_org_name_entity ON custom_fields (organization_id, name, entity_type);
