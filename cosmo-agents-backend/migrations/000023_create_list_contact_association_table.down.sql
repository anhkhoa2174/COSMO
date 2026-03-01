-- Restore legacy state by migrating data back to array column
DO $$
BEGIN
    -- Add the legacy array column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'list_contacts' AND column_name = 'contact_ids'
    ) THEN
        ALTER TABLE list_contacts ADD COLUMN contact_ids UUID[];
    END IF;
    
    -- Migrate data back from association table to array column
    UPDATE list_contacts 
    SET contact_ids = (
        SELECT array_agg(lca.contact_id)
        FROM list_contact_association lca 
        WHERE lca.list_contact_id = list_contacts.id
    )
    WHERE EXISTS (
        SELECT 1 FROM list_contact_association lca 
        WHERE lca.list_contact_id = list_contacts.id
    );
END $$;

DROP INDEX IF EXISTS idx_list_contact_association;
DROP TABLE IF EXISTS list_contact_association;
