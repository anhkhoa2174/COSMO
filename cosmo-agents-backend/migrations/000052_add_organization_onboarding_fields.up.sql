-- Onboarding step 3 asked which CRM the team uses and how they currently handle
-- leads, but had nowhere to store the answers. These columns give them a home.
ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS crm TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS lead_handling TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS lead_handling_other TEXT NOT NULL DEFAULT '';
