-- Core bootstrap tables to satisfy FKs for later migrations

-- Contacts
CREATE TABLE IF NOT EXISTS contacts (
	id UUID PRIMARY KEY,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	user_id UUID NOT NULL,
	source_id TEXT,
	hubspot_id TEXT,
	source TEXT,
	first_name TEXT,
	last_name TEXT,
	email TEXT,
	phone TEXT,
	company TEXT,
	job_title TEXT,
	address TEXT,
	city TEXT,
	country TEXT,
	state TEXT,
	zip TEXT,
	CONSTRAINT fk_contacts_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_contacts_source_id_source ON contacts (source_id, source);
CREATE INDEX IF NOT EXISTS ix_contacts_user_id ON contacts (user_id);

-- List Contacts
CREATE TABLE IF NOT EXISTS list_contacts (
	id UUID PRIMARY KEY,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	user_id UUID NOT NULL,
	name TEXT,
	contact_ids UUID[] DEFAULT '{}'::uuid[],
	source TEXT,
	source_id TEXT,
	hubspot_id TEXT,
	CONSTRAINT fk_list_contacts_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_list_contacts_source_id_source_user_id ON list_contacts (source_id, source, user_id);
CREATE INDEX IF NOT EXISTS ix_list_contacts_user_id ON list_contacts (user_id);

-- Campaigns
CREATE TABLE IF NOT EXISTS campaigns (
	id UUID PRIMARY KEY,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	name TEXT,
	playbook TEXT,
	user_id UUID NOT NULL,
	list_contact_id UUID,
	cmetadata JSONB NOT NULL DEFAULT '{}'::jsonb,
	CONSTRAINT fk_campaigns_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	CONSTRAINT fk_campaigns_list_contact_id FOREIGN KEY (list_contact_id) REFERENCES list_contacts(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS ix_campaigns_user_id ON campaigns (user_id);

-- Templates (for ordering before tasks and emails)
CREATE TABLE IF NOT EXISTS templates (
	id UUID PRIMARY KEY,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	user_id UUID NOT NULL,
	campaign_id UUID,
	type TEXT,
	category TEXT,
	subject TEXT,
	content TEXT,
	cmetadata JSONB NOT NULL DEFAULT '{}'::jsonb,
	position DOUBLE PRECISION NOT NULL,
	send_after INTEGER NOT NULL DEFAULT 0,
	CONSTRAINT fk_templates_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	CONSTRAINT fk_templates_campaign_id FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
	CONSTRAINT uq_templates_type UNIQUE (user_id, campaign_id, type)
);
CREATE INDEX IF NOT EXISTS idx_templates_user_id ON templates (user_id);
CREATE INDEX IF NOT EXISTS idx_templates_campaign_id ON templates (campaign_id);
CREATE INDEX IF NOT EXISTS idx_templates_category_campaign_id ON templates (category, campaign_id);
