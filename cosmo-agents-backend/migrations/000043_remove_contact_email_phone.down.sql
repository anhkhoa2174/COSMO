-- Restore email and phone columns
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS email VARCHAR(255) DEFAULT 'N/A';
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS phone VARCHAR(50) DEFAULT 'N/A';

-- Restore backed up emails from profile
UPDATE contacts
SET email = profile->>'backup_email'
WHERE profile->>'backup_email' IS NOT NULL
  AND profile->>'backup_email' != '';

-- Restore backed up phones from profile
UPDATE contacts
SET phone = profile->>'backup_phone'
WHERE profile->>'backup_phone' IS NOT NULL
  AND profile->>'backup_phone' != '';
