-- Add composite index for conversations filtering by agent_id and is_deleted with ordering by updated_at
-- This index covers the most common query pattern in GetConversations endpoint
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_conversations_agent_updated 
ON conversations (agent_id, updated_at DESC) 
WHERE is_deleted = false;