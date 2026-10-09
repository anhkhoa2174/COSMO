DROP INDEX IF EXISTS idx_chat_messages_session;
ALTER TABLE chat_messages DROP COLUMN IF EXISTS session_id;
DROP INDEX IF EXISTS idx_chat_sessions_user_recent;
DROP TABLE IF EXISTS chat_sessions;
