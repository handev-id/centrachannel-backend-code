ALTER TABLE campaigns
  ADD COLUMN IF NOT EXISTS type VARCHAR(255) NOT NULL DEFAULT 'broadcast',
  ADD COLUMN IF NOT EXISTS sending_option JSONB,
  ADD COLUMN IF NOT EXISTS stats JSONB,
  ADD COLUMN IF NOT EXISTS agent_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS sender_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS recipient_list_id INTEGER,
  ADD COLUMN IF NOT EXISTS template_id INTEGER;

DROP TABLE IF EXISTS campaign_contacts;

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

ALTER TABLE campaigns
  ADD CONSTRAINT fk_campaigns_recipient_list FOREIGN KEY (recipient_list_id) REFERENCES campaign_recipient_lists(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_campaigns_template FOREIGN KEY (template_id) REFERENCES campaign_templates(id) ON DELETE SET NULL;
