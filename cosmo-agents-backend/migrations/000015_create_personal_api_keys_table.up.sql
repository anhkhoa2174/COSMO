CREATE TABLE IF NOT EXISTS personal_api_keys (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id UUID NOT NULL,
    name TEXT NOT NULL,
    hashed_key TEXT NOT NULL,
    prefix TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    CONSTRAINT fk_personal_api_keys_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_personal_api_keys_user_id ON personal_api_keys (user_id);
