CREATE TABLE IF NOT EXISTS tenants (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    domain     VARCHAR(255) NOT NULL UNIQUE,
    logo       JSONB,
    address    TEXT,
    phone      VARCHAR(20),
    email      VARCHAR(255),
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    settings   JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_domain ON tenants(domain);
