CREATE TABLE IF NOT EXISTS domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(253) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT domains_name_normalized CHECK (name <> '' AND name = lower(btrim(name))),
    CONSTRAINT domains_default_active CHECK (NOT is_default OR is_active)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_domains_name ON domains (lower(name));
CREATE INDEX IF NOT EXISTS idx_domains_tenant_id ON domains (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_domains_default_per_tenant ON domains (tenant_id) WHERE is_default;
