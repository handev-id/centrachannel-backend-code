CREATE TABLE IF NOT EXISTS conversation_tag (
    id              SERIAL PRIMARY KEY,
    tenant_id       INTEGER     NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    conversation_id INTEGER     NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    tag_id          INTEGER     NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(conversation_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_conversation_tag_tenant ON conversation_tag(tenant_id);
CREATE INDEX IF NOT EXISTS idx_conversation_tag_conv ON conversation_tag(conversation_id);
