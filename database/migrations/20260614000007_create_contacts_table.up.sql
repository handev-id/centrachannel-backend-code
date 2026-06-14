CREATE TABLE IF NOT EXISTS contacts (
    id                   SERIAL PRIMARY KEY,
    tenant_id            INTEGER      NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    first_name           VARCHAR(255) NOT NULL,
    last_name            VARCHAR(255),
    username             VARCHAR(255),
    email                VARCHAR(255),
    phone                VARCHAR(255),
    avatar               JSONB,
    country              VARCHAR(255),
    bio                  TEXT,
    occupation           VARCHAR(255),
    category             VARCHAR(255),
    category_description TEXT,
    gender               VARCHAR(255),
    date_of_birth        DATE,
    province_of_origin   VARCHAR(255),
    facebook             VARCHAR(255),
    instagram            VARCHAR(255),
    whatsapp             VARCHAR(255),
    x                    VARCHAR(255),
    tiktok               VARCHAR(255),
    status               TEXT         NOT NULL DEFAULT 'individual',
    institution_name     VARCHAR(255),
    merged_to_id         INTEGER      REFERENCES contacts(id) ON DELETE SET NULL,
    deleted_at           TIMESTAMPTZ,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_contacts_status CHECK (status IN ('individual', 'institution'))
);

CREATE INDEX IF NOT EXISTS idx_contacts_tenant ON contacts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_contacts_deleted_at ON contacts(deleted_at);
