-- Revert Migration 000033

-- Drop interaction indexes then table
DROP INDEX IF EXISTS idx_interactions_ai_analysis;
DROP INDEX IF EXISTS idx_interactions_occurred;
DROP INDEX IF EXISTS idx_interactions_type;
DROP INDEX IF EXISTS idx_interactions_campaign;
DROP INDEX IF EXISTS idx_interactions_contact;
DROP TABLE IF EXISTS interactions;

-- Drop contact ↔ segment score indexes then table
DROP INDEX IF EXISTS idx_contact_segment_status;
DROP INDEX IF EXISTS idx_contact_segment_fit_score;
DROP INDEX IF EXISTS idx_contact_segment_segmentation;
DROP INDEX IF EXISTS idx_contact_segment_contact;
DROP TABLE IF EXISTS contact_segment_scores;

-- Drop segmentation indexes then table
DROP INDEX IF EXISTS idx_segmentations_criteria;
DROP INDEX IF EXISTS idx_segmentations_active;
DROP INDEX IF EXISTS idx_segmentations_priority;
DROP INDEX IF EXISTS idx_segmentations_user_id;
DROP TABLE IF EXISTS segmentations;

-- Drop contact JSONB indexes
DROP INDEX IF EXISTS idx_contacts_scores;
DROP INDEX IF EXISTS idx_contacts_ai_insights;
DROP INDEX IF EXISTS idx_contacts_confirmed_facts;

-- Remove added columns from contacts
ALTER TABLE contacts
    DROP COLUMN IF EXISTS scores,
    DROP COLUMN IF EXISTS insight_validation,
    DROP COLUMN IF EXISTS ai_insights,
    DROP COLUMN IF EXISTS confirmed_facts;
