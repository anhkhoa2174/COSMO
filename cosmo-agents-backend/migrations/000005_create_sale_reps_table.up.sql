CREATE TABLE IF NOT EXISTS sale_reps (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    first_name TEXT,
    last_name TEXT,
    email TEXT NOT NULL,
    calendar_link TEXT,
    picture TEXT,
    user_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    CONSTRAINT fk_salereps_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_salereps_organization_id FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_salereps_user_id_email ON sale_reps (user_id, email);
CREATE INDEX IF NOT EXISTS idx_salereps_organization_id ON sale_reps (organization_id);
