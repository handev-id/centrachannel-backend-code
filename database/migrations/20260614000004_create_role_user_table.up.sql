CREATE TABLE IF NOT EXISTS role_user (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, user_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_role_user_tenant_id ON role_user(tenant_id);
CREATE INDEX IF NOT EXISTS idx_role_user_user_id ON role_user(user_id);
CREATE INDEX IF NOT EXISTS idx_role_user_role_id ON role_user(role_id);
