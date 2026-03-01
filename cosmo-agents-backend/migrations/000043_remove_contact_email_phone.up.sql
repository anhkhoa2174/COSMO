-- Remove email and phone columns from contacts table
-- These fields are replaced by contact_information logic based on source

-- First, backup any non-placeholder emails to profile.email for reference
UPDATE contacts
SET profile = jsonb_set(
    COALESCE(profile, '{}'),
    '{backup_email}',
    to_jsonb(email)
)
WHERE email IS NOT NULL
  AND email != 'N/A'
  AND email NOT LIKE 'unknown-%'
  AND email NOT LIKE '%@linkedin.placeholder'
  AND email NOT LIKE '%@na.local';

-- Backup phones to profile.phone for reference
UPDATE contacts
SET profile = jsonb_set(
    COALESCE(profile, '{}'),
    '{backup_phone}',
    to_jsonb(phone)
)
WHERE phone IS NOT NULL
  AND phone != 'N/A'
  AND phone != '';

-- Drop email and phone columns
ALTER TABLE contacts DROP COLUMN IF EXISTS email;
ALTER TABLE contacts DROP COLUMN IF EXISTS phone;
