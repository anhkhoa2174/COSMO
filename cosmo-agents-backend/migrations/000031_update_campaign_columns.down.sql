ALTER TABLE campaigns
    DROP CONSTRAINT IF EXISTS fk_campaigns_agent_id,
    DROP CONSTRAINT IF EXISTS fk_campaigns_list_contact_id,
    DROP CONSTRAINT IF EXISTS fk_campaigns_organization_id;

ALTER TABLE campaigns
    DROP COLUMN IF EXISTS _sale_rep_node_states,
    DROP COLUMN IF EXISTS agent_id,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS schedule,
    DROP COLUMN IF EXISTS list_contact_id,
    DROP COLUMN IF EXISTS organization_id,
    DROP COLUMN IF EXISTS is_deleted,
    DROP COLUMN IF EXISTS deleted_at;

-- Only drop cmetadata if there was no original column.
-- Since the legacy schema already had cmetadata, we leave it in place.
