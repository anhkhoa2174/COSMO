-- Add status and missing_fields columns to contacts table
-- status: 'ready' (has all required fields) or 'pending' (missing required fields)
-- missing_fields: JSON array of field names that are missing

ALTER TABLE contacts
ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'pending',
ADD COLUMN IF NOT EXISTS missing_fields JSONB DEFAULT '[]';

-- Create index for filtering by status
CREATE INDEX IF NOT EXISTS idx_contacts_status ON contacts(status);

-- Update existing contacts: set status based on required fields
-- Required: first_name, last_name, email (valid), company, job_title
UPDATE contacts
SET
    status = CASE
        WHEN (first_name IS NOT NULL AND first_name != '' AND first_name != 'N/A')
         AND (last_name IS NOT NULL AND last_name != '' AND last_name != 'N/A')
         AND (email IS NOT NULL AND email != '' AND email != 'N/A' AND email NOT LIKE 'unknown-%')
         AND (company IS NOT NULL AND company != '' AND company != 'N/A')
         AND (job_title IS NOT NULL AND job_title != '' AND job_title != 'N/A')
        THEN 'ready'
        ELSE 'pending'
    END,
    missing_fields = (
        SELECT jsonb_agg(field)
        FROM (
            SELECT 'first_name' AS field WHERE first_name IS NULL OR first_name = '' OR first_name = 'N/A'
            UNION ALL
            SELECT 'last_name' WHERE last_name IS NULL OR last_name = '' OR last_name = 'N/A'
            UNION ALL
            SELECT 'email' WHERE email IS NULL OR email = '' OR email = 'N/A' OR email LIKE 'unknown-%'
            UNION ALL
            SELECT 'company' WHERE company IS NULL OR company = '' OR company = 'N/A'
            UNION ALL
            SELECT 'job_title' WHERE job_title IS NULL OR job_title = '' OR job_title = 'N/A'
        ) AS missing
    )
WHERE is_deleted = false;

-- Set empty array for contacts with no missing fields
UPDATE contacts
SET missing_fields = '[]'
WHERE missing_fields IS NULL AND is_deleted = false;
