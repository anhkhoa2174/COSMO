-- The contact list had an index built for a query it no longer runs.
--
-- idx_contacts_user_filtering_sort covers (user_id, is_deleted, updated_at, id)
-- and was the right shape when the list was "this representative's contacts".
-- Contact search now filters by organization instead, so that every member of
-- an organization sees the same book:
--
--     rawFilter := map[string]interface{}{
--         "is_deleted":      false,
--         "organization_id": organizationID,
--     }
--
-- No index leads with organization_id, so the query had nothing to use. On a
-- seeded 100,000-contact organization the planner chose a parallel sequential
-- scan followed by a top-N heapsort: 7,769 buffers and 205 ms for a single
-- page of 50, with no concurrent load at all. NFR-02 budgets 300 ms at p95 for
-- exactly this query at exactly this size, so it was failing before the first
-- concurrent user arrived.
--
-- With this index the same page is an index scan touching 8 buffers in 0.3 ms.
--
-- Column order follows the query: organization_id and is_deleted are equality
-- predicates and come first, updated_at DESC supplies the sort so no sort node
-- is needed, and id breaks ties so that pagination cannot skip or repeat a row
-- when two contacts share a timestamp.

CREATE INDEX IF NOT EXISTS idx_contacts_org_filtering_sort
    ON contacts (organization_id, is_deleted, updated_at DESC, id);

COMMENT ON INDEX idx_contacts_org_filtering_sort IS
    'Serves the paginated contact list, which filters by organization and sorts by updated_at DESC.';
