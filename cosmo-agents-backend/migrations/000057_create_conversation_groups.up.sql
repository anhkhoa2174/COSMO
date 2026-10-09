-- Personal groups for the AI Inbox. A representative files conversations under
-- names of their own choosing (a customer, a company, a product), so groups
-- belong to one user and are never shared: two people can both have a
-- "VIP" group and neither sees the other's.
CREATE TABLE IF NOT EXISTS conversation_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    color       TEXT NOT NULL DEFAULT 'slate',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT conversation_groups_name_not_blank CHECK (btrim(name) <> '')
);

-- One user cannot have two groups that read the same; "VIP" and "vip" count
-- as the same name.
CREATE UNIQUE INDEX IF NOT EXISTS uq_conversation_groups_user_name
    ON conversation_groups (user_id, lower(btrim(name)));

-- A conversation can sit in several of a user's groups at once.
CREATE TABLE IF NOT EXISTS conversation_group_members (
    group_id        UUID NOT NULL REFERENCES conversation_groups(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, conversation_id)
);

CREATE INDEX IF NOT EXISTS idx_conversation_group_members_conversation
    ON conversation_group_members (conversation_id);
