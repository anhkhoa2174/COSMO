-- Revert lifecycle stages back to legacy values
UPDATE contacts SET business_stage = 'PRE_SALES' WHERE business_stage = 'LEAD';
UPDATE contacts SET business_stage = 'SALES' WHERE business_stage = 'OPPORTUNITY';
UPDATE contacts SET business_stage = 'POST_SALES' WHERE business_stage = 'CUSTOMER';

-- Restore original default
ALTER TABLE contacts ALTER COLUMN business_stage SET DEFAULT 'PRE_SALES';
