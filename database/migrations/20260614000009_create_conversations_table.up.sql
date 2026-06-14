CREATE TABLE IF NOT EXISTS conversations (
    id            SERIAL PRIMARY KEY,
    tenant_id     INTEGER     NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    status        TEXT        NOT NULL,
    profile_id    INTEGER     NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    agent_id      INTEGER     REFERENCES users(id) ON DELETE SET NULL,
    channel_id    INTEGER     NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    last_agent_id INTEGER     REFERENCES users(id) ON DELETE SET NULL,
    unread_count  INTEGER     NOT NULL DEFAULT 0,
    last_message  JSONB,
    last_activity TIMESTAMPTZ,
    last_seen     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_conversations_status CHECK (status IN ('unassigned', 'assigned', 'resolved'))
);

CREATE INDEX IF NOT EXISTS idx_conversations_tenant ON conversations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_conversations_status ON conversations(tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_conversations_agent ON conversations(tenant_id, agent_id);
CREATE INDEX IF NOT EXISTS idx_conversations_profile ON conversations(profile_id);
