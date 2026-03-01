-- Tasks table depends on contacts, campaigns, templates
CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    contact_id UUID NOT NULL,
    campaign_id UUID NOT NULL,
    template_id UUID NOT NULL,
    schedule_at TIMESTAMPTZ,
    responded_at TIMESTAMPTZ,
    triggered_at TIMESTAMPTZ,
    done_at TIMESTAMPTZ,
    priority TEXT DEFAULT 'medium',
    payload JSONB DEFAULT '{"agent":null,"template":null}'::jsonb,
    status TEXT NOT NULL DEFAULT 'pending',
    error TEXT,
    CONSTRAINT fk_tasks_contact_id FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE,
    CONSTRAINT fk_tasks_campaign_id FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    CONSTRAINT fk_tasks_template_id FOREIGN KEY (template_id) REFERENCES templates(id) ON DELETE CASCADE,
    CONSTRAINT uq_tasks UNIQUE (contact_id, campaign_id, template_id)
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks (status);

CREATE TABLE IF NOT EXISTS task_updates (
    id UUID PRIMARY KEY,
    task_id UUID NOT NULL,
    update_type TEXT DEFAULT 'update',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_task_updates_task_id FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);
