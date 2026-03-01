-- Remove GIN indexes for array columns

DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_labels_gin;
DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_intents_gin;
DROP INDEX CONCURRENTLY IF EXISTS idx_emails_labels_gin;
DROP INDEX CONCURRENTLY IF EXISTS idx_emails_intents_gin;
DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_agent_labels;
DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_agent_intents;