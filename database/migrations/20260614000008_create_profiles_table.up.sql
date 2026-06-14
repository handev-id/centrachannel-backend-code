CREATE TABLE IF NOT EXISTS profiles (
    id                         SERIAL PRIMARY KEY,
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
    updated_at                 TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_profiles_channel_external ON profiles(channel_id, external_id);
CREATE INDEX IF NOT EXISTS idx_profiles_contact ON profiles(contact_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_unique_active
    ON profiles(channel_id, external_id) WHERE deleted_at IS NULL;
