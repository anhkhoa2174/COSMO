CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    user_id UUID NOT NULL,
    gmail_thread_id TEXT UNIQUE,
    labels TEXT[] NOT NULL DEFAULT '{}'::text[],
    replied BOOLEAN NOT NULL DEFAULT FALSE,
    status TEXT NOT NULL DEFAULT 'unread',
    campaign_id UUID,
    assignee_id UUID,
    intents TEXT[] NOT NULL DEFAULT '{}'::text[],
    cmetadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    agent_id UUID,
    CONSTRAINT fk_conversations_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_conversations_campaign_id FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    CONSTRAINT fk_conversations_assignee_id FOREIGN KEY (assignee_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_conversations_agent_id FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations (user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_status ON conversations (status);

-- Emails (depends on conversations, campaigns, users)
CREATE TABLE IF NOT EXISTS emails (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    user_id UUID NOT NULL,
    conversation_id UUID,
    gmail_message_id TEXT,
    from_email TEXT,
    to_email TEXT,
    subject TEXT,
    content TEXT,
    attachments UUID[] NOT NULL DEFAULT '{}'::uuid[],
    labels TEXT[] NOT NULL DEFAULT '{}'::text[],
    status TEXT NOT NULL DEFAULT 'sending',
    campaign_id UUID,
    intents TEXT[] NOT NULL DEFAULT '{}'::text[],
    CONSTRAINT fk_emails_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_emails_conversation_id FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    CONSTRAINT fk_emails_campaign_id FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_emails_user_id_gmail_message_id ON emails (user_id, gmail_message_id);
CREATE INDEX IF NOT EXISTS idx_emails_user_id ON emails (user_id);
CREATE INDEX IF NOT EXISTS idx_emails_status ON emails (status);

CREATE TABLE IF NOT EXISTS knowledges (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    source_type TEXT NOT NULL DEFAULT 'upload',
    summary_pair TEXT[] NOT NULL DEFAULT '{}'::text[],
    embedding_gid TEXT UNIQUE,
    coze_dataset_id TEXT UNIQUE,
    user_id UUID NOT NULL,
    cmetadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT fk_knowledges_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_knowledges_user_id ON knowledges (user_id);

CREATE TABLE IF NOT EXISTS template_knowledges (
    id UUID PRIMARY KEY,
    template_id UUID NOT NULL,
    knowledge_id UUID NOT NULL,
    CONSTRAINT fk_template_knowledges_template_id FOREIGN KEY (template_id) REFERENCES templates(id) ON DELETE CASCADE,
    CONSTRAINT fk_template_knowledges_knowledge_id FOREIGN KEY (knowledge_id) REFERENCES knowledges(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_template_knowledges_pair ON template_knowledges (template_id, knowledge_id);
