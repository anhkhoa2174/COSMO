-- Add contact_information column to contacts table
-- This field stores the primary contact method:
-- - For LinkedIn sources: stores linkedin_url
-- - For Apollo/other sources: stores email
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS contact_information VARCHAR(500) DEFAULT '';

-- Backfill contact_information from profile for existing contacts
-- For LinkedIn sources, use linkedin_url from profile
UPDATE contacts
SET contact_information = profile->>'linkedin_url'
WHERE (source = 'LinkedIn' OR source = 'linkedin_extension' OR source = 'linkedin_connections')
  AND profile->>'linkedin_url' IS NOT NULL
  AND profile->>'linkedin_url' != ''
  AND (contact_information IS NULL OR contact_information = '');

-- For other sources, use email from profile
UPDATE contacts
SET contact_information = profile->>'email'
WHERE source NOT IN ('LinkedIn', 'linkedin_extension', 'linkedin_connections')
  AND profile->>'email' IS NOT NULL
  AND profile->>'email' != ''
  AND profile->>'email' != 'N/A'
  AND (contact_information IS NULL OR contact_information = '');

-- Create index for contact_information lookups
CREATE INDEX IF NOT EXISTS idx_contacts_contact_information ON contacts(contact_information) WHERE contact_information != '';
