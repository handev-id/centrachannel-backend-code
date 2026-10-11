CREATE TABLE IF NOT EXISTS tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    logo       JSONB,
    address    TEXT,
    phone      VARCHAR(20),
    email      VARCHAR(255),
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    settings   JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
