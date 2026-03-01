CREATE TABLE IF NOT EXISTS operations (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'in_progress',
    input JSONB,
    output JSONB
);
