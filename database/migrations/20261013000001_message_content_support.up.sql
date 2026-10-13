ALTER TABLE messages ADD COLUMN edited_at TIMESTAMPTZ, ADD COLUMN quote JSONB;

-- Latest provider mutation per target/actor. Keeping removals prevents stale
-- reactions from reappearing; keeping edits also handles arrival before target.
CREATE TABLE message_mutations (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL,
    target_provider_id VARCHAR(255) NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('edit','reaction')),
    actor_key TEXT NOT NULL,
    payload JSONB NOT NULL,
    event_at TIMESTAMPTZ NOT NULL,
    event_id TEXT NOT NULL,
    PRIMARY KEY (tenant_id, profile_id, channel_id, target_provider_id, kind, actor_key)
);
