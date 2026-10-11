CREATE TABLE IF NOT EXISTS conversation_tag (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID     NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    conversation_id UUID     NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    tag_id          UUID     NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(conversation_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_conversation_tag_tenant ON conversation_tag(tenant_id);
CREATE INDEX IF NOT EXISTS idx_conversation_tag_conv ON conversation_tag(conversation_id);
