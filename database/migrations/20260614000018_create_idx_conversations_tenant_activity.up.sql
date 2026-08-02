CREATE INDEX IF NOT EXISTS idx_conversations_tenant_activity
    ON conversations (tenant_id, last_activity DESC NULLS LAST, id DESC);
