ALTER TABLE campaigns
    DROP COLUMN IF EXISTS deleted_at,
    DROP COLUMN IF EXISTS is_deleted;
