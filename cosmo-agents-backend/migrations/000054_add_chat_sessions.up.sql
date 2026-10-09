-- Ask COSMO kept nothing: the widget held its thread in browser memory, so a
-- refresh lost the conversation. These two changes give it sessions.
--
-- chat_messages already existed but had no way to group a thread — every row
-- was just (user, role, content), so a history could be listed but not
-- reconstructed as separate conversations. The column is nullable so the rows
-- written before sessions existed stay valid and simply belong to no session.

CREATE TABLE IF NOT EXISTS chat_sessions (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title       TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The session list is always "this user's, most recently used first", which is
-- the only way it is read.
CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_recent
    ON chat_sessions (user_id, updated_at DESC);

ALTER TABLE chat_messages
    ADD COLUMN IF NOT EXISTS session_id UUID REFERENCES chat_sessions(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_chat_messages_session
    ON chat_messages (session_id, created_at);

COMMENT ON COLUMN chat_messages.session_id IS
    'The Ask COSMO conversation this message belongs to. NULL for messages written before sessions existed.';
