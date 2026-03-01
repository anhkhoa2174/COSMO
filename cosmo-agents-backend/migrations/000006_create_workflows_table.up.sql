CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    user_id UUID NOT NULL,
    nodes TEXT[] NOT NULL DEFAULT '{}'::text[],
    edges JSONB NOT NULL DEFAULT '[]'::jsonb,
    state JSONB NOT NULL DEFAULT '{}'::jsonb,
    cmetadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT fk_workflows_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_workflows_user_id ON workflows (user_id);
