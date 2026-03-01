-- Remove composite index for conversations filtering by agent_id and ordering
DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_agent_updated;