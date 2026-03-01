-- Add deleted_at column to contacts table for soft delete support
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Create index on deleted_at for better query performance
CREATE INDEX IF NOT EXISTS idx_contacts_deleted_at ON contacts (deleted_at);
