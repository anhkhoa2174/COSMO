-- Remove composite index for conversations (only drop what this migration created)
-- Note: GIN indexes are managed by migration 000019
DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_agent_id_not_deleted;