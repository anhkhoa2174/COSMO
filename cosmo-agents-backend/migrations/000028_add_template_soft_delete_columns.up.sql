-- Migration 000028: Add soft delete to templates and update contacts schema

-- Part 1: Add soft delete columns to templates table
ALTER TABLE templates
    ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Ensure default false for existing rows
UPDATE templates SET is_deleted = FALSE WHERE is_deleted IS NULL;

-- Part 2: Add missing columns to contacts table

-- Add is_deleted column
ALTER TABLE contacts 
ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT false;

-- Add profile column
ALTER TABLE contacts 
ADD COLUMN IF NOT EXISTS profile JSONB DEFAULT '{}'::jsonb;

-- Add do_not_contact column
ALTER TABLE contacts 
ADD COLUMN IF NOT EXISTS do_not_contact BOOLEAN DEFAULT false;

-- Add organization_id column
ALTER TABLE contacts 
ADD COLUMN IF NOT EXISTS organization_id UUID;

-- Add tags column
ALTER TABLE contacts 
ADD COLUMN IF NOT EXISTS tags JSONB DEFAULT '{}'::jsonb;

-- Add foreign key constraint for organization_id
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'fk_contacts_organization_id'
    ) THEN
        ALTER TABLE contacts 
        ADD CONSTRAINT fk_contacts_organization_id 
        FOREIGN KEY (organization_id) 
        REFERENCES organizations(id) 
        ON DELETE CASCADE;
    END IF;
END $$;

-- Update unique constraint to match expected schema
DROP INDEX IF EXISTS idx_contacts_source_id_source;
CREATE UNIQUE INDEX IF NOT EXISTS uq_contacts_source_id_source_user_id 
ON contacts (source_id, source, user_id);

-- Create GIN index for profile jsonb column
CREATE INDEX IF NOT EXISTS idx_contacts_profile_gin 
ON contacts USING gin (profile jsonb_path_ops);

-- Create composite index for user filtering and sorting
CREATE INDEX IF NOT EXISTS idx_contacts_user_filtering_sort 
ON contacts (user_id, is_deleted, updated_at, id) 
INCLUDE (profile);
