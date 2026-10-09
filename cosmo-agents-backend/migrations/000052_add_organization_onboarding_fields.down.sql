ALTER TABLE organizations
    DROP COLUMN IF EXISTS crm,
    DROP COLUMN IF EXISTS lead_handling,
    DROP COLUMN IF EXISTS lead_handling_other;
