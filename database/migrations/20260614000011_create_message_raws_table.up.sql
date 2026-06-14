CREATE TABLE IF NOT EXISTS message_raws (
    id                               SERIAL PRIMARY KEY,
    tenant_id                        INTEGER      NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    webhook_message_id_by_external_id VARCHAR(255) NOT NULL UNIQUE,
    data                             JSONB,
    created_at                       TIMESTAMPTZ,
    updated_at                       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_message_raws_tenant ON message_raws(tenant_id);
