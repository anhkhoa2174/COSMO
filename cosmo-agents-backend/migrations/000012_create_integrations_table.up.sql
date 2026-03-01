CREATE TABLE IF NOT EXISTS integrations (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id UUID NOT NULL,
    source TEXT NOT NULL,
    source_id TEXT NOT NULL,
    credential JSONB,
    config JSONB,
    CONSTRAINT fk_integrations_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_integrations_user_source ON integrations (user_id, source);
