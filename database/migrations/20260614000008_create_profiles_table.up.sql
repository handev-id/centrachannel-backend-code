CREATE TABLE IF NOT EXISTS profiles (
    id                         SERIAL PRIMARY KEY,
    tenant_id                  INTEGER      NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    external_id                VARCHAR(255) NOT NULL,
    username                   VARCHAR(255),
    display_name               VARCHAR(255),
    is_main                    BOOLEAN      NOT NULL DEFAULT TRUE,
    linked_device_whatsapp_id  VARCHAR(255),
    merged_from_contact_id     INTEGER      REFERENCES contacts(id) ON DELETE SET NULL,
    contact_id                 INTEGER      NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    channel_id                 INTEGER      NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    deleted_at                 TIMESTAMPTZ,
    created_at                 TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT profiles_tenant_contact_fkey
        FOREIGN KEY (tenant_id, contact_id) REFERENCES contacts(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_profiles_channel_external ON profiles(channel_id, external_id);
CREATE INDEX IF NOT EXISTS idx_profiles_contact ON profiles(contact_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_unique_active
    ON profiles(tenant_id, channel_id, external_id) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_profiles_tenant_contact ON profiles(tenant_id, contact_id);
