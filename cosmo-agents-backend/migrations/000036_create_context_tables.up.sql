-- Organization Context - shared by all users in the org
CREATE TABLE IF NOT EXISTS org_contexts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID UNIQUE NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- ICP (Ideal Customer Profile) definition
    icp_definition TEXT,

    -- Target criteria (industries, company sizes, etc.)
    target_criteria JSONB DEFAULT '{}',

    -- Company knowledge base
    company_knowledge JSONB DEFAULT '{}',

    -- Common pain points for target market
    common_pain_points JSONB DEFAULT '[]',

    -- Common goals for target market
    common_goals JSONB DEFAULT '[]',

    -- Competitors information
    competitors JSONB DEFAULT '[]',

    -- Messaging guidelines
    messaging_guidelines TEXT,

    -- Custom AI instructions for the org
    ai_instructions TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User Context - private to each user
CREATE TABLE IF NOT EXISTS user_contexts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- User preferences for AI responses
    preferences JSONB DEFAULT '{}',

    -- Communication style: professional, casual, formal
    communication_style VARCHAR(50) DEFAULT 'professional',

    -- Preferred email length: short, medium, long
    preferred_email_length VARCHAR(20) DEFAULT 'medium',

    -- User's personal notes
    personal_notes TEXT,

    -- Recent contacts (for quick context)
    recent_contacts JSONB DEFAULT '[]',

    -- Custom AI instructions for this user
    ai_instructions TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Conversation History - for context continuity
CREATE TABLE IF NOT EXISTS conversation_histories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Optional: link to specific contact being discussed
    contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,

    -- Session ID to group related messages
    session_id VARCHAR(100),

    -- Message role: user, assistant, system
    role VARCHAR(20) NOT NULL,

    -- Message content
    content TEXT NOT NULL,

    -- Tools used in this message (if assistant)
    tools_used JSONB DEFAULT '[]',

    -- Metadata (tokens used, latency, etc.)
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_user_contexts_org ON user_contexts(organization_id);
CREATE INDEX IF NOT EXISTS idx_conversation_histories_user ON conversation_histories(user_id);
CREATE INDEX IF NOT EXISTS idx_conversation_histories_session ON conversation_histories(session_id);
CREATE INDEX IF NOT EXISTS idx_conversation_histories_contact ON conversation_histories(contact_id);
CREATE INDEX IF NOT EXISTS idx_conversation_histories_created ON conversation_histories(created_at);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_context_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_org_contexts_updated_at
    BEFORE UPDATE ON org_contexts
    FOR EACH ROW EXECUTE FUNCTION update_context_updated_at();

CREATE TRIGGER update_user_contexts_updated_at
    BEFORE UPDATE ON user_contexts
    FOR EACH ROW EXECUTE FUNCTION update_context_updated_at();
