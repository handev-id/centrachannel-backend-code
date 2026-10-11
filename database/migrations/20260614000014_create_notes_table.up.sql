CREATE TABLE IF NOT EXISTS notes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID     NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    text            TEXT        NOT NULL,
    date            DATE,
    conversation_id UUID     NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         UUID     REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notes_tenant ON notes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_notes_conversation ON notes(conversation_id);
