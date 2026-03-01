CREATE TABLE IF NOT EXISTS facebook_tokens (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id UUID NOT NULL,
    fb_user_id TEXT NOT NULL,
    token_type TEXT NOT NULL,
    access_token TEXT NOT NULL,
    token_expires_at TIMESTAMPTZ,
    page_id TEXT,
    page_name TEXT,
    CONSTRAINT fk_facebook_tokens_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT uq_user_token_type UNIQUE (user_id, token_type),
    CONSTRAINT uq_page_id UNIQUE (page_id)
);
