-- Add business_stage column to contacts table
-- Business stages: PRE_SALES, SALES, POST_SALES
-- Defines the company-wide pipeline stage for a contact

ALTER TABLE contacts ADD COLUMN IF NOT EXISTS business_stage VARCHAR(20) NOT NULL DEFAULT 'PRE_SALES';

-- Create index for business_stage filtering
CREATE INDEX IF NOT EXISTS idx_contacts_business_stage ON contacts(business_stage);

-- Comment for documentation
COMMENT ON COLUMN contacts.business_stage IS 'Pipeline stage: PRE_SALES (prospecting), SALES (negotiation), POST_SALES (customer expansion)';
