-- Migration 000001: Create agents table
-- Prerequisite tables (users, organizations, etc.) are created in migration 000000

-- Ensure referenced tables exist (minimal schemas)
-- Users
CREATE TABLE IF NOT EXISTS users (
	id UUID PRIMARY KEY,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
	deleted_at TIMESTAMPTZ,
	email TEXT UNIQUE NOT NULL,
	name TEXT,
	picture TEXT,
	credentials JSON,
	last_history_id TEXT,
	provider TEXT,
	phone_number TEXT[] DEFAULT '{}'::text[],
	staff_emails TEXT[] DEFAULT '{}'::text[],
	hubspot_credentials JSON,
	job_title TEXT,
	hubspot_field_mapping JSON,
	ui_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
	meta_llived_access_token TEXT
);

-- Organizations
CREATE TABLE IF NOT EXISTS organizations (
	id UUID PRIMARY KEY,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
	deleted_at TIMESTAMPTZ,
	user_id UUID,
	name TEXT,
	company_url TEXT,
	company_description TEXT,
	company_targeting_persona TEXT[] DEFAULT '{}'::text[],
	value_offering TEXT,
	CONSTRAINT fk_organizations_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Agents table depends on users and organizations
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    user_id UUID NOT NULL,
    organization_id UUID,
    name TEXT,
    cmetadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    persona TEXT[] DEFAULT '{}'::text[],
    email TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    email_provider TEXT,
    signature TEXT NOT NULL DEFAULT E'Best regard,\n\n{sender_name}\n\n{organization_name}',
    picture TEXT,
    credentials JSON,
    last_history_id TEXT,
    daily_limit INTEGER DEFAULT 50,
    max_daily_limit INTEGER DEFAULT 500,
    valid_cred BOOLEAN DEFAULT TRUE,
    emails_sent_today INTEGER DEFAULT 0,
    CONSTRAINT fk_agents_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_agents_organization_id FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agents_user_id_email ON agents (user_id, email);
CREATE INDEX IF NOT EXISTS idx_agents_user_id ON agents (user_id);
CREATE INDEX IF NOT EXISTS idx_agents_organization_id ON agents (organization_id);
CREATE INDEX IF NOT EXISTS idx_agents_email ON agents (email) WHERE is_deleted = false;
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents (status) WHERE is_deleted = false;
CREATE INDEX IF NOT EXISTS idx_agents_daily_limits ON agents (daily_limit, emails_sent_today) WHERE valid_cred = true AND is_deleted = false;
CREATE INDEX IF NOT EXISTS idx_agents_valid_cred ON agents (valid_cred) WHERE is_deleted = false;
