CREATE TABLE IF NOT EXISTS messages (
    id                      SERIAL PRIMARY KEY,
    tenant_id               INTEGER      NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    text                    TEXT,
    attachment              JSONB,
    status                  TEXT         NOT NULL,
    sender_id               INTEGER      NOT NULL,
    sender_type             TEXT         NOT NULL,
    webhook_message_id      VARCHAR(255),
    webhook_message_reply_id VARCHAR(255),
    conversation_id         INTEGER      NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_messages_status CHECK (status IN ('sent', 'delivered', 'read', 'unread')),
    CONSTRAINT chk_messages_sender_type CHECK (sender_type IN ('contact', 'user', 'ai'))
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_tenant ON messages(tenant_id);
