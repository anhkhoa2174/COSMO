DO $$
BEGIN
    -- organization_id
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'campaigns' AND column_name = 'organization_id'
    ) THEN
        ALTER TABLE campaigns
            ADD COLUMN organization_id UUID;
    END IF;

    -- list_contact_id
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'campaigns' AND column_name = 'list_contact_id'
    ) THEN
        ALTER TABLE campaigns
            ADD COLUMN list_contact_id UUID;
    END IF;

    -- schedule
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'campaigns' AND column_name = 'schedule'
    ) THEN
        ALTER TABLE campaigns
            ADD COLUMN schedule TIMESTAMPTZ;
    END IF;

    -- status
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'campaigns' AND column_name = 'status'
    ) THEN
        ALTER TABLE campaigns
            ADD COLUMN status TEXT NOT NULL DEFAULT 'draft';
    END IF;

    -- agent_id
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'campaigns' AND column_name = 'agent_id'
    ) THEN
        ALTER TABLE campaigns
            ADD COLUMN agent_id UUID;
    END IF;

    -- cmetadata column (ensure exists with default)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'campaigns' AND column_name = 'cmetadata'
    ) THEN
        ALTER TABLE campaigns
            ADD COLUMN cmetadata JSONB NOT NULL DEFAULT '{"config":[]}'::jsonb;
    ELSE
        ALTER TABLE campaigns
            ALTER COLUMN cmetadata SET DEFAULT '{"config":[]}'::jsonb;
        UPDATE campaigns
            SET cmetadata = '{"config":[]}'::jsonb
            WHERE cmetadata IS NULL;
    END IF;

    -- sale rep node states
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'campaigns' AND column_name = '_sale_rep_node_states'
    ) THEN
        ALTER TABLE campaigns
            ADD COLUMN _sale_rep_node_states JSONB;
    END IF;

    -- is_deleted
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'campaigns' AND column_name = 'is_deleted'
    ) THEN
        ALTER TABLE campaigns
            ADD COLUMN is_deleted BOOLEAN NOT NULL DEFAULT FALSE;
    END IF;

    -- deleted_at
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'campaigns' AND column_name = 'deleted_at'
    ) THEN
        ALTER TABLE campaigns
            ADD COLUMN deleted_at TIMESTAMPTZ;
    END IF;
END
$$;

-- Ensure foreign key constraints exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_campaigns_organization_id'
    ) THEN
        ALTER TABLE campaigns
            ADD CONSTRAINT fk_campaigns_organization_id
            FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_campaigns_list_contact_id'
    ) THEN
        ALTER TABLE campaigns
            ADD CONSTRAINT fk_campaigns_list_contact_id
            FOREIGN KEY (list_contact_id) REFERENCES list_contacts(id) ON DELETE SET NULL;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_campaigns_agent_id'
    ) THEN
        ALTER TABLE campaigns
            ADD CONSTRAINT fk_campaigns_agent_id
            FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE SET NULL;
    END IF;
END
$$;
