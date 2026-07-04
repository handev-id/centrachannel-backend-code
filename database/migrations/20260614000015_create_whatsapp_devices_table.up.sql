CREATE TABLE IF NOT EXISTS whatsapp_devices (
    id            SERIAL PRIMARY KEY,
    tenant_id     INTEGER      NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name          VARCHAR(255) NOT NULL,
    country_code  VARCHAR(255) NOT NULL DEFAULT '62',
    phone         VARCHAR(255) NOT NULL,
    whatsapp_id   VARCHAR(255) NOT NULL,
    status        TEXT         NOT NULL DEFAULT 'DISCONNECTED',
    created_at    TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ,
    UNIQUE(tenant_id, whatsapp_id),
    CONSTRAINT chk_whatsapp_devices_status CHECK (status IN ('CONNECTED', 'DISCONNECTED', 'CONNECTING'))
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_devices_tenant ON whatsapp_devices(tenant_id);
