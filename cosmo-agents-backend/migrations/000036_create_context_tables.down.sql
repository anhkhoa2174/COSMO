DROP TRIGGER IF EXISTS update_user_contexts_updated_at ON user_contexts;
DROP TRIGGER IF EXISTS update_org_contexts_updated_at ON org_contexts;
DROP FUNCTION IF EXISTS update_context_updated_at();

DROP TABLE IF EXISTS conversation_histories;
DROP TABLE IF EXISTS user_contexts;
DROP TABLE IF EXISTS org_contexts;
