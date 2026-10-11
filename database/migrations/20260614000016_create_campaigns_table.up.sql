CREATE TABLE IF NOT EXISTS campaign_templates (
    id             SERIAL PRIMARY KEY,
    tenant_id      INTEGER      NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name           VARCHAR(255) NOT NULL,
    type           VARCHAR(255),
    template_type  VARCHAR(255),
    category       VARCHAR(255),
    language       VARCHAR(255),
    content        JSONB        NOT NULL,
    variables      JSONB        NOT NULL DEFAULT '[]',
    quality        VARCHAR(255),
    account_id     INTEGER,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_templates_tenant ON campaign_templates(tenant_id);

CREATE TABLE IF NOT EXISTS campaign_recipient_lists (
    id          SERIAL PRIMARY KEY,
    tenant_id   INTEGER      NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    source      VARCHAR(255) NOT NULL DEFAULT 'manual',
    status      VARCHAR(255),
    channel_id  INTEGER REFERENCES channels(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_recipient_lists_tenant ON campaign_recipient_lists(tenant_id);

CREATE TABLE IF NOT EXISTS campaign_recipient_contacts (
    id                          SERIAL PRIMARY KEY,
    first_name                  VARCHAR(255) NOT NULL,
    last_name                   VARCHAR(255),
    username                    VARCHAR(255),
    institution                 VARCHAR(255),
    email                       VARCHAR(255),
    phone                       VARCHAR(255),
    campaign_recipient_list_id  INTEGER NOT NULL REFERENCES campaign_recipient_lists(id) ON DELETE CASCADE,
    master_contact_id           INTEGER REFERENCES contacts(id) ON DELETE SET NULL,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_campaign_recipient_contacts_list ON campaign_recipient_contacts(campaign_recipient_list_id);

CREATE TABLE campaigns (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(255) NOT NULL DEFAULT 'broadcast',
    description TEXT,
    message_template TEXT NOT NULL,
    channel_id INTEGER NOT NULL REFERENCES channels(id),
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    sending_option JSONB,
    stats JSONB,
    scheduled_at TIMESTAMP,
    sent_count INTEGER NOT NULL DEFAULT 0,
    total_count INTEGER NOT NULL DEFAULT 0,
    agent_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    sender_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    recipient_list_id INTEGER,
    template_id INTEGER,
    created_by INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_campaigns_recipient_list FOREIGN KEY (recipient_list_id) REFERENCES campaign_recipient_lists(id) ON DELETE SET NULL,
    CONSTRAINT fk_campaigns_template FOREIGN KEY (template_id) REFERENCES campaign_templates(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS campaign_recipients (
    id                  SERIAL PRIMARY KEY,
    campaign_id         INTEGER      NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    recipient_contact_id INTEGER     NOT NULL REFERENCES campaign_recipient_contacts(id) ON DELETE CASCADE,
    status              TEXT         NOT NULL DEFAULT 'pending',
    failed_reason       VARCHAR(255),
    delivery_time       TIMESTAMPTZ,
    open_time           TIMESTAMPTZ,
    click_time          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_campaign_recipients_status CHECK (status IN ('pending', 'delivered', 'failed', 'read'))
);

CREATE INDEX IF NOT EXISTS idx_campaign_recipients_campaign ON campaign_recipients(campaign_id);
