-- Create many-to-many association between list_contacts and contacts
CREATE TABLE IF NOT EXISTS list_contact_association (
    id UUID PRIMARY KEY,
    list_contact_id UUID,
    contact_id UUID,
    CONSTRAINT fk_lca_list_contact_id FOREIGN KEY (list_contact_id) REFERENCES list_contacts(id) ON DELETE CASCADE,
    CONSTRAINT fk_lca_contact_id FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_list_contact_association ON list_contact_association (list_contact_id, contact_id);

-- Migrate existing contact_ids data to association table (if column exists and has data)
DO $$
BEGIN
    -- Check if contact_ids column exists before attempting migration
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'list_contacts' AND column_name = 'contact_ids'
    ) THEN
        -- Migrate data from array column to association table
        INSERT INTO list_contact_association (id, list_contact_id, contact_id)
        SELECT 
            gen_random_uuid(),
            lc.id,
            unnest(lc.contact_ids) as contact_id
        FROM list_contacts lc
        WHERE lc.contact_ids IS NOT NULL AND array_length(lc.contact_ids, 1) > 0
        ON CONFLICT (list_contact_id, contact_id) DO NOTHING;
        
        -- Remove legacy array column after data migration
        ALTER TABLE list_contacts DROP COLUMN contact_ids;
    END IF;
END $$;
