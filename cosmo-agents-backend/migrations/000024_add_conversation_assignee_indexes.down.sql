-- Remove indexes for conversation assignee queries

DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_assignee_not_null;
DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_assignee_filtering;
DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_user_status;