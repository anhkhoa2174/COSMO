-- Migration: Merge first_name and last_name into single name column
-- This migration consolidates the contact name fields into a single column

-- Step 1: Add the name column if it doesn't exist
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS name VARCHAR(255) DEFAULT 'N/A';

-- Step 2: Migrate existing data - combine first_name and last_name into name
UPDATE contacts
SET name = CASE
    WHEN first_name IS NOT NULL AND first_name != '' AND first_name != 'N/A'
         AND last_name IS NOT NULL AND last_name != '' AND last_name != 'N/A'
    THEN TRIM(first_name || ' ' || last_name)
    WHEN first_name IS NOT NULL AND first_name != '' AND first_name != 'N/A'
    THEN TRIM(first_name)
    WHEN last_name IS NOT NULL AND last_name != '' AND last_name != 'N/A'
    THEN TRIM(last_name)
    ELSE 'N/A'
END
WHERE name IS NULL OR name = '' OR name = 'N/A';

-- Step 3: Drop the old columns
ALTER TABLE contacts DROP COLUMN IF EXISTS first_name;
ALTER TABLE contacts DROP COLUMN IF EXISTS last_name;
