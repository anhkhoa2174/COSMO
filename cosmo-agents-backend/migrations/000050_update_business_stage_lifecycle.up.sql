-- Migrate business_stage from legacy values to lifecycle stages
UPDATE contacts SET business_stage = 'LEAD' WHERE business_stage = 'PRE_SALES';
UPDATE contacts SET business_stage = 'OPPORTUNITY' WHERE business_stage = 'SALES';
UPDATE contacts SET business_stage = 'CUSTOMER' WHERE business_stage = 'POST_SALES';

-- Update default
ALTER TABLE contacts ALTER COLUMN business_stage SET DEFAULT 'LEAD';
