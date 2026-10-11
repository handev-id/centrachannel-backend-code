CREATE TABLE IF NOT EXISTS tags (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID      NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    color      VARCHAR(255),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_tags_tenant ON tags(tenant_id);
