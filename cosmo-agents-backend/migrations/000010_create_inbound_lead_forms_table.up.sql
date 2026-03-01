CREATE TABLE IF NOT EXISTS inbound_lead_forms (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id UUID,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    ui_metadata JSONB,
    CONSTRAINT fk_inbound_lead_forms_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_inbound_lead_forms_user_id ON inbound_lead_forms (user_id);

CREATE TABLE IF NOT EXISTS form_fields (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    form_id UUID NOT NULL,
    custom_field_id UUID,
    display_name TEXT NOT NULL,
    is_system_field BOOLEAN NOT NULL DEFAULT FALSE,
    system_field_name TEXT,
    ui_metadata JSONB,
    is_required BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT fk_form_fields_form_id FOREIGN KEY (form_id) REFERENCES inbound_lead_forms(id) ON DELETE CASCADE,
    CONSTRAINT fk_form_fields_custom_field_id FOREIGN KEY (custom_field_id) REFERENCES custom_fields(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_form_field_custom ON form_fields (form_id, custom_field_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_form_field_system ON form_fields (form_id, system_field_name);
CREATE INDEX IF NOT EXISTS idx_form_fields_form_id ON form_fields (form_id);

CREATE TABLE IF NOT EXISTS inbound_lead_form_list_contact_association (
    id UUID PRIMARY KEY,
    inbound_lead_form_id UUID NOT NULL,
    list_contact_id UUID NOT NULL,
    CONSTRAINT fk_inbound_lead_form_assoc_form FOREIGN KEY (inbound_lead_form_id) REFERENCES inbound_lead_forms(id) ON DELETE CASCADE,
    CONSTRAINT fk_inbound_lead_form_assoc_list FOREIGN KEY (list_contact_id) REFERENCES list_contacts(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_inbound_lead_form_list_contact ON inbound_lead_form_list_contact_association (inbound_lead_form_id, list_contact_id);
