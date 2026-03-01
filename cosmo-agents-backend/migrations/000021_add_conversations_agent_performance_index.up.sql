-- Add composite index for conversations performance optimization
-- This index optimizes the FindByAgentIDWithFilters query which filters on (agent_id, is_deleted)
-- Note: GIN indexes for labels/intents are created in migration 000019
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_conversations_agent_id_not_deleted 
ON conversations (agent_id, is_deleted) 
WHERE is_deleted = false;