-- A knowledge document had no notion of what it was about, so retrieval could
-- not prefer the pricing sheet for a pricing question. Existing rows become
-- 'other', which retrieval treats as "no preference" — they keep being found
-- exactly as before.
ALTER TABLE knowledges
    ADD COLUMN IF NOT EXISTS type TEXT NOT NULL DEFAULT 'other';

ALTER TABLE knowledges
    DROP CONSTRAINT IF EXISTS knowledges_type_check;
ALTER TABLE knowledges
    ADD CONSTRAINT knowledges_type_check
    CHECK (type IN ('pricing', 'product', 'case_study', 'faq', 'other'));

COMMENT ON COLUMN knowledges.type IS
    'What the document is about: pricing, product, case_study, faq or other. Retrieval prefers the matching type for an inbound reply''s intent.';
