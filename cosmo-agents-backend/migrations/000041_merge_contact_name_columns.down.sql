-- Rollback: Restore first_name and last_name columns from name column

-- Step 1: Add back the first_name and last_name columns
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS first_name VARCHAR(255) DEFAULT 'N/A';
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS last_name VARCHAR(255) DEFAULT 'N/A';

-- Step 2: Attempt to split the name back into first_name and last_name
-- Uses the first word as first_name and the rest as last_name
UPDATE contacts
SET
    first_name = CASE
        WHEN name IS NOT NULL AND name != '' AND name != 'N/A'
        THEN SPLIT_PART(name, ' ', 1)
        ELSE 'N/A'
    END,
    last_name = CASE
        WHEN name IS NOT NULL AND name != '' AND name != 'N/A'
             AND POSITION(' ' IN name) > 0
        THEN TRIM(SUBSTRING(name FROM POSITION(' ' IN name) + 1))
        ELSE 'N/A'
    END;

-- Step 3: Drop the name column
ALTER TABLE contacts DROP COLUMN IF EXISTS name;
