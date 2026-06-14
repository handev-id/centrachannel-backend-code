ALTER TABLE campaigns
  DROP CONSTRAINT IF EXISTS fk_campaigns_template,
  DROP CONSTRAINT IF EXISTS fk_campaigns_recipient_list;

DROP TABLE IF EXISTS campaign_recipients;
DROP TABLE IF EXISTS campaign_recipient_contacts;
DROP TABLE IF EXISTS campaign_recipient_lists;
DROP TABLE IF EXISTS campaign_templates;

CREATE TABLE campaign_contacts (
    id SERIAL PRIMARY KEY,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    contact_id INTEGER NOT NULL REFERENCES contacts(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    sent_at TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE campaigns
  DROP COLUMN IF EXISTS type,
  DROP COLUMN IF EXISTS sending_option,
  DROP COLUMN IF EXISTS stats,
  DROP COLUMN IF EXISTS agent_id,
  DROP COLUMN IF EXISTS sender_id,
  DROP COLUMN IF EXISTS recipient_list_id,
  DROP COLUMN IF EXISTS template_id;
