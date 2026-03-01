-- Add indexes to optimize conversation assignee queries and filtering
-- Note: Removed CONCURRENTLY to allow running inside transaction

-- Index for conversations where assignee_id is NOT NULL (used in GetDistinctAssignees)
CREATE INDEX IF NOT EXISTS idx_conversations_assignee_not_null 
ON conversations (user_id, assignee_id) 
WHERE assignee_id IS NOT NULL AND is_deleted = false;

-- Index for conversation assignment filtering (used in SearchConversations)
CREATE INDEX IF NOT EXISTS idx_conversations_assignee_filtering 
ON conversations (assignee_id, is_deleted) 
WHERE is_deleted = false;

-- Composite index for user conversations with status filtering
CREATE INDEX IF NOT EXISTS idx_conversations_user_status 
ON conversations (user_id, status, is_deleted) 
WHERE is_deleted = false;