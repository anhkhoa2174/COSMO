-- Add GIN indexes for array columns to optimize array filtering queries
-- These indexes are essential for efficient PostgreSQL array operations using ANY() operator
-- Note: Removed CONCURRENTLY to allow running inside transaction

-- Index for conversations.labels array column (used in GetConversations filtering)
CREATE INDEX IF NOT EXISTS idx_conversations_labels_gin ON conversations USING GIN (labels);

-- Index for conversations.intents array column (used in GetConversations filtering)  
CREATE INDEX IF NOT EXISTS idx_conversations_intents_gin ON conversations USING GIN (intents);

-- Index for emails.labels array column (used in email filtering and metrics)
CREATE INDEX IF NOT EXISTS idx_emails_labels_gin ON emails USING GIN (labels);

-- Index for emails.intents array column (used in email filtering)
CREATE INDEX IF NOT EXISTS idx_emails_intents_gin ON emails USING GIN (intents);

-- Composite index for agent_id + array columns for better performance on filtered queries
CREATE INDEX IF NOT EXISTS idx_conversations_agent_labels ON conversations (agent_id) WHERE labels IS NOT NULL AND array_length(labels, 1) > 0;
CREATE INDEX IF NOT EXISTS idx_conversations_agent_intents ON conversations (agent_id) WHERE intents IS NOT NULL AND array_length(intents, 1) > 0;