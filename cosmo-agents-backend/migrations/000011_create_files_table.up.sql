CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    user_id UUID,
    filename TEXT,
    mime_type TEXT,
    s3_key TEXT NOT NULL,
    size BIGINT,
    CONSTRAINT fk_files_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_files_user_id ON files (user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_files_s3_key ON files (s3_key);
