CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    user_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    name TEXT NOT NULL DEFAULT 'admin',
    job_title TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    CONSTRAINT fk_roles_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_roles_organization_id FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    CONSTRAINT chk_roles_name CHECK (name IN ('admin','member')),
    CONSTRAINT chk_roles_status CHECK (status IN ('pending','active'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_user_org ON roles (user_id, organization_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_roles_name ON roles (user_id, organization_id, name);
